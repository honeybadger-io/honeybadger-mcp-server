# MCP Server Usage Analytics — Design

**Date:** 2026-08-06
**Status:** Approved, ready for implementation planning

## Problem

The MCP server has no usage analytics. The only observability is `slog` text to stderr:
session register/unregister (session ID only), request errors, and a debug-level
`Processing request` line that records the MCP method but not which tool was called
(`internal/hbmcp/server.go:46-58`).

Three questions are unanswerable today:

1. **Which tools get used.** The catalog is 33 tools registered through the registrar,
   plus `search_tools` — 34 total. We cannot tell which earn their place, which are
   dead weight, or whether the discovery layer is used at all.
2. **Which customers use it.** `Claims` parses the access token but keeps only
   `Scopes` (`internal/hbmcp/claims.go:12-14`); every identifying claim is discarded.
3. **Whether it works.** No error rates, no latency, no signal on tools the LLM calls
   with malformed arguments.

## Scope

**In scope:** the hosted HTTP/OAuth server only.

**Out of scope:** stdio. It runs on the customer's machine, and instrumenting it means
a phone-home with the disclosure and opt-out obligations that implies. Accepted
consequence: self-hosted stdio usage stays invisible, so adoption figures cover hosted
users only.

**Also out of scope:** adding a `User-Agent` to api-go for server-side attribution of
stdio traffic, and a `mcp.session_start` connection event. The latter was considered
and rejected: in stateless mode it may count per-request rather than per-connection,
producing a number that cannot be trusted.

## Architecture

Four components, each independently testable.

### `internal/analytics` (new package)

Owns the destination; knows nothing about MCP.

```go
type Event struct {
    Type string
    Data map[string]any
}

type Notice struct {
    Err         error
    Fingerprint string
    Context     map[string]any
}

type Sink interface {
    Emit(Event)
    Notify(Notice) string
    Flush()
}
```

`Emit` returns nothing by design — a caller must not be able to fail a tool call by
emitting. `Notify` returns the token because that is the only place it is obtainable
(`honeybadger-go@v0.9.0/client.go:92`).

`Notify` takes a `Notice` struct rather than honeybadger-go's variadic
`extra ...any`, so every library type stays inside this package and `hbmcp` never
imports honeybadger-go — the mirror of "analytics knows nothing about MCP". `Flush` is
on the interface so the shutdown path does not need a handle on the client.

These are one interface rather than two because both methods are backed by the same
`*honeybadger.Client`. That one returns a value and the other does not is a property
of the methods, not a reason for separate types. Two implementations: `hbSink`
wrapping a honeybadger-go client, and `nopSink`.

Because the interface is this small, `hbmcp` tests use a recording fake and never
touch the network.

### `instrumenter` decorator (in `hbmcp`)

The decorator needs more than a sink: `EffectiveReadOnly` requires `cfg`
(`internal/hbmcp/server.go:17`) and the event carries the build version. So it is a
method on a small struct built once at server construction, not a free function with a
long parameter list:

```go
type instrumenter struct {
    sink    analytics.Sink // nopSink when analytics is off
    cfg     *config.Config
    version string
}

func (i *instrumenter) wrap(tool mcp.Tool, h server.ToolHandlerFunc) server.ToolHandlerFunc
```

`wrap` times the call, classifies the outcome, reads identity from the context, emits,
and returns the handler's result unchanged. It takes the whole `mcp.Tool` rather than
just the name so it can build the argument allowlist from `tool.InputSchema.Properties`
once at registration.

**With analytics off, `wrap` returns the handler untouched.** Wrapping and discarding
the result would still cost a context value, a timer, and a panic intercept per call —
and the intercept re-panics, which rewrites the stack `server.WithRecovery` sees. Since
stdio is supposed to be unchanged, the decorator has to be absent there, not merely
inert.

`toolRegistrar` holds an `instrumenter` and applies the decorator in `AddTool`
(`internal/hbmcp/tool_search.go:30`). `registerSearchTool` receives the same
`instrumenter` and wraps its own handler at its own registration site.

**`search_tools` registration is deliberately left as-is.** It is registered directly
against the server (`internal/hbmcp/server.go:86`) because it takes the completed
catalog by value and must run after every other registration. It also excludes itself
from what it searches — note the asymmetry with `server.go:88`, where the landing-page
catalog is `append(r.catalog, searchToolInfo)`, *with* `search_tools`.

