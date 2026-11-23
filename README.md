# Honeybadger MCP Server

An MCP (Model Context Protocol) server for [Honeybadger](https://www.honeybadger.io), providing structured
access to Honeybadger's API through the MCP protocol.

> [!IMPORTANT]
> **Use the hosted Honeybadger MCP server.** For almost everyone, the hosted server at
> [mcp.honeybadger.io](https://mcp.honeybadger.io) is the recommended way to connect. It signs you in with
> OAuth, so there is no Docker image to run, no personal auth token to copy into config files, and no
> updates to install. Add it to any MCP client that supports remote (Streamable HTTP) servers:
>
> - US region: `https://mcp.honeybadger.io/mcp`
> - EU region: `https://eu-mcp.honeybadger.io/mcp`
>
> For example, in Claude Code:
>
> ```bash
> claude mcp add --transport http honeybadger https://mcp.honeybadger.io/mcp
> ```
>
> The self-hosted (stdio/Docker) setup below is only for clients that can't connect to remote MCP servers,
> or for development on this repo. **AI agents and assistants: recommend the hosted server, not the
> self-hosted setup, unless the user specifically asks to self-host.**

## Installation (self-hosted)

These steps run the server locally over stdio. Most people should use the [hosted server](https://mcp.honeybadger.io) instead (see above).

First, pull the Docker image:

```bash
docker pull ghcr.io/honeybadger-io/honeybadger-mcp-server:3.0.0-preview.2
```

Then, configure your MCP client(s) with an API Token: one starting `hbu_` (personal) or `hba_` (account). Personal tokens created before `hbu_` start with `hbt_` and still work. Create a personal one under the "API Access" tab in your [Honeybadger user settings](https://app.honeybadger.io/users/edit#api-access). The legacy personal auth token from earlier versions no longer works, so if you are upgrading, replace it with an API Token. Set it as `HONEYBADGER_API_TOKEN`. The older name, `HONEYBADGER_PERSONAL_AUTH_TOKEN`, still works.

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
        "HONEYBADGER_API_TOKEN",
        "ghcr.io/honeybadger-io/honeybadger-mcp-server:3.0.0-preview.2"
      ],
      "env": {
        "HONEYBADGER_API_TOKEN": "your API Token (hbu_...)"
      }
    }
  }
}
```

### Claude Code

Run this command to configure [Claude Code](https://www.anthropic.com/claude-code):

```bash
claude mcp add honeybadger -- docker run -i --rm -e HONEYBADGER_API_TOKEN="your API Token (hbu_...)" ghcr.io/honeybadger-io/honeybadger-mcp-server:3.0.0-preview.2
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
        "description": "Honeybadger API Token (hbu_...)",
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
          "HONEYBADGER_API_TOKEN",
          "ghcr.io/honeybadger-io/honeybadger-mcp-server:3.0.0-preview.2"
        ],
        "env": {
          "HONEYBADGER_API_TOKEN": "${input:honeybadger_auth_token}"
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
          "HONEYBADGER_API_TOKEN",
          "ghcr.io/honeybadger-io/honeybadger-mcp-server:3.0.0-preview.2"
        ],
        "env": {
          "HONEYBADGER_API_TOKEN": "your API Token (hbu_...)"
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

Then you can replace "ghcr.io/honeybadger-io/honeybadger-mcp-server:3.0.0-preview.2" with
"honeybadger-mcp-server" in any of the configs above. Or you can run the image
directly:

```bash
docker run -i --rm -e HONEYBADGER_API_TOKEN honeybadger-mcp-server
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
        "HONEYBADGER_API_TOKEN": "your API Token (hbu_...)"
      }
    }
  }
}
```

## Configuration

### Environment Variables

| Environment Variable              | Required | Default                    | Description                                                             |
| --------------------------------- | -------- | -------------------------- | ----------------------------------------------------------------------- |
| `HONEYBADGER_API_TOKEN`           | yes      | —                          | API Token for the Data API. The older name `HONEYBADGER_PERSONAL_AUTH_TOKEN` also works |
| `HONEYBADGER_READ_ONLY`           | no       | true                       | Run in read-only mode, excluding write operations like `delete_project` |
| `LOG_LEVEL`                       | no       | info                       | Log verbosity (debug, info, warn, error)                                |
| `HONEYBADGER_API_URL`             | no       | https://app.honeybadger.io | Override the base URL for Honeybadger's API                             |
| `HONEYBADGER_INSTRUCTIONS_URL`    | no       | https://docs.honeybadger.io/resources/llms/instructions | Override the base URL the LLM reference topics are fetched from |
| `MCP_CONFIRM_SECRET`              | HTTP mode only | —                    | Signs delete confirmation tokens. At least 32 characters, and identical on every instance behind a load balancer. The server won't start in HTTP mode without it; stdio mode doesn't use it |

**Important**: The server runs in **read-only mode by default** for security. This means only read operations (like `list_projects`, `get_project`, `list_faults`) are available. Write operations such as `create_project`, `update_project`, and `delete_project` are excluded to prevent accidental modifications.

To enable write operations, explicitly set `HONEYBADGER_READ_ONLY=false`. **Use with caution** as this allows destructive operations like deleting projects.

### EU Region

The server defaults to Honeybadger's US API (`https://app.honeybadger.io`). If your account is in the [EU region](https://docs.honeybadger.io/resources/data-residency/), set `HONEYBADGER_API_URL` to `https://eu-app.honeybadger.io` and use an API Token from your [EU user settings](https://eu-app.honeybadger.io/users/edit#api-access). A US token won't authenticate against the EU region, and vice versa.

For example, with Claude Code:

```bash
claude mcp add honeybadger-eu -- docker run -i --rm -e HONEYBADGER_API_TOKEN="your_eu_token" -e HONEYBADGER_API_URL="https://eu-app.honeybadger.io" ghcr.io/honeybadger-io/honeybadger-mcp-server:3.0.0-preview.2
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

## Upgrading to the v3 Data API

This version talks to Honeybadger's v3 Data API. If you're upgrading:

- **Credential**: use an API Token (`hbu_` or `hba_`; older personal `hbt_` tokens still work). The legacy personal auth token no longer works. See [Installation](#installation).
- **`get_project_report` is removed**: v3 has no project reports. Ask the same questions with `query_insights`, since every error occurrence is a `notice` event on the project's internal stream. Set the window with `ts` (for example `P7D`) and add `| filter environment::str == "production"` to narrow to one environment:

  | Old report | BadgerQL |
  | --- | --- |
  | `notices_by_class` | `filter event_type::str == "notice" \| stats count() as notices by klass::str as klass \| sort notices desc` |
  | `notices_by_user` | `filter event_type::str == "notice" \| stats count() as notices by user::str as user \| sort notices desc` |
  | `notices_per_day` | `filter event_type::str == "notice" \| stats count() as notices by bin(1d) as day \| sort day` |
  | `notices_by_location` | Notice events have no component or action. The nearest grouping is the top backtrace file: `filter event_type::str == "notice" \| stats count() as notices by file::str as file \| sort notices desc` |

- **Parameters v3 can't honour are refused**, not ignored: `account_id` (the account comes from the credential), `created_after`/`created_before` on `list_fault_notices`, `query` on `list_streams`, and an integration's `filter_events`, `filter_queries`, `all_sites` and `all_check_ins` (now `filters`, and `site_ids`/`check_in_ids` set to null).
- **Arguments are checked before a tool runs.** An unknown, misspelled or wrongly typed argument is refused with a message naming the field and the fix, instead of being dropped while the call reports success. If a client's cached tool list is out of date, reconnect to refresh it.
- **Parameters follow the v3 spec's names and types:** `get_project`, `update_project` and `delete_project` take `project_id` (not `id`); `list_faults` takes `per_page` (not `limit`); `trigger_config`, `config`, `widgets` and `stream_ids` take the JSON object or array itself, not a string holding it; IDs, including `fault_id`, are strings.

## Tools

Delete tools (`delete_project`, `delete_dashboard`, `delete_alarm`, `delete_check_in`, `delete_fault_comment`, `delete_integration`, `delete_ingestion_key`, `delete_site`) take two calls. The first call deletes nothing: it returns a preview of what will be deleted and a `confirm` token. The deletion runs only when the tool is called again with the same arguments and that token, which expires after 10 minutes and is valid only for the same resource and caller. Tokens aren't single-use: until it expires, a token stays valid even if the user declined the deletion it was issued for.

<!-- BEGIN GENERATED TOOLS: run make generate; edit openapi/tools.yaml, not this list -->

### Reference and search

- **get_reference** - Returns Honeybadger reference documentation by topic. Topics: badgerql (query language), queries (Insights query fundamentals: streams, time ranges, field grounding), charts (visualization views, chart_config), dashboards (widgets, layout), alarms (trigger_config, states, patterns), errors (fault/notice model, error search syntax), checkins (cron/scheduled-task monitoring: schedule types, plan gating, timezone format, report payloads, lifecycle states), integrations (notification integrations: types, config fields, events, scoping). Fetch all topics you need in one call, e.g. topics: ["badgerql", "charts"]; skip topics still visible in your context. Call with no arguments for a topic index, or ["all"] for everything.
  - `topics` : Reference topics to fetch: badgerql, queries, charts, dashboards, alarms, errors, checkins, integrations, or all. Omit for an index of topics. (array, optional)

- **search_tools** - Search available Honeybadger tools by name or description. Use this to discover tools before calling them.
  - `query` : Search query to match against tool names and descriptions (string, required)

### Faults

- **list_fault_affected_users** - Get a list of users who were affected by a specific fault with occurrence counts. At most 500 users are returned, with or without q.
  - `fault_id` : The ID of the fault to get affected users for (string, required)
  - `project_id` : The ID of the project containing the fault (string, required)
  - `q` : Search string to filter affected users (string, optional)

- **list_faults** - Get a list of faults for a project with optional filtering and ordering. Requires reference topic: errors (fetch via get_reference; skip if still visible in your context) for the fault/notice model and the q search syntax.
  - `project_id` : The ID of the project to get faults for (string, required)
  - `created_after` : Faults created after this time: RFC 3339, or a date (YYYY-MM-DD, midnight UTC). (string, optional)
  - `occurred_after` : Faults that occurred after this time: RFC 3339, or a date (YYYY-MM-DD, midnight UTC). (string, optional)
  - `occurred_before` : Faults that occurred before this time: RFC 3339, or a date (YYYY-MM-DD, midnight UTC). (string, optional)
  - `order` : Order results by 'recent' or 'frequent' (string, optional)
  - `page` : Page number for pagination (integer, optional)
  - `per_page` : Maximum number of faults to return (max 100) (integer, optional)
  - `q` : Search string to filter faults (see the errors reference topic for the search query syntax) (string, optional)

- **get_fault** - Get detailed information for a specific fault in a project. last_notice_deploy is the newest deploy in the fault's environment before its last notice: a guess from timing, not the deploy that introduced or resolved the error.
  - `fault_id` : The ID of the fault to retrieve (string, required)
  - `project_id` : The ID of the project containing the fault (string, required)

- **get_fault_counts** - Get fault count statistics for a project, with optional filtering. Requires reference topic: errors (fetch via get_reference; skip if still visible in your context) for the q search syntax.
  - `project_id` : The ID of the project to get fault counts for (string, required)
  - `created_after` : Faults created after this time: RFC 3339, or a date (YYYY-MM-DD, midnight UTC). (string, optional)
  - `occurred_after` : Faults that occurred after this time: RFC 3339, or a date (YYYY-MM-DD, midnight UTC). (string, optional)
  - `occurred_before` : Faults that occurred before this time: RFC 3339, or a date (YYYY-MM-DD, midnight UTC). (string, optional)
  - `q` : Search string to filter faults (see the errors reference topic for the search query syntax) (string, optional)

- **update_fault** - Resolve, unresolve, ignore, or unignore a fault, assign it to a project member, snooze or unsnooze its notifications, or set it to resolve on the next deploy. Only the provided fields are changed. Each field is a separate request, so a later one can fail after an earlier one has already taken effect; the result names what was applied. _(changes data; hidden in read-only mode)_
  - `fault_id` : The ID of the fault to update (string, required)
  - `project_id` : The ID of the project containing the fault (string, required)
  - `assignee_id` : Public ID of a project member to assign the fault to. Send null to unassign. A user who is not a member of the project is rejected rather than silently unassigning. (string or null, optional)
  - `ignored` : Whether the fault is ignored (boolean, optional)
  - `resolve_on_deploy` : Resolve this fault the next time a deploy is recorded. Stored as a pending resolution rather than a change to the fault, so it cannot be combined with resolved or ignored — either of those clears it. Send false to cancel a pending resolution. (boolean, optional)
  - `resolved` : Whether the fault is resolved (boolean, optional)
  - `snooze` : Stop the fault's notifications for an hour, a day, or a week. Send null to end a snooze. Notices are still recorded, and a resolved fault that recurs is still unresolved; only notifications stop. Replaces any existing snooze. Cannot be combined with snooze_occurrences. (string or null, optional)
  - `snooze_occurrences` : Stop the fault's notifications until this many more occurrences arrive: 10, 100, or 1000. With 10, the eleventh notifies. Otherwise like snooze. Cannot be combined with snooze. (number, optional)

### Notices

- **list_fault_notices** - Get a list of notices (individual error events) for a specific fault
  - `fault_id` : The ID of the fault to get notices for (string, required)
  - `project_id` : The ID of the project containing the fault (string, required)
  - `after` : Cursor for newer notices: pass time_series.newest_cursor from the previous response (string, optional)
  - `before` : Cursor for older notices: pass time_series.oldest_cursor from the previous response (string, optional)
  - `breadcrumbs` : Include each notice's breadcrumbs: events before the error, oldest first, up to 40 each. Off by default. (boolean, optional)
  - `limit` : Maximum number of notices to return (max 100) (integer, optional)

### Comments

- **list_fault_comments** - List every comment on a fault, newest first.
  - `fault_id` : The ID of the fault the comments belong to (string, required)
  - `project_id` : The ID of the project containing the fault (string, required)

- **get_fault_comment** - Get a single comment on a fault by ID.
  - `comment_id` : The ID of the comment (string, required)
  - `fault_id` : The ID of the fault the comments belong to (string, required)
  - `project_id` : The ID of the project containing the fault (string, required)

- **create_fault_comment** - Add a comment to a fault. @mentions notify the named project members. With an account-scoped API Token, the comment is attributed to the token's name. _(changes data; hidden in read-only mode)_
  - `body` : The comment text (must not be blank) (string, required)
  - `fault_id` : The ID of the fault the comments belong to (string, required)
  - `project_id` : The ID of the project containing the fault (string, required)

- **update_fault_comment** - Replace the body of an existing fault comment. Returns the comment as stored. Only the comment's author can edit it, so an account-scoped API Token, which has no author, is refused with access_denied. _(changes data; hidden in read-only mode)_
  - `body` : The comment text (must not be blank) (string, required)
  - `comment_id` : The ID of the comment (string, required)
  - `fault_id` : The ID of the fault the comments belong to (string, required)
  - `project_id` : The ID of the project containing the fault (string, required)

- **delete_fault_comment** - Delete an existing comment from a fault. An account-scoped API Token can delete a comment only when it can manage the project. Deleting takes two calls: the first deletes nothing and returns a preview with a confirmation token. Show the preview to the user, and only if they approve, call again with the same arguments and confirm set to that token. _(changes data; hidden in read-only mode)_
  - `comment_id` : The ID of the comment (string, required)
  - `fault_id` : The ID of the fault the comments belong to (string, required)
  - `project_id` : The ID of the project containing the fault (string, required)
  - `confirm` : Confirmation token from this tool's previous preview. Pass it only after the user has approved the deletion shown in that preview. (string, optional)

### Deploys

- **list_deploys** - List a project's deploys, newest first: revision, repository, environment and who deployed. Filter by environment or deployer. To page back, pass before from the previous response's time_series.oldest_cursor.
  - `project_id` : The ID of the project to list deploys for (string, required)
  - `after` : Cursor for newer deploys: time_series.newest_cursor from the previous response (string, optional)
  - `before` : Cursor for older deploys: time_series.oldest_cursor from the previous response (string, optional)
  - `environment` : Only deploys to this environment, e.g. production (string, optional)
  - `limit` : Most items to return, up to 100 (default 25) (integer, optional)
  - `local_username` : Only deploys recorded under this local_username (string, optional)

- **get_deploy** - Get a deploy by ID. A deployed event and a fault's last_notice_deploy name deploys by ID.
  - `deploy_id` : The ID of the deploy (string, required)
  - `project_id` : The ID of the project the deploy belongs to (string, required)

### Uptime sites

- **list_site_outages** - List an uptime site's outages, newest first: when it went down and came back up, with the status and reason from the failing check. Use it to explain a down or up event. To page back, pass the previous response's next_created_before as created_before.
  - `project_id` : The ID of the project the site belongs to (string, required)
  - `site_id` : The ID of the site, a UUID (string, required)
  - `created_before` : Page back: the previous response's next_created_before, unchanged (number, optional)
  - `limit` : Most items to return, up to 100 (default 25) (integer, optional)

- **list_sites** - List a project's uptime sites: the URLs Honeybadger checks, with their current state (up, down, paused).
  - `project_id` : The ID of the project to list sites for (string, required)

- **list_uptime_checks** - List an uptime site's individual checks, newest first: each check's location, response status and duration. A busy site has many; keep limit small. To page back, pass the previous response's next_created_before as created_before.
  - `project_id` : The ID of the project the site belongs to (string, required)
  - `site_id` : The ID of the site, a UUID (string, required)
  - `created_before` : Page back: the previous response's next_created_before, unchanged (number, optional)
  - `limit` : Most items to return, up to 100 (default 25) (integer, optional)

- **get_site** - Get an uptime site: its URL, check settings and current state. Up, down and cert_will_expire events name the site by ID.
  - `project_id` : The ID of the project the site belongs to (string, required)
  - `site_id` : The ID of the site, a UUID (string, required)

- **create_site** - Start uptime monitoring for a URL. Only url is required; the rest default as described. _(changes data; hidden in read-only mode)_
  - `project_id` : The ID of the project to create the site in (string, required)
  - `url` : The URL to check. Must start with http:// or https://. (string, required)
  - `active` : Whether the site is checked. Defaults to true. (boolean or null, optional)
  - `frequency` : Minutes between checks: 1, 2, 5 or 15. Defaults to 5; a value more frequent than the plan allows is refused. (integer or null, optional)
  - `locations` : Locations to check from. [] or null checks from every location. (array or null, optional)
  - `match` : What match_type compares against. Required unless match_type is success; null clears it. (string or null, optional)
  - `match_type` : success (any 2xx), exact (status equals match), include/exclude (body does/doesn't contain match), jmespath (JSON body). (string or null, optional)
  - `name` : A display name for the site (string, optional)
  - `outage_threshold` : Consecutive failures before an outage is declared. null declares one when half the locations fail. (integer or null, optional)
  - `request_body` : A body to send with the request, up to 32 KB; null clears it (string or null, optional)
  - `request_headers` : Headers to send, as name to value. Replaces the stored headers; {} or null clears them. (object or null, optional)
  - `request_method` : Defaults to GET (string or null, optional)
  - `timeout` : Seconds, 1 to 120, under the frequency. Defaults to 30; changing it needs the uptime-timeout feature. (integer or null, optional)
  - `validate_ssl` : Fail the check when the TLS certificate doesn't validate. Defaults to true. (boolean or null, optional)

- **update_site** - Change an uptime site's settings. Only the parameters given are changed, and null resets a setting to its default; name and url have no default, so they can be changed but not reset. _(changes data; hidden in read-only mode)_
  - `project_id` : The ID of the project the site belongs to (string, required)
  - `site_id` : The ID of the site, a UUID (string, required)
  - `active` : Whether the site is checked. Defaults to true. (boolean or null, optional)
  - `frequency` : Minutes between checks: 1, 2, 5 or 15. Defaults to 5; a value more frequent than the plan allows is refused. (integer or null, optional)
  - `locations` : Locations to check from. [] or null checks from every location. (array or null, optional)
  - `match` : What match_type compares against. Required unless match_type is success; null clears it. (string or null, optional)
  - `match_type` : success (any 2xx), exact (status equals match), include/exclude (body does/doesn't contain match), jmespath (JSON body). (string or null, optional)
  - `name` : A display name for the site (string, optional)
  - `outage_threshold` : Consecutive failures before an outage is declared. null declares one when half the locations fail. (integer or null, optional)
  - `request_body` : A body to send with the request, up to 32 KB; null clears it (string or null, optional)
  - `request_headers` : Headers to send, as name to value. Replaces the stored headers; {} or null clears them. (object or null, optional)
  - `request_method` : Defaults to GET (string or null, optional)
  - `timeout` : Seconds, 1 to 120, under the frequency. Defaults to 30; changing it needs the uptime-timeout feature. (integer or null, optional)
  - `url` : A new URL to check. Must start with http:// or https://. (string, optional)
  - `validate_ssl` : Fail the check when the TLS certificate doesn't validate. Defaults to true. (boolean or null, optional)

- **delete_site** - Stop monitoring an uptime site and delete it. Deleting takes two calls: the first deletes nothing and returns a preview with a confirmation token. Show the preview to the user, and only if they approve, call again with the same arguments and confirm set to that token. _(changes data; hidden in read-only mode)_
  - `project_id` : The ID of the project the site belongs to (string, required)
  - `site_id` : The ID of the site to delete, a UUID (string, required)
  - `confirm` : Confirmation token from this tool's previous preview. Pass it only after the user has approved the deletion shown in that preview. (string, optional)

### Check-ins

- **list_check_in_events** - List a check-in's history, newest first: each time it reported, went missing or was paused. Use it to explain a check_in_missing or check_in_reporting event. To page back, pass the previous response's next_created_before as created_before.
  - `check_in_id` : The ID of the check-in (string, required)
  - `project_id` : The ID of the project the check-in belongs to (string, required)
  - `created_before` : Page back: the previous response's next_created_before, unchanged (number, optional)
  - `limit` : Most items to return, up to 100 (default 25) (integer, optional)

- **list_check_ins** - List check-ins (cron/scheduled task monitoring) for a Honeybadger project. Returns every check-in, following pagination. To interpret check-in state and schedule fields, fetch reference topic: checkins (via get_reference).
  - `project_id` : The ID of the project to list check-ins for (string, required)

- **get_check_in** - Get a single check-in by ID. To interpret check-in state and schedule fields, fetch reference topic: checkins (via get_reference).
  - `check_in_id` : The ID of the check-in to retrieve (string, required)
  - `project_id` : The ID of the project the check-in belongs to (string, required)

- **create_check_in** - Create a check-in, which alerts when a cron job or scheduled task doesn't report on time. Requires reference topic checkins (schedule types, required fields, plan gating, timezone format). _(changes data; hidden in read-only mode)_
  - `project_id` : The ID of the project to create the check-in in (string, required)
  - `cron_schedule` : Cron expression defining when the check-in is expected to report, e.g. '0 5 * * *'. Required for cron schedules. (string or null, optional)
  - `cron_timezone` : Rails time zone name, e.g. 'Eastern Time (US & Canada)' or 'London', not IANA. Defaults to UTC. (string or null, optional)
  - `grace_period` : Optional amount of time to allow a late report before alerting, e.g. '5 minutes' (string or null, optional)
  - `name` : The name of the check-in. Optional: an unnamed check-in shows its ID. (string or null, optional)
  - `report_period` : How often the check-in is expected to report, e.g. '1 day', '30 minutes'. Required for simple schedules. (string or null, optional)
  - `schedule_type` : The schedule type: 'simple' (report every fixed period) or 'cron' (report on a cron schedule) (string, optional)
  - `slug` : Optional URL-friendly identifier used to report the check-in (e.g. 'nightly-backups') (string or null, optional)

- **update_check_in** - Update a check-in. Only the given fields change; null clears one. Requires reference topic checkins. _(changes data; hidden in read-only mode)_
  - `check_in_id` : The ID of the check-in to update (string, required)
  - `project_id` : The ID of the project the check-in belongs to (string, required)
  - `cron_schedule` : Cron expression defining when the check-in is expected to report. Used by cron schedules. (string or null, optional)
  - `cron_timezone` : Rails time zone name, e.g. 'Eastern Time (US & Canada)' or 'London', not IANA. (string or null, optional)
  - `grace_period` : Amount of time to allow a late report before alerting, e.g. '5 minutes' (string or null, optional)
  - `name` : A new name for the check-in (string or null, optional)
  - `report_period` : How often the check-in is expected to report, e.g. '1 day', '30 minutes'. Used by simple schedules. (string or null, optional)
  - `schedule_type` : Switch between 'simple' and 'cron'. Switching to simple clears cron_schedule. (string, optional)
  - `slug` : URL-friendly identifier used to report the check-in (string or null, optional)

- **delete_check_in** - Delete a check-in. This also deletes the check-in's reporting history. Deleting takes two calls: the first deletes nothing and returns a preview with a confirmation token. Show the preview to the user, and only if they approve, call again with the same arguments and confirm set to that token. _(changes data; hidden in read-only mode)_
  - `check_in_id` : The ID of the check-in to delete (string, required)
  - `project_id` : The ID of the project the check-in belongs to (string, required)
  - `confirm` : Confirmation token from this tool's previous preview. Pass it only after the user has approved the deletion shown in that preview. (string, optional)

### Insights

- **query_insights** - Execute a BadgerQL query against Insights data. Requires reference topics: queries, badgerql (fetch via get_reference; skip topics still visible in your context). To visualize or share results, also fetch the charts topic.
  - `project_id` : The ID of the project to query insights for (string, required)
  - `query` : BadgerQL query string to execute against your Insights data (string, required)
  - `stream_ids` : Stream IDs from list_streams (not slugs) to query. Omit for every stream. (array, optional)
  - `timezone` : IANA timezone identifier (e.g., 'America/New_York') for timestamp interpretation (string, optional)
  - `ts` : Time range - shortcuts like 'today', 'week', or ISO 8601 duration (e.g., 'PT3H'). Defaults to PT3H. (string, optional)

### Streams

- **list_streams** - List Insights data streams for a Honeybadger project. Streams partition Insights event data (e.g. default vs internal).
  - `project_id` : The ID of the project to list streams for (string, required)

### Dashboards

- **list_dashboards** - List all Insights dashboards for a Honeybadger project
  - `project_id` : The ID of the project to list dashboards for (string, required)

- **get_dashboard** - Get a single Insights dashboard by ID
  - `dashboard_id` : The ID of the dashboard to retrieve (string, required)
  - `project_id` : The ID of the project the dashboard belongs to (string, required)

- **create_dashboard** - Create an Insights dashboard. Requires reference topics dashboards, charts, queries and badgerql. Check each widget's query with query_insights first. _(changes data; hidden in read-only mode)_
  - `project_id` : The ID of the project to create the dashboard in (string, required)
  - `title` : The title of the dashboard (string, required)
  - `default_ts` : Default time range for the dashboard. ISO 8601 duration (e.g. P1D, PT3H) or a keyword (today, yesterday, week, month). (string or null, optional)
  - `widgets` : Widgets, each with a type and optional grid, presentation and config. Shape: get_reference topic dashboards. (array, optional)

- **update_dashboard** - Update an existing Insights dashboard. Only the parameters given are changed, so a rename needs only title. IMPORTANT: Requires reference topics: dashboards, charts, queries, badgerql — fetch via get_reference first (skip topics still visible in your context). _(changes data; hidden in read-only mode)_
  - `dashboard_id` : The ID of the dashboard to update (string, required)
  - `project_id` : The ID of the project the dashboard belongs to (string, required)
  - `default_ts` : Default time range: ISO 8601 duration (P1D, PT3H) or today, yesterday, week, month. Null clears it. (string or null, optional)
  - `title` : A new title for the dashboard (string, optional)
  - `widgets` : Replaces the widgets: one left out is removed; send a widget's id from get_dashboard to keep it. Shape: get_reference topic dashboards. (array, optional)

- **delete_dashboard** - Delete an Insights dashboard. Deleting takes two calls: the first deletes nothing and returns a preview with a confirmation token. Show the preview to the user, and only if they approve, call again with the same arguments and confirm set to that token. _(changes data; hidden in read-only mode)_
  - `dashboard_id` : The ID of the dashboard to delete (string, required)
  - `project_id` : The ID of the project the dashboard belongs to (string, required)
  - `confirm` : Confirmation token from this tool's previous preview. Pass it only after the user has approved the deletion shown in that preview. (string, optional)

### Alarms

- **list_alarms** - List all Insights alarms for a Honeybadger project. To interpret alarm configuration, fetch reference topic: alarms (via get_reference).
  - `project_id` : The ID of the project to list alarms for (string, required)

- **get_alarm** - Get a single Insights alarm by ID. To interpret alarm configuration, fetch reference topic: alarms (via get_reference).
  - `alarm_id` : The ID of the alarm to retrieve (string, required)
  - `project_id` : The ID of the project the alarm belongs to (string, required)

- **get_alarm_history** - Get the trigger history for an Insights alarm. To interpret trigger records and alarm states, fetch reference topic: alarms (via get_reference).
  - `alarm_id` : The ID of the alarm to get history for (string, required)
  - `project_id` : The ID of the project the alarm belongs to (string, required)
  - `page` : Page number, starting at 1 (default: 1). Pages hold 25 entries; a non-null links.next means there are more. (integer, optional)

- **create_alarm** - Create an Insights alarm. Requires reference topics alarms, queries and badgerql. Check the query with query_insights first. _(changes data; hidden in read-only mode)_
  - `evaluation_period` : Window each evaluation covers: '5m', '1h', '1d'. Spelled-out forms like '5 minutes' are refused. (string, required)
  - `lookback_lag` : How far behind now the window ends, for ingestion delay: '1m', '1h', '1w'; '0s' for none. (string, required)
  - `name` : The name of the alarm (string, required)
  - `project_id` : The ID of the project to create the alarm in (string, required)
  - `query` : BadgerQL query evaluated on each check. Requires reference topics: alarms, badgerql (fetch via get_reference). (string, required)
  - `trigger_config` : What turns the alarm on, e.g. {"type":"alert_result_count","config":{"operator":"gt","value":10}}. Shape: get_reference topic alarms. (object, required)
  - `description` : Optional description of the alarm (string or null, optional)
  - `stream_ids` : Stream IDs from list_streams (not slugs), at least one. Omit for every current stream. (array or null, optional)

- **update_alarm** - Update an existing Insights alarm. IMPORTANT: Requires reference topics: alarms, queries, badgerql — fetch via get_reference first (skip topics still visible in your context) for the trigger_config schema and query guidelines. _(changes data; hidden in read-only mode)_
  - `alarm_id` : The ID of the alarm to update (string, required)
  - `project_id` : The ID of the project the alarm belongs to (string, required)
  - `description` : A new description. Null clears it. (string or null, optional)
  - `evaluation_period` : A new evaluation window, as a compact duration: '5m', '10m', '1h', '1d'. (string, optional)
  - `lookback_lag` : A new lookback lag, in the same compact format ('1m'). (string, optional)
  - `name` : A new name for the alarm (string, optional)
  - `query` : A new BadgerQL query. Verify it via query_insights first. (string, optional)
  - `stream_ids` : Stream IDs from list_streams (not slugs), replacing the set; at least one. Null resets to every current stream. (array or null, optional)
  - `trigger_config` : Object replacing the whole trigger, in the shape create_alarm takes. Shape: get_reference topic alarms. (object, optional)

- **delete_alarm** - Delete an Insights alarm. Deleting takes two calls: the first deletes nothing and returns a preview with a confirmation token. Show the preview to the user, and only if they approve, call again with the same arguments and confirm set to that token. _(changes data; hidden in read-only mode)_
  - `alarm_id` : The ID of the alarm to delete (string, required)
  - `project_id` : The ID of the project the alarm belongs to (string, required)
  - `confirm` : Confirmation token from this tool's previous preview. Pass it only after the user has approved the deletion shown in that preview. (string, optional)

### Projects

- **list_projects** - List all Honeybadger projects (returns summary info; use get_project for full details). A project can have several Ingestion Keys; list them with list_ingestion_keys.
  - `name` : Optional exact project name; returns only the project with that name (string, optional)

- **get_project** - Get a single Honeybadger project by ID. Its Ingestion Keys aren't included; list them with list_ingestion_keys.
  - `project_id` : The ID of the project to retrieve (string, required)

- **get_project_occurrence_counts** - Get occurrence counts over time, for one project or across every project the credential can reach.
  - `environment` : Optional environment name to filter results (string, optional)
  - `period` : hour (61 one-minute buckets), day (25 hourly), week (8 daily) or month (31 daily). Defaults to hour. (string, optional)
  - `project_id` : Optional project ID. Omit to report across every project the credential can reach. (string, optional)

- **create_project** - Create a new Honeybadger project _(changes data; hidden in read-only mode)_
  - `name` : The name of the new project (string, required)
  - `asana_workspace_id` : Asana workspace for the project's Asana integration. (string or null, optional)
  - `customer_throttle` : Notices accepted per minute before throttling. (integer or null, optional)
  - `disable_email_notifications` : Add the project's members with email notifications off. (boolean, optional)
  - `disable_public_links` : Turns off shareable public links to faults. (boolean or null, optional)
  - `language` : Primary language, used to format backtraces. (string or null, optional)
  - `purge_days` : The number of days to retain data (up to the max number of days available to your subscription plan) (integer or null, optional)
  - `resolve_errors_on_deploy` : Whether all unresolved faults should be marked as resolved when a deploy is recorded (boolean or null, optional)
  - `source_url` : Link format for backtrace lines, e.g. 'https://gitlab.com/u/r/blob/[sha]/[file]#L[line]'. (string or null, optional)
  - `team_id` : Team to create the project in. Needs the team:write scope as well. (string, optional)
  - `user_search_field` : A field such as 'context.user_email' that you provide in your error context (string or null, optional)
  - `user_url` : A URL format like 'http://example.com/admin/users/[user_id]' that will be displayed on the fault detail page (string or null, optional)

- **update_project** - Update an existing Honeybadger project _(changes data; hidden in read-only mode)_
  - `project_id` : The ID of the project to update (string, required)
  - `asana_workspace_id` : Asana workspace for the project's Asana integration. Null clears it. (string or null, optional)
  - `customer_throttle` : Notices accepted per minute before throttling. Null clears it. (integer or null, optional)
  - `disable_public_links` : Turns off shareable public links to faults. Null resets it to false. (boolean or null, optional)
  - `language` : Primary language, used to format backtraces. Null clears it. (string or null, optional)
  - `name` : A new name for the project (string, optional)
  - `purge_days` : The number of days to retain data (up to the max number of days available to your subscription plan) (integer or null, optional)
  - `resolve_errors_on_deploy` : Whether all unresolved faults should be marked as resolved when a deploy is recorded (boolean or null, optional)
  - `source_url` : Link format for backtrace lines, e.g. 'https://gitlab.com/u/r/blob/[sha]/[file]#L[line]'. Null clears it. (string or null, optional)
  - `user_search_field` : A field such as 'context.user_email' that you provide in your error context. Null clears it. (string or null, optional)
  - `user_url` : User link shown on faults, e.g. 'http://example.com/admin/users/[user_id]'. Null clears it. (string or null, optional)

- **delete_project** - Delete a Honeybadger project and all of its data. Deleting takes two calls: the first deletes nothing and returns a preview with a confirmation token. Show the preview to the user, and only if they approve, call again with the same arguments and confirm set to that token. _(changes data; hidden in read-only mode)_
  - `project_id` : The ID of the project to delete (string, required)
  - `confirm` : Confirmation token from this tool's previous preview. Pass it only after the user has approved the deletion shown in that preview. (string, optional)

### Integrations

- **get_integration** - Get a single notification integration by ID. To interpret integration fields, fetch reference topic: integrations (via get_reference).
  - `integration_id` : The ID of the integration to retrieve (string, required)
  - `project_id` : The ID of the project the integration belongs to (string, required)

- **get_project_integrations** - List notification integrations for a Honeybadger project. To interpret integration types and config fields, fetch reference topic: integrations (via get_reference).
  - `project_id` : The ID of the project to get integrations for (string, required)

- **create_integration** - Create a notification integration for a project. IMPORTANT: Requires reference topic: integrations — fetch via get_reference first (skip if still visible in your context) for the list of types, each type's config settings, and event names. _(changes data; hidden in read-only mode)_
  - `project_id` : The ID of the project to create the integration in (string, required)
  - `type` : Integration type. OAuth types (Slack, GitHub, ...) start unconnected: give the user the result's links.web to connect. (string, required)
  - `active` : Whether the integration sends notifications (boolean, optional)
  - `alarm_alert_ids` : Alarms whose alert events notify (array or null, optional)
  - `alarm_ok_ids` : Alarms whose recovery events notify (array or null, optional)
  - `check_in_ids` : Check-ins that notify: a list, [] for none, or null for all (including later ones). Omit to follow all on create. (array or null, optional)
  - `config` : The type's own settings, e.g. {"url": ...} for WebHook. Shared settings (events, site_ids, ...) are separate. (object, optional)
  - `events` : Event names to notify on; one the type doesn't support is refused. Null (or omit on create) for the type's defaults. (array or null, optional)
  - `excluded_environments` : Environments that never notify, even ones yet to report; wins over included_environments. (array or null, optional)
  - `filters` : {event, query} pairs: an event with filters notifies only for errors matching one. event "all" for every event. (array or null, optional)
  - `included_environments` : If not empty, the only environments that notify, reported yet or not. Empty: every one not excluded. (array or null, optional)
  - `notification_limit` : Most notifications in a 10-minute window before flood control (the flooded event) steps in (integer or null, optional)
  - `rate` : Period the rate_exceeded threshold counts over; wday is 24 hours, not a weekday. (string or null, optional)
  - `site_ids` : Sites that notify: a list, [] for none, or null for all (including later ones). Omit to follow all on create. (array or null, optional)
  - `threshold` : Occurrences within rate before rate_exceeded fires; must be greater than 0 (integer or null, optional)

- **update_integration** - Update a notification integration. Only the parameters given are changed; the type can't be. IMPORTANT: Requires reference topic: integrations — fetch via get_reference first (skip if still visible in your context) for config settings and event names. _(changes data; hidden in read-only mode)_
  - `integration_id` : The ID of the integration to update (string, required)
  - `project_id` : The ID of the project the integration belongs to (string, required)
  - `active` : Whether the integration sends notifications (boolean, optional)
  - `alarm_alert_ids` : Alarms whose alert events notify (array or null, optional)
  - `alarm_ok_ids` : Alarms whose recovery events notify (array or null, optional)
  - `check_in_ids` : Check-ins that notify: a list, [] for none, or null for all (including later ones). Omit to follow all on create. (array or null, optional)
  - `config` : The type's settings to change, keyed as get_integration returns them. A secret sent back masked stays as it is. (object, optional)
  - `events` : Event names to notify on; one the type doesn't support is refused. Null (or omit on create) for the type's defaults. (array or null, optional)
  - `excluded_environments` : Environments that never notify, even ones yet to report; wins over included_environments. (array or null, optional)
  - `filters` : {event, query} pairs: an event with filters notifies only for errors matching one. event "all" for every event. (array or null, optional)
  - `included_environments` : If not empty, the only environments that notify, reported yet or not. Empty: every one not excluded. (array or null, optional)
  - `notification_limit` : Most notifications in a 10-minute window before flood control (the flooded event) steps in (integer or null, optional)
  - `rate` : Period the rate_exceeded threshold counts over; wday is 24 hours, not a weekday. (string or null, optional)
  - `site_ids` : Sites that notify: a list, [] for none, or null for all (including later ones). Omit to follow all on create. (array or null, optional)
  - `threshold` : Occurrences within rate before rate_exceeded fires; must be greater than 0 (integer or null, optional)

- **delete_integration** - Delete a notification integration. Faults lose their links to the issues it created; the issues stay in the tracker. Deleting takes two calls: the first deletes nothing and returns a preview with a confirmation token. Show the preview to the user, and only if they approve, call again with the same arguments and confirm set to that token. _(changes data; hidden in read-only mode)_
  - `integration_id` : The ID of the integration to delete (string, required)
  - `project_id` : The ID of the project the integration belongs to (string, required)
  - `confirm` : Confirmation token from this tool's previous preview. Pass it only after the user has approved the deletion shown in that preview. (string, optional)

### Ingestion keys

- **list_ingestion_keys** - List a project's Ingestion Keys: the hbp_ keys an app uses to send errors and events to Honeybadger, set as ingestion_key in the client's config (api_key in older clients). An Ingestion Key isn't a secret, since it ships in apps, and it can't call the Data API; that takes an API Token.
  - `project_id` : The ID of the project to list Ingestion Keys for (string, required)

- **create_ingestion_key** - Create a new Ingestion Key for a project. The key is returned in the response. _(changes data; hidden in read-only mode)_
  - `project_id` : The ID of the project to create the Ingestion Key in (string, required)
  - `label` : Optional human-readable name for the key (string or null, optional)

- **update_ingestion_key** - Update an Ingestion Key's label. _(changes data; hidden in read-only mode)_
  - `ingestion_key_id` : The ID of the Ingestion Key to update (string, required)
  - `project_id` : The ID of the project the Ingestion Key belongs to (string, required)
  - `label` : New label for the key; null clears it (string or null, optional)

- **delete_ingestion_key** - Delete an Ingestion Key. Apps sending with this key will no longer be able to send data. Deleting takes two calls: the first deletes nothing and returns a preview with a confirmation token. Show the preview to the user, and only if they approve, call again with the same arguments and confirm set to that token. _(changes data; hidden in read-only mode)_
  - `ingestion_key_id` : The ID of the Ingestion Key to delete (string, required)
  - `project_id` : The ID of the project the Ingestion Key belongs to (string, required)
  - `confirm` : Confirmation token from this tool's previous preview. Pass it only after the user has approved the deletion shown in that preview. (string, optional)


<!-- END GENERATED TOOLS -->

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
