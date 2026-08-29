package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterAlarmTools registers all alarm-related MCP tools
// RegisterAlarmTools registers the alarm tools, all on v3.
func RegisterAlarmTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	// list_alarms tool
	r.AddTool(
		mcp.NewTool("list_alarms",
			mcp.WithTitleAnnotation("List Alarms"),
			mcp.WithDescription("List all Insights alarms for a Honeybadger project. To interpret alarm configuration, fetch reference topic: alarms (via get_reference)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to list alarms for"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListAlarms(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_alarm tool
	r.AddTool(
		mcp.NewTool("get_alarm",
			mcp.WithTitleAnnotation("Get Alarm"),
			mcp.WithDescription("Get a single Insights alarm by ID. To interpret alarm configuration, fetch reference topic: alarms (via get_reference)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the alarm belongs to"),
			),
			mcp.WithString("alarm_id",
				mcp.Required(),
				mcp.Description("The ID of the alarm to retrieve"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetAlarm(ctx, v3ClientFor(ctx), req)
		},
	)

	// create_alarm tool
	r.AddTool(
		mcp.NewTool("create_alarm",
			mcp.WithTitleAnnotation("Create Alarm"),
			mcp.WithDescription("Create a new Insights alarm for a Honeybadger project. IMPORTANT: Requires reference topics: alarms, queries, badgerql — fetch via get_reference first (skip topics still visible in your context) for the trigger_config schema and query guidelines. Verify the query returns the expected results via query_insights before creating the alarm."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the alarm in"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the alarm"),
			),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("BadgerQL query evaluated on each check. Requires reference topics: alarms, badgerql (fetch via get_reference)."),
			),
			mcp.WithString("evaluation_period",
				mcp.Description("Window each evaluation covers, as a compact duration: "+
					"'5m', '10m', '1h', '1d'. Spelled-out forms like '5 minutes' are rejected."),
			),
			mcp.WithString("lookback_lag",
				mcp.Description("How far behind now the evaluation window ends, allowing for "+
					"ingestion delay. Same compact format as evaluation_period ('1m'). Required "+
					"in practice: the API refuses a create with a blank lookback_lag."),
			),
			mcp.WithString("trigger_config",
				mcp.Description(`JSON object describing what turns the alarm on, e.g. {"type":"alert_result_count","config":{"operator":"gt","value":10}}. Operators are named (gt, lt) rather than symbolic. Without a trigger the alarm is created but never fires. The alarms reference topic has the full list of types.`),
			),
			mcp.WithString("stream_ids",
				mcp.Description("JSON array of stream IDs the query runs against. Omit to use every stream on the project."),
			),
			mcp.WithString("description",
				mcp.Description("Optional description of the alarm"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateAlarm(ctx, v3ClientFor(ctx), req)
		},
	)

	// update_alarm tool
	r.AddTool(
		mcp.NewTool("update_alarm",
			mcp.WithTitleAnnotation("Update Alarm"),
			mcp.WithDescription("Update an existing Insights alarm. IMPORTANT: Requires reference topics: alarms, queries, badgerql — fetch via get_reference first (skip topics still visible in your context) for the trigger_config schema and query guidelines."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the alarm belongs to"),
			),
			mcp.WithString("alarm_id",
				mcp.Required(),
				mcp.Description("The ID of the alarm to update"),
			),
			mcp.WithString("name",
				mcp.Description("A new name for the alarm"),
			),
			mcp.WithString("description",
				mcp.Description("A new description. Pass an empty string to clear it."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateAlarm(ctx, v3ClientFor(ctx), req)
		},
	)

	// delete_alarm tool
	r.AddTool(
		mcp.NewTool("delete_alarm",
			mcp.WithTitleAnnotation("Delete Alarm"),
			mcp.WithDescription("Delete an Insights alarm."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the alarm belongs to"),
			),
			mcp.WithString("alarm_id",
				mcp.Required(),
				mcp.Description("The ID of the alarm to delete"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteAlarm(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_alarm_history tool
	r.AddTool(
		mcp.NewTool("get_alarm_history",
			mcp.WithTitleAnnotation("Get Alarm History"),
			mcp.WithDescription("Get the trigger history for an Insights alarm. To interpret trigger records and alarm states, fetch reference topic: alarms (via get_reference)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the alarm belongs to"),
			),
			mcp.WithString("alarm_id",
				mcp.Required(),
				mcp.Description("The ID of the alarm to get history for"),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination (default: 0)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetAlarmHistory(ctx, v3ClientFor(ctx), req)
		},
	)
}

func handleListAlarms(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	response, err := client.Alarms.List(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list alarms: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetAlarm(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	alarmID := req.GetString("alarm_id", "")
	if alarmID == "" {
		return mcp.NewToolResultError("alarm_id is required"), nil
	}

	alarm, err := client.Alarms.Get(ctx, projectID, alarmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get alarm: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(alarm)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// alarmUpdateOnlyName lists the fields v3's alarm update cannot carry.
//
// Create takes the query, evaluation period, trigger and streams; update takes
// only name and description. So an alarm's behaviour cannot be changed after it
// exists, and a request trying to is refused rather than silently applying just
// the name.
var alarmUpdateOnlyName = []string{
	"query", "evaluation_period", "trigger_config", "lookback_lag", "stream_ids",
}

func handleCreateAlarm(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	params := apiv3.AlarmParams{
		Name:             name,
		Query:            query,
		EvaluationPeriod: req.GetString("evaluation_period", ""),
		LookbackLag:      req.GetString("lookback_lag", ""),
		Description:      req.GetString("description", ""),
	}

	// stream_ids and trigger_config arrive as JSON strings, matching how v2's tool
	// took them.
	if raw := req.GetString("stream_ids", ""); raw != "" {
		if err := json.Unmarshal([]byte(raw), &params.StreamIDs); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse stream_ids JSON: %v", err)), nil
		}
	}
	if raw := req.GetString("trigger_config", ""); raw != "" {
		var trigger struct {
			Type   string `json:"type"`
			Config struct {
				Operator string  `json:"operator"`
				Value    float32 `json:"value"`
			} `json:"config"`
		}
		if err := json.Unmarshal([]byte(raw), &trigger); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse trigger_config JSON: %v", err)), nil
		}
		params.Trigger = &apiv3.AlarmTrigger{
			Type:     trigger.Type,
			Operator: trigger.Config.Operator,
			Value:    trigger.Config.Value,
		}
	}

	alarm, err := client.Alarms.Create(ctx, projectID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create alarm: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(alarm)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleUpdateAlarm changes an alarm's name or description.
//
// Those are the only fields v3's update schema declares, so a request touching
// the alarm's behaviour is refused instead of applying a subset.
func handleUpdateAlarm(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	alarmID := req.GetString("alarm_id", "")
	if alarmID == "" {
		return mcp.NewToolResultError("alarm_id is required"), nil
	}
	if msg := rejectUnsupported(req, alarmUpdateOnlyName, "updating an alarm",
		"v3's alarm update accepts only name and description — delete and recreate the alarm "+
			"to change how it fires"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	// Presence, not emptiness: pointing at "" is how a caller clears a description,
	// while omitting the field leaves it alone.
	args := req.GetArguments()
	var params apiv3.AlarmUpdateParams
	if v, ok := args["name"].(string); ok {
		params.Name = &v
	}
	if v, ok := args["description"].(string); ok {
		params.Description = &v
	}
	if params.Name == nil && params.Description == nil {
		return mcp.NewToolResultError("at least one of name or description is required"), nil
	}

	alarm, err := client.Alarms.Update(ctx, projectID, alarmID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update alarm: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(alarm)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteAlarm(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	alarmID := req.GetString("alarm_id", "")
	if alarmID == "" {
		return mcp.NewToolResultError("alarm_id is required"), nil
	}

	err := client.Alarms.Delete(ctx, projectID, alarmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete alarm: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(map[string]any{"deleted": true, "alarm_id": alarmID})
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetAlarmHistory(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	alarmID := req.GetString("alarm_id", "")
	if alarmID == "" {
		return mcp.NewToolResultError("alarm_id is required"), nil
	}

	// This endpoint passes the query service's paging through, so it takes page
	// but no per_page.
	var opts []apiv3.Option
	if page := req.GetInt("page", 0); page > 0 {
		opts = append(opts, apiv3.Page(page, 0))
	}

	// ListHistory returns only the rows, so the page the caller asked for is
	// echoed back with them — otherwise there is no way to tell whether another
	// page exists without probing for it.
	page := req.GetInt("page", 0)
	if page == 0 {
		page = 1
	}

	response, err := client.Alarms.ListHistory(ctx, projectID, alarmID, opts...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get alarm history: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(map[string]any{
		"results":    response,
		"page":       page,
		"page_size":  len(response),
		"pagination": "Only the requested page is returned. Ask for the next page to find out whether more rows exist.",
	})
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
