# Spec-generated tool definitions with strict arguments

Status: draft for Kevin's approval. Date: 2026-11-20. Branch: `strict-tool-args`.

## Intent

When the v3 OpenAPI spec changes in honeybadger, updating the MCP server should
be as mechanical as updating api-go, terraform and the docs: vendor the new
bundle, run `make generate`, read the diff. Today the tool definitions are
written by hand, so every spec change starts with research into what the MCP
should change, and things slip. The survey for this design found spec fields the
MCP never exposed (project `team_id`, `customer_throttle`, `asana_workspace_id`;
`created_after` on outages and uptime checks; paging on fault comments;
`unmapped` on notices; `schedule_type` on `update_check_in`). All were in the
spec a month before the v3 migration, and nothing records leaving them out:
they were never added when the v2 tools were moved onto v3.

The same work closes a second leak. The MCP drops unknown, misspelled,
wrong-typed and stale arguments silently: `update_fault` with `{"resolve": true}`
next to a valid assignee reports success without resolving. The v3 API refuses
unknown fields with a 422, but api-go can't send them, so the MCP is the one
layer that hides an agent's mistake. Every call is now checked against the tool's
definition before the handler runs, and refused with a message the agent can act
on. Kevin's position: Postel's law is the wrong default when the caller is an
AI agent.

## Goals

1. Tool input schemas are generated from a vendored copy of the v3 bundle plus a
   hand-written overlay. The spec owns the shape (names, types, enums,
   nullability, required, nesting); the overlay owns the words and the few
   deliberate differences.
2. A spec field is exposed by default. Leaving one out takes an exclusion with a
   written reason.
3. After a spec update, `make generate` either succeeds with a reviewable diff
   or fails naming exactly what needs a human: a new field without a
   description, an override that no longer matches the spec, a field the pinned
   api-go doesn't have.
4. Every tool call is checked against the tool's schema before its handler runs.
   Refusals name the field and the fix.
5. Descriptions are terse and written for agents. Detail lives in `get_reference`
   topics.

## Non-goals

- Generating handlers. Handlers stay hand-written Go: previews, confirmation,
  date conversion and combining operations are code, not generated.
- Changing api-go. The MCP doesn't read the spec through api-go and doesn't need
  an api-go release to see a spec change.
- Preview operations (`x-preview`). The default bundle doesn't contain them; see
  "Vendoring".
- An ignore-list for keys some client injects. Add one only when the refusal log
  shows a real client doing it.
- Enabling mcp-go's `WithInputSchemaValidation`.

## Anti-goals (failure even if it works)

- A tool is renamed. Tool names are public: the Connectors Directory listing,
  skills and clients call them by name. Every one of today's 56 names survives.
- The overlay turns into a second hand-written schema. If overrides keep
  growing, fix the spec or the API first (the rule from the terraform `specgen`
  work: a generator holds extraction plus structural checks).
- An argument is silently dropped anywhere on the path from the client to the
  API.
- Refusals so generic the agent can't fix the call (mcp-go's
  `input schema validation failed: at '': additional properties 'resolve' not allowed`).

## Constraints

- Vendoring follows the terraform provider exactly: `make update-spec` copies
  `../../honeybadger/openapi/v3/bundled.yaml` (or `$HONEYBADGER_DIR`), records
  branch, commit, date and sha256 in `openapi/README.md`, and a check fails if the
  file was edited by hand. Vendor from the same honeybadger commit as the other
  consumers when a change spans them.
- Generated code refers to api-go's types by name, so `GOWORK=off go build`
  fails when the vendored spec and the pinned api-go disagree on a field. As in
  terraform's `check-spec.sh`, the bundle is not compared with api-go's bundle:
  reworded spec text must not need an api-go release.
- `GOWORK=off go test ./...` passes; CI runs `make generate` and fails on a
  diff.
- CLAUDE.md rules still hold: every tool has a title, `readOnlyHint` and
  `destructiveHint`; delete tools gate on `confirm`.
- `get_account_usage` is being added by hand in the `account-usage-tool` session.
  Whichever lands second rebases. Its operation `getAccountUsage` is in the spec,
  so it becomes a generated tool with no overrides.

## Decided by Kevin

- Generate the definitions from the spec, vendored in this repo, modelled on the
  terraform provider. Don't tie MCP spec updates to api-go releases.
