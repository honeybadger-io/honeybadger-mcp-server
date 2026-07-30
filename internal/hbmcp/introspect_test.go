package hbmcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/honeybadger-io/api-go/apiv3"
)

// countingFetcher records how often the API was actually asked.
func countingFetcher(info *apiv3.TokenInfo, err error) (TokenInfoFetcher, *int) {
	calls := 0
	return func(ctx context.Context, token string) (*apiv3.TokenInfo, error) {
		calls++
		return info, err
	}, &calls
}

// The whole point: repeated requests with the same credential must not each ask
// the API.
func TestIntrospectionCacheServesRepeatCalls(t *testing.T) {
	fetch, calls := countingFetcher(&apiv3.TokenInfo{AccountID: "Ab3kL9"}, nil)
	c := NewIntrospectionCache(fetch, 0, 0, 0)

	for i := 0; i < 10; i++ {
		info, err := c.Get(context.Background(), "hbt_abc")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if info.AccountID != "Ab3kL9" {
			t.Fatalf("AccountID = %q", info.AccountID)
		}
	}
	if *calls != 1 {
		t.Errorf("fetched %d times, want 1", *calls)
	}
}

// Different credentials must never share an entry.
func TestIntrospectionCacheSeparatesCredentials(t *testing.T) {
	var asked []string
	c := NewIntrospectionCache(func(ctx context.Context, token string) (*apiv3.TokenInfo, error) {
		asked = append(asked, token)
		return &apiv3.TokenInfo{AccountID: "acct_for_" + token}, nil
	}, 0, 0, 0)

	first, _ := c.Get(context.Background(), "hbt_one")
	second, _ := c.Get(context.Background(), "hbt_two")

	if first.AccountID == second.AccountID {
		t.Fatal("two credentials shared a cache entry")
	}
	if len(asked) != 2 {
		t.Errorf("asked %v, want both credentials fetched", asked)
	}
}

// A stale entry must be refetched, so a revoked scope stops applying.
func TestIntrospectionCacheExpires(t *testing.T) {
	fetch, calls := countingFetcher(&apiv3.TokenInfo{AccountID: "Ab3kL9"}, nil)
	c := NewIntrospectionCache(fetch, 60*time.Second, 0, 0)

	base := time.Now()
	c.now = func() time.Time { return base }

	if _, err := c.Get(context.Background(), "hbt_abc"); err != nil {
		t.Fatal(err)
	}
	c.now = func() time.Time { return base.Add(61 * time.Second) }
	if _, err := c.Get(context.Background(), "hbt_abc"); err != nil {
		t.Fatal(err)
	}

	if *calls != 2 {
		t.Errorf("fetched %d times, want 2 — the entry should have expired", *calls)
	}
}

// A rejected credential is cached only briefly, so fixing it takes effect
// quickly, but a flood of bad tokens still cannot be amplified upstream.
func TestIntrospectionCacheNegativeTTLIsShorter(t *testing.T) {
	boom := errors.New("unauthorized")
	fetch, calls := countingFetcher(nil, boom)
	c := NewIntrospectionCache(fetch, 60*time.Second, 5*time.Second, 0)

	base := time.Now()
	c.now = func() time.Time { return base }

	if _, err := c.Get(context.Background(), "hbt_bad"); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the fetch error", err)
	}
	// Still inside the negative window: no second call.
	c.now = func() time.Time { return base.Add(3 * time.Second) }
	if _, err := c.Get(context.Background(), "hbt_bad"); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if *calls != 1 {
		t.Errorf("fetched %d times inside the negative window, want 1", *calls)
	}

	// Past it, and well before the positive TTL, it asks again.
	c.now = func() time.Time { return base.Add(6 * time.Second) }
	if _, err := c.Get(context.Background(), "hbt_bad"); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if *calls != 2 {
		t.Errorf("fetched %d times, want 2 after the negative window", *calls)
	}
}

// Unique tokens must not grow the cache without limit — that would be a memory
// exhaustion vector rather than a cache.
func TestIntrospectionCacheIsBounded(t *testing.T) {
	c := NewIntrospectionCache(func(ctx context.Context, token string) (*apiv3.TokenInfo, error) {
		return &apiv3.TokenInfo{AccountID: token}, nil
	}, 0, 0, 8)

	for i := 0; i < 100; i++ {
		if _, err := c.Get(context.Background(), fmt.Sprintf("hbt_%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if got := c.Len(); got > 8 {
		t.Errorf("cache holds %d entries, want at most 8", got)
	}
}

// The cache key must not be the credential itself.
func TestIntrospectionCacheDoesNotRetainTheToken(t *testing.T) {
	c := NewIntrospectionCache(func(ctx context.Context, token string) (*apiv3.TokenInfo, error) {
		return &apiv3.TokenInfo{}, nil
	}, 0, 0, 0)

	const secret = "hbt_super_secret_value"
	if _, err := c.Get(context.Background(), secret); err != nil {
		t.Fatal(err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.entries {
		if strings.Contains(key, "secret") || key == secret {
			t.Fatalf("cache key %q contains the credential", key)
		}
		if len(key) != 64 {
			t.Errorf("key %q is not a sha256 digest", key)
		}
	}
}

// An empty credential is not something to ask the API about.
func TestIntrospectionCacheIgnoresEmptyToken(t *testing.T) {
	fetch, calls := countingFetcher(&apiv3.TokenInfo{}, nil)
	c := NewIntrospectionCache(fetch, 0, 0, 0)

	info, err := c.Get(context.Background(), "")
	if info != nil || err != nil {
		t.Errorf("got (%v, %v), want (nil, nil)", info, err)
	}
	if *calls != 0 {
		t.Errorf("fetched %d times for an empty credential, want 0", *calls)
	}
}

// Hosted means concurrent. The cache must survive parallel use under -race.
func TestIntrospectionCacheConcurrentUse(t *testing.T) {
	c := NewIntrospectionCache(func(ctx context.Context, token string) (*apiv3.TokenInfo, error) {
		return &apiv3.TokenInfo{AccountID: token}, nil
	}, 0, 0, 64)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			token := fmt.Sprintf("hbt_%d", i%10)
			for j := 0; j < 20; j++ {
				info, err := c.Get(context.Background(), token)
				if err != nil {
					t.Errorf("Get: %v", err)
					return
				}
				if info.AccountID != token {
					t.Errorf("got %q for %q — entries crossed", info.AccountID, token)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}
