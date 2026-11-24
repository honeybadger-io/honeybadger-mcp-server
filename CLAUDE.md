# Contributor notes

## Tool definitions come from the spec

Tool definitions are generated, not written in Go: `make generate` builds
`internal/hbmcp/zz_generated_tools.go` (and README's tool reference) from the
vendored spec, `openapi/bundled.yaml`, and the overlay, `openapi/tools.yaml`.
The spec owns each tool's shape; the overlay holds titles, descriptions and the
few deliberate differences, each with its reason (the header of
`openapi/tools.yaml` lists them). Don't edit the generated file or the bundle.

- **Spec update:** `make update-spec` copies honeybadger's bundle, then
  `make generate`. Generation fails naming each tool or parameter that needs
  attention, such as a new field with no description.
- **New tool for an operation:** add its overlay entry (operation, title,
  description, a description per parameter), run `make generate`, add the
  operation to `toolOperations` in `toolscopes.go`, and register the handler
  with `r.AddTool(generatedTool("name"), handler)`. Read-only and destructive
  hints come from the operation's scopes.
- **Handlers** read only what the definition declares. Body arguments decode
  into the api-go params with `bodyFrom`. `TestEveryAdvertisedParameterReachesTheRequest`
  fails for any advertised parameter a handler doesn't send.
- **Every call is checked** against the tool's schema before its handler runs
  (`strictArgs`); handlers don't re-check types or unknown keys.

## A tool no operation fits

Mark it `handwritten: true` in the overlay and declare it in Go with
`mcp.NewTool(...)`, in this order:

1. `mcp.WithTitleAnnotation("Title Case Name")` — a short, human-readable title.
   Required by the Anthropic Connectors Directory; a missing title is an
   automatic review rejection.
2. `mcp.WithDescription(...)` — what the tool does.
3. `mcp.WithReadOnlyHintAnnotation(...)` and `mcp.WithDestructiveHintAnnotation(...)`
   — both are required on every tool. Read-only tools:
   `mcp.WithReadOnlyHintAnnotation(true)` / `mcp.WithDestructiveHintAnnotation(false)`.
   Tools that create, update, or delete:
   `mcp.WithReadOnlyHintAnnotation(false)` / `mcp.WithDestructiveHintAnnotation(true)`.

`TestAllToolsHaveTitleAndAnnotations` in `internal/hbmcp/server_test.go` fails
the build if any tool is missing a title, `readOnlyHint`, or `destructiveHint`,
and `TestToolsComeFromTheOverlay` fails for a tool the overlay doesn't list.

## Delete tools

Delete tools must not delete on the first call. Set `confirm: true` in the
overlay (it adds the `confirm` parameter and note), and gate the handler on
`deletionConfirmed(...)`; when it returns false, look up the resource and return
`deletionPreview(...)` so the user sees what will be deleted. Pass the same tool
name and ids to both (see `handleDeleteProject` in `projects.go`).
`destructiveHint` is only a hint to the client; it doesn't stop the call.
`TestDeleteToolsRequireConfirmation` in `internal/hbmcp/confirm_test.go` fails
for any `delete_*` tool that deletes without a valid token or isn't listed in
its `deleteToolArgs`.

Run `GOWORK=off go test ./...` before committing.

Handlers live in `internal/hbmcp/` grouped by domain (`faults.go`, `alarms.go`,
`projects.go`, `dashboards.go`, `checkins.go`, `insights.go`, `streams.go`,
`reference.go`) and are registered from `internal/hbmcp/server.go`.
