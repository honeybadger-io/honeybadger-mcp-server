# MCP Server Usage Analytics — Design

**Date:** 2026-08-06
**Status:** Approved, ready for implementation planning

## Problem

The MCP server has no usage analytics. The only observability is `slog` text to stderr:
session register/unregister (session ID only), request errors, and a debug-level
`Processing request` line that records the MCP method but not which tool was called
(`internal/hbmcp/server.go:46-58`).

Three questions are unanswerable today:

1. **Which tools get used.** The catalog is 39 tools registered through the registrar,
   plus `search_tools`. We cannot tell which earn their place, which are dead weight,
   or whether the discovery layer is used at all.
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

Three components, each independently testable.

### `internal/analytics` (new package)

Owns the destination; knows nothing about MCP.

```go
type Event struct {
    Type string
    Data map[string]any
}

type Emitter interface {
    Emit(Event)
}
```

`Emit` returns nothing by design — a caller must not be able to fail a tool call by
emitting. Two implementations: `hbEmitter` wrapping a honeybadger-go client, and
`nopEmitter`.

Because the interface is this small, `hbmcp` tests use a recording fake and never
touch the network.

### `instrumented` decorator (in `hbmcp`)

```go
func instrumented(em analytics.Emitter, name string, h server.ToolHandlerFunc) server.ToolHandlerFunc
```

Times the call, classifies the outcome, reads identity from the context, emits, and
returns the handler's result unchanged.

`toolRegistrar` holds an `Emitter` and applies the decorator in `AddTool`
(`internal/hbmcp/tool_search.go:30`). `registerSearchTool` receives the same emitter
and wraps its own handler at its own registration site.

**`search_tools` registration is deliberately left as-is.** It is registered directly
against the server (`internal/hbmcp/server.go:85`) because it takes the completed
catalog by value and must run after every other registration. It also excludes itself
from what it searches — note the asymmetry with `server.go:87`, where the landing-page
catalog is `append(r.catalog, searchToolInfo)`, *with* `search_tools`. Routing it
through the registrar would append it to `r.catalog` while the handler holds an
earlier by-value slice header. That happens to work today and would break silently
the first time someone reorders registration. Keeping `instrumented` a standalone
function avoids the restructuring entirely.

### Wiring in `NewServerWithCatalog`

The emitter is selected once at construction. See "Activation invariant" below.

## Event schema

One event type, `mcp.tool_call`, emitted once per handler invocation.

| Field | Source | Notes |
|---|---|---|
| `tool` | registration name | |
| `outcome` | see classification | |
| `duration_ms` | measured | |
| `arg_names` | request arguments | names only, sorted |
| `user_id` | `sub` claim | hashid |
| `account_id` | `account_id` claim | hashid, optional |
| `client_id` | `client_id` claim | OAuth app uid, optional |
| `project_id` | `project_id` claim | optional |
| `read_only` | `EffectiveReadOnly(ctx, cfg)` | effective scope, not the startup flag |
| `server_version` | build version | |
| `error_id` | notice token | `internal_error` only, omitted when empty |
| `client_name` / `client_version` | MCP `initialize` | best-effort, often absent |
| `session_id` | `ClientSessionFromContext` | best-effort, often absent |

### Argument capture

Names only, never values. Records which parameters were supplied — answering whether
pagination and filter arguments are used, and catching malformed calls — with no
customer data in the stream. Sorted, so cardinality stays stable.

### Outcome classification

The handler signature already makes the distinction that matters:

| Return | Outcome | Meaning |
|---|---|---|
| `(result, nil)`, `IsError == false` | `ok` | |
| `(result, nil)`, `IsError == true` | `tool_error` | Call reached us and we rejected it — e.g. `mcp.NewToolResultError("query is required")` (`tool_search.go:72`). A spike in one tool means its schema or description needs work. |
| `(_, err)` | `internal_error` | Go-level failure: our bug or the API misbehaving. |
| panic | `panic` | Emitted from a `defer`; `server.WithRecovery()` (`server.go:64`) still handles the panic itself. |

Collapsing `tool_error` and `internal_error` into a single "error" would destroy the
value of the metric: the first is a product signal, the second an outage signal.

## Identity

The Honeybadger AS mints these claims
(`../honeybadger/config/initializers/doorkeeper.rb:114-129`):

