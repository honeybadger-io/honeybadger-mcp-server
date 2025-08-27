# Honeybadger MCP Server

An MCP (Model Context Protocol) server for [Honeybadger](https://www.honeybadger.io), providing structured
access to Honeybadger's API through the MCP protocol.

## Installation

First, pull the Docker image:

```bash
docker pull ghcr.io/honeybadger-io/honeybadger-mcp-server:latest
```

Then, configure your MCP client(s). You can find your personal auth token under the "Authentication" tab in your [Honeybadger User settings](https://app.honeybadger.io/users/edit).

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

| Environment Variable              | Required | Default                    | Description                                 |
| --------------------------------- | -------- | -------------------------- | ------------------------------------------- |
| `HONEYBADGER_PERSONAL_AUTH_TOKEN` | yes      | —                          | API token for Honeybadger                   |
| `LOG_LEVEL`                       | no       | info                       | Log verbosity (debug, info, warn, error)    |
| `HONEYBADGER_API_URL`             | no       | https://app.honeybadger.io | Override the base URL for Honeybadger's API |

### Command Line Options

When running the server via the CLI you can configure the server with command-line flags:

```bash
# Run with custom configuration
./honeybadger-mcp-server stdio --auth-token your_token --log-level debug --api-url https://custom.honeybadger.io

# Get help
./honeybadger-mcp-server stdio --help
```

### Configuration File

You can also use a configuration file at `~/.honeybadger-mcp-server.yaml`:

```yaml
auth-token: "your_token_here"
log-level: "info"
api-url: "https://app.honeybadger.io"
```

## Tools

### Projects

- **list_projects** - List all Honeybadger projects

  - `account_id` : Account ID to filter projects by specific account (string, optional)

- **get_project** - Get detailed information for a single project by ID

  - `id` : The ID of the project to retrieve (number, required)

- **create_project** - Create a new Honeybadger project

  - `account_id` : The account ID to associate the project with (string, required)
  - `name` : The name of the new project (string, required)
  - `resolve_errors_on_deploy` : Whether all unresolved faults should be marked as resolved when a deploy is recorded (boolean, optional)
  - `disable_public_links` : Whether to allow fault details to be publicly shareable via a button on the fault detail page (boolean, optional)
  - `user_url` : A URL format like 'http://example.com/admin/users/[user_id]' that will be displayed on the fault detail page (string, optional)
  - `source_url` : A URL format like 'https://gitlab.com/username/reponame/blob/[sha]/[file]#L[line]' that is used to link lines in the backtrace to your git browser (string, optional)
  - `purge_days` : The number of days to retain data (up to the max number of days available to your subscription plan) (number, optional)
  - `user_search_field` : A field such as 'context.user_email' that you provide in your error context (string, optional)

- **update_project** - Update an existing Honeybadger project

  - `id` : The ID of the project to update (number, required)
  - `name` : The name of the project (string, optional)
  - `resolve_errors_on_deploy` : Whether all unresolved faults should be marked as resolved when a deploy is recorded (boolean, optional)
  - `disable_public_links` : Whether to allow fault details to be publicly shareable via a button on the fault detail page (boolean, optional)
  - `user_url` : A URL format like 'http://example.com/admin/users/[user_id]' that will be displayed on the fault detail page (string, optional)
  - `source_url` : A URL format like 'https://gitlab.com/username/reponame/blob/[sha]/[file]#L[line]' that is used to link lines in the backtrace to your git browser (string, optional)
  - `purge_days` : The number of days to retain data (up to the max number of days available to your subscription plan) (number, optional)
  - `user_search_field` : A field such as 'context.user_email' that you provide in your error context (string, optional)

- **delete_project** - Delete a Honeybadger project

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

## Development

Run the tests:

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
