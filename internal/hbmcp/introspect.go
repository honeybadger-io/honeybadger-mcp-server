package hbmcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/honeybadger-io/api-go/apiv3"
)

// TokenInfoFetcher describes a credential by asking the API. In production this
// is apiv3's Tokens.Get; tests substitute their own.
type TokenInfoFetcher func(ctx context.Context, token string) (*apiv3.TokenInfo, error)

// Defaults for NewIntrospectionCache.
const (
	// DefaultIntrospectionTTL is short on purpose. The cache exists to stop one
	// API call per request, not to hold authorization state — a revoked scope
	// should stop mattering in seconds.
	DefaultIntrospectionTTL = 60 * time.Second

	// DefaultNegativeTTL is shorter still: a rejected credential is usually a
	// user fixing their configuration, and they should not wait a minute to see
	// it work. It is non-zero so a flood of bad tokens cannot be turned into an
	// equal flood of upstream calls.
	DefaultNegativeTTL = 5 * time.Second

	// DefaultIntrospectionSize bounds the cache. Without a bound, a caller
	// sending unique tokens could grow it without limit, which is a memory
	// exhaustion vector rather than a caching strategy.
	DefaultIntrospectionSize = 2048
)

type introspectionEntry struct {
	info      *apiv3.TokenInfo
	err       error
	expiresAt time.Time
	// lastUsed drives eviction; the newest entries survive.
	lastUsed time.Time
}

// inFlight is a fetch in progress. Callers arriving for the same credential wait
// on it rather than starting their own.
type inFlight struct {
	done chan struct{}
	info *apiv3.TokenInfo
	err  error
}

// IntrospectionCache remembers what a credential permits, briefly.
//
// It is a cache, not session state: losing it costs one API call and never
// changes an outcome, because the API remains the only thing that authorizes a
// request. That is what keeps the hosted server stateless in the sense that
// matters — no request depends on state a previous request created.
//
// Credentials are keyed by SHA-256 digest, so the cache never holds a token it
// could leak, and two different tokens cannot share an entry.
type IntrospectionCache struct {
	fetch       TokenInfoFetcher
	ttl         time.Duration
	negativeTTL time.Duration
	maxEntries  int

	// now is swappable so tests can advance time instead of sleeping.
	now func() time.Time

	mu       sync.Mutex
	entries  map[string]*introspectionEntry
	inflight map[string]*inFlight
}

// NewIntrospectionCache returns a cache with the given fetcher. Zero values for
// ttl, negativeTTL, or maxEntries take the package defaults.
func NewIntrospectionCache(fetch TokenInfoFetcher, ttl, negativeTTL time.Duration, maxEntries int) *IntrospectionCache {
	if ttl <= 0 {
		ttl = DefaultIntrospectionTTL
	}
	if negativeTTL <= 0 {
		negativeTTL = DefaultNegativeTTL
	}
	if maxEntries <= 0 {
		maxEntries = DefaultIntrospectionSize
	}
	return &IntrospectionCache{
		fetch:       fetch,
		ttl:         ttl,
		negativeTTL: negativeTTL,
		maxEntries:  maxEntries,
		now:         time.Now,
		entries:     make(map[string]*introspectionEntry),
		inflight:    make(map[string]*inFlight),
	}
}

// Get describes the credential, from cache when it is fresh.
//
// Failures are returned as they arrive and cached only briefly, so a caller can
// tell a rejected credential from a working one without this cache deciding
// anything on its own.
func (c *IntrospectionCache) Get(ctx context.Context, token string) (*apiv3.TokenInfo, error) {
	if token == "" {
		return nil, nil
	}
	key := digest(token)

	// One fetch per credential, however many callers arrive at once.
	//
	// Without this, a burst of requests carrying the same uncached token issues one
	// upstream call each — and at capacity each completion also evicts an unrelated
	// entry, pushing other tenants into misses and amplifying the burst further.
	info, err, hit, waiter := c.claim(key)
	if hit {
		return info, err
	}
	if waiter != nil {
		<-waiter.done
		return waiter.info, waiter.err
	}
	call := c.inFlightFor(key)

	info, err = c.fetch(ctx, token)
	c.store(key, info, err)

	c.mu.Lock()
	delete(c.inflight, key)
	c.mu.Unlock()

	call.info, call.err = info, err
	close(call.done)
	return info, err
}

// claim decides, under one lock, which of three things happens: the cache answers,
// the caller waits on a fetch already running, or the caller becomes the fetcher.
//
// Checking the cache and registering the fetch must be atomic. When they were two
// separate locked sections, a caller could miss the cache before the winner stored
// its result and then register a second fetch after the winner had cleaned up —
// which is exactly the duplicate call this exists to prevent.
func (c *IntrospectionCache) claim(key string) (info *apiv3.TokenInfo, err error, hit bool, waitOn *inFlight) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[key]; ok {
		if c.now().After(entry.expiresAt) {
			delete(c.entries, key)
		} else {
			entry.lastUsed = c.now()
			return entry.info, entry.err, true, nil
		}
	}
	if existing, ok := c.inflight[key]; ok {
		return nil, nil, false, existing
	}
	c.inflight[key] = &inFlight{done: make(chan struct{})}
	return nil, nil, false, nil
}

// inFlightFor returns the fetch this caller registered in claim.
func (c *IntrospectionCache) inFlightFor(key string) *inFlight {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inflight[key]
}

func (c *IntrospectionCache) store(key string, info *apiv3.TokenInfo, err error) {
	ttl := c.ttl
	if err != nil {
		ttl = c.negativeTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Only evict when adding a key the cache does not already hold. Refreshing an
	// existing entry must never cost another tenant theirs.
	if _, replacing := c.entries[key]; !replacing && len(c.entries) >= c.maxEntries {
		c.evictOldestLocked()
	}
	now := c.now()
	c.entries[key] = &introspectionEntry{
		info:      info,
		err:       err,
		expiresAt: now.Add(ttl),
		lastUsed:  now,
	}
}

// evictOldestLocked drops the least recently used entry. Called with mu held.
func (c *IntrospectionCache) evictOldestLocked() {
	var oldestKey string
	var oldest time.Time
	for k, e := range c.entries {
		if oldestKey == "" || e.lastUsed.Before(oldest) {
			oldestKey, oldest = k, e.lastUsed
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// Len reports how many entries are held. For tests and metrics.
func (c *IntrospectionCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// digest keys a credential without retaining it.
func digest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