Keeping the decorator independent of the registrar means none of that has to be
disturbed — `registerSearchTool` simply calls `wrap` itself.
An earlier draft justified this by claiming that routing `search_tools` through the
registrar would introduce a slice-aliasing bug; that was wrong. The handler's copied
slice header keeps its own length, so appending `searchToolInfo` to `r.catalog`
afterwards remains invisible to it either way, and the must-run-last ordering
constraint is inherent to needing a complete catalog rather than something the
registrar would create. The real argument is simply that a decorator applied at each
registration site requires no restructuring of registration at all.

### Upstream recorder

A `http.RoundTripper` that wraps the api-go client's transport and records the
upstream call's status, error, and duration into a per-call record held in the
context. See "Outcome classification" for why this exists — briefly, handler return
values cannot distinguish an API outage from a rejected call, and the transport can.

The record is deliberately flat — `{status int, transportErr bool, durationMs int64}`
— because **every handler makes exactly one upstream call.** Verified across all 33:
the files with more call expressions than handlers are either/or branches
(`projects.go:254/256`, `projects.go:437/440`), and api-go performs a single
`httpClient.Do` with no retry logic (`api-go@v0.8.0/client.go:122`).

The one concession to a future fan-out handler is that a recorded failure is sticky:
a later success cannot overwrite an earlier failure. That is two lines and removes the
only way this could misclassify silently, which is cheaper than either a worst-outcome
ranking lattice or a comment asking future authors to be careful.

Attached in `newClientFactory` (`internal/hbmcp/server.go:90`) via
`hbapi.Client.WithHTTPClient` (`api-go@v0.8.0/client.go:82`). When analytics is off,
no record is placed in the context and the recorder is not attached, so the stdio path
is byte-for-byte unchanged.

**`WithHTTPClient` replaces api-go's default client**, which carries
`Timeout: 30 * time.Second` (`api-go@v0.8.0/client.go:39-41`). The replacement must
set the same timeout explicitly; supplying a bare `&http.Client{}` would silently
remove the upstream timeout and let a hung API call pin a request indefinitely.

### Wiring in `NewServerWithCatalog`

The `Sink` is selected once at construction and handed to the `instrumenter`. See
"Activation invariant" below.

## Event schema

One event type, `mcp.tool_call`, emitted once per handler invocation.

| Field | Source | Notes |
|---|---|---|
| `tool` | registration name | |
| `outcome` | see classification | |
| `duration_ms` | measured | |
| `arg_names` | request arguments | names only, sorted, allowlisted against the tool schema |
| `unknown_arg_count` | request arguments | count of undeclared keys; never their names |
| `user_id` | `sub` claim | hashid |
| `account_id` | `account_id` claim | hashid, optional |
| `client_id` | `client_id` claim | OAuth app uid, optional |
| `project_scoped` | presence of `project_id` claim | boolean, not the ID itself |
| `read_only` | `EffectiveReadOnly(ctx, cfg)` | effective scope, not the startup flag |
| `upstream_status` | recording RoundTripper | HTTP status from the API, when a call was made |
| `upstream_ms` | recording RoundTripper | time spent in the upstream call |
| `server_version` | build version | |
| `error_id` | notice token | `internal_error` only, omitted when empty |

### Argument capture

Names only, never values. Records which parameters were supplied — answering whether
pagination and filter arguments are used, and catching malformed calls.

**Names are allowlisted against the tool's declared schema.** Arguments arrive as an
arbitrary `map[string]any` (`mcp-go/mcp/tools.go:61,70`) and this server enables no
input-schema validation (`internal/hbmcp/server.go:60`), so a caller can send any key
it likes. An unvalidated key is both an unbounded-cardinality risk and a PII vector —
a client could put customer data in the *name*, not just the value.

The decorator therefore builds a set from `tool.InputSchema.Properties` once at
registration and emits only names in that set. Anything else is counted, not named:

- `arg_names` — declared parameters that were supplied, sorted.
- `unknown_arg_count` — how many undeclared keys were sent, as an integer.

This bounds cardinality to the declared schema and makes it impossible for caller-
controlled text to reach the analytics stream, while still surfacing malformed calls.
Sorting only removes order permutations; the allowlist is what actually bounds the
field.

### Outcome classification

The goal is to separate a product signal (the LLM called us wrong) from an outage
signal (the upstream API is failing). The handler return value alone **cannot** make
that distinction in this codebase.

