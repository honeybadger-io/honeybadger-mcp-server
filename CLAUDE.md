# Contributor notes

## Adding a new MCP tool

Every tool registered with the server must declare, in this order inside
`mcp.NewTool(...)`:

1. `mcp.WithTitleAnnotation("Title Case Name")` — a short, human-readable title.
   Required by the Anthropic Connectors Directory; a missing title is an
   automatic review rejection.
2. `mcp.WithDescription(...)` — what the tool does.
3. `mcp.WithReadOnlyHintAnnotation(...)` and `mcp.WithDestructiveHintAnnotation(...)`
   — both are required on every tool. Read-only tools: `ReadOnly(true)` /
   `Destructive(false)`. Tools that create, update, or delete: `ReadOnly(false)`
   / `Destructive(true)`.

`TestAllToolsHaveTitleAndAnnotations` in `internal/hbmcp/server_test.go` fails
the build if any tool (including hidden aliases) is missing a title,
`readOnlyHint`, or `destructiveHint`. Run `go test ./...` before committing.

Tools live in `internal/hbmcp/` grouped by domain (`faults.go`, `alarms.go`,
`projects.go`, `dashboards.go`, `checkins.go`, `insights.go`, `reference.go`)
and are registered from `internal/hbmcp/server.go`.
