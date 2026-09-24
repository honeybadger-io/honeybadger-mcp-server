# MCP Server Usage Analytics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Emit one `mcp.tool_call` event to Honeybadger Insights per tool invocation on the hosted HTTP server, carrying tool name, outcome, duration, opaque customer identity, and upstream API status.

**Architecture:** A `Sink` interface in a new `internal/analytics` package owns the Honeybadger client and knows nothing about MCP. An `instrumenter` in `hbmcp` decorates every registered tool handler, timing the call and classifying its outcome. Because handlers convert upstream API failures into `IsError: true, nil`, outcome classification cannot rely on the handler's return value alone — a recording `http.RoundTripper` on the api-go client captures the real upstream status into a per-call record in the context. Everything is gated so stdio is untouched.

**Tech Stack:** Go 1.25.5, `honeybadger-go v0.9.0` (new dependency), `mcp-go v0.55.1`, `api-go v0.8.0`.

**Spec:** `docs/superpowers/specs/2026-08-06-mcp-usage-analytics-design.md`

## Global Constraints

- **Build and test with `GOWORK=off`.** A parent `go.work` pins a stale `../api-go` checkout. Every `go build` / `go test` / `go get` command in this plan must be prefixed with it.
- **stdio must remain byte-for-byte unchanged.** Analytics activates only when `TransportMode == config.TransportHTTP` **and** `HONEYBADGER_API_KEY` is non-empty.
- **Never reference package-level `honeybadger.Notify` / `Event` / `Configure` / `Flush`.** Only a client instance built inside `internal/analytics`. Ambient `HONEYBADGER_API_KEY` seeds honeybadger-go's `DefaultClient` at package init, so a stdio user's own key would otherwise send our telemetry to their project.
- **`hbmcp` must not import `honeybadger-go`.** All library types stay inside `internal/analytics`.
- **Never put argument *values* in events.** Argument names only, allowlisted against the tool's declared input schema.
- **Emitting must never fail a tool call.** `Sink.Emit` returns nothing; all errors are swallowed inside the sink.
- Existing tests must stay green with no API key set.

---

### Task 1: Extend `Claims` with identity fields

