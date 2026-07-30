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
func RegisterAlarmTools(r *toolRegistrar, clientFor ClientFactory, v3ClientFor V3ClientFactory) {
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
		// The alarm's configuration parameters are not advertised: v3's schema
		// cannot carry them, and required parameters the handler always refuses
		// would make these tools unusable for a conforming caller. create_alarm
		// refuses outright for the same reason — an alarm with no query never
		// fires. Both handlers still reject the fields if an older caller sends
		// them.
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
				mcp.Required(),
				mcp.Description("The name of the alarm"),
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

	response, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.ListResponse[apiv3.Alarm], error) {
			return client.Alarms.List(ctx, projectID, inAccount(accountID)...)
		})
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

	alarm, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.Alarm, error) {
			return client.Alarms.Get(ctx, projectID, alarmID, inAccount(accountID)...)
		})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get alarm: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(alarm)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// alarmConfigFields are the fields that make an alarm actually alarm. v3's write
// schema declares only name, so none of them can be sent.
var alarmConfigFields = []string{
	"query", "evaluation_period", "trigger_config", "lookback_lag",
	"stream_ids", "description",
}

// handleCreateAlarm refuses, deliberately.
//
// v3's create schema declares only name. An alarm with no query, evaluation
// period, or trigger configuration does not fire — creating one would leave a
// broken alarm in the account that looks real. Unlike a project, which is still a
// project with only a name, there is no useful alarm to create here, so this
// reports the gap instead of half-doing the job.
func handleCreateAlarm(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(
		"Creating alarms is not available on the v3 API yet: its create schema accepts only a " +
			"name, so the query, evaluation period and trigger configuration an alarm needs to " +
			"fire cannot be sent, and the result would be an alarm that never fires. " +
			"Create the alarm in the Honeybadger UI."), nil
}

// handleUpdateAlarm renames an alarm, which is all v3's schema permits.
//
// A rename is a genuinely useful subset, so unlike create this is allowed —
// but a request touching the alarm's configuration is refused rather than
// silently applying only the name.
func handleUpdateAlarm(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	alarmID := req.GetString("alarm_id", "")
	if alarmID == "" {
		return mcp.NewToolResultError("alarm_id is required"), nil
	}
	if msg := rejectUnsupported(req, alarmConfigFields, "updating an alarm",
		"v3's alarm write schema accepts only name; change the rest in the Honeybadger UI"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	alarm, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.Alarm, error) {
			return client.Alarms.Update(ctx, projectID, alarmID, name, inAccount(accountID)...)
		})
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

	_, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (any, error) {
			return nil, client.Alarms.Delete(ctx, projectID, alarmID, inAccount(accountID)...)
		})
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

	response, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) ([]apiv3.AlarmHistoryEntry, error) {
			return client.Alarms.ListHistory(ctx, projectID, alarmID, append(opts, inAccount(accountID)...)...)
		})
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
