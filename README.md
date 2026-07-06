# Honeybadger MCP Server

An MCP (Model Context Protocol) server for [Honeybadger](https://www.honeybadger.io), providing structured
access to Honeybadger's API through the MCP protocol.

## Installation

First, pull the Docker image:

```bash
docker pull ghcr.io/honeybadger-io/honeybadger-mcp-server:latest
```

Then, configure your MCP client(s). You can find your personal auth token under the "Authentication" tab in your [Honeybadger user settings](https://app.honeybadger.io/users/edit#authentication).

### Cursor, Windsurf, and Claude Desktop

Put this config in `~/.cursor/mcp.json` for [Cursor](https://docs.cursor.com/context/model-context-protocol), or `~/.codeium/windsurf/mcp_config.json` for [Windsurf](https://docs.windsurf.com/windsurf/cascade/mcp). See Anthropic's [MCP quickstart guide](https://modelcontextprotocol.io/quickstart/user) for how to locate your `claude_desktop_config.json` for Claude Desktop:

```json
{
  "mcpServers": {
    "honeybadger": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e",
        "HONEYBADGER_PERSONAL_AUTH_TOKEN",
        "ghcr.io/honeybadger-io/honeybadger-mcp-server"
      ],
      "env": {
        "HONEYBADGER_PERSONAL_AUTH_TOKEN": "your personal auth token"
      }
    }
  }
}
```

### Claude Code

Run this command to configure [Claude Code](https://www.anthropic.com/claude-code):

```bash
claude mcp add honeybadger -- docker run -i --rm -e HONEYBADGER_PERSONAL_AUTH_TOKEN="HONEYBADGER_PERSONAL_AUTH_TOKEN" ghcr.io/honeybadger-io/honeybadger-mcp-server:latest
```

### VS Code

Add the following to your [user settings](https://code.visualstudio.com/docs/configure/settings#_settings-json-file) or `.vscode/mcp.json` in your workspace:

```json
{
  "mcp": {
    "inputs": [
      {
        "type": "promptString",
        "id": "honeybadger_auth_token",
        "description": "Honeybadger Personal Auth Token",
        "password": true
      }
    ],
    "servers": {
      "honeybadger": {
        "command": "docker",
        "args": [
          "run",
          "-i",
          "--rm",
          "-e",
          "HONEYBADGER_PERSONAL_AUTH_TOKEN",
          "ghcr.io/honeybadger-io/honeybadger-mcp-server"
        ],
        "env": {
          "HONEYBADGER_PERSONAL_AUTH_TOKEN": "${input:honeybadger_auth_token}"
        }
      }
    }
  }
}
```

See [Use MCP servers in VS Code](https://code.visualstudio.com/docs/copilot/chat/mcp-servers) for more info.

### Zed

Add the following to your Zed settings file in `~/.config/zed/settings.json`:

```json
{
  "context_servers": {
    "honeybadger": {
      "command": {
        "path": "docker",
        "args": [
          "run",
          "-i",
          "--rm",
          "-e",
          "HONEYBADGER_PERSONAL_AUTH_TOKEN",
          "ghcr.io/honeybadger-io/honeybadger-mcp-server"
        ],
        "env": {
          "HONEYBADGER_PERSONAL_AUTH_TOKEN": "your personal auth token"
        }
      },
      "settings": {}
    }
  }
}
```

### Building Docker locally

To build the Docker image and run it locally:

```bash
git clone git@github.com:honeybadger-io/honeybadger-mcp-server.git
cd honeybadger-mcp-server
docker build -t honeybadger-mcp-server .
```

Then you can replace "ghcr.io/honeybadger-io/honeybadger-mcp-server" with
"honeybadger-mcp-server" in any of the configs above. Or you can run the image
directly:

```bash
docker run -i --rm -e HONEYBADGER_PERSONAL_AUTH_TOKEN honeybadger-mcp-server
```

### Building from source

If you don't have Docker, you can build the server from source:

```bash
git clone git@github.com:honeybadger-io/honeybadger-mcp-server.git
cd honeybadger-mcp-server
go build -o honeybadger-mcp-server ./cmd/honeybadger-mcp-server
```

And then configure your MCP client to run the server directly:

```json
{
  "mcpServers": {
    "honeybadger": {
      "command": "/path/to/honeybadger-mcp-server",
      "args": ["stdio"],
      "env": {
        "HONEYBADGER_PERSONAL_AUTH_TOKEN": "your personal auth token"
      }
    }
  }
}
```

## Configuration

### Environment Variables

| Environment Variable              | Required | Default                    | Description                                                             |
| --------------------------------- | -------- | -------------------------- | ----------------------------------------------------------------------- |
| `HONEYBADGER_PERSONAL_AUTH_TOKEN` | yes      | —                          | API token for Honeybadger                                               |
| `HONEYBADGER_READ_ONLY`           | no       | true                       | Run in read-only mode, excluding write operations like `delete_project` |
| `LOG_LEVEL`                       | no       | info                       | Log verbosity (debug, info, warn, error)                                |
| `HONEYBADGER_API_URL`             | no       | https://app.honeybadger.io | Override the base URL for Honeybadger's API                             |

**Important**: The server runs in **read-only mode by default** for security. This means only read operations (like `list_projects`, `get_project`, `list_faults`) are available. Write operations such as `create_project`, `update_project`, and `delete_project` are excluded to prevent accidental modifications.

To enable write operations, explicitly set `HONEYBADGER_READ_ONLY=false`. **Use with caution** as this allows destructive operations like deleting projects.

### EU Region

The server defaults to Honeybadger's US API (`https://app.honeybadger.io`). If your account is in the [EU region](https://docs.honeybadger.io/resources/data-residency/), set `HONEYBADGER_API_URL` to `https://eu-app.honeybadger.io` and use a personal auth token from your [EU user settings](https://eu-app.honeybadger.io/users/edit#authentication). A US token won't authenticate against the EU region, and vice versa.

For example, with Claude Code:

```bash
claude mcp add honeybadger-eu -- docker run -i --rm -e HONEYBADGER_PERSONAL_AUTH_TOKEN="your_eu_token" -e HONEYBADGER_API_URL="https://eu-app.honeybadger.io" ghcr.io/honeybadger-io/honeybadger-mcp-server:latest
```

To use both regions at once, run two servers with distinct names (for example `honeybadger-us` and `honeybadger-eu`), each with its own token and API URL.

### Command Line Options

When running the server via the CLI you can configure the server with command-line flags:

```bash
# Run with custom configuration
./honeybadger-mcp-server stdio --auth-token your_token --log-level debug --api-url https://custom.honeybadger.io

# Enable write operations (use with caution)
./honeybadger-mcp-server stdio --auth-token your_token --read-only=false

# Get help
./honeybadger-mcp-server stdio --help
```

The `--read-only` flag defaults to `true`. Set `--read-only=false` to enable write operations like `create_project`, `update_project`, and `delete_project`.

### Configuration File

You can also use a configuration file at `~/.honeybadger-mcp-server.yaml`:

```yaml
auth-token: "your_token_here"
log-level: "info"
api-url: "https://app.honeybadger.io"
read-only: true
```

## Tools

### Projects

- **list_projects** - List all Honeybadger projects
  - `account_id` : Account ID to filter projects by specific account (string, optional)

- **get_project** - Get detailed information for a single project by ID
  - `id` : The ID of the project to retrieve (number, required)

- **create_project** - Create a new Honeybadger project _(requires `read-only=false`)_
  - `account_id` : The account ID to associate the project with (string, required)
  - `name` : The name of the new project (string, required)
  - `resolve_errors_on_deploy` : Whether all unresolved faults should be marked as resolved when a deploy is recorded (boolean, optional)
  - `disable_public_links` : Whether to allow fault details to be publicly shareable via a button on the fault detail page (boolean, optional)
  - `user_url` : A URL format like 'http://example.com/admin/users/[user_id]' that will be displayed on the fault detail page (string, optional)
  - `source_url` : A URL format like 'https://gitlab.com/username/reponame/blob/[sha]/[file]#L[line]' that is used to link lines in the backtrace to your git browser (string, optional)
  - `purge_days` : The number of days to retain data (up to the max number of days available to your subscription plan) (number, optional)
  - `user_search_field` : A field such as 'context.user_email' that you provide in your error context (string, optional)

- **update_project** - Update an existing Honeybadger project _(requires `read-only=false`)_
  - `id` : The ID of the project to update (number, required)
  - `name` : The name of the project (string, optional)
  - `resolve_errors_on_deploy` : Whether all unresolved faults should be marked as resolved when a deploy is recorded (boolean, optional)
  - `disable_public_links` : Whether to allow fault details to be publicly shareable via a button on the fault detail page (boolean, optional)
  - `user_url` : A URL format like 'http://example.com/admin/users/[user_id]' that will be displayed on the fault detail page (string, optional)
  - `source_url` : A URL format like 'https://gitlab.com/username/reponame/blob/[sha]/[file]#L[line]' that is used to link lines in the backtrace to your git browser (string, optional)
  - `purge_days` : The number of days to retain data (up to the max number of days available to your subscription plan) (number, optional)
  - `user_search_field` : A field such as 'context.user_email' that you provide in your error context (string, optional)

- **delete_project** - Delete a Honeybadger project _(requires `read-only=false`)_
  - `id` : The ID of the project to delete (number, required)

- **get_project_occurrence_counts** - Get occurrence counts for all projects or a specific project
  - `project_id` : Project ID to get occurrence counts for a specific project (number, optional)
  - `period` : Time period for grouping data: 'hour', 'day', 'week', or 'month'. Defaults to 'hour' (string, optional)
  - `environment` : Environment name to filter results (string, optional)

- **get_project_integrations** - Get a list of integrations (channels) for a Honeybadger project
  - `project_id` : The ID of the project to get integrations for (number, required)

- **get_project_report** - Get report data for a Honeybadger project
  - `project_id` : The ID of the project to get report data for (number, required)
  - `report` : The type of report to get: 'notices_by_class', 'notices_by_location', 'notices_by_user', or 'notices_per_day' (string, required)
  - `start` : Start date/time in ISO 8601 format for the beginning of the reporting period (string, optional)
  - `stop` : Stop date/time in ISO 8601 format for the end of the reporting period (string, optional)
  - `environment` : Environment name to filter results (string, optional)

### Faults

- **list_faults** - Get a list of faults for a project with optional filtering and ordering
  - `project_id` : The ID of the project to get faults for (number, required)
  - `q` : Search string to filter faults (string, optional)
  - `created_after` : Filter faults created after this timestamp (string, optional)
  - `occurred_after` : Filter faults that occurred after this timestamp (string, optional)
  - `occurred_before` : Filter faults that occurred before this timestamp (string, optional)
  - `limit` : Maximum number of faults to return (max 25) (number, optional)
  - `order` : Order results by 'recent' or 'frequent' (string, optional)
  - `page` : Page number for pagination (number, optional)

- **get_fault** - Get detailed information for a specific fault in a project
  - `project_id` : The ID of the project containing the fault (number, required)
  - `fault_id` : The ID of the fault to retrieve (number, required)

- **get_fault_counts** - Get fault count statistics for a project with optional filtering
  - `project_id` : The ID of the project to get fault counts for (number, required)
  - `q` : Search string to filter faults (string, optional)
  - `created_after` : Filter faults created after this timestamp (string, optional)
  - `occurred_after` : Filter faults that occurred after this timestamp (string, optional)
  - `occurred_before` : Filter faults that occurred before this timestamp (string, optional)

- **list_fault_notices** - Get a list of notices (individual error events) for a specific fault
  - `project_id` : The ID of the project containing the fault (number, required)
  - `fault_id` : The ID of the fault to get notices for (number, required)
  - `created_after` : Filter notices created after this timestamp (string, optional)
  - `created_before` : Filter notices created before this timestamp (string, optional)
  - `limit` : Maximum number of notices to return (max 25) (number, optional)

- **list_fault_affected_users** - Get a list of users who were affected by a specific fault with occurrence counts
  - `project_id` : The ID of the project containing the fault (number, required)
  - `fault_id` : The ID of the fault to get affected users for (number, required)
  - `q` : Search string to filter affected users (string, optional)

### Insights

- **query_insights** - Execute a BadgerQL query against Insights data
  - `project_id` : The ID of the project to query insights for (number, required)
  - `query` : BadgerQL query string to execute against your Insights data (string, required)
  - `ts` : Time range - shortcuts like 'today', 'week', or ISO 8601 duration (e.g., 'PT3H'). Defaults to PT3H (string, optional)
  - `timezone` : IANA timezone identifier (e.g., 'America/New_York') for timestamp interpretation (string, optional)

- **get_insights_reference** - Returns the Honeybadger Insights reference covering BadgerQL query syntax, available functions, common patterns, shareable URLs, and dashboard configuration. Call this before working with Insights, alarm, or dashboard tools. (no parameters)

### Dashboards

- **list_dashboards** - List all Insights dashboards for a project
  - `project_id` : The ID of the project to list dashboards for (number, required)

- **get_dashboard** - Get a single Insights dashboard by ID
  - `project_id` : The ID of the project the dashboard belongs to (number, required)
  - `dashboard_id` : The ID of the dashboard to retrieve (string, required)

- **create_dashboard** - Create a new Insights dashboard _(requires `read-only=false`)_
  - `project_id` : The ID of the project to create the dashboard in (number, required)
  - `title` : The title of the dashboard (string, required)
  - `widgets` : JSON array of widget objects. Call `get_insights_reference` for the full widget schema and examples. Each widget needs a `type` (`insights_vis`, `alarms`, `errors`, `deployments`, `checkins`, `uptime`) and optionally `grid` ({x,y,w,h}), `presentation` ({title, subtitle}), and `config` (type-specific settings) (string, required)
  - `default_ts` : Default time range for the dashboard. ISO 8601 duration (e.g., P1D, PT3H) or keyword (today, yesterday, week, month) (string, optional)

- **update_dashboard** - Update an existing Insights dashboard _(requires `read-only=false`)_
  - `project_id` : The ID of the project the dashboard belongs to (number, required)
  - `dashboard_id` : The ID of the dashboard to update (string, required)
  - `title` : The title of the dashboard (string, required)
  - `widgets` : JSON array of widget objects (see `create_dashboard`) (string, required)
  - `default_ts` : Default time range for the dashboard (string, optional)

- **delete_dashboard** - Delete an Insights dashboard _(requires `read-only=false`)_
  - `project_id` : The ID of the project the dashboard belongs to (number, required)
  - `dashboard_id` : The ID of the dashboard to delete (string, required)

### Alarms

- **list_alarms** - List all Insights alarms for a project
  - `project_id` : The ID of the project to list alarms for (number, required)

- **get_alarm** - Get a single Insights alarm by ID
  - `project_id` : The ID of the project the alarm belongs to (number, required)
  - `alarm_id` : The ID of the alarm to retrieve (string, required)

- **create_alarm** - Create a new Insights alarm _(requires `read-only=false`)_. Call `get_insights_reference` first for the full alarm documentation, `trigger_config` schema, and query guidelines.
  - `project_id` : The ID of the project to create the alarm in (number, required)
  - `name` : The name of the alarm (string, required)
  - `query` : BadgerQL query for the alarm. The alarm system wraps the query to count results automatically (string, required)
  - `evaluation_period` : How often the alarm is evaluated (e.g., 5m, 1h, 1d). Minimum 1m (string, required)
  - `trigger_config` : JSON object defining when to trigger the alarm, e.g. `{"type": "alert_result_count", "config": {"operator": "gt", "value": 10}}` (string, required)
  - `lookback_lag` : Delay before evaluating to allow data to arrive (e.g., 1m, or 0s for no lag) (string, required)
  - `description` : Optional description of the alarm (string, optional)
  - `stream_ids` : Optional JSON array of stream IDs to query (defaults to `["default"]`) (string, optional)

- **update_alarm** - Update an existing Insights alarm _(requires `read-only=false`)_. Call `get_insights_reference` first for the full alarm documentation.
  - `project_id` : The ID of the project the alarm belongs to (number, required)
  - `alarm_id` : The ID of the alarm to update (string, required)
  - `name` : The name of the alarm (string, required)
  - `query` : BadgerQL query for the alarm (string, required)
  - `evaluation_period` : How often the alarm is evaluated (e.g., 5m, 1h, 1d). Minimum 1m (string, required)
  - `trigger_config` : JSON object defining when to trigger the alarm (string, required)
  - `lookback_lag` : Delay before evaluating to allow data to arrive (e.g., 1m, 0s for no lag) (string, required)
  - `description` : Optional description of the alarm (string, optional)
  - `stream_ids` : Optional JSON array of stream IDs to query (string, optional)

- **delete_alarm** - Delete an Insights alarm _(requires `read-only=false`)_
  - `project_id` : The ID of the project the alarm belongs to (number, required)
  - `alarm_id` : The ID of the alarm to delete (string, required)

- **get_alarm_history** - Get the trigger history for an Insights alarm
  - `project_id` : The ID of the project the alarm belongs to (number, required)
  - `alarm_id` : The ID of the alarm to get history for (string, required)
  - `page` : Page number for pagination (default: 0) (number, optional)

### Tool Search

- **search_tools** - Search available Honeybadger tools by name or description. Use this to discover tools before calling them. In read-only mode, only read-only tools are returned.
  - `query` : Search query to match against tool names and descriptions (string, required)

## Development

### Local Development Setup

This project uses the [`api-go`](https://github.com/honeybadger-io/api-go) library for API interactions. For local development, you'll need to set up a Go workspace to work with both repositories simultaneously.

From the parent directory containing both `honeybadger-mcp-server` and `api-go`:

```bash
# Initialize the workspace (if not already done)
go work init
go work use ./honeybadger-mcp-server
go work use ./api-go

# The go.work file is gitignored and won't be committed
```

Now you can work on both repositories and changes to `api-go` will be immediately reflected when working on the MCP server.

### Working with Dependencies

When using the workspace, Go uses the local `api-go` directory instead of fetching from GitHub. However, `go.sum` must still contain checksums for the published `api-go` module to support:
- CI/CD builds (which don't have the workspace)
- Developers who clone only this repository
- Docker builds

**When to use `GOWORK=off`:**

```bash
# Update dependencies and go.sum with published module checksums
GOWORK=off go mod tidy

# Install a specific version of a dependency
GOWORK=off go get github.com/some/package@v1.2.3

# Test the build as if no workspace exists (simulates CI/end-user builds)
GOWORK=off go build ./...
GOWORK=off go test ./...
```

The `GOWORK=off` flag temporarily disables the workspace, ensuring that `go.sum` contains the correct checksums for the published modules.

### Running Tests

```bash
go test ./...
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add my amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