- Include spec fields by default; exclusions carry a reason.
- The overlay supplies the descriptions, written for agents, separate from the
  human docs.
- Descriptions are terse by default; the agent reads `get_reference` topics for
  more.
- Nested schemas can be overridden to advertise a shallow type and point to a
  reference topic.
- Strict enforcement is part of this work, with its own messages. From the
  original brief, still standing:
  - one shared check wraps registration and reads the same schema `tools/list`
    advertises
  - unknown key: refused, naming field and tool, with did-you-mean and the
    stale-tools hint
  - `staleSchemaFields` folds in, so v2-era names keep their specific reason
  - type mismatch: refused, naming field, expected type and received type; null
    accepted only where the schema allows it
  - a number for an ID: dedicated message (IDs are strings; use the list tool;
    reconnect if stale)
  - `WithStrictInputSchemaDefault` (generated schemas also set
    `additionalProperties: false` themselves)
  - not `WithInputSchemaValidation`: it runs before tool middleware and its
    generic message would replace ours
  - log every refusal (tool, field, reason, client name and version) for the
    first release
- Date filters stay an override: the spec takes Unix seconds, the MCP takes an
  RFC 3339 timestamp or a bare date and converts. This dates from #26/#29
  (Feb 2026) and #54.

## Design

### Files

```
openapi/bundled.yaml          vendored bundle (never edited by hand)
openapi/README.md             provenance: branch, commit, date, sha256
openapi/tools.yaml            the overlay
scripts/update-spec.sh        copy + provenance (terraform's script, adapted)
scripts/check-spec.sh         sha256 check
internal/toolgen/             the generator (package + cmd)
internal/hbmcp/zz_generated_tools.go   output: one mcp.Tool per tool, plus api-go field checks
```

`make update-spec`, `make check-spec`, `make generate` (runs check-spec, then
the generator via `go generate`).

### From bundle to tool

For each tool in the overlay:

1. Look up its operation(s) by operationId.
2. Collect path parameters, query parameters and request-body properties into
   one flat argument object. Resolve `$ref` and `allOf`; drop `readOnly`
   properties. Required = required path and query parameters plus the body's
   `required` list.
3. Convert each schema to JSON Schema (OpenAPI 3.1 schemas already are): keep
   `type`, `enum`, `items`, `properties`, `additionalProperties`, nullability,
   `minimum`/`maximum`, `minLength`/`maxLength`, `pattern`; inline references;
   strip OpenAPI-only keys (`example`, `x-*`, `readOnly`, `discriminator`
   mapping noise). Every object gets `additionalProperties: false` unless the
   spec gives a schema for it (e.g. `request_headers`: values are strings).
4. Apply the overlay.
5. Annotations from the operations' scopes (`apiv3.OperationScopes`): read-only
   and not destructive when every operation needs only a `:read` scope;
   otherwise not read-only and destructive. Not from the HTTP method:
   `runInsightsQuery` is a POST that only reads, and read-only mode would hide
   `query_insights`. A test checks the derived annotations equal today's for
   every tool. Title from the overlay. (My call.)
6. Emit `zz_generated_tools.go`: the `mcp.Tool` values (full input schema as
   `RawInputSchema`), and for each body field a compile-time reference to the
   api-go params field it maps to (`var _ = apiv3.SiteUpdateParams{}.Frequency`
   or equivalent), so a mismatch fails the build.

Registration becomes `r.AddTool(generatedTools["update_site"], handleUpdateSite)`.
A test fails if a generated tool has no handler, a handler has no generated tool,
or the overlay names a tool whose operation isn't in `toolOperations` (the scope
map stays, and the generator checks it against the overlay instead of
duplicating it).

### The overlay

```yaml
list_faults:
  operation: listFaults
  title: List Faults
  description: "List a project's faults. Search syntax: get_reference topic errors."
  params:
    project_id: { description: "Project ID, from list_projects." }
    q:          { description: "Search query." }
    occurred_after:
      as: date
      description: "Only faults that occurred after this time (RFC 3339 or YYYY-MM-DD)."
      reason: "Agents write dates reliably and epoch seconds unreliably. Converted in the handler."

create_dashboard:
  operation: createDashboard
  params:
    widgets:
      shallow: dashboards
      description: "The dashboard's widgets."

delete_site:
  operation: deleteSite
  confirm: true

update_fault:
  handwritten: true   # MCP-shaped: resolved/ignored booleans, snooze null = unsnooze

get_project_occurrence_counts:
  operations: [getProjectOccurrences, listAccountOccurrences]   # project_id optional: absent = whole account

get_reference:
  handwritten: true
```

