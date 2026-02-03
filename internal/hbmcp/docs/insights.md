# Honeybadger Insights Reference

## Cheat Sheet

```
stats count()                                    # count events
stats count() by field::str                      # count grouped by field
stats count() by bin(1h)                         # time series (hourly)
filter field::str == "value"                     # filter events
filter field::int >= 400 | stats count()         # filter then aggregate
stats avg(field::int) by group::str              # average by group
| sort avg desc | limit 10                       # sort + limit results
| only @ts, field1, field2                       # restrict output columns
```

Pipe `|` joins functions. Type-hint fields once with `::int`, `::str`, `::float`, `::bool` to tell BadgerQL which storage bucket to find the data in.

## Quickstart

### How to call query_insights

Call the `query_insights` tool with these arguments:

```json
{
  "project_id": 12345,
  "query": "stats count() by event_type::str | sort count desc",
  "ts": "P1D",
  "timezone": "America/New_York"
}
```

- `project_id` (required) — integer project ID.
- `query` (required) — BadgerQL query string.
- `ts` (optional) — time range. Defaults to `PT3H` (last 3 hours). Use `"week"`, `"P1D"`, etc. for wider windows. Format is ISO 8601 duration: `PT{n}H` (hours), `P{n}D` (days), `P{n}W` (weeks), or shortcuts `"today"`, `"yesterday"`, `"week"`, `"month"`.
- `timezone` (optional) — IANA timezone name for timestamp interpretation.

The response contains `results` (array of row objects) and `meta` (query info, field names, schema, row counts, time range). An optional `error` field may be omitted on success and is present as `{"message": "..."}` when an inline error occurs.

### Minimal working query

```
stats count()
```

This returns the total event count for the project within the default `PT3H` window. Start here to verify the project has data.

### Starter queries — copy and adapt

**1. What event types exist?** (run this first to discover available data)
```
stats count() by event_type::str
| sort count desc
```

**2. What fields are on an event type?** (inspect one event to see its fields)
```
filter event_type::str == "request.handled"
| limit 1
```

**3. Error count over time:**
```
filter event_type::str == "notice"
| stats count() by bin(1h)
```

**4. Top controllers by request count:**
```
filter event_type::str == "process_action.action_controller"
| stats count() by controller::str
| sort count desc
| limit 10
```

**5. Latency percentiles by endpoint:**
```
filter event_type::str == "process_action.action_controller"
| stats
    percentile(50, duration::float) as p50,
    percentile(95, duration::float) as p95,
    percentile(99, duration::float) as p99
  by controller::str, action::str
| sort p95 desc
```

**6. Unique users affected by faults:**
```
filter event_type::str == "notice"
| stats unique(user::str) as `Affected Users` by fault_id::int
| sort `Affected Users` desc
| limit 10
```

**7. Status code breakdown over time:**
```
filter event_type::str == "process_action.action_controller"
| fields cond(
    status::int between 200 and 299, "2XX",
    status::int between 300 and 399, "3XX",
    status::int between 400 and 499, "4XX",
    status::int >= 500, "5XX",
    "OTHER"
  ) as status_group
| stats count() by status_group, bin(1h) as `time`
| sort time asc
```

**8. Apdex score by controller:**
```
filter event_type::str == "request.handled"
| stats round(apdex(duration::int, 200000), 3) as apdex by controller::str
| sort apdex asc
```

### Best practices for writing queries

- **Start with `stats count()`** to verify data exists before building complex queries.
- **Discover fields** by filtering to one event type and using `| limit 1` to inspect a sample event.
- **Hint types once** — `filter status::int > 400 | stats count() by status` works; don't re-hint `status` after the first use.
- **Keep output small** — always end with `| sort ... | limit N` and use `| only` to restrict columns. Large unbounded results are slow and hard to read.
- **Use `event_type::str`** to filter to the kind of event you care about. Common event types include `notice`, `request.handled`, `process_action.action_controller`, `sql.active_record`, `perform.sidekiq`, `uptime_check`.

### Common pitfalls

- **Missing pipes**: Every function after the first must be preceded by `|`. Multi-line queries need `|` at the start of each new function line.
- **Wrong type hint**: Data is stored per type, so `status::str` looks in the string bucket — if the data is actually stored as an integer, you'll get empty results or a type error. Use `::int` for numbers, `::str` for strings.
- **Forgetting type hints**: Without a hint, BadgerQL doesn't know which storage bucket to look in, which can produce errors or empty results.
- **`bin()` without interval**: `bin()` auto-selects bucket size based on the query time range. Pass an explicit interval like `bin(1h)` for predictable output.
- **`ts` defaults to `PT3H`** (last 3 hours): If you're looking for older data, set `ts` explicitly (e.g., `"week"`, `"P1D"`, `"P30D"`).

