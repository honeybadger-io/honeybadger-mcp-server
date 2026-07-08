# Charts & Visualizations

How to visualize Insights query results: the available view types, their `chart_config` options, and how to construct shareable URLs that link directly to query results in the Insights UI. The same `view` and `chart_config` values are used by `insights_vis` dashboard widgets.

## Shareable URL Structure

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

