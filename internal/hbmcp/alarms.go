package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterAlarmTools registers all alarm-related MCP tools
func RegisterAlarmTools(r *toolRegistrar, client *hbapi.Client) {
	// list_alarms tool
	r.AddTool(
		mcp.NewTool("list_alarms",
			mcp.WithDescription("List all Insights alarms for a Honeybadger project. Call get_insights_reference for alarm documentation."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to list alarms for"),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListAlarms(ctx, client, req)
		},
	)

	// get_alarm tool
	r.AddTool(
		mcp.NewTool("get_alarm",
			mcp.WithDescription("Get a single Insights alarm by ID. Call get_insights_reference for alarm documentation."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the alarm belongs to"),
				mcp.Min(1),
			),
			mcp.WithString("alarm_id",
				mcp.Required(),
				mcp.Description("The ID of the alarm to retrieve"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetAlarm(ctx, client, req)
		},
	)

	// create_alarm tool
	r.AddTool(
		mcp.NewTool("create_alarm",
			mcp.WithDescription("Create a new Insights alarm for a Honeybadger project. IMPORTANT: Call get_insights_reference first for full alarm documentation, trigger_config schema, and query guidelines."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the alarm in"),
				mcp.Min(1),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the alarm"),
			),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("BadgerQL query for the alarm (e.g., 'filter event_type::str == \"notice\"'). The alarm system wraps the query to count results automatically."),
			),
			mcp.WithString("evaluation_period",
				mcp.Required(),
				mcp.Description("How often the alarm is evaluated (e.g., 5m, 1h, 1d). Minimum 1m."),
			),
			mcp.WithString("trigger_config",
				mcp.Required(),
				mcp.Description("JSON object defining when to trigger the alarm. Example: {\"type\": \"alert_result_count\", \"config\": {\"operator\": \"gt\", \"value\": 10}}"),
			),
			mcp.WithString("description",
				mcp.Description("Optional description of the alarm"),
			),
			mcp.WithString("stream_ids",
				mcp.Description("Optional JSON array of stream IDs to query (defaults to [\"default\"])"),
			),
			mcp.WithString("lookback_lag",
				mcp.Required(),
				mcp.Description("Delay before evaluating to allow data to arrive (e.g., 1m, or 0s for no lag)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateAlarm(ctx, client, req)
		},
	)

	// update_alarm tool
	r.AddTool(
		mcp.NewTool("update_alarm",
			mcp.WithDescription("Update an existing Insights alarm. IMPORTANT: Call get_insights_reference first for full alarm documentation, trigger_config schema, and query guidelines."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the alarm belongs to"),
				mcp.Min(1),
			),
			mcp.WithString("alarm_id",
				mcp.Required(),
				mcp.Description("The ID of the alarm to update"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the alarm"),
			),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("BadgerQL query for the alarm"),
			),
			mcp.WithString("evaluation_period",
				mcp.Required(),
				mcp.Description("How often the alarm is evaluated (e.g., 5m, 1h, 1d). Minimum 1m."),
			),
			mcp.WithString("trigger_config",
				mcp.Required(),
				mcp.Description("JSON object defining when to trigger the alarm"),
			),
			mcp.WithString("description",
				mcp.Description("Optional description of the alarm"),
			),
			mcp.WithString("stream_ids",
				mcp.Description("Optional JSON array of stream IDs to query"),
			),
			mcp.WithString("lookback_lag",
				mcp.Required(),
				mcp.Description("Delay before evaluating to allow data to arrive (e.g., 1m, 0s for no lag)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateAlarm(ctx, client, req)
		},
	)

	// delete_alarm tool
	r.AddTool(
		mcp.NewTool("delete_alarm",
			mcp.WithDescription("Delete an Insights alarm. Call get_insights_reference for alarm documentation."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the alarm belongs to"),
				mcp.Min(1),
			),
			mcp.WithString("alarm_id",
				mcp.Required(),
				mcp.Description("The ID of the alarm to delete"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteAlarm(ctx, client, req)
		},
	)

	// get_alarm_history tool
	r.AddTool(
		mcp.NewTool("get_alarm_history",
			mcp.WithDescription("Get the trigger history for an Insights alarm. Call get_insights_reference for alarm documentation."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the alarm belongs to"),
				mcp.Min(1),
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
			return handleGetAlarmHistory(ctx, client, req)
		},
	)
}

func handleListAlarms(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
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

func handleGetAlarm(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
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

func handleCreateAlarm(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
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

	evaluationPeriod := req.GetString("evaluation_period", "")
	if evaluationPeriod == "" {
		return mcp.NewToolResultError("evaluation_period is required"), nil
	}

	triggerConfigJSON := req.GetString("trigger_config", "")
	if triggerConfigJSON == "" {
		return mcp.NewToolResultError("trigger_config is required"), nil
	}

	var triggerConfig map[string]interface{}
	if err := json.Unmarshal([]byte(triggerConfigJSON), &triggerConfig); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse trigger_config JSON: %v", err)), nil
	}

	lookbackLag := req.GetString("lookback_lag", "")
	if lookbackLag == "" {
		return mcp.NewToolResultError("lookback_lag is required"), nil
	}

	alarmReq := hbapi.AlarmRequest{
		Name:             name,
		Query:            query,
		EvaluationPeriod: evaluationPeriod,
		TriggerConfig:    triggerConfig,
		Description:      req.GetString("description", ""),
		LookbackLag:      lookbackLag,
	}

	// Parse optional stream_ids
	streamIDsJSON := req.GetString("stream_ids", "")
	if streamIDsJSON != "" {
		var streamIDs []string
		if err := json.Unmarshal([]byte(streamIDsJSON), &streamIDs); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse stream_ids JSON: %v", err)), nil
		}
		alarmReq.StreamIDs = streamIDs
	}

	alarm, err := client.Alarms.Create(ctx, projectID, alarmReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create alarm: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(alarm)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateAlarm(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	alarmID := req.GetString("alarm_id", "")
	if alarmID == "" {
		return mcp.NewToolResultError("alarm_id is required"), nil
	}

	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	evaluationPeriod := req.GetString("evaluation_period", "")
	if evaluationPeriod == "" {
		return mcp.NewToolResultError("evaluation_period is required"), nil
	}

	triggerConfigJSON := req.GetString("trigger_config", "")
	if triggerConfigJSON == "" {
		return mcp.NewToolResultError("trigger_config is required"), nil
	}

	var triggerConfig map[string]interface{}
	if err := json.Unmarshal([]byte(triggerConfigJSON), &triggerConfig); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse trigger_config JSON: %v", err)), nil
	}

	lookbackLag := req.GetString("lookback_lag", "")
	if lookbackLag == "" {
		return mcp.NewToolResultError("lookback_lag is required"), nil
	}

	alarmReq := hbapi.AlarmRequest{
		Name:             name,
		Query:            query,
		EvaluationPeriod: evaluationPeriod,
		TriggerConfig:    triggerConfig,
		Description:      req.GetString("description", ""),
		LookbackLag:      lookbackLag,
	}

	// Parse optional stream_ids
	streamIDsJSON := req.GetString("stream_ids", "")
	if streamIDsJSON != "" {
		var streamIDs []string
		if err := json.Unmarshal([]byte(streamIDsJSON), &streamIDs); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse stream_ids JSON: %v", err)), nil
		}
		alarmReq.StreamIDs = streamIDs
	}

	result, err := client.Alarms.Update(ctx, projectID, alarmID, alarmReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update alarm: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteAlarm(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	alarmID := req.GetString("alarm_id", "")
	if alarmID == "" {
		return mcp.NewToolResultError("alarm_id is required"), nil
	}

	result, err := client.Alarms.Delete(ctx, projectID, alarmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete alarm: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetAlarmHistory(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	alarmID := req.GetString("alarm_id", "")
	if alarmID == "" {
		return mcp.NewToolResultError("alarm_id is required"), nil
	}

	page := req.GetInt("page", 0)

	response, err := client.Alarms.History(ctx, projectID, alarmID, page)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get alarm history: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
