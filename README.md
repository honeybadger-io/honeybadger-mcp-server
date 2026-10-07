# Honeybadger MCP Server

An MCP (Model Context Protocol) server for [Honeybadger](https://www.honeybadger.io), providing structured
access to Honeybadger's API through the MCP protocol.

## Installation

First, pull the Docker image:

```bash
docker pull ghcr.io/honeybadger-io/honeybadger-mcp-server:1.5.1
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
        "ghcr.io/honeybadger-io/honeybadger-mcp-server:1.5.1"
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
claude mcp add honeybadger -- docker run -i --rm -e HONEYBADGER_PERSONAL_AUTH_TOKEN="HONEYBADGER_PERSONAL_AUTH_TOKEN" ghcr.io/honeybadger-io/honeybadger-mcp-server:1.5.1
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
          "ghcr.io/honeybadger-io/honeybadger-mcp-server:1.5.1"
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
          "ghcr.io/honeybadger-io/honeybadger-mcp-server:1.5.1"
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

Then you can replace "ghcr.io/honeybadger-io/honeybadger-mcp-server:1.5.1" with
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
| `HONEYBADGER_INSTRUCTIONS_URL`    | no       | https://docs.honeybadger.io/resources/llms/instructions | Override the base URL the LLM reference topics are fetched from |
| `MCP_CONFIRM_SECRET`              | HTTP mode only | —                    | Signs delete confirmation tokens. At least 32 characters, and identical on every instance behind a load balancer. The server won't start in HTTP mode without it; stdio mode doesn't use it |

**Important**: The server runs in **read-only mode by default** for security. This means only read operations (like `list_projects`, `get_project`, `list_faults`) are available. Write operations such as `create_project`, `update_project`, and `delete_project` are excluded to prevent accidental modifications.

To enable write operations, explicitly set `HONEYBADGER_READ_ONLY=false`. **Use with caution** as this allows destructive operations like deleting projects.

### EU Region

The server defaults to Honeybadger's US API (`https://app.honeybadger.io`). If your account is in the [EU region](https://docs.honeybadger.io/resources/data-residency/), set `HONEYBADGER_API_URL` to `https://eu-app.honeybadger.io` and use a personal auth token from your [EU user settings](https://eu-app.honeybadger.io/users/edit#authentication). A US token won't authenticate against the EU region, and vice versa.

For example, with Claude Code:

```bash
claude mcp add honeybadger-eu -- docker run -i --rm -e HONEYBADGER_PERSONAL_AUTH_TOKEN="your_eu_token" -e HONEYBADGER_API_URL="https://eu-app.honeybadger.io" ghcr.io/honeybadger-io/honeybadger-mcp-server:1.5.1
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

Delete tools (`delete_project`, `delete_dashboard`, `delete_alarm`, `delete_check_in`, `delete_fault_comment`, `delete_integration`, `delete_project_key`, `delete_site`) take two calls. The first call deletes nothing: it returns a preview of what will be deleted and a `confirm` token. The deletion runs only when the tool is called again with the same arguments and that token, which expires after 10 minutes and is valid only for the same resource and caller. Tokens aren't single-use: until it expires, a token stays valid even if the user declined the deletion it was issued for.

### Reference

- **get_reference** - Returns Honeybadger reference documentation for LLMs, organized into non-overlapping topics: `badgerql` (query language), `queries` (Insights query fundamentals), `charts` (visualization views, `chart_config`), `dashboards` (widget schema, grid layout), `alarms` (`trigger_config` schema, states, patterns), and `errors` (fault/notice model, error search syntax). Topics are fetched from the [docs site](https://docs.honeybadger.io/resources/llms/instructions/) and cached in memory. Tool descriptions declare which topics they require.
  - `topics` : Reference topics to fetch, e.g. `["badgerql", "charts"]`. Use `["all"]` for everything; omit for an index of topics (array of strings, optional)

### Projects

- **list_projects** - List all Honeybadger projects
  - `name` : Exact project name; returns only that project (string, optional)

- **get_project** - Get detailed information for a single project by ID
  - `id` : The ID of the project to retrieve (string, required)

- **create_project** - Create a new Honeybadger project _(requires `read-only=false`)_
  - `name` : The name of the new project (string, required)
  - `resolve_errors_on_deploy` : Whether all unresolved faults should be marked as resolved when a deploy is recorded (boolean, optional)
  - `disable_public_links` : Whether to allow fault details to be publicly shareable via a button on the fault detail page (boolean, optional)
  - `user_url` : A URL format like 'http://example.com/admin/users/[user_id]' that will be displayed on the fault detail page (string, optional)
  - `source_url` : A URL format like 'https://gitlab.com/username/reponame/blob/[sha]/[file]#L[line]' that is used to link lines in the backtrace to your git browser (string, optional)
  - `purge_days` : The number of days to retain data (up to the max number of days available to your subscription plan) (number, optional)
  - `user_search_field` : A field such as 'context.user_email' that you provide in your error context (string, optional)

- **update_project** - Update an existing Honeybadger project _(requires `read-only=false`)_
  - `id` : The ID of the project to update (string, required)
  - `name` : The name of the project (string, optional)
  - `resolve_errors_on_deploy` : Whether all unresolved faults should be marked as resolved when a deploy is recorded (boolean, optional)
  - `disable_public_links` : Whether to allow fault details to be publicly shareable via a button on the fault detail page (boolean, optional)
  - `user_url` : A URL format like 'http://example.com/admin/users/[user_id]' that will be displayed on the fault detail page (string, optional)
  - `source_url` : A URL format like 'https://gitlab.com/username/reponame/blob/[sha]/[file]#L[line]' that is used to link lines in the backtrace to your git browser (string, optional)
  - `purge_days` : The number of days to retain data (up to the max number of days available to your subscription plan) (number, optional)
  - `user_search_field` : A field such as 'context.user_email' that you provide in your error context (string, optional)

- **delete_project** - Delete a Honeybadger project _(requires `read-only=false`)_
  - `id` : The ID of the project to delete (string, required)
  - `confirm` : Confirmation token from the preview returned by the first call (string, optional)

- **get_project_occurrence_counts** - Get occurrence counts for all projects or a specific project
  - `project_id` : Project ID to get occurrence counts for a specific project (string, optional)
  - `period` : Window to report over: 'hour' (61 one-minute buckets), 'day' (25 hourly), 'week' (8 daily), or 'month' (31 daily). Defaults to 'hour' (string, optional)
  - `environment` : Environment name to filter results (string, optional)

- **get_project_integrations** - Get a list of integrations (channels) for a Honeybadger project
  - `project_id` : The ID of the project to get integrations for (string, required)

- **get_integration** - Get a single notification integration
  - `project_id` : The ID of the project the integration belongs to (string, required)
  - `integration_id` : The ID of the integration (string, required)

- **create_integration** - Create a notification integration _(requires `read-only=false`)_
  - `project_id` : The ID of the project to create the integration in (string, required)
  - `type` : Integration type, such as `WebHook`, `Email`, `PagerDutyV2` or `Slack`. OAuth types start unconnected; the user connects them from the result's `links.web`, and they can be active from the start (string, required)
  - `config` : JSON object of the type's own settings, e.g. `{"url": "https://example.com/hook"}` for `WebHook` (string, optional)
  - `active`, `events`, `rate`, `threshold`, `notification_limit`, `site_ids`, `check_in_ids`, `alarm_alert_ids`, `alarm_ok_ids`, `included_environments`, `excluded_environments`, `filters` : the settings every type shares, as their own parameters rather than inside `config`. `filters` is an ordered list of `{"event", "query"}` objects (`event` can be `all`). `site_ids`/`check_in_ids` take `null` to follow every site or check-in, including ones added later (the default on create), `[]` for none, or a list (optional)

- **update_integration** - Update a notification integration _(requires `read-only=false`)_
  - `project_id`, `integration_id` : Which integration (string, required)
  - `config` and the shared settings as for `create_integration`; only those given change, and the type can't (optional)

- **delete_integration** - Delete a notification integration and its tickets _(requires `read-only=false`)_
  - `project_id`, `integration_id` : Which integration (string, required)
  - `confirm` : Confirmation token from the preview returned by the first call (string, optional)

- **list_faults** - Get a list of faults for a project with optional filtering and ordering. Fetch the `errors` reference topic (via `get_reference`) for the fault/notice model and the `q` search syntax.
  - `project_id` : The ID of the project to get faults for (string, required)
  - `q` : Search string to filter faults (string, optional)
  - `created_after` : Filter faults created after this time: an RFC 3339 timestamp or a date (midnight UTC) (string, optional)
  - `occurred_after` : Filter faults that occurred after this time: an RFC 3339 timestamp or a date (midnight UTC) (string, optional)
  - `occurred_before` : Filter faults that occurred before this time: an RFC 3339 timestamp or a date (midnight UTC) (string, optional)
  - `limit` : Maximum number of faults to return (max 25) (number, optional)
  - `order` : Order results by 'recent' or 'frequent' (string, optional)
  - `page` : Page number for pagination (number, optional)

- **get_fault** - Get detailed information for a specific fault in a project
  - `project_id` : The ID of the project containing the fault (string, required)
  - `fault_id` : The ID of the fault to retrieve (string, required)

- **update_fault** - Update a fault's resolved, ignored, assignee, or resolve-on-deploy state. Only the provided fields are changed.
  - `project_id` : The ID of the project containing the fault (string, required)
  - `fault_id` : The ID of the fault to update (string, required)
  - `resolved` : Whether the fault is resolved (boolean, optional)
  - `ignored` : Whether the fault is ignored (boolean, optional)
  - `assignee_id` : Public ID of a project member to assign the fault to; null to remove the current assignee; omit to leave unchanged (string or null, optional)
  - `resolve_on_deploy` : Mark the fault to be resolved automatically on next deploy (boolean, optional)

- **get_fault_counts** - Get fault count statistics for a project with optional filtering. Fetch the `errors` reference topic (via `get_reference`) for the `q` search syntax.
  - `project_id` : The ID of the project to get fault counts for (string, required)
  - `q` : Search string to filter faults (string, optional)
  - `created_after` : Filter faults created after this time: an RFC 3339 timestamp or a date (midnight UTC) (string, optional)
  - `occurred_after` : Filter faults that occurred after this time: an RFC 3339 timestamp or a date (midnight UTC) (string, optional)
  - `occurred_before` : Filter faults that occurred before this time: an RFC 3339 timestamp or a date (midnight UTC) (string, optional)

- **list_fault_notices** - Get a list of notices (individual error events) for a specific fault
  - `project_id` : The ID of the project containing the fault (string, required)
  - `fault_id` : The ID of the fault to get notices for (string, required)
  - `before` : Cursor for older notices: `time_series.oldest_cursor` from the previous response (string, optional)
  - `after` : Cursor for newer notices: `time_series.newest_cursor` from the previous response (string, optional)
  - `limit` : Maximum number of notices to return (max 25) (number, optional)

- **list_fault_affected_users** - Get a list of users who were affected by a specific fault with occurrence counts. At most 500 users are returned, with or without `q`.
  - `project_id` : The ID of the project containing the fault (string, required)
  - `fault_id` : The ID of the fault to get affected users for (string, required)
  - `q` : Search string to filter affected users (string, optional)

### Fault Comments

- **list_fault_comments** - List every comment on a fault, newest first.
  - `project_id` : The ID of the project containing the fault (string, required)
  - `fault_id` : The ID of the fault (string, required)

- **get_fault_comment** - Get a single comment on a fault by ID.
  - `project_id` : The ID of the project containing the fault (string, required)
  - `fault_id` : The ID of the fault (string, required)
  - `comment_id` : The ID of the comment (string, required)

- **create_fault_comment** - Add a comment to a fault.
  - `project_id` : The ID of the project containing the fault (string, required)
  - `fault_id` : The ID of the fault (string, required)
  - `body` : Non-blank comment text (string, required)

- **update_fault_comment** - Replace the body of an existing fault comment. Returns the comment as stored.
  - `project_id` : The ID of the project containing the fault (string, required)
  - `fault_id` : The ID of the fault (string, required)
  - `comment_id` : The ID of the comment (string, required)
  - `body` : Non-blank comment text (string, required)

- **delete_fault_comment** - Delete an existing comment from a fault.
  - `project_id` : The ID of the project containing the fault (string, required)
  - `fault_id` : The ID of the fault (string, required)
  - `comment_id` : The ID of the comment (string, required)
  - `confirm` : Confirmation token from the preview returned by the first call (string, optional)

Creating, updating, and deleting comments require write access (`--read-only=false` in stdio mode or the `write` scope in HTTP mode). With an account-scoped API Token (`hba_`), a new comment is attributed to the token's name; updates are refused with `access_denied`, since only a comment's author can edit it; and deletes need permission to manage the project.

### Insights

- **query_insights** - Execute a BadgerQL query against Insights data
  - `project_id` : The ID of the project to query insights for (string, required)
  - `query` : BadgerQL query string to execute against your Insights data (string, required)
  - `ts` : Time range - shortcuts like 'today', 'week', or ISO 8601 duration (e.g., 'PT3H'). Defaults to PT3H (string, optional)
  - `timezone` : IANA timezone identifier (e.g., 'America/New_York') for timestamp interpretation (string, optional)
  - `stream_ids` : List of stream IDs to restrict the query to specific Insights streams. Use `list_streams` to discover a project's stream IDs. Omit to query all streams (array of strings, optional)

### Streams

- **list_streams** - List Insights data streams for a project
  - `project_id` : The ID of the project to list streams for (string, required)

### Dashboards

- **list_dashboards** - List all Insights dashboards for a project
  - `project_id` : The ID of the project to list dashboards for (string, required)

- **get_dashboard** - Get a single Insights dashboard by ID
  - `project_id` : The ID of the project the dashboard belongs to (string, required)
  - `dashboard_id` : The ID of the dashboard to retrieve (string, required)

- **create_dashboard** - Create a new Insights dashboard _(requires `read-only=false`)_
  - `project_id` : The ID of the project to create the dashboard in (string, required)
  - `title` : The title of the dashboard (string, required)
  - `widgets` : JSON array of widget objects. The `dashboards` reference topic has the full widget schema and examples. Each widget needs a `type` (`insights_vis`, `alarms`, `errors`, `deployments`, `checkins`, `uptime`) and optionally `grid` ({x,y,w,h}), `presentation` ({title, subtitle}), and `config` (type-specific settings) (string, required)
  - `default_ts` : Default time range for the dashboard. ISO 8601 duration (e.g., P1D, PT3H) or keyword (today, yesterday, week, month) (string, optional)

- **update_dashboard** - Update an existing Insights dashboard _(requires `read-only=false`)_
  - `project_id` : The ID of the project the dashboard belongs to (string, required)
  - `dashboard_id` : The ID of the dashboard to update (string, required)
  - `title` : A new title (string, optional)
  - `widgets` : JSON array of widget objects that replaces the dashboard's widgets; keep each widget's `id` to keep its identity (string, optional)
  - `default_ts` : Default time range for the dashboard; an empty string clears it (string, optional)

  Only the fields given change, so a rename needs only `title`.

- **delete_dashboard** - Delete an Insights dashboard _(requires `read-only=false`)_
  - `project_id` : The ID of the project the dashboard belongs to (string, required)
  - `dashboard_id` : The ID of the dashboard to delete (string, required)
  - `confirm` : Confirmation token from the preview returned by the first call (string, optional)

### Alarms

- **list_alarms** - List all Insights alarms for a project
  - `project_id` : The ID of the project to list alarms for (string, required)

- **get_alarm** - Get a single Insights alarm by ID
  - `project_id` : The ID of the project the alarm belongs to (string, required)
  - `alarm_id` : The ID of the alarm to retrieve (string, required)

- **create_alarm** - Create a new Insights alarm _(requires `read-only=false`)_. Fetch reference topics `alarms`, `queries`, and `badgerql` first (via `get_reference`) for the `trigger_config` schema and query guidelines.
  - `project_id` : The ID of the project to create the alarm in (string, required)
  - `name` : The name of the alarm (string, required)
  - `query` : BadgerQL query for the alarm. The alarm system wraps the query to count results automatically (string, required)
  - `evaluation_period` : How often the alarm is evaluated (e.g., 5m, 1h, 1d). Minimum 1m (string, required)
  - `trigger_config` : JSON object defining when to trigger the alarm, e.g. `{"type": "alert_result_count", "config": {"operator": "gt", "value": 10}}` (string, required)
  - `lookback_lag` : Delay before evaluating to allow data to arrive (e.g., 1m, or 0s for no lag) (string, required)
  - `description` : Optional description of the alarm (string, optional)
  - `stream_ids` : JSON array of stream IDs to query, naming at least one; an empty array is refused. Omit to query every current stream (string, optional)

- **update_alarm** - Update an existing Insights alarm _(requires `read-only=false`)_. Fetch reference topics `alarms`, `queries`, and `badgerql` first (via `get_reference`).
  - `project_id` : The ID of the project the alarm belongs to (string, required)
  - `alarm_id` : The ID of the alarm to update (string, required)
  - `name` : A new name for the alarm (string, optional)
  - `description` : A new description; an empty string clears it (string, optional)
  - `query` : A new BadgerQL query (string, optional)
  - `evaluation_period` : A new evaluation window, as a compact duration such as `5m` or `1h` (string, optional)
  - `lookback_lag` : A new lookback lag, such as `1m` (string, optional)
  - `stream_ids` : JSON array of stream IDs, replacing the current set (at least one); `null` resets the alarm to every current stream (string, optional)
  - `trigger_config` : JSON object replacing the whole trigger, in the same shape `create_alarm` takes (string, optional)

  Provide at least one field.

- **delete_alarm** - Delete an Insights alarm _(requires `read-only=false`)_
  - `project_id` : The ID of the project the alarm belongs to (string, required)
  - `alarm_id` : The ID of the alarm to delete (string, required)
  - `confirm` : Confirmation token from the preview returned by the first call (string, optional)

- **get_alarm_history** - Get the trigger history for an Insights alarm
  - `project_id` : The ID of the project the alarm belongs to (string, required)
  - `alarm_id` : The ID of the alarm to get history for (string, required)
  - `page` : Page number, starting at 1 (default: 1). Pages hold 25 entries; a non-null `links.next` means there are more (number, optional)

### Check-Ins

- **list_check_ins** - List check-ins (cron/scheduled task monitoring) for a project. Returns every check-in, following pagination
  - `project_id` : The ID of the project to list check-ins for (string, required)

- **get_check_in** - Get a single check-in by ID
  - `project_id` : The ID of the project the check-in belongs to (string, required)
  - `check_in_id` : The ID of the check-in to retrieve (string, required)

- **create_check_in** - Create a new check-in for a project _(requires `read-only=false`)_
  - `project_id` : The ID of the project to create the check-in in (string, required)
  - `name` : The name of the check-in; an unnamed check-in shows its ID (string, optional)
  - `schedule_type` : The schedule type: `simple` (report every fixed period) or `cron` (report on a cron schedule) (string, required)
  - `slug` : Optional URL-friendly identifier used to report the check-in, e.g. `nightly-backups` (string, optional)
  - `report_period` : How often the check-in is expected to report, e.g. `1 day`, `30 minutes`. Required for simple schedules (string, optional)
  - `grace_period` : Amount of time to allow a late report before alerting, e.g. `5 minutes` (string, optional)
  - `cron_schedule` : Cron expression defining when the check-in is expected to report, e.g. `0 5 * * *`. Required for cron schedules (string, optional)
  - `cron_timezone` : Timezone for the cron schedule (defaults to UTC) (string, optional)

- **update_check_in** - Update an existing check-in; only the provided fields are changed, and fields cannot be cleared once set. The schedule type cannot be changed after creation _(requires `read-only=false`)_
  - `project_id` : The ID of the project the check-in belongs to (string, required)
  - `check_in_id` : The ID of the check-in to update (string, required)
  - `name` : The name of the check-in (string, optional)
  - `slug` : URL-friendly identifier used to report the check-in (string, optional)
  - `report_period` : How often the check-in is expected to report. Used by simple schedules (string, optional)
  - `grace_period` : Amount of time to allow a late report before alerting (string, optional)
  - `cron_schedule` : Cron expression defining when the check-in is expected to report. Used by cron schedules (string, optional)
  - `cron_timezone` : Timezone for the cron schedule (string, optional)

- **delete_check_in** - Delete a check-in and its reporting history _(requires `read-only=false`)_
  - `project_id` : The ID of the project the check-in belongs to (string, required)
  - `check_in_id` : The ID of the check-in to delete (string, required)
  - `confirm` : Confirmation token from the preview returned by the first call (string, optional)

- **list_check_in_events** - List a check-in's history, newest first: each time it reported, went missing or was paused
  - `project_id` : The ID of the project the check-in belongs to (string, required)
  - `check_in_id` : The ID of the check-in (string, required)
  - `limit` : Most events to return, up to 100 (number, optional)
  - `created_before` : Page back: the previous response's `next_created_before`, unchanged (number, optional)

### Uptime Sites

- **list_sites** - List a project's uptime sites and their current state
  - `project_id` : The ID of the project (string, required)

- **get_site** - Get an uptime site's URL, check settings and state
  - `project_id` : The ID of the project (string, required)
  - `site_id` : The ID of the site, a UUID (string, required)

- **create_site** - Start uptime monitoring for a URL _(requires `read-only=false`)_
  - `project_id` : The ID of the project (string, required)
  - `url` : The URL to check, starting with http:// or https:// (string, required)
  - `name`, `frequency` (1, 2, 5 or 15 minutes), `locations` (Virginia, Oregon, London, Frankfurt, Singapore; `[]` or `null` for every location), `match_type` (`success`, `exact`, `include`, `exclude`, `jmespath`), `match`, `request_method`, `request_body`, `request_headers`, `timeout`, `outage_threshold`, `validate_ssl`, `active` : check settings (optional)

- **update_site** - Change an uptime site's settings; `null` resets a setting to its default _(requires `read-only=false`)_
  - `project_id`, `site_id` (required), and any of `url` and the settings `create_site` takes (optional)

- **delete_site** - Stop monitoring an uptime site and delete it _(requires `read-only=false`)_
  - `project_id`, `site_id` (string, required)
  - `confirm` : Confirmation token from the preview returned by the first call (string, optional)

- **list_site_outages** - List a site's outages, newest first, with the status and reason from the failing check
  - `project_id`, `site_id` (string, required)
  - `limit`, `created_before` : paging, as for `list_check_in_events` (number, optional)

- **list_uptime_checks** - List a site's individual checks, newest first: location, response status and duration
  - `project_id`, `site_id` (string, required)
  - `limit`, `created_before` : paging, as for `list_check_in_events` (number, optional)

### Deploys

- **list_deploys** - List a project's deploys, newest first
  - `project_id` : The ID of the project (string, required)
  - `environment` : Only deploys to this environment (string, optional)
  - `local_username` : Only deploys recorded under this username (string, optional)
  - `limit` : Most deploys to return, up to 100 (number, optional)
  - `before` / `after` : Cursors from the previous response's `time_series.oldest_cursor` / `newest_cursor` (string, optional)

- **get_deploy** - Get a deploy by ID, e.g. one a `deployed` event or a fault's `last_notice_deploy` names
  - `project_id` : The ID of the project (string, required)
  - `deploy_id` : The ID of the deploy (string, required)

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
