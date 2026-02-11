# Honeybadger Insights Reference

## Cheat Sheet

```
stats count() as count                           # count events
stats count() as count by field::str             # count grouped by field
stats count() as count by bin(1h)                # time series (hourly)
filter field::str == "value"                     # filter events
filter field::int >= 400 | stats count() as count  # filter then aggregate
stats avg(field::int) as avg by group::str       # average by group
| sort avg desc | limit 10                       # sort + limit results
| only @ts, field1, field2                       # restrict output columns
```

**Always alias aggregate functions** — `stats count() as count`, not `stats count()`. Without an alias, the output column is literally `count()`, which is hard to reference in `sort`, `filter`, or a second `stats`.

Pipe `|` joins functions. Type-hint fields once with `::int`, `::str`, `::float`, `::bool` to tell BadgerQL which storage bucket to find the data in.

## Quickstart

### How to call query_insights

Call the `query_insights` tool with these arguments:

```json
{
  "project_id": 12345,
  "query": "stats count() as count by event_type::str | sort count desc",
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

**1. What event types exist?** (always run this first to discover available data)
```
stats count() as count by event_type::str
| sort count desc
```

**2. What fields are on an event type?** (this single query returns one sample event per event type — showing all field names, values, and actual types so you can pick the right `::type` hint)
```
fields @preview | limit 1 by event_type::str
```

This returns one sample for **every** event type in one call. Do NOT run separate per-event-type queries — use this single query to see all fields across all event types at once. Different event types have different fields. For example, `process_action.action_controller` events have `controller`, `action`, `duration` but likely not `country` or `session_id`. Custom events like `checkout.intent` may have `country`, `gateway`, etc. Run this query before writing analytical queries to avoid wasting round trips guessing at field names or type hints.

**3. When investigating a problem**, break down by every low-cardinality categorical field. Review `@preview` and identify fields that represent categories or dimensions (e.g., types, statuses, names, regions) rather than unique identifiers (e.g., IDs, timestamps, emails). Include each noteworthy field as a `by` field in your analytical queries, or explicitly state why it is irrelevant. Skip high-cardinality fields like IDs or free-text values. Presenting conclusions without checking all available dimensions is an incomplete analysis — do not summarize findings until every noteworthy field has been considered or ruled out.

**4. Error count over time:**
```
filter event_type::str == "notice"
| stats count() as count by bin(1h)
```

**5. Top controllers by request count:**
```
filter event_type::str == "process_action.action_controller"
| stats count() as count by controller::str
| sort count desc
| limit 10
```

**6. Latency percentiles by endpoint:**
```
filter event_type::str == "process_action.action_controller"
| stats
    percentile(50, duration::float) as p50,
    percentile(95, duration::float) as p95,
    percentile(99, duration::float) as p99
  by controller::str, action::str
| sort p95 desc
```

**7. Unique users affected by faults:**
```
filter event_type::str == "notice"
| stats unique(user::str) as `Affected Users` by fault_id::int
| sort `Affected Users` desc
| limit 10
```

**8. Status code breakdown over time:**
```
filter event_type::str == "process_action.action_controller"
| fields cond(
    status::int between 200 and 299, "2XX",
    status::int between 300 and 399, "3XX",
    status::int between 400 and 499, "4XX",
    status::int >= 500, "5XX",
    "OTHER"
  ) as status_group
| stats count() as count by status_group, bin(1h) as `time`
| sort time asc
```

**9. Apdex score by controller:**
```
filter event_type::str == "request.handled"
| stats round(apdex(duration::int, 200000), 3) as apdex by controller::str
| sort apdex asc
```

### Best practices for writing queries

- **Always run the discovery queries first** (starter queries 1 and 2 above). Run `@preview` for every event type you plan to query. This tells you exactly which fields exist and what types they are, so you don't waste queries guessing.
- **Pick type hints from `@preview` output** — if `@preview` shows `duration: 28.7`, use `::float`, not `::int`. If it shows `status: 200`, use `::int`. Getting this wrong returns empty results and costs a round trip.
- **Don't assume fields exist across event types** — each event type has its own set of fields. A field like `country` may exist on `checkout.intent` but not on `process_action.action_controller`. Always check `@preview` first rather than querying with `filter isNotNull(field::str)` to probe for fields.
- **Hint types once** — `filter status::int > 400 | stats count() as count by status` works; don't re-hint `status` after the first use.
- **Keep output small** — always end with `| sort ... | limit N` and use `| only` to restrict columns. Large unbounded results are slow and hard to read.
- **Use `event_type::str`** to filter to the kind of event you care about. Common event types include `notice`, `request.handled`, `process_action.action_controller`, `sql.active_record`, `perform.sidekiq`, `uptime_check`.

### Correlating across event types

When fields you need are spread across different event types, group by a shared key (e.g. `session_id`) and use `first()`/`last()` to pull fields from different event types into the same aggregation row. These functions skip null values, so if a field only exists on one event type, `first()` will find it regardless of row order.

```
filter event_type::str in ["process_action.action_controller", "checkout.intent"]
| stats first(country::str) as country,
        first(duration::float) as duration,
        first(controller::str) as controller
    by session_id::str
