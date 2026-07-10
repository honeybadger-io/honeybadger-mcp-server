package hbmcp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

var testSets = []struct {
	name    string
	tokens  int
	content string
}{
	{"badgerql", 7593, "# BadgerQL Reference\n\nstats count() as count"},
	{"queries", 1525, "# Insights queries\n\nStreams and time ranges."},
	{"charts", 1645, "# Charts\n\nchart_config fields per view."},
	{"dashboards", 1224, "# Dashboards\n\nWidget structure and grid."},
	{"alarms", 999, "# Alarms\n\ntrigger_config and states."},
	{"errors", 2108, "# Errors\n\nFaults, notices, and search syntax."},
}

// newDocsServer serves a fake docs site: index.json plus one .txt per set,
// with ETag support. hits counts requests per path.
func newDocsServer(t *testing.T, hits *map[string]*atomic.Int64, fail *atomic.Bool) *httptest.Server {
	t.Helper()
	counters := make(map[string]*atomic.Int64)
	mux := http.NewServeMux()

	serve := func(path, etag, body string) {
		c := &atomic.Int64{}
		counters[path] = c
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			c.Add(1)
			if fail != nil && fail.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("ETag", etag)
			_, _ = w.Write([]byte(body))
		})
	}

	var entries []string
	for _, s := range testSets {
		entries = append(entries, fmt.Sprintf(`{"name":%q,"title":%q,"description":"About %s","approx_tokens":%d,"url":"ignored","page":"ignored"}`, s.name, s.name, s.name, s.tokens))
		serve("/instructions/"+s.name+".txt", `W/"`+s.name+`-v1"`, s.content)
	}
	serve("/instructions/index.json", `W/"index-v1"`, `{"instructions":[`+strings.Join(entries, ",")+`]}`)

	if hits != nil {
		*hits = counters
	}
	return httptest.NewServer(mux)
}

func testFetcher(baseURL string) *referenceFetcher {
	return newReferenceFetcher(baseURL+"/instructions", slog.New(slog.DiscardHandler))
}

func referenceRequest(topics ...string) mcp.CallToolRequest {
	args := map[string]interface{}{}
	if topics != nil {
		list := make([]interface{}, len(topics))
		for i, t := range topics {
			list[i] = t
		}
		args["topics"] = list
	}
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: args},
	}
}

func TestHandleGetReference_Index(t *testing.T) {
	server := newDocsServer(t, nil, nil)
	defer server.Close()
	f := testFetcher(server.URL)

	result, err := handleGetReference(context.Background(), f, referenceRequest())
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	resultText := getResultText(result)
	for _, s := range testSets {
		if !strings.Contains(resultText, s.name) {
			t.Errorf("index should list topic %q", s.name)
		}
		if strings.Contains(resultText, s.content) {
			t.Errorf("index should not include full content of topic %q", s.name)
		}
	}
	if !strings.Contains(resultText, "7593 tokens") {
		t.Error("index should include token counts from index.json")
	}
}

func TestHandleGetReference_ComposeStableOrder(t *testing.T) {
	server := newDocsServer(t, nil, nil)
	defer server.Close()
	f := testFetcher(server.URL)

	// Request in reverse order; composition must follow index.json order.
	result, err := handleGetReference(context.Background(), f, referenceRequest("charts", "badgerql"))
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	resultText := getResultText(result)
	badgerqlPos := strings.Index(resultText, "# BadgerQL Reference")
	chartsPos := strings.Index(resultText, "# Charts")
	if badgerqlPos == -1 || chartsPos == -1 {
		t.Fatal("expected both topic contents in result")
	}
	if badgerqlPos > chartsPos {
		t.Error("badgerql should precede charts regardless of request order")
	}
	if strings.Contains(resultText, "# Alarms") {
		t.Error("unrequested topics should not be included")
	}
}

func TestHandleGetReference_All(t *testing.T) {
	server := newDocsServer(t, nil, nil)
	defer server.Close()
	f := testFetcher(server.URL)

	result, err := handleGetReference(context.Background(), f, referenceRequest("all"))
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	resultText := getResultText(result)
	for _, s := range testSets {
		if !strings.Contains(resultText, s.content) {
			t.Errorf("\"all\" should include full content of topic %q", s.name)
		}
	}
}