### Error handling

Errors are returned in the response `error` field as `{"message": "..."}`. Common errors:

- `"query timed out"` — query too broad; add filters or reduce time range.
- Type errors (e.g., `"integer vs string"`) — fix the `::type` hint on the offending field.
- `"X is not a valid internal lookup"` — the `@field` name is misspelled; valid ones are `@ts`, `@id`, `@size`, `@stream.name`, `@stream.id`, `@received_ts`, `@query.start_at`, `@query.end_at`.

When you get an error, fix the query and retry — don't give up after one attempt.

---

## Query Structure

Functions are combined with the pipe operator `|`:

```
fields status_code::int, controller::str
| filter status_code > 400
| stats count() by controller
| sort count desc
| limit 10
```

## Type Hinting

Each field's data is stored in a separate bucket per type. The `::type` hint tells BadgerQL which bucket to look in — it's not a conversion, it's a lookup directive:

```
fields status_code::int
| filter email::str like "%example.com%"
| stats avg(duration::float) by controller::str
```

Available type hints: `int`, `float`, `str`, `bool`

You only need to hint a field once per query — subsequent uses remember the hint. If you hint a field with the wrong type, you'll get empty results (no data in that bucket) or a type error.

Nested fields use dot notation: `site.name::str`, `user.id::int`

Array fields use `[*]` to expand elements: `tags[*]::str`

Use backticks for aliases with spaces: `` as `Affected Users` ``

## Built-in Fields

- `@ts` — Event timestamp (datetime).
- `@id` — Event ID (string).
- `@received_ts` — Timestamp when the event was received (datetime).
- `@stream.id` — Stream ID (string).
- `@stream.name` — Stream name (string).
- `@size` — Event size in bytes (integer).
- `@query.start_at` — Start time of the query (datetime).
- `@query.end_at` — End time of the query (datetime).

## Base Functions

### fields

```
fields expr [as alias][, ...]
```

Add computed or renamed fields.

```
fields a as b
fields duration::int / 1000 as duration_sec
fields concat(first_name::str, " ", last_name::str) as full_name
```

### filter

```
filter boolean_expr [and|or ...]*
```

Exclude events that don't match the condition.

```
filter status_code::int >= 400
filter controller::str == "UsersController" and action::str == "show"
filter email::str match /.*@example\.com/
```

### sort

```
sort expr [desc|asc][, ...]*
```

Order results. Default is ascending.

```
sort count desc
sort name asc, created_at desc
```

### stats

```
stats agg_expr[, ...]* by [expr][, ...]*
```

Group and aggregate data. The `by` clause specifies grouping fields. Aggregate expressions must use an aggregate function (count, sum, avg, min, max, unique, percentile, first, last, apdex).

```
stats count()
stats count() by status_code::int
stats avg(duration::int), count() by controller::str, action::str
stats percentile(95, duration::int) by bin(1h)
```

### limit

```
limit integer
```

Restrict the number of returned results.

```
limit 25
```

### parse

```
parse expr /regex/
```

Extract fields from a string using named capture groups in a regular expression (re2 syntax).

```
parse controller::str /(?<prefix>\w+)Controller/
```

### only

```
only field[, field]*
```

Restrict and order the final output fields. Use this to keep responses small and relevant.

```
only @ts, controller, status_code, duration
```

## Expression Functions

### Comparison

| Function | Syntax | Description |
|----------|--------|-------------|
| `>` | `a > b` | Greater than |
| `<` | `a < b` | Less than |
| `==` | `a == b` | Equal |
| `!=` / `<>` | `a != b` | Not equal |
| `<=` | `a <= b` | Less than or equal |
| `>=` | `a >= b` | Greater than or equal |
| `between` | `a between b and c` | Inclusive range |
| `not between` | `a not between b and c` | Outside range |
| `in` | `a in [v1, v2, ...]` | Value in array |
| `not in` | `a not in [v1, v2, ...]` | Value not in array |
| `isNull` | `isNull(expr)` | True if null |
| `isNotNull` | `isNotNull(expr)` | True if not null |
| `like` | `a like "pattern"` | Wildcard match (case-sensitive). `%` = any chars, `_` = one char |
| `not like` | `a not like "pattern"` | Negated like |
| `ilike` | `a ilike "pattern"` | Wildcard match (case-insensitive) |
| `not ilike` | `a not ilike "pattern"` | Negated ilike |
| `match` | `a match /regex/` | Regex match (re2 syntax) |
| `not match` | `a not match /regex/` | Negated regex match |
| `either` | `either(a, b, c)` | Returns first non-null value (coalesce) |