| filter isNotNull(country) and controller == "CheckoutController"
| stats avg(duration) as avg_dur, count() as cnt by country
| sort avg_dur desc
```

Note: double aggregation (`stats | filter | stats`) works, but the second `stats` and any `filter` between them must reference the **aliases** produced by the first `stats`, not the original field names with type hints. After the first `stats`, the original fields no longer exist — only the aliases do.

### Common pitfalls

- **Unaliased aggregate functions**: `stats count()` produces a column named `count()`, not `count`. Always alias: `stats count() as count`. Without an alias, `sort count desc` will fail because there is no column named `count`.
- **Missing pipes**: Every function after the first must be preceded by `|`. Multi-line queries need `|` at the start of each new function line.
- **Wrong type hint**: Data is stored per type, so `status::str` looks in the string bucket — if the data is actually stored as an integer, you'll get empty results or a type error. Use `::int` for whole numbers, `::float` for decimals, `::str` for strings. Check `@preview` output to see actual values and pick the right hint. A common mistake is using `::int` for a field that is actually a float (e.g., `duration`).
- **Forgetting type hints**: Without a hint, BadgerQL doesn't know which storage bucket to look in, which can produce errors or empty results.
- **Assuming fields exist on an event type**: Don't blindly query fields like `country::str` or `session_id::str` on event types that may not have them. Run the `@preview` discovery query first. If you need data from multiple event types, see "Correlating across event types" above.
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
| stats count() as count by controller
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
stats count() as count
stats count() as count by status_code::int
stats avg(duration::int) as avg, count() as count by controller::str, action::str
stats percentile(95, duration::int) as p95 by bin(1h)
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
    "query": "stats count() as count by name::str",
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
| stats count() as count by bin(1h)
```