`update_fault` stays hand-written (my call). Its nine operations don't merge
into one schema: `assignee_id` is nullable in one and required in another,
`snooze`/`snooze_occurrences` are the spec's `time`/`count` with null meaning a
different operation, and a merge would pull in the bulk fields (`fault_ids`,
`q`, `all`, `dry_run`). A hand-written definition is still strict-checked, but
the generator can't report its drift. The bulk fields stay out of scope.

Override kinds, and the only ones:

| Key | Effect | Fails generation when |
| --- | --- | --- |
| `description` | tool or parameter text | missing on any tool or exposed parameter; over the length cap without `long: <reason>` |
| `exclude: <reason>` | leave a spec field out | the field isn't in the spec |
| `rename: <name>` + `reason` | MCP name for a spec field | the field isn't in the spec |
| `as: date` + `reason` | MCP type converted in code | the field isn't in the spec, or isn't a Unix-seconds number |
| `shallow: <topic>` | advertise only the top-level type; description ends `Shape: get_reference topic <topic>.` | the topic doesn't exist in `get_reference` |
| `confirm: true` | adds the `confirm` parameter and the confirm note | the operation isn't a DELETE |
| `operations: [...]` | several operations, parameters merged; a parameter is required only if every operation requires it | two operations give one name different schemas |
| `handwritten: true` | definition stays in Go; still strict-checked; operations, if any, stay in `toolOperations` for scopes | — |

`shallow` advertises less than it enforces: the check validates the full nested
schema from the spec, and a nested refusal names the path and the topic. The
advertised type never contradicts the enforced one.

Length caps (my call): 300 characters for a tool description, 120 for a
parameter. Over the cap needs `long: <reason>`.

### Strict check

- At registration, compile each tool's input schema (generated or hand-written)
  with `santhosh-tekuri/jsonschema/v6`, already a dependency through mcp-go.
- `toolRegistrar.AddTool` wraps the handler: validate the arguments; on failure,
  translate the validator's error kinds into the messages below, log, and return
  a tool error without calling the handler. `search_tools` registers through the
  same wrapper.
- `staleSchemaFields` becomes a map of tool → field → reason, consulted when an
  unknown key matches. The handler calls to `rejectStaleSchemaFields`, and the
  `refuseNonStrings` calls the check makes redundant, are removed.
- Log at WARN: tool, field, kind (`unknown`, `stale`, `type`, `null`, `id_number`,
  `nested`), client name and version from the session if available.

Messages (exact):

```
update_fault has no parameter "resolve". Did you mean "resolved"? The call was not made. If you expected "resolve" to exist, your tool list may be stale; reconnect to refresh tool schemas.

update_fault has no parameter "colour". Its parameters are: assignee_id, fault_id, ignored, project_id, resolve_on_deploy, resolved, snooze, snooze_occurrences. The call was not made. If you expected "colour" to exist, your tool list may be stale; reconnect to refresh tool schemas.

list_projects no longer accepts "account_id": v3 takes the account from the credential. The call was not made. Your tool list is stale; reconnect to refresh tool schemas.

update_site: frequency must be one of 1, 5, 15 or null; got a string ("60"). The call was not made.

update_site: name can't be null; omit it to leave the setting unchanged. The call was not made.

list_deploys: limit must be a whole number; got 2.5. The call was not made.

get_site: project_id must be a string; got the number 12345. IDs are strings now: call list_projects for the current one. If your tool definitions show project_id as a number, they are stale; reconnect to refresh tool schemas.

create_dashboard: widgets[0].chart has no field "colour". The call was not made. See get_reference topic "dashboards".

update_site refused these arguments; the call was not made:
- frequncy: no such parameter. Did you mean "frequency"?
- timeout: must be a whole number or null; got a string ("30").
If you expected a parameter to exist, your tool list may be stale; reconnect to refresh tool schemas.
```

Did-you-mean: the closest declared name within edit distance 2. ID message:
applies to `id` and `*_id`; names `list_<resource>s` when that tool is
registered, otherwise "call the tool that lists them".

## Behaviour changes callers will see