Every handler converts upstream failures into `mcp.NewToolResultError(...), nil` —
see `alarms.go:200` (API error), `alarms.go:205` (marshal failure), `projects.go:259`.
A returned Go error is vanishingly rare. Classifying on the return value alone would
therefore file API outages as `tool_error` and leave `internal_error` permanently
empty, inverting the exact signal this design exists to produce.

**Upstream outcome is captured at the transport layer instead.** `newClientFactory`
(`internal/hbmcp/server.go:90`) already builds an `hbapi.Client` per call, and api-go
exposes `WithHTTPClient` (`api-go@v0.8.0/client.go:82`). The factory attaches an
`http.Client` whose `Transport` is a recording `RoundTripper` that writes the upstream
status code — or the transport error — into a per-call record carried in the context.
The decorator creates that record before invoking the handler and reads it afterwards.

This needs no handler changes and cannot be missed at an individual call site, which
is why it is preferred over a per-site helper: an omitted helper call would
misclassify silently and no test would catch it.

Resulting classification:

| Condition | Outcome | Meaning |
|---|---|---|
| Handler returns `(result, nil)`, `IsError == false` | `ok` | |
| `IsError == true`, no upstream failure recorded | `tool_error` | We rejected the call before or independently of the API — e.g. `mcp.NewToolResultError("query is required")` (`tool_search.go:72`). A spike in one tool means its schema or description needs work. |
| Upstream failure recorded (5xx, or transport error) | `internal_error` | The API failed. Calls `Notify`. |
| Upstream 4xx recorded | `tool_error` | Bad request or authorization failure — attributable to the call, not an outage. |
| Handler returns `(_, err)` | `internal_error` | Go-level failure. Calls `Notify`. |
| Handler panics | `panic` | Calls `Notify`. |

Two extra fields fall out of the recorder and are worth keeping:

| Field | Notes |
|---|---|
| `upstream_status` | HTTP status from the API, when a call was made. Makes 429s and 5xx directly visible. |
| `upstream_ms` | Time spent in the upstream call, so slow tools can be attributed to the API rather than to us. |

Every handler makes exactly one upstream call, so no aggregation is needed — see
"Upstream recorder" for the verification and for the sticky-failure rule that keeps a
hypothetical future fan-out handler from misclassifying.

## Identity

The Honeybadger AS mints these claims
(`../honeybadger/config/initializers/doorkeeper.rb:114-129`):

```ruby
{ ver:, iss:, sub:, iat:, exp:, jti:, client_id:, scope:, account_id:, project_id:, aud: }.compact
```

`sub` is `User.encode_id` and `account_id` is `Account.encode_id` — both hashids
(`hashid-rails`), so those two are already opaque and there is nothing for us to hash.

**`project_id` is the exception: the AS emits it raw** (`doorkeeper.rb:126`), with no
`encode_id`. Rather than put a bare database ID in the analytics stream, we record
only its *presence* as `project_scoped`. Because the payload is `.compact`ed, presence
of the claim is exactly the scoped-vs-account distinction — the only thing the field
was wanted for — so this loses no stated capability while removing the raw-ID
exception entirely. `Claims` therefore does not carry `ProjectID`.

`client_id` is the OAuth application uid, and is the sole client dimension. It travels
in the token, so it survives stateless mode. It is *not* universal, however:
`opts[:application]&.uid` is safe-navigated and the payload is `.compact`ed
(`doorkeeper.rb:123,128`), so tokens minted without an application omit it entirely.

**MCP `initialize` clientInfo and `session_id` are deliberately not captured.** The
hosted server runs stateless by default (`cmd/honeybadger-mcp-server/main.go:72`,
`--stateless` defaults to `true` and is recommended for horizontal scaling), and
stateless requests get a fresh session with no persisted client info
(`mcp-go@v0.55.1/server/streamable_http.go:1587-1595`). Those fields would be empty in
substantially all production events, so capturing them buys nothing and costs session
plumbing plus its tests. If the hosted deploy ever moves to stateful and client-name
breakdown becomes useful, they can be added back cheaply.

`Claims` gains `Subject`, `AccountID`, `ClientID`, and `ProjectScoped`.

**The payload ends in `.compact`**, so `account_id`, `project_id`, and `client_id` are
genuinely absent on some tokens. Parsing must treat each as optional, and their
absence must never fail token validation — it may only leave the event field empty.

## Error reporting

`internal_error` also calls `Notify` on our client, so error detail goes to error
tracking rather than the analytics stream. Error messages can carry API response
bodies, which are customer data; keeping them out of the event and inside error
tracking means existing redaction and grouping apply. This also closes a real gap:
today `AddOnError` only writes to stderr.