URL:
```
/projects/5/insights/query?query=filter+status%3A%3Aint+%3E%3D+500%0A%7C+stats+count%28%29+as+count+by+bin%281h%29&view=line&xField=bin(1h)&ts=2026-01-22T00:00:00/2026-01-29T00:00:00
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

**Before creating or updating a dashboard, test every widget query with `query_insights` first.** Dashboard creation will not validate queries, so broken queries will silently produce empty or erroring widgets. Run each query individually to confirm it returns the expected data, then use those validated queries in the widget configs.

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
  "widgets": "[{\"type\":\"insights_vis\",\"presentation\":{\"title\":\"Error Rate\"},\"grid\":{\"x\":0,\"y\":0,\"w\":6,\"h\":4},\"config\":{\"query\":\"filter event_type::str == \\\"notice\\\" | stats count() as count by bin(1h)\",\"vis\":{\"view\":\"line\"}}},{\"type\":\"errors\",\"presentation\":{\"title\":\"Recent Errors\"},\"grid\":{\"x\":6,\"y\":0,\"w\":6,\"h\":4},\"config\":{\"limit\":10,\"sort\":\"last_seen_desc\"}}]"
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

# Alarms

Alarms monitor Insights queries and trigger notifications when conditions are met. Use the alarm tools to create, update, list, get, delete alarms, and view their trigger history.

## Tools

| Tool | Description |
|------|-------------|
| `list_alarms` | List all alarms for a project |
| `get_alarm` | Get a single alarm by ID |
| `create_alarm` | Create a new alarm |
| `update_alarm` | Update an existing alarm |
| `delete_alarm` | Delete an alarm |
| `get_alarm_history` | Get trigger history for an alarm |

## Creating an Alarm

Call `create_alarm` with:

- `project_id` (required) — integer project ID.
- `name` (required) — alarm name string.
- `query` (required) — BadgerQL query string. For count-based alarms (`alert_result_count`), this can be any query that returns rows (e.g., a `filter`); the system automatically wraps it to count matching results per evaluation period.
- `evaluation_period` (required) — how often the alarm is evaluated (e.g., `5m`, `1h`, `1d`). Minimum `1m`.
- `trigger_config` (required) — JSON string defining when to trigger the alarm (see below).
- `description` (optional) — alarm description.
- `stream_ids` (optional) — JSON array of stream IDs to query (defaults to `["default"]`).
- `lookback_lag` (required) — delay before evaluating to allow data to arrive (e.g., `1m`, or `0s` for no lag).

### Example

```json
{
  "project_id": 12345,
  "name": "High Error Rate",
  "description": "Triggers when error count exceeds 100 in 5 minutes",
  "query": "filter event_type::str == \"notice\"",
  "evaluation_period": "5m",
  "lookback_lag": "1m",
  "trigger_config": "{\"type\": \"alert_result_count\", \"config\": {\"operator\": \"gt\", \"value\": 100}}"
}
```

## Trigger Config

The `trigger_config` defines when an alarm transitions to the triggered state.

### Structure

```json
{
  "type": "alert_result_count",
  "config": {
    "operator": "gt",
    "value": 100
  }
}
```

### Types

| Type | Description |
|------|-------------|
| `alert_result_count` | Triggers based on the count of events matching the query |

### Config Fields

| Field | Type | Description |
|-------|------|-------------|
| `operator` | string | Comparison operator (required) |
| `value` | integer | Threshold value to compare against (required, >= 0) |

### Operators

| Operator | Description |
|----------|-------------|
| `gt` | Greater than |
| `gte` | Greater than or equal |
| `lt` | Less than |
| `lte` | Less than or equal |
| `eq` | Equal |
| `neq` | Not equal |

### Examples

**Trigger when error count exceeds 50:**
```json
{"type": "alert_result_count", "config": {"operator": "gt", "value": 50}}
```

**Trigger when count drops below threshold:**
```json
{"type": "alert_result_count", "config": {"operator": "lt", "value": 10}}
```

**Trigger when exactly zero events (missing heartbeat):**
```json
{"type": "alert_result_count", "config": {"operator": "eq", "value": 0}}
```

**Trigger when any events exist:**
```json
{"type": "alert_result_count", "config": {"operator": "neq", "value": 0}}
```

## Alarm States

| State | Description |
|-------|-------------|
| `ok` | Query result does not meet trigger condition |
| `alarm` | Query result meets trigger condition (alarm is triggered) |

Note: An alarm with a non-null `error` field indicates the query failed to execute. Check the `error` field for details.

## Query Guidelines for Alarms

The alarm system automatically wraps your query to count results per evaluation period. Your query should filter and/or aggregate events; the system handles the final counting.

**Good queries:**
```
filter event_type::str == "notice"
filter status::int >= 500
filter event_type::str == "request.handled" and duration::int > 5000
```

**Queries with stats work too** (system counts the result rows):
```
filter event_type::str == "notice" | stats count() as count by fault_id::int
```

## Alarm History

Use `get_alarm_history` to see past trigger events. Each trigger record includes:

| Field | Description |
|-------|-------------|
| `id` | Trigger event ID |
| `state` | State after evaluation (`ok`, `alarm`, `error`) |
| `result` | Query result that caused this state |
| `created_at` | When the evaluation occurred |

## Updating an Alarm

Call `update_alarm` with `project_id`, `alarm_id`, and the fields to update. All required fields (`name`, `query`, `evaluation_period`, `trigger_config`) must be provided even if unchanged.

## Evaluation Timing

- **evaluation_period** — How often the alarm checks (e.g., `5m`, `1h`, `1d`). Minimum `1m`, must be more granular than a week.
- **lookback_lag** — Delay before evaluating to allow data to arrive (e.g., `1m`, or `0s` for no lag).

The alarm evaluates at each period boundary, looking back over the `evaluation_period` duration (offset by `lookback_lag` if set).

## Common Alarm Patterns

**Error spike detection:**
```
name: "Error Spike"
query: "filter event_type::str == \"notice\""
evaluation_period: "5m"
lookback_lag: "1m"
trigger_config: {"type": "alert_result_count", "config": {"operator": "gt", "value": 100}}
```

**Slow requests:**
```
name: "Slow Requests"
query: "filter event_type::str == \"request.handled\" and duration::int > 5000"
evaluation_period: "5m"
lookback_lag: "1m"
trigger_config: {"type": "alert_result_count", "config": {"operator": "gt", "value": 10}}
```

**Missing heartbeat (no events in period):**
```
name: "Missing Heartbeat"
query: "filter event_type::str == \"heartbeat\""
evaluation_period: "10m"
lookback_lag: "0m"
trigger_config: {"type": "alert_result_count", "config": {"operator": "eq", "value": 0}}
```

**5xx errors:**
```
name: "Server Errors"
query: "filter event_type::str == \"request.handled\" and status::int >= 500"
evaluation_period: "5m"
lookback_lag: "1m"
trigger_config: {"type": "alert_result_count", "config": {"operator": "gt", "value": 0}}
```

## Presenting Alarm Links

After creating or updating an alarm, present the user with a direct link so they can view it in the Honeybadger UI:

```
https://app.honeybadger.io/projects/{project_id}/insights/alarms/{alarm_id}
```
