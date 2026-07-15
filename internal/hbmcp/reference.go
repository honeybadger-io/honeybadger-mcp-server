package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// referenceTopicNames is the tool-contract copy of the topic list: it appears
// in tool descriptions and server instructions, which are rendered before any
// network call is possible. The docs site's index.json is authoritative at
// request time — validation and composition order come from it, so a set
// added to the docs is servable before it's named here.
var referenceTopicNames = []string{"badgerql", "queries", "charts", "dashboards", "alarms", "errors"}

// instructionSet mirrors one entry of the docs site's index.json.
type instructionSet struct {
	Name         string `json:"name"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ApproxTokens int    `json:"approx_tokens"`
}

type instructionIndex struct {
	Instructions []instructionSet `json:"instructions"`
}

type cacheEntry struct {
	body      string
	etag      string
	fetchedAt time.Time
}

// referenceFetcher pulls reference content from the docs site with an
// in-memory cache. There is deliberately no embedded fallback: the docs site
// is the single source of truth, and a cold-cache fetch failure surfaces as a
// tool error rather than silently serving content that may have drifted.
type referenceFetcher struct {
	baseURL string
	ttl     time.Duration
	client  *http.Client
	logger  *slog.Logger

	mu      sync.Mutex
	entries map[string]*cacheEntry
	flights map[string]chan struct{}
}

func newReferenceFetcher(baseURL string, logger *slog.Logger) *referenceFetcher {
	return &referenceFetcher{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		ttl:     5 * time.Minute,
		client:  &http.Client{Timeout: 5 * time.Second},
		logger:  logger,
		entries: make(map[string]*cacheEntry),
		flights: make(map[string]chan struct{}),
	}
}

// get returns the body served at baseURL/<path>, from cache when fresh.
// Expired entries are revalidated with If-None-Match; if the refresh fails
// the stale copy is served (and the failure logged) so a docs-site blip
// doesn't break tool calls mid-session. Only a cold cache returns an error.
func (f *referenceFetcher) get(ctx context.Context, path string) (string, error) {
	for {
		f.mu.Lock()
		if e, ok := f.entries[path]; ok && time.Since(e.fetchedAt) < f.ttl {
			body := e.body
			f.mu.Unlock()
			return body, nil
		}
		if ch, ok := f.flights[path]; ok {
			// Another call is already fetching this path; wait and re-check.
			f.mu.Unlock()
			select {
			case <-ch:
				continue
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		ch := make(chan struct{})
		f.flights[path] = ch
		stale := f.entries[path]
		f.mu.Unlock()

		body, etag, status, err := f.fetch(ctx, path, stale)

		f.mu.Lock()
		delete(f.flights, path)
		close(ch)
		now := time.Now()
		switch {
		case err == nil && status == http.StatusOK:
			f.entries[path] = &cacheEntry{body: body, etag: etag, fetchedAt: now}
			f.mu.Unlock()
			return body, nil
		case err == nil && status == http.StatusNotModified && stale != nil:
			stale.fetchedAt = now
			f.mu.Unlock()
			return stale.body, nil
		default:
			if err == nil {
				err = fmt.Errorf("unexpected status %d", status)
			}
			if stale != nil {
				stale.fetchedAt = now
				f.mu.Unlock()
				f.logger.Warn("Reference refresh failed, serving stale copy", "path", path, "error", err)
				return stale.body, nil
			}
			f.mu.Unlock()
			return "", fmt.Errorf("failed to fetch reference from %s/%s: %w", f.baseURL, path, err)
		}
	}
}

func (f *referenceFetcher) fetch(ctx context.Context, path string, stale *cacheEntry) (body, etag string, status int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.baseURL+"/"+path, nil)
	if err != nil {
		return "", "", 0, err
	}
	if stale != nil && stale.etag != "" {
		req.Header.Set("If-None-Match", stale.etag)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotModified {
		return "", "", resp.StatusCode, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", resp.StatusCode, nil
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", 0, err
	}
	return string(bodyBytes), resp.Header.Get("ETag"), resp.StatusCode, nil
}

func (f *referenceFetcher) index(ctx context.Context) (*instructionIndex, error) {
	body, err := f.get(ctx, "index.json")
	if err != nil {
		return nil, err
	}
	var idx instructionIndex
	if err := json.Unmarshal([]byte(body), &idx); err != nil {
		return nil, fmt.Errorf("invalid reference index: %w", err)
	}
	if len(idx.Instructions) == 0 {
		return nil, fmt.Errorf("reference index at %s/index.json lists no instruction sets", f.baseURL)
	}
	return &idx, nil
}

// RegisterReferenceTools registers the get_reference documentation tool
func RegisterReferenceTools(r *toolRegistrar, fetcher *referenceFetcher) {
	r.AddTool(
		mcp.NewTool("get_reference",
			mcp.WithTitleAnnotation("Get Reference Documentation"),
			mcp.WithDescription("Returns Honeybadger reference documentation by topic: badgerql (query language), queries (Insights query fundamentals: streams, time ranges, field grounding), charts (visualization views, chart_config), dashboards (widgets, layout), alarms (trigger_config, states, patterns), errors (fault/notice model, error search syntax). Fetch all topics you need in one call, e.g. topics: [\"badgerql\", \"charts\"]; skip topics still visible in your context. Call with no arguments for a topic index, or [\"all\"] for everything."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithArray("topics",
				mcp.Description("Reference topics to fetch: badgerql, queries, charts, dashboards, alarms, errors, or all. Omit for an index of topics."),
				mcp.WithStringItems(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetReference(ctx, fetcher, req)
		},
	)
}

func handleGetReference(ctx context.Context, f *referenceFetcher, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idx, err := f.index(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	requested := req.GetStringSlice("topics", nil)
	if len(requested) == 0 {
		return mcp.NewToolResultText(renderIndex(idx)), nil
	}

	want := make(map[string]bool, len(requested))
	for _, name := range requested {
		if name == "all" {
			for _, set := range idx.Instructions {
				want[set.Name] = true
			}
			continue
		}
		valid := false
		for _, set := range idx.Instructions {
			if set.Name == name {
				valid = true
				break
			}
		}
		if !valid {
			var names []string
			for _, set := range idx.Instructions {
				names = append(names, set.Name)
			}
			return mcp.NewToolResultError(fmt.Sprintf("unknown topic %q; valid topics: %s, or \"all\"", name, strings.Join(names, ", "))), nil
		}
		want[name] = true
	}

	// Compose in index order so output is stable regardless of request order.
	var parts []string
	for _, set := range idx.Instructions {
		if !want[set.Name] {
			continue
		}
		content, err := f.get(ctx, set.Name+".txt")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		parts = append(parts, content)
	}
	return mcp.NewToolResultText(strings.Join(parts, "\n\n---\n\n")), nil
}

func renderIndex(idx *instructionIndex) string {
	var sb strings.Builder
	sb.WriteString("Honeybadger reference topics. Fetch the ones you need in a single call: get_reference with topics: [\"name\", ...]. Use [\"all\"] for everything. Skip any topic whose content is still visible in your context.\n")
	for _, set := range idx.Instructions {
		fmt.Fprintf(&sb, "\n- %s (~%d tokens) — %s", set.Name, set.ApproxTokens, set.Description)
	}
	return sb.String()
}

// ServerInstructions is the always-on context sent to clients at session
// init: the reference topic map, the fetch-once rule, and the cross-tool
// fundamentals. Keep it small — every session pays for it. Interface facts
// belong in reference topics, not here; procedural methodology belongs in
// client-side skills, not the server.
func ServerInstructions() string {
	var sb strings.Builder
	sb.WriteString("Honeybadger provides error tracking, uptime and check-in (cron/scheduled task) monitoring, and Insights (log/event analytics queried with BadgerQL).\n\n")
	sb.WriteString("Reference documentation is split into non-overlapping topics served by the get_reference tool: ")
	sb.WriteString(strings.Join(referenceTopicNames, ", "))
	sb.WriteString(". Tool descriptions state which topics they require. Fetch required topics in a single get_reference call before using those tools, and never re-fetch a topic whose content is still visible in your context. Call get_reference with no arguments for the topic index.\n\n")
	sb.WriteString("Fundamentals: discover before you analyze — resolve projects with list_projects, verify data exists with a minimal count query, and preview event fields before writing analytical queries. Build queries incrementally with narrow time ranges and small limits, then widen. Verify a query returns the expected shape via query_insights before embedding it in a dashboard widget or alarm trigger. After running queries or creating dashboards and alarms, give the user Honeybadger UI links to the results.")
	return sb.String()
}