### Logic

| Function | Syntax | Description |
|----------|--------|-------------|
| `and` | `a and b` | Logical AND |
| `or` | `a or b` | Logical OR |
| `not` | `not(expr)` | Logical NOT |
| `if` | `if(cond, then, else)` | Conditional branch |
| `cond` | `cond(bool, val, bool, val, ..., default)` | Multi-branch conditional (like if/else-if) |

### Arithmetic

| Function | Syntax | Description |
|----------|--------|-------------|
| `+` | `a + b` | Addition |
| `-` | `a - b` | Subtraction |
| `*` | `a * b` | Multiplication |
| `/` | `a / b` | Division (returns float) |
| `%` | `a % b` | Modulo |
| `abs` | `abs(n)` | Absolute value |
| `pow` | `pow(base, exp)` | Power |
| `log` | `log(n)` | Natural logarithm |
| `log2` | `log2(n)` | Base-2 logarithm |
| `log10` | `log10(n)` | Base-10 logarithm |
| `round` | `round(n[, decimals])` | Round to decimals (default 0) |
| `floor` | `floor(n[, decimals])` | Floor |
| `ceil` | `ceil(n[, decimals])` | Ceiling |
| `exp` | `exp(n)` | e^n |

### String

| Function | Syntax | Description |
|----------|--------|-------------|
| `trim` | `trim(s)` | Remove leading/trailing whitespace |
| `concat` | `concat(s1, s2, ...)` | Concatenate strings |
| `lowercase` | `lowercase(s)` | To lowercase |
| `uppercase` | `uppercase(s)` | To uppercase |
| `substring` | `substring(s, start, length)` | Extract substring (1-indexed start) |
| `length` | `length(s)` | String length |
| `replace` | `replace(s, match, replacement)` | Replace all occurrences (string or regex) |
| `replaceFirst` | `replaceFirst(s, match, replacement)` | Replace first occurrence (string or regex) |
| `startsWith` | `startsWith(s, prefix)` | True if string starts with prefix |
| `toHumanString` | `toHumanString(n, type)` | Format number for display |

`toHumanString` type options: `"number"` (default, comma-separated), `"bytes"`, `"short"`, `"milliseconds"`, `"microseconds"`.

### Dates & Times

| Function | Syntax | Description |
|----------|--------|-------------|
| `now` | `now()` | Current datetime |
| `toTimezone` | `toTimezone(dt, "zone")` | Convert to timezone |
| `toYear` | `toYear(dt)` | Extract year (integer) |
| `toHour` | `toHour(dt)` | Extract hour 0-23 (integer) |
| `toDay` | `toDay(dt)` | Extract day of month 1-31 (integer) |
| `toDayOfWeek` | `toDayOfWeek(dt)` | Day of week 1-7 (1=Monday) |
| `formatDate` | `formatDate(fmt[, dt])` | Format datetime. `dt` defaults to `@ts` |
| `bin` | `bin([interval[, dt]])` | Bucket into time intervals. Defaults to auto-sized bins on `@ts` |
| `toStartOf` | `toStartOf(interval[, dt])` | Start of interval |
| `toEndOf` | `toEndOf(interval[, dt])` | End of interval |

Interval syntax: `1m` (minute), `5m`, `15m`, `30m`, `1h`, `6h`, `1d`, `1w`, `1M` (month).

### Aggregate

Used inside `stats` expressions.

| Function | Syntax | Description |
|----------|--------|-------------|
| `count` | `count([expr])` | Count events. Optional boolean/field filters results. |
| `sum` | `sum(n)` | Sum numeric field |
| `avg` | `avg(n)` | Average |
| `min` | `min(expr)` | Minimum value |
| `max` | `max(expr)` | Maximum value |
| `unique` | `unique(expr)` | Count distinct values |
| `percentile` | `percentile(pct, n)` | Percentile (0-100). Approximated. |
| `first` | `first(expr)` | First encountered value |
| `last` | `last(expr)` | Last encountered value |
| `apdex` | `apdex(responseTime, threshold)` | Apdex score |

### Conversion

| Function | Syntax | Description |
|----------|--------|-------------|
| `toInt` | `toInt(expr)` | Convert to integer |
| `toFloat` | `toFloat(expr)` | Convert to float |
| `toString` | `toString(expr)` | Convert to string |
| `toDate` | `toDate(expr)` | Convert to date |
| `toDateTime` | `toDateTime(expr)` | Convert to datetime |
| `toUnix` | `toUnix(dt)` | Datetime to Unix timestamp (ms) |

