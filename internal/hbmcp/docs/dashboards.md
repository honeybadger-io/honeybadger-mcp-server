# Dashboards

Dashboards are collections of widgets displayed on a project's Insights page. Use the dashboard tools to create, update, list, get, and delete dashboards.

## Tools

| Tool | Description |
|------|-------------|
| `list_dashboards` | List all dashboards for a project |
| `get_dashboard` | Get a single dashboard by ID |
| `create_dashboard` | Create a new dashboard |
| `update_dashboard` | Update an existing dashboard |
| `delete_dashboard` | Delete a dashboard |

## Creating a Dashboard

**Before creating or updating a dashboard, test every widget query with `query_insights` first.** Dashboard creation will not validate queries, so broken queries will silently produce empty or erroring widgets. Run each query individually to confirm it returns the expected data, then use those validated queries in the widget configs.

Call `create_dashboard` with:

- `project_id` (required) — integer project ID.
- `title` (required) — dashboard title string (max 255 characters).
- `widgets` (required) — JSON string containing an array of widget objects.
- `default_ts` (optional) — default time range. ISO 8601 duration (e.g., `P1D`, `PT3H`) or keyword (`today`, `yesterday`, `week`, `month`).

### Example

```json
{
  "project_id": 12345,
  "title": "Production Overview",
  "default_ts": "P1D",
  "widgets": "[{\"type\":\"insights_vis\",\"presentation\":{\"title\":\"Error Rate\"},\"grid\":{\"x\":0,\"y\":0,\"w\":6,\"h\":4},\"config\":{\"query\":\"filter event_type::str == \\\"notice\\\" | stats count() as count by bin(1h)\",\"vis\":{\"view\":\"line\"}}},{\"type\":\"errors\",\"presentation\":{\"title\":\"Recent Errors\"},\"grid\":{\"x\":6,\"y\":0,\"w\":6,\"h\":4},\"config\":{\"limit\":10,\"sort\":\"last_seen_desc\"}}]"
}
```

## Widget Structure

Each widget object has:

| Field | Required | Description |
|-------|----------|-------------|
| `type` | yes | Widget type (see below) |
| `id` | no | Stable widget identifier. Omit when creating; the server assigns one. Preserve existing `id`s when updating so widget state/history is retained |
| `grid` | no | Layout position: `{x, y, w, h}` (integers, grid units) |
| `presentation` | no | Display options: `{title, subtitle}` |
| `config` | no | Type-specific configuration |

## Widget Types

### `insights_vis` — Insights Query Visualization

Displays a BadgerQL query result as a chart or table.

**Config:**

| Field | Description |
|-------|-------------|
| `query` | BadgerQL query string |
| `streams` | Array of stream names: `["default"]`, `["internal"]`, or both |
| `vis` | Visualization settings (see below) |

**`vis` object:**

| Field | Required | Description |
|-------|----------|-------------|
| `view` | yes | Chart type: `table`, `billboard`, `line`, `area`, `bar`, `histogram`, `scatter`, `heatmap`, `pie` |
| `chart_config` | no | View-specific options (see the `charts` reference topic) |

**Example — line chart of error counts:**
```json
{
  "type": "insights_vis",
  "presentation": {"title": "Errors Over Time"},
  "config": {
    "query": "filter event_type::str == \"notice\" | stats count() as count by bin(1h)",
    "vis": {"view": "line"}
  }
}
```

**Example — bar chart of top controllers:**
```json
{
  "type": "insights_vis",
  "presentation": {"title": "Top Controllers"},
  "config": {
    "query": "filter event_type::str == \"process_action.action_controller\" | stats count() as count by controller::str | sort count desc | limit 10",
    "vis": {"view": "bar", "chart_config": {"categoryField": "controller", "valueField": "count"}}
  }
}
```

**Example — billboard showing total requests:**
```json
{
  "type": "insights_vis",
  "presentation": {"title": "Total Requests"},
  "config": {
    "query": "filter event_type::str == \"request.handled\" | stats count() as count",
    "vis": {"view": "billboard"}
  }
}
```

### `errors` — Error List

| Config Field | Description |
|-------------|-------------|
| `limit` | Max errors to show (integer) |
| `query` | Search string to filter errors |
| `sort` | Sort order: `last_seen_desc`, `last_seen_asc`, `times_desc`, `times_asc` |

### `alarms` — Alarm Status

| Config Field | Description |
|-------------|-------------|
| `limit` | Max alarms to show (integer) |
| `filter_state` | Filter: `all`, `triggered`, `ok` |

### `deployments` — Deploy History

| Config Field | Description |
|-------------|-------------|
| `limit` | Max deploys to show (integer) |
| `override_time` | Whether to use custom time range (boolean) |
| `ts` | Custom time range string |

### `checkins` — Check-In Status

| Config Field | Description |
|-------------|-------------|
| `limit` | Max check-ins to show (integer) |
| `sort_order` | Sort by: `state_name`, `name`, `last_reported` |

### `uptime` — Uptime Monitors

| Config Field | Description |
|-------------|-------------|
| `limit` | Max monitors to show (integer) |

## Updating a Dashboard

Call `update_dashboard` with `project_id`, `dashboard_id`, `title`, `widgets`, and optionally `default_ts`. The entire widget array is replaced — include all widgets, not just changed ones.

## Grid Layout

The dashboard uses a 12-column grid. Widget positions are set via `grid`:

- `x` — column offset (0–11)
- `y` — row offset (0 = top)
- `w` — width in columns (1–12)
- `h` — height in row units

**Widgets must not overlap.** A widget's `y` value must be >= the `y + h` of any widget above it in the same columns. Plan the layout on paper first — sketch out each row, tracking the next available `y` for each column before assigning positions. Overlapping or misaligned widgets will render incorrectly.

Example two-column layout:
```
Row 0:  Widget A: {x:0, y:0, w:6, h:4}    Widget B: {x:6, y:0, w:6, h:4}
Row 4:  Widget C: {x:0, y:4, w:12, h:4}
Row 8:  Widget D: {x:0, y:8, w:6, h:3}    Widget E: {x:6, y:8, w:6, h:3}
```

Note: widgets in the same row that sit side-by-side should share the same `y` value. The next row starts at `y + h` of the tallest widget in the current row.

## Presenting Dashboard Links

After creating or updating a dashboard, present the user with a direct link so they can view it in the Honeybadger UI:

```
https://app.honeybadger.io/projects/{project_id}/insights/dashboards/{dashboard_id}
```