**Files:**
- Modify: `internal/hbmcp/claims.go`
- Test: `internal/hbmcp/claims_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `Claims` struct gains `Subject string`, `AccountID string`, `ClientID string`, `ProjectScoped bool`. Populated by the existing `ParseAccessToken(raw string, keyfunc jwt.Keyfunc, expectedIssuer, expectedAudience string) (*Claims, error)`.

**Background:** The Honeybadger authorization server mints this payload (`../honeybadger/config/initializers/doorkeeper.rb:114-129`):

```ruby
{ ver:, iss:, sub:, iat:, exp:, jti:, client_id:, scope:, account_id:, project_id:, aud: }.compact
```

`sub` and `account_id` are hashids (opaque). `project_id` is a raw database ID, so we record only whether it is present, never its value. The payload is `.compact`ed, so `account_id`, `client_id`, and `project_id` are absent on some tokens — their absence must never fail validation.

- [ ] **Step 1: Write the failing tests**

Add to `internal/hbmcp/claims_test.go`:

```go
func TestParseAccessToken_PopulatesIdentity(t *testing.T) {
	key, kf := testKey(t)
	claims := baseClaims()
	claims["sub"] = "user_abc123"
	claims["account_id"] = "acct_xyz789"
	claims["client_id"] = "app-uid-42"
	claims["project_id"] = float64(31337)
	raw := signedToken(t, key, claims)

	got, err := ParseAccessToken(raw, kf, "http://localhost:3001", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Subject != "user_abc123" {
		t.Errorf("Subject = %q, want %q", got.Subject, "user_abc123")
	}
	if got.AccountID != "acct_xyz789" {
		t.Errorf("AccountID = %q, want %q", got.AccountID, "acct_xyz789")
	}
	if got.ClientID != "app-uid-42" {
		t.Errorf("ClientID = %q, want %q", got.ClientID, "app-uid-42")
	}
	if !got.ProjectScoped {
		t.Error("ProjectScoped = false, want true")
	}
}

// The AS .compacts the payload, so optional claims are genuinely absent on
// some tokens. Their absence must leave fields empty, never fail validation.
func TestParseAccessToken_OptionalIdentityClaimsAbsent(t *testing.T) {
	key, kf := testKey(t)
	claims := baseClaims()
	claims["sub"] = "user_abc123"
	raw := signedToken(t, key, claims)

	got, err := ParseAccessToken(raw, kf, "http://localhost:3001", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AccountID != "" {
		t.Errorf("AccountID = %q, want empty", got.AccountID)
	}
	if got.ClientID != "" {
		t.Errorf("ClientID = %q, want empty", got.ClientID)
	}
	if got.ProjectScoped {
		t.Error("ProjectScoped = true, want false")
	}
	if !got.HasScope("read") {
		t.Error("scopes should still parse when identity claims are absent")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `GOWORK=off go test ./internal/hbmcp/ -run TestParseAccessToken -v`
Expected: FAIL — `got.Subject undefined (type *Claims has no field or method Subject)`

- [ ] **Step 3: Add the fields and parse them**

In `internal/hbmcp/claims.go`, replace the `Claims` struct:

```go
type Claims struct {
	Scopes []string
	// Subject and AccountID are hashids from the AS — opaque, not raw
	// database IDs. ProjectScoped records only whether the token carried a
	// project_id, because that claim (unlike the other two) is emitted raw.
	Subject       string
	AccountID     string
	ClientID      string
	ProjectScoped bool
}
```

Then replace the tail of `ParseAccessToken` (currently `scope, _ := mc["scope"].(string)` and the `return`):

```go
	mc, _ := tok.Claims.(jwt.MapClaims)
	scope, _ := mc["scope"].(string)
	sub, _ := mc["sub"].(string)
	accountID, _ := mc["account_id"].(string)
	clientID, _ := mc["client_id"].(string)
	_, projectScoped := mc["project_id"]
	return &Claims{
		Scopes:        strings.Fields(scope),
		Subject:       sub,
		AccountID:     accountID,
		ClientID:      clientID,
		ProjectScoped: projectScoped,
	}, nil
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `GOWORK=off go test ./internal/hbmcp/ -v -run TestParseAccessToken`
Expected: PASS, including the pre-existing `TestParseAccessToken_Success` and `TestParseAccessToken_MissingPrefix`.

- [ ] **Step 5: Commit**

```bash
git add internal/hbmcp/claims.go internal/hbmcp/claims_test.go
git commit -m "Parse identity claims from the access token

sub and account_id are hashids and safe to record as-is. project_id is
emitted raw by the AS, so only its presence is kept."
```

---

### Task 2: `internal/analytics` package

**Files:**
- Create: `internal/analytics/sink.go`
- Create: `internal/analytics/honeybadger.go`
- Test: `internal/analytics/honeybadger_test.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type Event struct { Type string; Data map[string]any }`
  - `type Notice struct { Err error; Fingerprint string; Context map[string]any }`
  - `type Sink interface { Emit(Event); Notify(Notice) string; Flush() }`
  - `func NewNopSink() Sink`
  - `func NewHoneybadgerSink(apiKey, env string) Sink`

**Design note:** `Notify` takes a `Notice` struct rather than honeybadger-go's variadic `extra ...any`. This keeps every honeybadger-go type inside this package, so `hbmcp` never imports the library — the mirror of "analytics knows nothing about MCP".

- [ ] **Step 1: Add the dependency**

```bash
GOWORK=off go get github.com/honeybadger-io/honeybadger-go@v0.9.0
```

- [ ] **Step 2: Write `sink.go`**

```go
// Package analytics owns the telemetry destination. It knows nothing about
// MCP, and it is the only package permitted to import honeybadger-go.
package analytics

// Event is one analytics event. Data must never contain customer data.
type Event struct {
	Type string
	Data map[string]any
}

// Notice is an error report. Fingerprint controls grouping; Context carries
// searchable metadata that is deliberately kept out of Event.
type Notice struct {
	Err         error
	Fingerprint string
	Context     map[string]any
}

// Sink is the telemetry destination.
//
// Emit returns nothing by design: a caller must never be able to fail a tool
// call by emitting. Notify returns the notice token for correlation, which is
// the only place it is obtainable.
type Sink interface {
	Emit(Event)
	Notify(Notice) string
	Flush()
}

type nopSink struct{}

func (nopSink) Emit(Event)         {}
func (nopSink) Notify(Notice) string { return "" }
func (nopSink) Flush()             {}

// NewNopSink returns a Sink that discards everything. Used whenever analytics
// is not activated, so no honeybadger client is ever constructed.
func NewNopSink() Sink { return nopSink{} }
```

- [ ] **Step 3: Write `honeybadger.go`**

```go
package analytics

import (
	honeybadger "github.com/honeybadger-io/honeybadger-go"
)

type hbSink struct {
	client *honeybadger.Client
}

// NewHoneybadgerSink builds a dedicated client. It never touches
// honeybadger.DefaultClient, which is seeded from ambient HONEYBADGER_API_KEY
// at package init and could otherwise carry a customer's own key.
func NewHoneybadgerSink(apiKey, env string) Sink {
	return newHoneybadgerSink(honeybadger.Configuration{
		APIKey: apiKey,
		Env:    env,
	})
}

func newHoneybadgerSink(cfg honeybadger.Configuration) *hbSink {
	return &hbSink{client: honeybadger.New(cfg)}
}

func (s *hbSink) Emit(e Event) {
	// Swallowed by design. With the async worker this returns nil anyway;
	// delivery failures surface through the library's own drop logging.
	_ = s.client.Event(e.Type, e.Data)
}

func (s *hbSink) Notify(n Notice) string {
	extra := []any{honeybadger.Fingerprint{Content: n.Fingerprint}}
	if len(n.Context) > 0 {
		extra = append(extra, honeybadger.Context(n.Context))
	}
	token, _ := s.client.Notify(n.Err, extra...)
	return token
}

func (s *hbSink) Flush() { s.client.Flush() }
```

- [ ] **Step 4: Write the failing test**

Create `internal/analytics/honeybadger_test.go`. Note that `Configuration.Backend` cannot be faked from outside honeybadger-go — `Backend.Event` takes `[]*eventPayload`, an unexported type — so the test points `Endpoint` at an `httptest.Server` and uses `Sync: true` for determinism.

```go
package analytics

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	honeybadger "github.com/honeybadger-io/honeybadger-go"
)

func TestHoneybadgerSink_EmitPostsEvent(t *testing.T) {
	type captured struct {
		path string
		body []byte
	}
	got := make(chan captured, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- captured{path: r.URL.Path, body: b}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	sink := newHoneybadgerSink(honeybadger.Configuration{
		APIKey:   "test-key",
		Env:      "test",
		Endpoint: srv.URL,
		Sync:     true,
	})

	sink.Emit(Event{Type: "mcp.tool_call", Data: map[string]any{
		"tool":    "list_faults",
		"outcome": "ok",
	}})

	c := <-got
	if c.path != "/v1/events" {
		t.Errorf("path = %q, want /v1/events", c.path)
	}
	var payload map[string]any
	if err := json.Unmarshal(c.body, &payload); err != nil {
		t.Fatalf("body is not JSON: %v (body=%s)", err, c.body)
	}
	if payload["event_type"] != "mcp.tool_call" {
		t.Errorf("event_type = %v, want mcp.tool_call", payload["event_type"])
	}
	if payload["tool"] != "list_faults" {
		t.Errorf("tool = %v, want list_faults", payload["tool"])
	}
}

func TestHoneybadgerSink_NotifyReturnsToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	sink := newHoneybadgerSink(honeybadger.Configuration{
		APIKey:   "test-key",
		Env:      "test",
		Endpoint: srv.URL,
		Sync:     true,
	})

	token := sink.Notify(Notice{
		Err:         errors.New("boom"),
		Fingerprint: "mcp_tool_call:list_faults",
		Context:     map[string]any{"tool": "list_faults"},
	})
	if token == "" {
		t.Error("Notify returned an empty token, want a correlation token")
	}
}

func TestNopSink_NotifyReturnsEmpty(t *testing.T) {
	s := NewNopSink()
	s.Emit(Event{Type: "x"})
	s.Flush()
	if token := s.Notify(Notice{Err: errors.New("boom")}); token != "" {
		t.Errorf("token = %q, want empty", token)
	}
}
```

- [ ] **Step 5: Run the tests**

Run: `GOWORK=off go test ./internal/analytics/ -v`
Expected: PASS. If `event_type` is not the key honeybadger-go uses for the event type, read `/Users/ben/go/pkg/mod/github.com/honeybadger-io/honeybadger-go@v0.9.0/events.go` for the actual serialized field name and fix the assertion — do not change the production code to match a guess.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/analytics/
git commit -m "Add analytics package wrapping honeybadger-go

Sink is the only interface hbmcp needs, and this package is the only one
that imports honeybadger-go. Notify takes a Notice struct so library types
never leak outward."
```

---

### Task 3: Config fields and env binding

**Files:**
- Modify: `internal/config/config.go`
- Modify: `cmd/honeybadger-mcp-server/main.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `config.Config` gains `HoneybadgerAPIKey string` and `HoneybadgerEnv string`. `config.Load`'s signature is **unchanged** — the two new fields are set by `loadConfigFromFlags` in `main.go` after `Load` returns, keeping config construction in one function without growing an already 6-parameter positional call.

- [ ] **Step 1: Write the failing test**

Add to `internal/config/config_test.go`:

```go
func TestConfig_AnalyticsFieldsDefaultEmpty(t *testing.T) {
	cfg, err := Load("token", "https://api.example.com", "", "info", false, TransportStdio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HoneybadgerAPIKey != "" {
		t.Errorf("HoneybadgerAPIKey = %q, want empty by default", cfg.HoneybadgerAPIKey)
	}
	if cfg.HoneybadgerEnv != "" {
		t.Errorf("HoneybadgerEnv = %q, want empty by default", cfg.HoneybadgerEnv)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `GOWORK=off go test ./internal/config/ -run TestConfig_AnalyticsFields -v`
Expected: FAIL — `cfg.HoneybadgerAPIKey undefined`

- [ ] **Step 3: Add the fields**

In `internal/config/config.go`, extend the `Config` struct:

```go
type Config struct {
	AuthToken       string
	APIURL          string
	InstructionsURL string
	LogLevel        string
	ReadOnly        bool
	TransportMode   string
	// Analytics destination. Empty disables analytics entirely; see
	// hbmcp.newSink for the activation gate.
	HoneybadgerAPIKey string
	HoneybadgerEnv    string
}
```

- [ ] **Step 4: Bind the environment variables**

In `cmd/honeybadger-mcp-server/main.go`, add to the `BindEnv` block (after the `read-only` line, around line 141):

```go
	_ = viper.BindEnv("honeybadger-api-key", "HONEYBADGER_API_KEY")
	_ = viper.BindEnv("honeybadger-env", "HONEYBADGER_ENV")
```

Then in `loadConfigFromFlags`, replace the `return config.Load(...)` tail with:

```go
	cfg, err := config.Load(
		viper.GetString("auth-token"),
		viper.GetString("api-url"),
		viper.GetString("instructions-url"),
		viper.GetString("log-level"),
		readOnly,
		transportMode,
	)
	if err != nil {
		return nil, err
	}
	cfg.HoneybadgerAPIKey = viper.GetString("honeybadger-api-key")
	cfg.HoneybadgerEnv = viper.GetString("honeybadger-env")
	return cfg, nil
}
```

- [ ] **Step 5: Run the tests**

Run: `GOWORK=off go test ./internal/config/ ./cmd/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go cmd/honeybadger-mcp-server/main.go
git commit -m "Add analytics config fields and env bindings"
```

---

### Task 4: Upstream recorder

**Files:**
- Create: `internal/hbmcp/upstream.go`
- Test: `internal/hbmcp/upstream_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type upstreamRecord struct` with method `record(status int, transportErr bool, d time.Duration)` and `snapshot() (called bool, status int, transportErr bool, durationMs int64)`
  - `func withUpstreamRecord(ctx context.Context) (context.Context, *upstreamRecord)`
  - `func upstreamRecordFromContext(ctx context.Context) *upstreamRecord`
  - `func newRecordingHTTPClient() *http.Client`

**Why this exists:** Every handler converts upstream API failures into `mcp.NewToolResultError(...), nil` (e.g. `alarms.go:200`, `projects.go:259`). Classifying on the handler's return value alone would file an API outage as a product signal. The transport sees the truth.

**Why it is flat:** Every handler makes exactly one upstream call — the files with more call expressions than handlers are either/or branches (`projects.go:254/256`, `projects.go:437/440`), and api-go performs a single `httpClient.Do` with no retries (`api-go@v0.8.0/client.go:122`). The sticky-failure rule is a cheap guard so a future fan-out handler cannot silently misclassify.

- [ ] **Step 1: Write the failing test**

Create `internal/hbmcp/upstream_test.go`:

```go
package hbmcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRecordingTransport_RecordsStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, rec := withUpstreamRecord(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := newRecordingHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	called, status, transportErr, _ := rec.snapshot()
	if !called {
		t.Fatal("called = false, want true")
	}
	if status != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", status)
	}
	if transportErr {
		t.Error("transportErr = true, want false")
	}
}

func TestRecordingTransport_RecordsTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing is listening now

	ctx, rec := withUpstreamRecord(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if _, err := newRecordingHTTPClient().Do(req); err == nil {
		t.Fatal("expected a transport error")
	}

	called, _, transportErr, _ := rec.snapshot()
	if !called {
		t.Fatal("called = false, want true")
	}
	if !transportErr {
		t.Error("transportErr = false, want true")
	}
}

// A later success must not mask an earlier failure.
func TestUpstreamRecord_FailureIsSticky(t *testing.T) {
	rec := &upstreamRecord{}
	rec.record(500, false, 5*time.Millisecond)
	rec.record(200, false, 1*time.Millisecond)

	_, status, _, _ := rec.snapshot()
	if status != 500 {
		t.Errorf("status = %d, want 500 (failure should be sticky)", status)
	}
}

func TestUpstreamRecord_NilSafeWhenAnalyticsOff(t *testing.T) {
	// No record in the context: the transport must not panic.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := newRecordingHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if rec := upstreamRecordFromContext(context.Background()); rec != nil {
		t.Error("expected no record in a bare context")
	}
}

// api-go's default client carries a 30s timeout; replacing the client must
// not silently drop it (api-go@v0.8.0/client.go:39-41).
func TestRecordingHTTPClient_PreservesTimeout(t *testing.T) {
	if got := newRecordingHTTPClient().Timeout; got != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `GOWORK=off go test ./internal/hbmcp/ -run 'TestRecording|TestUpstream' -v`
Expected: FAIL — `undefined: withUpstreamRecord`

- [ ] **Step 3: Write the implementation**

Create `internal/hbmcp/upstream.go`:

```go
package hbmcp

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// apiClientTimeout mirrors api-go's default http.Client timeout
// (api-go@v0.8.0/client.go:39-41). WithHTTPClient replaces that client
// wholesale, so failing to set this would remove the upstream timeout and
// let a hung API call pin a request indefinitely.
const apiClientTimeout = 30 * time.Second

// upstreamRecord captures the outcome of the API call a tool handler makes.
// Handlers turn API failures into IsError results with a nil error, so the
// handler's return value cannot distinguish an outage from a rejected call.
//
// Every handler makes exactly one upstream call, so no aggregation is needed.
// The sticky-failure rule below is a guard for a future fan-out handler: once
// a failure is recorded, a later success cannot overwrite it.
type upstreamRecord struct {
	mu           sync.Mutex
	called       bool
	status       int
	transportErr bool
	durationMs   int64
}

func (r *upstreamRecord) record(status int, transportErr bool, d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.called && (r.transportErr || r.status >= http.StatusInternalServerError) {
		return
	}
	r.called = true
	r.status = status
	r.transportErr = transportErr
	r.durationMs = d.Milliseconds()
}

func (r *upstreamRecord) snapshot() (called bool, status int, transportErr bool, durationMs int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.called, r.status, r.transportErr, r.durationMs
}

type upstreamKey struct{}

func withUpstreamRecord(ctx context.Context) (context.Context, *upstreamRecord) {
	rec := &upstreamRecord{}
	return context.WithValue(ctx, upstreamKey{}, rec), rec
}

func upstreamRecordFromContext(ctx context.Context) *upstreamRecord {
	rec, _ := ctx.Value(upstreamKey{}).(*upstreamRecord)
	return rec
}

type recordingTransport struct {
	base http.RoundTripper
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := upstreamRecordFromContext(req.Context())
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	if rec == nil {
		return resp, err
	}
	if err != nil {
		rec.record(0, true, time.Since(start))
	} else {
		rec.record(resp.StatusCode, false, time.Since(start))
	}
	return resp, err
}

// newRecordingHTTPClient builds the client handed to api-go via
// WithHTTPClient. api-go builds its requests with
// http.NewRequestWithContext using the handler's context, so the record
// placed by the instrumenter is reachable here.
func newRecordingHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   apiClientTimeout,
		Transport: &recordingTransport{base: http.DefaultTransport},
	}
}
```

- [ ] **Step 4: Run the tests**

Run: `GOWORK=off go test ./internal/hbmcp/ -run 'TestRecording|TestUpstream' -v`
Expected: PASS (5 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/hbmcp/upstream.go internal/hbmcp/upstream_test.go
git commit -m "Record upstream API outcome at the transport layer

Handlers return API failures as IsError with a nil error, so the handler
return value cannot distinguish an outage from a rejected call. Preserves
api-go's 30s client timeout, which WithHTTPClient would otherwise drop."
```

---

### Task 5: The `instrumenter` decorator

**Files:**
- Create: `internal/hbmcp/instrument.go`
- Test: `internal/hbmcp/instrument_test.go`

**Interfaces:**
- Consumes: `analytics.Sink`, `analytics.Event`, `analytics.Notice` (Task 2); `Claims.Subject/AccountID/ClientID/ProjectScoped` (Task 1); `withUpstreamRecord`, `upstreamRecord.snapshot` (Task 4); `config.Config`.
- Produces:
  - `type instrumenter struct { sink analytics.Sink; cfg *config.Config; version string }`
  - `func newInstrumenter(sink analytics.Sink, cfg *config.Config, version string) *instrumenter`
  - `func (i *instrumenter) wrap(tool mcp.Tool, h server.ToolHandlerFunc) server.ToolHandlerFunc`

- [ ] **Step 1: Write the failing test**

Create `internal/hbmcp/instrument_test.go`:

```go
package hbmcp

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/analytics"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type recordingSink struct {
	events  []analytics.Event
	notices []analytics.Notice
	token   string
}

func (s *recordingSink) Emit(e analytics.Event) { s.events = append(s.events, e) }
func (s *recordingSink) Notify(n analytics.Notice) string {
	s.notices = append(s.notices, n)
	return s.token
}
func (s *recordingSink) Flush() {}

func testTool() mcp.Tool {
	return mcp.NewTool("list_faults",
		mcp.WithDescription("test"),
		mcp.WithNumber("project_id", mcp.Description("project")),
		mcp.WithNumber("limit", mcp.Description("limit")),
	)
}

func testInstrumenter(sink analytics.Sink) *instrumenter {
	return newInstrumenter(sink, &config.Config{TransportMode: config.TransportHTTP}, "1.2.3")
}

func callWrapped(t *testing.T, sink *recordingSink, args map[string]any, h server.ToolHandlerFunc) analytics.Event {
	t.Helper()
	wrapped := testInstrumenter(sink).wrap(testTool(), h)
	req := mcp.CallToolRequest{}
	req.Params.Name = "list_faults"
	req.Params.Arguments = args
	ctx := WithClaims(context.Background(), &Claims{
		Scopes:        []string{"read"},
		Subject:       "user_abc",
		AccountID:     "acct_xyz",
		ClientID:      "app-42",
		ProjectScoped: true,
	})
	_, _ = wrapped(ctx, req)
	if len(sink.events) != 1 {
		t.Fatalf("got %d events, want 1", len(sink.events))
	}
	return sink.events[0]
}

func TestWrap_OKOutcomeAndIdentity(t *testing.T) {
	sink := &recordingSink{}
	ev := callWrapped(t, sink, map[string]any{"project_id": float64(1)},
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("ok"), nil
		})

	if ev.Type != "mcp.tool_call" {
		t.Errorf("Type = %q, want mcp.tool_call", ev.Type)
	}
	if ev.Data["outcome"] != "ok" {
		t.Errorf("outcome = %v, want ok", ev.Data["outcome"])
	}
	if ev.Data["tool"] != "list_faults" {
		t.Errorf("tool = %v, want list_faults", ev.Data["tool"])
	}
	if ev.Data["user_id"] != "user_abc" {
		t.Errorf("user_id = %v, want user_abc", ev.Data["user_id"])
	}
	if ev.Data["account_id"] != "acct_xyz" {
		t.Errorf("account_id = %v, want acct_xyz", ev.Data["account_id"])
	}
	if ev.Data["client_id"] != "app-42" {
		t.Errorf("client_id = %v, want app-42", ev.Data["client_id"])
	}
	if ev.Data["project_scoped"] != true {
		t.Errorf("project_scoped = %v, want true", ev.Data["project_scoped"])
	}
	if ev.Data["server_version"] != "1.2.3" {
		t.Errorf("server_version = %v, want 1.2.3", ev.Data["server_version"])
	}
	if len(sink.notices) != 0 {
		t.Errorf("got %d notices on the ok path, want 0", len(sink.notices))
	}
}

func TestWrap_ToolErrorOutcome(t *testing.T) {
	sink := &recordingSink{}
	ev := callWrapped(t, sink, nil,
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultError("project_id is required"), nil
		})

	if ev.Data["outcome"] != "tool_error" {
		t.Errorf("outcome = %v, want tool_error", ev.Data["outcome"])
	}
	if len(sink.notices) != 0 {
		t.Errorf("tool_error must not Notify; got %d notices", len(sink.notices))
	}
}

func TestWrap_InternalErrorOnReturnedError(t *testing.T) {
	sink := &recordingSink{token: "tok-123"}
	ev := callWrapped(t, sink, nil,
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return nil, errors.New("boom")
		})

	if ev.Data["outcome"] != "internal_error" {
		t.Errorf("outcome = %v, want internal_error", ev.Data["outcome"])
	}
	if ev.Data["error_id"] != "tok-123" {
		t.Errorf("error_id = %v, want tok-123", ev.Data["error_id"])
	}
	if len(sink.notices) != 1 {
		t.Fatalf("got %d notices, want 1", len(sink.notices))
	}
	if sink.notices[0].Fingerprint != "mcp_tool_call:list_faults" {
		t.Errorf("Fingerprint = %q, want mcp_tool_call:list_faults", sink.notices[0].Fingerprint)
	}
}

// A 5xx from the API is an outage, even though the handler reports it as a
// tool result error with a nil Go error.
func TestWrap_UpstreamServerErrorIsInternalError(t *testing.T) {
	sink := &recordingSink{token: "tok-500"}
	ev := callWrapped(t, sink, nil,
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if rec := upstreamRecordFromContext(ctx); rec != nil {
				rec.record(http.StatusInternalServerError, false, 3*time.Millisecond)
			}
			return mcp.NewToolResultError("Failed to list faults: HTTP 500"), nil
		})

	if ev.Data["outcome"] != "internal_error" {
		t.Errorf("outcome = %v, want internal_error", ev.Data["outcome"])
	}
	if ev.Data["upstream_status"] != 500 {
		t.Errorf("upstream_status = %v, want 500", ev.Data["upstream_status"])
	}
	if len(sink.notices) != 1 {
		t.Errorf("got %d notices, want 1", len(sink.notices))
	}
}

// A 4xx is attributable to the call, not to an outage.
func TestWrap_UpstreamClientErrorIsToolError(t *testing.T) {
	sink := &recordingSink{}
	ev := callWrapped(t, sink, nil,
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if rec := upstreamRecordFromContext(ctx); rec != nil {
				rec.record(http.StatusNotFound, false, time.Millisecond)
			}
			return mcp.NewToolResultError("Failed: HTTP 404"), nil
		})

	if ev.Data["outcome"] != "tool_error" {
		t.Errorf("outcome = %v, want tool_error", ev.Data["outcome"])
	}
	if len(sink.notices) != 0 {
		t.Errorf("4xx must not Notify; got %d notices", len(sink.notices))
	}
}

func TestWrap_ArgNamesAllowlistedAndSorted(t *testing.T) {
	sink := &recordingSink{}
	ev := callWrapped(t, sink, map[string]any{
		"project_id":            float64(1),
		"limit":                 float64(10),
		"customer@example.com":  "injected",
		"another_undeclared":    true,
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	})

	names, ok := ev.Data["arg_names"].([]string)
	if !ok {
		t.Fatalf("arg_names is %T, want []string", ev.Data["arg_names"])
	}
	if len(names) != 2 || names[0] != "limit" || names[1] != "project_id" {
		t.Errorf("arg_names = %v, want [limit project_id]", names)
	}
	if ev.Data["unknown_arg_count"] != 2 {
		t.Errorf("unknown_arg_count = %v, want 2", ev.Data["unknown_arg_count"])
	}
	for _, n := range names {
		if n == "customer@example.com" {
			t.Fatal("undeclared argument name leaked into the event")
		}
	}
}

func TestWrap_PanicIsRecordedAndRepanics(t *testing.T) {
	sink := &recordingSink{token: "tok-panic"}
	wrapped := testInstrumenter(sink).wrap(testTool(),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			panic("kaboom")
		})

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("panic did not propagate; WithRecovery would never see it")
			}
		}()
		req := mcp.CallToolRequest{}
		req.Params.Name = "list_faults"
		_, _ = wrapped(context.Background(), req)
	}()

	if len(sink.events) != 1 {
		t.Fatalf("got %d events, want 1", len(sink.events))
	}
	if sink.events[0].Data["outcome"] != "panic" {
		t.Errorf("outcome = %v, want panic", sink.events[0].Data["outcome"])
	}
	if len(sink.notices) != 1 {
		t.Errorf("panic must Notify; got %d notices", len(sink.notices))
	}
}

func TestWrap_ResultAndErrorPassThroughUnchanged(t *testing.T) {
	sink := &recordingSink{}
	want := mcp.NewToolResultText("payload")
	wantErr := errors.New("boom")
	wrapped := testInstrumenter(sink).wrap(testTool(),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return want, wantErr
		})

	req := mcp.CallToolRequest{}
	req.Params.Name = "list_faults"
	got, err := wrapped(context.Background(), req)
	if got != want {
		t.Error("result was not passed through unchanged")
	}
	if !errors.Is(err, wantErr) {
		t.Error("error was not passed through unchanged")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `GOWORK=off go test ./internal/hbmcp/ -run TestWrap -v`
Expected: FAIL — `undefined: newInstrumenter`

- [ ] **Step 3: Write the implementation**

Create `internal/hbmcp/instrument.go`:

```go
package hbmcp

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/analytics"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	eventToolCall = "mcp.tool_call"

	outcomeOK            = "ok"
	outcomeToolError     = "tool_error"
	outcomeInternalError = "internal_error"
	outcomePanic         = "panic"
)

type instrumenter struct {
	sink    analytics.Sink
	cfg     *config.Config
	version string
}

func newInstrumenter(sink analytics.Sink, cfg *config.Config, version string) *instrumenter {
	return &instrumenter{sink: sink, cfg: cfg, version: version}
}

// wrap decorates a tool handler with usage analytics. It takes the whole
// mcp.Tool so the argument allowlist is built once at registration rather
// than per call.
func (i *instrumenter) wrap(tool mcp.Tool, h server.ToolHandlerFunc) server.ToolHandlerFunc {
	name := tool.Name
	allowed := allowedArgNames(tool)

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, rec := withUpstreamRecord(ctx)
		start := time.Now()

		// Observe panics, report them, then re-panic so
		// server.WithRecovery still handles the panic exactly as before.
		defer func() {
			if r := recover(); r != nil {
				i.emit(ctx, name, allowed, req, outcomePanic, start, rec,
					fmt.Errorf("panic in tool %s: %v", name, r))
				panic(r)
			}
		}()

		result, err := h(ctx, req)
		i.emit(ctx, name, allowed, req, i.classify(result, err, rec), start, rec, err)
		return result, err
	}
}

// classify separates a product signal (the caller got it wrong) from an
// outage signal (the API is failing). It cannot rely on the handler's return
// value alone: handlers convert upstream failures into IsError results with a
// nil error, so an outage is only visible in the upstream record.
func (i *instrumenter) classify(result *mcp.CallToolResult, err error, rec *upstreamRecord) string {
	if err != nil {
		return outcomeInternalError
	}
	called, status, transportErr, _ := rec.snapshot()
	if called && (transportErr || status >= 500) {
		return outcomeInternalError
	}
	if result != nil && result.IsError {
		return outcomeToolError
	}
	return outcomeOK
}

func (i *instrumenter) emit(
	ctx context.Context,
	name string,
	allowed map[string]struct{},
	req mcp.CallToolRequest,
	outcome string,
	start time.Time,
	rec *upstreamRecord,
	err error,
) {
	names, unknown := splitArgNames(req.GetArguments(), allowed)

	data := map[string]any{
		"tool":              name,
		"outcome":           outcome,
		"duration_ms":       time.Since(start).Milliseconds(),
		"arg_names":         names,
		"unknown_arg_count": unknown,
		"read_only":         EffectiveReadOnly(ctx, i.cfg),
		"server_version":    i.version,
	}

	if claims := ClaimsFromContext(ctx); claims != nil {
		data["user_id"] = claims.Subject
		data["account_id"] = claims.AccountID
		data["client_id"] = claims.ClientID
		data["project_scoped"] = claims.ProjectScoped
	}

	if called, status, transportErr, durationMs := rec.snapshot(); called {
		data["upstream_ms"] = durationMs
		if !transportErr {
			data["upstream_status"] = status
		}
	}

	// Error detail goes to error tracking, never into the event: messages can
	// carry API response bodies, which are customer data.
	if outcome == outcomeInternalError || outcome == outcomePanic {
		if err == nil {
			err = fmt.Errorf("tool %s failed upstream", name)
		}
		notice := analytics.Notice{
			Err: err,
			// Without this every tool's errors share the decorator's stack
			// frame and group together.
			Fingerprint: "mcp_tool_call:" + name,
			Context: map[string]any{
				"tool":    name,
				"outcome": outcome,
			},
		}
		if claims := ClaimsFromContext(ctx); claims != nil {
			notice.Context["account_id"] = claims.AccountID
			notice.Context["client_id"] = claims.ClientID
		}
		if token := i.sink.Notify(notice); token != "" {
			data["error_id"] = token
		}
	}

	i.sink.Emit(analytics.Event{Type: eventToolCall, Data: data})
}

// allowedArgNames builds the set of argument names declared by the tool's
// input schema. Arguments arrive as an arbitrary map and this server enables
// no schema validation, so an undeclared key is both an unbounded-cardinality
// risk and a place a caller could hide customer data.
func allowedArgNames(tool mcp.Tool) map[string]struct{} {
	allowed := make(map[string]struct{}, len(tool.InputSchema.Properties))
	for k := range tool.InputSchema.Properties {
		allowed[k] = struct{}{}
	}
	return allowed
}

func splitArgNames(args map[string]any, allowed map[string]struct{}) ([]string, int) {
	names := make([]string, 0, len(args))
	unknown := 0
	for k := range args {
		if _, ok := allowed[k]; ok {
			names = append(names, k)
			continue
		}
		unknown++
	}
	sort.Strings(names)
	return names, unknown
}
```

- [ ] **Step 4: Run the tests**

Run: `GOWORK=off go test ./internal/hbmcp/ -run TestWrap -v`
Expected: PASS (8 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/hbmcp/instrument.go internal/hbmcp/instrument_test.go
git commit -m "Add the tool-call instrumentation decorator

Classifies outcomes using the upstream record rather than the handler's
return value, allowlists argument names against the tool schema, and
fingerprints notices per tool so errors do not all group together."
```

---

### Task 6: Wire it up with the activation gate

**Files:**
- Modify: `internal/hbmcp/server.go`
- Modify: `internal/hbmcp/tool_search.go`
- Test: `internal/hbmcp/server_test.go`

**Interfaces:**
- Consumes: everything from Tasks 1–5.
- Produces:
  - `func newSink(cfg *config.Config) analytics.Sink`
  - `newToolRegistrar(s *server.MCPServer, inst *instrumenter) *toolRegistrar`
  - `registerSearchTool(s *server.MCPServer, catalog []ToolInfo, cfg *config.Config, inst *instrumenter)`
  - `NewServerWithCatalog(cfg *config.Config, version string) (*server.MCPServer, []ToolInfo, analytics.Sink)` — **note the new third return value**, needed so `main.go` can flush on shutdown.

- [ ] **Step 1: Write the failing test**

Add to `internal/hbmcp/server_test.go`:

```go
// The activation gate is the only thing separating our telemetry from a
// customer's own project: honeybadger-go seeds its DefaultClient from ambient
// HONEYBADGER_API_KEY, so in stdio a self-hosting customer's key is our
// configured key. stdio must never get a live sink.
func TestNewSink_StdioNeverActivates(t *testing.T) {
	t.Setenv("HONEYBADGER_API_KEY", "a-customers-own-key")
	cfg := &config.Config{
		TransportMode:     config.TransportStdio,
		HoneybadgerAPIKey: "a-customers-own-key",
	}
	if sink := newSink(cfg); sink != analytics.NewNopSink() {
		t.Errorf("stdio produced %T, want the no-op sink", sink)
	}
}

func TestNewSink_HTTPWithoutKeyIsNoop(t *testing.T) {
	cfg := &config.Config{TransportMode: config.TransportHTTP}
	if sink := newSink(cfg); sink != analytics.NewNopSink() {
		t.Errorf("http without a key produced %T, want the no-op sink", sink)
	}
}

func TestNewSink_HTTPWithKeyActivates(t *testing.T) {
	cfg := &config.Config{
		TransportMode:     config.TransportHTTP,
		HoneybadgerAPIKey: "our-key",
	}
	if sink := newSink(cfg); sink == analytics.NewNopSink() {
		t.Error("http with a key produced the no-op sink, want a live sink")
	}
}

func TestNewServerWithCatalog_StdioReturnsNopSink(t *testing.T) {
	cfg := &config.Config{
		AuthToken:         "t",
		TransportMode:     config.TransportStdio,
		HoneybadgerAPIKey: "a-customers-own-key",
	}
	_, _, sink := NewServerWithCatalog(cfg, "test")
	if sink != analytics.NewNopSink() {
		t.Errorf("stdio server got %T, want the no-op sink", sink)
	}
}
```

Add `"github.com/honeybadger-io/honeybadger-mcp-server/internal/analytics"` to that file's imports.

**Note:** `nopSink` is an empty struct, so `NewNopSink() == NewNopSink()` compares equal. That is what makes these identity assertions work.

- [ ] **Step 2: Run to verify it fails**

Run: `GOWORK=off go test ./internal/hbmcp/ -run TestNewSink -v`
Expected: FAIL — `undefined: newSink`

- [ ] **Step 3: Add the gate and thread the instrumenter through the registrar**

In `internal/hbmcp/tool_search.go`, change the registrar to hold an instrumenter:

```go
type toolRegistrar struct {
	server  *server.MCPServer
	inst    *instrumenter
	catalog []ToolInfo
}

func newToolRegistrar(s *server.MCPServer, inst *instrumenter) *toolRegistrar {
	return &toolRegistrar{
		server: s,
		inst:   inst,
	}
}

func (r *toolRegistrar) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	r.server.AddTool(tool, r.inst.wrap(tool, handler))
	r.catalog = append(r.catalog, ToolInfo{
		Name:        tool.Name,
		Description: tool.Description,
		ReadOnly:    tool.Annotations.ReadOnlyHint != nil && *tool.Annotations.ReadOnlyHint,
	})
}
```

Then change `registerSearchTool` to accept and apply the instrumenter. Its registration stays direct against the server — it needs the completed catalog and must exclude itself — so it wraps its own handler:

```go
func registerSearchTool(s *server.MCPServer, catalog []ToolInfo, cfg *config.Config, inst *instrumenter) {
	tool := mcp.NewTool(searchToolInfo.Name,
		mcp.WithTitleAnnotation("Search Tools"),
		mcp.WithDescription(searchToolInfo.Description),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query to match against tool names and descriptions"),
		),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Lift the existing closure body from tool_search.go:69-97 verbatim.
		// It starts with `query := strings.TrimSpace(...)` and ends with
		// `return mcp.NewToolResultText(sb.String()), nil`. Do not change a
		// line of it — the read-only catalog filtering and the
		// self-exclusion behaviour must stay exactly as they are.
	}
	s.AddTool(tool, inst.wrap(tool, handler))
}
```

This is a pure restructuring: the `mcp.NewTool(...)` call and the handler closure are both unchanged, they are just bound to named variables so `tool` can be passed to `wrap`.

- [ ] **Step 4: Add `newSink` and update `NewServerWithCatalog`**

In `internal/hbmcp/server.go`, add the gate:

```go
// newSink is the activation gate. Analytics runs only on the hosted HTTP
// transport with a key we configured. stdio always gets the no-op sink,
// whatever HONEYBADGER_API_KEY contains — in stdio that key belongs to the
// customer running the binary, and sending our telemetry to their project
// would consume their quota.
func newSink(cfg *config.Config) analytics.Sink {
	if cfg.TransportMode != config.TransportHTTP || cfg.HoneybadgerAPIKey == "" {
		return analytics.NewNopSink()
	}
	env := cfg.HoneybadgerEnv
	if env == "" {
		env = "production"
	}
	return analytics.NewHoneybadgerSink(cfg.HoneybadgerAPIKey, env)
}
```

Update `NewServer` and `NewServerWithCatalog`:

```go
func NewServer(cfg *config.Config, version string) *server.MCPServer {
	s, _, _ := NewServerWithCatalog(cfg, version)
	return s
}

func NewServerWithCatalog(cfg *config.Config, version string) (*server.MCPServer, []ToolInfo, analytics.Sink) {
	logger := logging.SetupLogger(cfg.LogLevel)

	sink := newSink(cfg)
	analyticsOn := sink != analytics.NewNopSink()
	if !analyticsOn {
		logger.Info("Usage analytics disabled", "reason", "no HONEYBADGER_API_KEY or non-http transport")
	}

	// ... existing hooks and serverOptions unchanged ...

	s := server.NewMCPServer("honeybadger-mcp-server", version, serverOptions...)

	inst := newInstrumenter(sink, cfg, version)
	clientFor := newClientFactory(cfg, analyticsOn)
	r := newToolRegistrar(s, inst)
	RegisterReferenceTools(r, newReferenceFetcher(cfg.InstructionsURL, logger))
	RegisterProjectTools(r, clientFor)
	RegisterFaultTools(r, clientFor)
	RegisterInsightsTools(r, clientFor)
	RegisterStreamTools(r, clientFor)
	RegisterDashboardTools(r, clientFor)
	RegisterAlarmTools(r, clientFor)
	RegisterCheckInTools(r, clientFor)
	registerSearchTool(s, r.catalog, cfg, inst)

	return s, append(r.catalog, searchToolInfo), sink
}
```

The `Register*` list above matches `internal/hbmcp/server.go:78-85` exactly. Only the
three lines around it change: `inst := ...`, the `newToolRegistrar` and
`newClientFactory` arguments, and `registerSearchTool`'s new parameter.

Attach the recording transport in the client factory:

```go
func newClientFactory(cfg *config.Config, analyticsOn bool) ClientFactory {
	if cfg.TransportMode == config.TransportHTTP {
		// No fallback to cfg.AuthToken — the 401 middleware must catch
		// bearer-less requests; a fallback would mask that regression.
		return func(ctx context.Context) *hbapi.Client {
			c := hbapi.NewClient().
				WithBaseURL(cfg.APIURL).
				WithBearerToken(AuthTokenFromContext(ctx))
			if analyticsOn {
				c = c.WithHTTPClient(newRecordingHTTPClient())
			}
			return c
		}
	}
	return func(ctx context.Context) *hbapi.Client {
		return hbapi.NewClient().
			WithBaseURL(cfg.APIURL).
			WithAuthToken(cfg.AuthToken)
	}
}
```

- [ ] **Step 5: Fix the other `NewServerWithCatalog` call site**

Run: `GOWORK=off go build ./...`
Expected: a compile error in `cmd/honeybadger-mcp-server/main.go` where `NewServerWithCatalog` is destructured into two values. Update it to three, capturing the sink:

```go
	mcpServer, toolCatalog, sink := hbmcp.NewServerWithCatalog(cfg, version)
```

Task 7 uses `sink`. Until then, silence the unused variable by proceeding directly to Task 7 in the same session, or temporarily assign `_ = sink`.

- [ ] **Step 6: Run the full suite**

Run: `GOWORK=off go test ./...`
Expected: PASS across all packages. Any existing test that calls `newToolRegistrar` or `registerSearchTool` needs its new argument — pass `newInstrumenter(analytics.NewNopSink(), &config.Config{}, "test")`.

- [ ] **Step 7: Commit**

```bash
git add internal/hbmcp/ cmd/honeybadger-mcp-server/main.go
git commit -m "Wire analytics through the registrar behind the activation gate

stdio always gets the no-op sink regardless of HONEYBADGER_API_KEY, which
in stdio belongs to the customer running the binary."
```

---

### Task 7: Flush on shutdown and document the configuration

**Files:**
- Modify: `cmd/honeybadger-mcp-server/main.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: `analytics.Sink` from `NewServerWithCatalog` (Task 6).
- Produces: nothing further.

**Why:** honeybadger-go batches events in a background worker. Without a flush, every deploy drops the last unflushed batch — on a frequently deployed service that is a systematic undercount, not a rounding error.

- [ ] **Step 1: Flush on the graceful-shutdown path**

In `runHTTP` in `cmd/honeybadger-mcp-server/main.go`, the shutdown block is at lines 355-372. Insert the flush immediately after the `mcpHandler.Shutdown` error check and before the `cancelBase()` call, so buffered events are sent while the process is still healthy:

```go
		if err := mcpHandler.Shutdown(shutdownCtx); err != nil {
			logger.Error("MCP shutdown error", "error", err)
		}
		// Events are batched in a background worker; without this the last
		// batch is lost on every deploy. No-op when analytics is off.
		sink.Flush()
		// End streaming request contexts so Shutdown's drain returns
		// promptly instead of timing out and TCP-resetting live streams.
		cancelBase()
```

- [ ] **Step 2: Verify it builds and the suite is green**

Run: `GOWORK=off go build ./... && GOWORK=off go test ./...`
Expected: PASS, and no unused-variable error for `sink`.

- [ ] **Step 3: Document the environment variables**

In `README.md`, the environment-variable table is at lines 162-168 with four columns (`Environment Variable | Required | Default | Description`). Append these two rows after the `HONEYBADGER_INSTRUCTIONS_URL` row:

```markdown
| `HONEYBADGER_API_KEY`             | no       | —                          | Project API key for usage analytics and error reporting. Hosted HTTP transport only — ignored in stdio mode. Unset disables analytics entirely. |
| `HONEYBADGER_ENV`                 | no       | production                 | Environment label attached to analytics events and errors                |
```

- [ ] **Step 4: Run gofmt and the full suite one final time**

```bash
GOWORK=off gofmt -l ./cmd ./internal
GOWORK=off go vet ./...
GOWORK=off go test ./...
```

Expected: `gofmt -l` prints nothing, vet is clean, all tests pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/honeybadger-mcp-server/main.go README.md
git commit -m "Flush buffered events on shutdown and document analytics config"
```

---

## Verification

After all tasks, confirm the whole feature end to end:

- [ ] `GOWORK=off go test ./...` — all packages pass
- [ ] `GOWORK=off go vet ./...` — clean
- [ ] `GOWORK=off gofmt -l ./cmd ./internal` — no output
- [ ] `grep -rn "honeybadger-go" internal/hbmcp/ cmd/` returns nothing — the library stays inside `internal/analytics`
- [ ] `grep -rn "honeybadger\.\(Notify\|Event\|Configure\|Flush\)(" internal/ cmd/` returns nothing — no package-level calls, only client instances
- [ ] `GOWORK=off go build -o /tmp/hb-mcp ./cmd/honeybadger-mcp-server && HONEYBADGER_API_KEY=fake /tmp/hb-mcp stdio --auth-token=x` starts and logs `Usage analytics disabled` — the stdio gate holds even with a key present