```ruby
{ ver:, iss:, sub:, iat:, exp:, jti:, client_id:, scope:, account_id:, project_id:, aud: }.compact
```

`sub` is `User.encode_id` and `account_id` is `Account.encode_id` — both hashids
(`hashid-rails`), so they are already opaque. No raw database IDs, no PII, and nothing
for us to hash.

`client_id` is the OAuth application uid. It identifies the connecting client app more
reliably than MCP's `initialize` clientInfo, because it is present on every request
including in stateless mode, where clientInfo is empty. It is therefore the primary
client dimension; `client_name`/`client_version` are supplementary.

`Claims` gains `Subject`, `AccountID`, `ClientID`, and `ProjectID`.

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
`error_id` for correlation. This costs nothing: `notice.Token` is generated
client-side in `newNotice` before delivery is pushed to the async worker
(`client.go:94-118`), so the token is available immediately without blocking.

Two honest limits:

- `Notify` returns `""` if a beforeNotify handler errors or the worker push fails, so
  `error_id` is omitted when empty.
- The token is minted before delivery, so an `error_id` can reference a notice that
  never landed (worker gave up after retries). Correlation is best-effort.

**Grouping requires an explicit fingerprint.** `newError(err, 2)` uses the error's own
stack only if it implements `Callers()` (`error.go:64-69`). Plain `fmt.Errorf` errors
— what our handlers and api-go return — do not, so the stack is generated from the
decorator's frame. Without intervention every internal error from all 40 tools would
share a top frame and group into one fault. Therefore pass
`honeybadger.Fingerprint{Content: "mcp_tool_call:" + tool}` (`notice.go:171`) so
grouping is per-tool, plus a `honeybadger.Context{}` carrying tool, account, and
client.

## Configuration

| Variable | Purpose |
|---|---|
| `HONEYBADGER_API_KEY` | Analytics/error destination. Bound via viper alongside the existing `BindEnv` calls (`cmd/honeybadger-mcp-server/main.go:137-147`). |
| `HONEYBADGER_ENV` | Passed as `Configuration.Env`. The library returns `""` when unset, so we substitute `production` ourselves — staging traffic must not pollute the data. |

`Config` gains `HoneybadgerAPIKey`. When the key is unset, the emitter is `nopEmitter`
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
   stdio always gets `nopEmitter`, whatever the key contains.
2. Package-level `honeybadger.Notify` / `Event` / `Configure` are never referenced.
   Only our own client instance, constructed inside the gate.
3. A test asserts that constructing a server in stdio mode with `HONEYBADGER_API_KEY`
   set yields a no-op emitter and never constructs a client.

Rule 3 is what prevents a future `honeybadger.Notify` call from quietly shipping a
customer's data to their own project.

## Failure handling

- `Emit` swallows errors by construction. The first failure logs at WARN via a
  `sync.Once`; subsequent failures log at DEBUG, so a misconfigured key does not
  produce one log line per tool call.
- The decorator emits from a `defer`, so a panicking handler still produces an event
  with `outcome: "panic"`.
- `Flush()` is called on the existing graceful-shutdown path
  (`cmd/honeybadger-mcp-server/main.go:356-375`). Without it, every deploy drops the
  last unflushed batch — on a frequently deployed service that is a systematic
  undercount, not a rounding error.
- Delivery robustness is inherited from honeybadger-go's `EventsWorker`: batching, a
  bounded queue that drops on overflow, and bounded retries.

## Testing

| Area | Approach |
|---|---|
| `instrumented` | Recording fake emitter; table test over `ok` / `tool_error` / `internal_error` / `panic`; argument-name extraction and sorting; identity extraction; absent optional claims. |
| Activation invariant | stdio + `HONEYBADGER_API_KEY` set yields `nopEmitter` and constructs no client. |
| `hbEmitter` | honeybadger-go's `Configuration.Backend` is injectable; a fake backend asserts payload shape without network. |
| Claims | Extend `claims_test.go` for the new fields, plus a regression test that a token lacking `account_id` / `client_id` / `project_id` still validates. |
| Existing suite | Default `nopEmitter` keeps every existing test green without a key. |

## Open items

None. The account-claim question is resolved against
`../honeybadger/config/initializers/doorkeeper.rb`.
