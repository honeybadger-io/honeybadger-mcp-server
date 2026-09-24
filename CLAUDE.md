# Contributor notes

## Adding a new MCP tool

Every tool registered with the server must declare, in this order inside
`mcp.NewTool(...)`:

1. `mcp.WithTitleAnnotation("Title Case Name")` — a short, human-readable title.
   Required by the Anthropic Connectors Directory; a missing title is an
   automatic review rejection.
2. `mcp.WithDescription(...)` — what the tool does.
3. `mcp.WithReadOnlyHintAnnotation(...)` and `mcp.WithDestructiveHintAnnotation(...)`
   — both are required on every tool. Read-only tools:
   `mcp.WithReadOnlyHintAnnotation(true)` / `mcp.WithDestructiveHintAnnotation(false)`.
   Tools that create, update, or delete:
   `mcp.WithReadOnlyHintAnnotation(false)` / `mcp.WithDestructiveHintAnnotation(true)`.

4. Delete tools must not delete on the first call. Declare `withConfirmParam()`,
   append `confirmNote` to the description, and gate the handler on
   `deletionConfirmed(...)`; when it returns false, look up the resource and
   return `deletionPreview(...)` so the user sees what will be deleted. Pass the
   same tool name and ids to both (see `handleDeleteProject` in `projects.go`).
   `destructiveHint` is only a hint to the client; it doesn't stop the call.
   `TestDeleteToolsRequireConfirmation` in
   `internal/hbmcp/confirm_test.go` fails for any `delete_*` tool that deletes
   without a valid token or isn't listed in its `deleteToolArgs`.

`TestAllToolsHaveTitleAndAnnotations` in `internal/hbmcp/server_test.go` fails
the build if any tool (including hidden aliases) is missing a title,
`readOnlyHint`, or `destructiveHint`. Run `go test ./...` before committing.

Tools live in `internal/hbmcp/` grouped by domain (`faults.go`, `alarms.go`,
`projects.go`, `dashboards.go`, `checkins.go`, `insights.go`, `streams.go`, `reference.go`)
and are registered from `internal/hbmcp/server.go`.