func TestHandleGetReference_DuplicateTopics(t *testing.T) {
	server := newDocsServer(t, nil, nil)
	defer server.Close()
	f := testFetcher(server.URL)

	result, err := handleGetReference(context.Background(), f, referenceRequest("alarms", "alarms"))
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if strings.Count(getResultText(result), "# Alarms") != 1 {
		t.Error("duplicate topic names should be deduplicated")
	}
}

func TestHandleGetReference_UnknownTopic(t *testing.T) {
	server := newDocsServer(t, nil, nil)
	defer server.Close()
	f := testFetcher(server.URL)

	result, err := handleGetReference(context.Background(), f, referenceRequest("badgerql", "nonsense"))
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for unknown topic")
	}
	resultText := getResultText(result)
	if !strings.Contains(resultText, "nonsense") {
		t.Error("error should name the unknown topic")
	}
	if !strings.Contains(resultText, "badgerql") {
		t.Error("error should list valid topic names")
	}
}

func TestReferenceFetcher_CachesWithinTTL(t *testing.T) {
	var hits map[string]*atomic.Int64
	server := newDocsServer(t, &hits, nil)
	defer server.Close()
	f := testFetcher(server.URL)

	for i := 0; i < 3; i++ {
		if _, err := handleGetReference(context.Background(), f, referenceRequest("badgerql")); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}

	if n := hits["/instructions/badgerql.txt"].Load(); n != 1 {
		t.Errorf("expected 1 fetch of badgerql.txt within TTL, got %d", n)
	}
	if n := hits["/instructions/index.json"].Load(); n != 1 {
		t.Errorf("expected 1 fetch of index.json within TTL, got %d", n)
	}
}

func TestReferenceFetcher_RevalidatesWithETag(t *testing.T) {
	var hits map[string]*atomic.Int64
	server := newDocsServer(t, &hits, nil)
	defer server.Close()
	f := testFetcher(server.URL)
	f.ttl = 0 // every call revalidates

	for i := 0; i < 2; i++ {
		result, err := handleGetReference(context.Background(), f, referenceRequest("badgerql"))
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if result.IsError {
			t.Fatalf("call %d returned tool error: %s", i, getResultText(result))
		}
		if !strings.Contains(getResultText(result), "# BadgerQL Reference") {
			t.Fatalf("call %d: content missing after revalidation", i)
		}
	}

	// Second round hits the server again (TTL 0) but gets 304s.
	if n := hits["/instructions/badgerql.txt"].Load(); n != 2 {
		t.Errorf("expected 2 requests (initial + revalidation), got %d", n)
	}
}

func TestReferenceFetcher_ServesStaleOnRefreshFailure(t *testing.T) {
	var fail atomic.Bool
	server := newDocsServer(t, nil, &fail)
	defer server.Close()
	f := testFetcher(server.URL)

	// Prime the cache, then break the docs site and expire the cache.
	if result, err := handleGetReference(context.Background(), f, referenceRequest("badgerql")); err != nil || result.IsError {
		t.Fatalf("priming call failed: err=%v result=%s", err, getResultText(result))
	}
	fail.Store(true)
	f.ttl = 0

	result, err := handleGetReference(context.Background(), f, referenceRequest("badgerql"))
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected stale content, got tool error: %s", getResultText(result))
	}
	if !strings.Contains(getResultText(result), "# BadgerQL Reference") {
		t.Error("stale cached content should be served when refresh fails")
	}
}

func TestReferenceFetcher_ColdCacheFailureIsError(t *testing.T) {
	var fail atomic.Bool
	fail.Store(true)
	server := newDocsServer(t, nil, &fail)
	defer server.Close()
	f := testFetcher(server.URL)

	result, err := handleGetReference(context.Background(), f, referenceRequest("badgerql"))
	if err != nil {
		t.Fatalf("handleGetReference() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error when docs site is unreachable and cache is cold")
	}
}

func TestServerInstructions(t *testing.T) {
	instructions := ServerInstructions()
	for _, name := range referenceTopicNames {
		if !strings.Contains(instructions, name) {
			t.Errorf("instructions should mention topic %q", name)
		}
	}
	if !strings.Contains(instructions, "get_reference") {
		t.Error("instructions should mention the get_reference tool")
	}
	// Instructions are always-on context for every session — keep them small.
	if len(instructions) > 2500 {
		t.Errorf("instructions too long: %d bytes", len(instructions))
	}
}