### URL

| Function | Syntax | Description |
|----------|--------|-------------|
| `urlParameter` | `urlParameter(url, "key")` | Extract query parameter value |
| `urlPath` | `urlPath(url)` | Extract URL path |
| `urlDomain` | `urlDomain(url)` | Extract hostname |

### JSON

| Function | Syntax | Description |
|----------|--------|-------------|
| `json` | `json(str, "$.path")` | Extract scalar value from JSON string using JSONPath |

### Array

| Function | Syntax | Description |
|----------|--------|-------------|
| `any` | `any(boolean_expr)` | True if any array element matches. Use `[*]` to expand. |
| `all` | `all(boolean_expr)` | True if all array elements match. Use `[*]` to expand. |

## Time Range Parameter (`ts`)

The `ts` parameter on query_insights controls the time window.

**Shortcuts:**

| Value | Meaning |
|-------|---------|
| `today` | Since start of today |
| `yesterday` | Since start of yesterday |
| `week` | Since start of this week (Sunday) |
| `month` | Since start of this month |

**ISO 8601 durations** (relative to now):

`P8D` (8 days ago), `P1W` (1 week ago), `PT3H` (3 hours ago), `PT30M` (30 minutes ago), etc.

**Absolute date**: `2021-12-14T22:14:08` (from that time to now)

**Absolute range**: `2021-12-10T00:00/2021-12-12T00:00` (between two times)

**Abbreviated relative range**: `P1W/P0D` (from 1 week ago to today)

## Response Schema

```json
{
  "results": [
    {"ts": "2024-01-01T00:00:00Z", "count": 42, "name": "web"}
  ],
  "meta": {
    "query": "stats count() by name::str",
    "fields": ["ts", "count", "name"],
    "schema": [
      {"name": "ts", "type": "DateTime"},
      {"name": "count", "type": "UInt64"},
      {"name": "name", "type": "String"}
    ],
    "rows": 1,
    "total_rows": 1,
    "start_at": "2024-01-01T00:00:00Z",
    "end_at": "2024-01-01T03:00:00Z"
  }
}
```

On failure, an `error` field is included: `{"message": "..."}` describing the issue.

# Shareable Insights URLs

Construct URLs to link directly to query results in the Insights UI.

## URL Structure

```
/projects/{project_id}/insights/query
  ?query={url_encoded_badgerql}
  &view={view_type}
  &ts={time_range}
  &stream_ids={stream_filter}
  &{view_specific_params}
```

## Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `query` | yes | URL-encoded BadgerQL query |
| `view` | no | Visualization type (see below) |
| `ts` | no | Time range as ISO 8601 interval: `{start}/{end}` |
| `stream_ids` | no | JSON array of stream IDs to filter |

## View Types and Chart Config

Each view type accepts a `chart_config` object with the fields below. All views also accept an optional `groups` object for custom series colors (see below).

### `table`

No additional config.

### `billboard`

| Field | Required | Description |
|-------|----------|-------------|
| `titleField` | no | Field for title text |
| `valueField` | conditional | Required when `groupType` is `"events"` |
| `titleURLField` | no | Field containing URL for title link |
| `subtitleField` | no | Field for subtitle text |
| `statusField` | no | Field for status indicator |
| `groupType` | no | `"fields"` or `"events"` |

### `line`

| Field | Required | Description |
|-------|----------|-------------|
| `xField` | no | X-axis field |
| `yField` | conditional | Required when `groupType` is `"events"` |
| `zField` | no | Series/group field |
| `colorField` | no | Field for line color |
| `xFieldUnit` | no | Unit for X-axis values |
| `yFieldUnit` | no | Unit for Y-axis values |
| `groupType` | no | `"fields"` or `"events"` |
| `yAxisLabel` | no | Left Y-axis label |
| `yAxisMin` | no | Left Y-axis minimum (number) |
| `yAxisMax` | no | Left Y-axis maximum (number) |
| `rightYAxisFormat` | no | Format for right Y-axis |
| `rightYAxisLabel` | no | Right Y-axis label |
| `rightYAxisMin` | no | Right Y-axis minimum (number) |
| `rightYAxisMax` | no | Right Y-axis maximum (number) |

Line `groups` entries also accept `axis`: `"left"` or `"right"` to assign a series to the right Y-axis.

### `area`

| Field | Required | Description |
|-------|----------|-------------|
| `xField` | no | X-axis field |
| `yField` | conditional | Required when `groupType` is `"events"` |
| `zField` | no | Series/group field |
| `groupType` | no | `"fields"` or `"events"` |
| `stacked` | no | Stack series (boolean) |