Breaking, at the parameter level (no tool is renamed). The strict check refuses
the old forms with a message naming the new one.

- `get_project`, `update_project`, `delete_project`: `id` → `project_id` (spec
  name). My call.
- `list_faults`: `limit` → `per_page` (spec name). My call; the alternative is a
  `rename` override.
- `trigger_config`, `widgets`, `config`: a JSON object or array, no longer a
  string holding JSON. `stream_ids` (alarms): an array or null, no longer a
  string. `widgets` and `trigger_config` are `shallow`.
- `fault_id`: string only; a number gets the ID message.
- Integer parameters are typed `integer`; numeric strings ("30") are refused
  everywhere (`purge_days`, `created_before`, `page`, `limit`).
- Every argument not in the definition is refused.

Additive, from include-by-default (each needs a description or an exclusion on
the first run): the drift fields listed under Intent.

Excluded on the first run, with reasons (my calls):

- `page`/`per_page` on tools whose handler fetches every page (`list_projects`,
  `list_sites`, `list_check_ins`, `list_dashboards`, `list_streams`,
  `list_ingestion_keys`, `get_project_integrations`,
  `get_project_occurrence_counts`): "the tool returns every page".

Descriptions: today's long descriptions are cut to the caps, with detail moved
into reference topics where a topic doesn't already carry it. Separate plan step.

## Vendoring and previews

`make update-spec` copies the default bundle, without preview operations, the
same as terraform (my call). A preview operation reaches the MCP when it leaves
preview.

The messages above are the exact wording and real cases; tests use them as
fixtures.

## Open questions for Kevin

1. **Every advertised field must reach the request?** Two gaps the api-go
   compile check doesn't cover:
   - a body field the hand-written handler doesn't copy: generated, accepted,
     dropped by the handler
   - a query parameter api-go has no `Option` for (`sort` on `listFaults`,
     `unmapped` on `listNotices` in v0.12.0): advertised, accepted, never sent

   Proposed: handlers decode body arguments generically into the api-go params
   struct (as `siteBody`/`decodeSettings` already do), so a new body field needs
   no handler edit; and a generated wire test gives every advertised parameter a
   sample value from its schema, calls the handler against a fake API, and fails
   naming `<tool>.<param>` for any that doesn't appear in the request. A query
   parameter api-go can't send is excluded with the reason "api-go has no
   option", until an api-go release adds one.
2. Commit this spec to the repo? The repo has no `docs/` folder yet.

## Left to the builder

- Generator internals, file layout inside `internal/toolgen`, YAML library.
- How the compile-time api-go field references are expressed.
- How the handlers read arguments once the definitions are generated (they keep
  their current helpers where those still fit).
- Updating tests that assert today's messages and leniencies.
- README's per-tool parameter lists (`## Tools`): regenerate from the same
  definitions or replace with a pointer to `tools/list`. My lean: generate them,
  since they drift the same way.

## How we'll know it works

Tests:

- A table test over every registered tool: an unknown key is refused by name;
  each declared type is refused when given another; null accepted only where
  allowed; every stale field gets its reason and the reconnect hint.
- `update_fault` with `resolve` next to a valid `assignee_id`, and `update_site`
  with `frequncy` next to a valid `name`: refused, no request made.
- A numeric `project_id` gets the ID message.
- Generator tests with a small fixture bundle: a new body field without a
  description fails; an exclusion of a field not in the spec fails; a `shallow`
  pointing at a missing topic fails; a renamed spec field breaks the api-go field
  check.

End to end, for real:

1. `make update-spec && make generate` against a honeybadger checkout; `git diff`
   shows only expected changes.
2. Add a throwaway field to a body schema in a local honeybadger checkout,
   re-bundle, `make update-spec && make generate`: generation fails naming
   `<tool>.<field>: no description`. Revert.
3. Run the server (`make build`, stdio), send `tools/list` over JSON-RPC: tools
   have `additionalProperties: false`, `update_site.frequency` is
   `[integer, null]` with the spec's enum, `create_dashboard.widgets` is shallow
   with the topic pointer.
4. `tools/call update_fault {"project_id": "...", "fault_id": "...", "resolve": true}`
   returns the did-you-mean refusal, and the log shows the WARN line. The same
   call with `resolved` resolves the fault on the local dev stack.
5. `GOWORK=off go test ./...` and `change-checks` pass.
