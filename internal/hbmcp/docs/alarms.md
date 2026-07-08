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