### `bar`

| Field | Required | Description |
|-------|----------|-------------|
| `categoryField` | yes | Category axis field |
| `valueField` | yes | Value axis field |
| `valueFieldUnit` | no | Unit for value axis |
| `labelField` | no | Label field |
| `groupType` | no | `"fields"` or `"events"` |
| `groupField` | no | Field to group bars by |
| `horizontal` | no | Horizontal bars (boolean) |
| `stacked` | no | Stack bars (boolean) |

### `histogram`

| Field | Required | Description |
|-------|----------|-------------|
| `xField` | yes | X-axis field |
| `yField` | yes | Y-axis field |
| `zField` | no | Series/group field |
| `xFieldUnit` | no | Unit for X-axis values |
| `yFieldUnit` | no | Unit for Y-axis values |

### `scatter`

| Field | Required | Description |
|-------|----------|-------------|
| `xField` | yes | X-axis field |
| `yField` | yes | Y-axis field |
| `groupField` | no | Field to color points by |
| `scaleField` | no | Field to size points by |

### `heatmap`

| Field | Required | Description |
|-------|----------|-------------|
| `xField` | yes | X-axis field |
| `yField` | yes | Y-axis field |
| `zField` | yes | Value/intensity field |
| `yFieldUnit` | no | Unit for Y-axis values |
| `steps` | no | Number of color steps (integer, min 1) |

Note: Heatmap queries must include `| sort xField, yField` to render correctly.

### `pie`

| Field | Required | Description |
|-------|----------|-------------|
| `nameField` | yes | Slice label field |
| `valueField` | yes | Slice value field |

### `groups` — Custom Series Colors

All chart types accept a `groups` object in `chart_config` to assign colors to specific series. Keys are series names, values are objects with a `color` string:

```json
{
  "view": "line",
  "chart_config": {
    "yField": "count",
    "groups": {
      "web": {"color": "#4A90D9"},
      "api": {"color": "#E74C3C"}
    }
  }
}
```

For `line` charts, group entries can also include `"axis": "right"` to plot a series on the right Y-axis.

## Example

Query:
```
filter status::int >= 500
| stats count() by bin(1h)
```

URL:
```
/projects/5/insights/query?query=filter+status%3A%3Aint+%3E%3D+500%0A%7C+stats+count%28%29+by+bin%281h%29&view=line&xField=bin(1h)&ts=2026-01-22T00:00:00/2026-01-29T00:00:00
```

When surfacing findings, include these links so users can view results in the UI, adjust the query, or share with their team.

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

Call `create_dashboard` with:

- `project_id` (required) — integer project ID.
- `title` (required) — dashboard title string.
- `widgets` (required) — JSON string containing an array of widget objects.
- `default_ts` (optional) — default time range. ISO 8601 duration (e.g., `P1D`, `PT3H`) or keyword (`today`, `yesterday`, `week`, `month`).

### Example

```json
{
  "project_id": 12345,
  "title": "Production Overview",
  "default_ts": "P1D",
  "widgets": "[{\"type\":\"insights_vis\",\"presentation\":{\"title\":\"Error Rate\"},\"grid\":{\"x\":0,\"y\":0,\"w\":6,\"h\":4},\"config\":{\"query\":\"filter event_type::str == \\\"notice\\\" | stats count() by bin(1h)\",\"vis\":{\"view\":\"line\"}}},{\"type\":\"errors\",\"presentation\":{\"title\":\"Recent Errors\"},\"grid\":{\"x\":6,\"y\":0,\"w\":6,\"h\":4},\"config\":{\"limit\":10,\"sort\":\"last_seen_desc\"}}]"
}
```

## Widget Structure

Each widget object has:

| Field | Required | Description |
|-------|----------|-------------|
| `type` | yes | Widget type (see below) |
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
| `chart_config` | no | View-specific options (same as Shareable URL params above) |

**Example — line chart of error counts:**
```json
{
  "type": "insights_vis",
  "presentation": {"title": "Errors Over Time"},
  "config": {
    "query": "filter event_type::str == \"notice\" | stats count() by bin(1h)",
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
    "query": "filter event_type::str == \"process_action.action_controller\" | stats count() by controller::str | sort count desc | limit 10",
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
    "query": "filter event_type::str == \"request.handled\" | stats count()",
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

Example two-column layout:
```
Widget A: {x:0, y:0, w:6, h:4}    Widget B: {x:6, y:0, w:6, h:4}
Widget C: {x:0, y:4, w:12, h:4}
```