**Ordering.** `Notify` is called before `Emit`, and its return value is recorded as
`error_id` for correlation. `notice.Token` is generated client-side in `newNotice`
before delivery is pushed to the async worker (`client.go:94-118`), so the token is
available immediately. Delivery itself is non-blocking
(`buffered_worker.go:33`), but note that `Notify` runs every registered
`BeforeNotify` handler synchronously first (`client.go:95`) — non-blocking holds
because we register none, not as an unconditional property of the library. Adding a
`BeforeNotify` handler later would put that work on the tool-call path.

Three honest limits:

- `Notify` returns `""` if a beforeNotify handler errors or the worker push fails, so
  `error_id` is omitted when empty.
- The token is minted before delivery, so an `error_id` can reference a notice that
  never landed (worker gave up after retries). Correlation is best-effort.
- Consequently `count(error_id)` in Insights is not a reliable error count. The faults
  list is the source of truth; `error_id` is for jumping from an event to its fault.

**Grouping requires an explicit fingerprint.** `newError(err, 2)` uses the error's own
stack only if it implements `Callers()` (`error.go:64-69`). Plain `fmt.Errorf` errors
— what our handlers and api-go return — do not, so the stack is generated from the
decorator's frame, which is identical for every tool.

That does not by itself collapse everything into one fault: default grouping hashes
the error class alongside the application frame, and the concrete class is preserved
(`error.go:59`), so distinct error types still separate. What it does destroy is the
*per-tool* distinction — the same class failing in `list_faults` and in
`create_alarm` becomes one fault. Passing
`honeybadger.Fingerprint{Content: "mcp_tool_call:" + tool}` (`notice.go:171`) restores
it, plus a `honeybadger.Context{}` carrying tool, account, and client.

The tradeoff is explicit and worth stating: a per-tool fingerprint groups by tool
*instead of* by error class, so every distinct failure mode within one tool collapses
into a single fault. That is the right trade here — "which tool is broken" is the
question this data exists to answer, and `honeybadger.Context` retains the detail —
but it is a trade, not a free win.

## Configuration

| Variable | Purpose |
|---|---|
| `HONEYBADGER_API_KEY` | Analytics/error destination. Bound via viper alongside the existing `BindEnv` calls (`cmd/honeybadger-mcp-server/main.go:137-147`). |
| `HONEYBADGER_ENV` | Passed as `Configuration.Env`. The library returns `""` when unset, so we substitute `production` ourselves — staging traffic must not pollute the data. |

`Config` gains `HoneybadgerAPIKey`. When the key is unset, the sink is `nopSink`
and startup logs one INFO line stating analytics is off, so a misconfigured deploy is
visible rather than silently mute.

### Activation invariant

honeybadger-go initializes `DefaultClient = New(Configuration{})` at package-var init
(`honeybadger.go:14`), and `newConfig` seeds `APIKey` from ambient
`HONEYBADGER_API_KEY` (`configuration.go:90`). `update` only overwrites non-empty
values (`configuration.go:38-40`), so an empty key leaves the ambient one in place.

Because we use the conventional variable name, a stdio user's own key *is* our
configured key. The hazard is not phone-home to us; it is the inverse — writing our
telemetry into the customer's project, against their quota. A non-empty-key check is
therefore necessary but **not sufficient**: it is satisfied by the customer's key just
as well as ours. And honeybadger-go v0.9.0 has no enable/disable flag for Insights
(only a `BeforeEvent` handler returning `ErrEventDropped`, `client.go:125-133`).

The transport gate is what actually separates our traffic from theirs, so it is
specified as a tested invariant:

1. The emitter is real only when `TransportMode == HTTP` **and** the key is non-empty.
   stdio always gets `nopSink`, whatever the key contains.
2. Package-level `honeybadger.Notify` / `Event` / `Configure` are never referenced.
   Only our own client instance, constructed inside the gate. Note that importing
   honeybadger-go always constructs `DefaultClient` at package-var init
   (`honeybadger.go:12`) — that is unavoidable and harmless on its own. The invariant
   is that we never construct the *dedicated telemetry client* outside the gate, and
   never route anything through `DefaultClient`.
3. A test asserts that constructing a server in stdio mode with `HONEYBADGER_API_KEY`
   set yields a no-op emitter and builds no telemetry client.

Rule 3 is what prevents a future `honeybadger.Notify` call from quietly shipping a
customer's data to their own project.

## Failure handling

