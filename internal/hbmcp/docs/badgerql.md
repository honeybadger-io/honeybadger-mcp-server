# BadgerQL Query Reference

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