- `Emit` swallows errors by construction. An earlier draft specified a `sync.Once`
  WARN on first emission failure; that is not implementable and has been dropped.
  With `Sync == false`, `Client.Event` discards the worker result and returns `nil`
  (`client.go:135`), queue overflow is invisible to the caller
  (`events_worker.go:70`), and backend failures happen later inside the worker
  (`events_worker.go:129`). The only failures `Emit` can observe are synchronous
  `BeforeEvent` rejections, which we do not register. Delivery-failure visibility
  therefore comes from the library's own drop logging, governed by
  `EventsDropLogInterval` (default 60s) — which is worth surfacing in the deploy's
  log configuration, since it is the sole signal that events are being lost.
- The decorator emits from a `defer`, so a panicking handler still produces an event
  with `outcome: "panic"`. Recovery middleware wraps the registered handler — and
  therefore wraps our decorator (`mcp-go/server/server.go:402,1996`) — so the `defer`
  runs during unwinding as intended. The decorator must `recover()` to observe the
  panic and then re-panic, so that `server.WithRecovery()` still handles it and
  behavior is unchanged.
- `Flush()` is called on the existing graceful-shutdown path
  (`cmd/honeybadger-mcp-server/main.go:356-375`). Without it, every deploy drops the
  last unflushed batch — on a frequently deployed service that is a systematic
  undercount, not a rounding error.
- Delivery robustness is inherited from honeybadger-go's `EventsWorker`: batching, a
  bounded queue that drops on overflow, and bounded retries.

## Testing

| Area | Approach |
|---|---|
| `instrumenter.wrap` | Recording fake emitter; table test over every outcome; argument-name allowlisting, sorting, and `unknown_arg_count`; identity extraction; absent optional claims; panic observed and re-panicked. |
| Activation invariant | stdio + `HONEYBADGER_API_KEY` set yields `nopSink` and builds no telemetry client. Importing honeybadger-go always constructs its package-level `DefaultClient`; the invariant is that nothing routes through it. |
| Upstream recorder | `httptest` server returning 200 / 4xx / 5xx / connection failure; assert outcome classification, sticky failure, and that the replacement client preserves the 30s timeout. |
| `hbSink` | Point `Configuration.Endpoint` at an `httptest.Server` with `Sync: true` and assert the posted NDJSON. Note: `Configuration.Backend` is *not* usable for this — `Backend.Event` takes `[]*eventPayload`, an unexported type, so no external package can implement the interface. |
| Claims | Extend `claims_test.go` for the new fields, plus a regression test that a token lacking `account_id` / `client_id` / `project_id` still validates. |
| Existing suite | Default `nopSink` keeps every existing test green without a key. |

## Open items

None. The `project_id` question is resolved by recording `project_scoped` instead of
the raw ID.

## Review history

This spec was reviewed adversarially against the cited sources before implementation
planning. The review overturned several claims in the first draft, which are corrected
above rather than quietly removed:

- The tool count was wrong (a `grep -c` matched `serve`**`r.AddTool(`** and included
  test files). 33 registrar tools plus `search_tools`, not 39.
- The slice-aliasing hazard used to justify leaving `search_tools` registration alone
  did not exist. The conclusion stands on simpler grounds.
- **Outcome classification did not work at all**, because handlers return upstream
  failures as `IsError: true, nil`. This drove the addition of the upstream recorder.
- `arg_names` was neither cardinality-bounded nor guaranteed PII-free, since argument
  keys are caller-controlled and unvalidated. Hence the schema allowlist.
- `project_id` is raw, contradicting the original blanket privacy claim. Resolved in
  the simplification pass by recording `project_scoped` instead.
- The `sync.Once` emission-failure warning was unimplementable against an async
  worker.
- Fingerprint grouping was overstated, and its real tradeoff was unstated.

A subsequent simplification pass then cut the following, with no loss against the
three goals:

- `client_name`, `client_version`, and `session_id` — the hosted server runs stateless
  by default (`main.go:72`), so all three would be empty in substantially every
  production event. `client_id` from the JWT carries client attribution instead.
- The separate `Notifier` interface, merged into `Sink`.
- Raw `project_id`, replaced by the `project_scoped` boolean.
- The worst-outcome ranking lattice and `upstream_ms` summing, replaced by a flat
  record with a sticky-failure rule, after verifying that every handler makes exactly
  one upstream call.

The pass also surfaced an omission: `WithHTTPClient` replaces api-go's default client
and would silently drop its 30-second timeout.
