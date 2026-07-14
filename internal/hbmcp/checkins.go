package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterCheckInTools registers all check-in-related MCP tools
func RegisterCheckInTools(r *toolRegistrar, clientFor ClientFactory) {
	// list_check_ins tool
	r.AddTool(
		mcp.NewTool("list_check_ins",
			mcp.WithTitleAnnotation("List Check-Ins"),
			mcp.WithDescription("List check-ins (cron/scheduled task monitoring) for a Honeybadger project. Returns the first 25 check-ins; pagination is not currently supported."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to list check-ins for"),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListCheckIns(ctx, clientFor(ctx), req)
		},
	)

	// get_check_in tool
	r.AddTool(
		mcp.NewTool("get_check_in",
			mcp.WithTitleAnnotation("Get Check-In"),
			mcp.WithDescription("Get a single check-in by ID"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the check-in belongs to"),
				mcp.Min(1),
			),
			mcp.WithString("check_in_id",
				mcp.Required(),
				mcp.Description("The ID of the check-in to retrieve"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetCheckIn(ctx, clientFor(ctx), req)
		},
	)

	// create_check_in tool
	r.AddTool(
		mcp.NewTool("create_check_in",
			mcp.WithTitleAnnotation("Create Check-In"),
			mcp.WithDescription("Create a new check-in for a Honeybadger project. Check-ins monitor cron jobs and scheduled tasks by alerting when an expected report doesn't arrive."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the check-in in"),
				mcp.Min(1),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the check-in"),
			),
			mcp.WithString("schedule_type",
				mcp.Required(),
				mcp.Description("The schedule type: 'simple' (report every fixed period) or 'cron' (report on a cron schedule)"),
				mcp.Enum("simple", "cron"),
			),
			mcp.WithString("slug",
				mcp.Description("Optional URL-friendly identifier used to report the check-in (e.g. 'nightly-backups')"),
			),
			mcp.WithString("report_period",
				mcp.Description("How often the check-in is expected to report, e.g. '1 day', '30 minutes'. Required for simple schedules."),
			),
			mcp.WithString("grace_period",
				mcp.Description("Optional amount of time to allow a late report before alerting, e.g. '5 minutes'"),
			),
			mcp.WithString("cron_schedule",
				mcp.Description("Cron expression defining when the check-in is expected to report, e.g. '0 5 * * *'. Required for cron schedules."),
			),
			mcp.WithString("cron_timezone",
				mcp.Description("Optional timezone for the cron schedule (defaults to UTC)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateCheckIn(ctx, clientFor(ctx), req)
		},
	)

	// update_check_in tool
	r.AddTool(
		mcp.NewTool("update_check_in",
			mcp.WithTitleAnnotation("Update Check-In"),
			mcp.WithDescription("Update an existing check-in. Only the provided fields are changed; fields cannot be cleared once set. The schedule type cannot be changed after creation."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the check-in belongs to"),
				mcp.Min(1),
			),
			mcp.WithString("check_in_id",
				mcp.Required(),
				mcp.Description("The ID of the check-in to update"),
			),
			mcp.WithString("name",
				mcp.Description("The name of the check-in"),
			),
			mcp.WithString("slug",
				mcp.Description("URL-friendly identifier used to report the check-in"),
			),
			mcp.WithString("report_period",
				mcp.Description("How often the check-in is expected to report, e.g. '1 day', '30 minutes'. Used by simple schedules."),
			),
			mcp.WithString("grace_period",
				mcp.Description("Amount of time to allow a late report before alerting, e.g. '5 minutes'"),
			),
			mcp.WithString("cron_schedule",
				mcp.Description("Cron expression defining when the check-in is expected to report. Used by cron schedules."),
			),
			mcp.WithString("cron_timezone",
				mcp.Description("Timezone for the cron schedule"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateCheckIn(ctx, clientFor(ctx), req)
		},
	)

	// delete_check_in tool
	r.AddTool(
		mcp.NewTool("delete_check_in",
			mcp.WithTitleAnnotation("Delete Check-In"),
			mcp.WithDescription("Delete a check-in. This also deletes the check-in's reporting history."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the check-in belongs to"),
				mcp.Min(1),
			),
			mcp.WithString("check_in_id",
				mcp.Required(),
				mcp.Description("The ID of the check-in to delete"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteCheckIn(ctx, clientFor(ctx), req)
		},
	)
}

func handleListCheckIns(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	checkIns, err := client.CheckIns.List(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list check-ins: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(checkIns)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetCheckIn(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	checkInID := req.GetString("check_in_id", "")
	if checkInID == "" {
		return mcp.NewToolResultError("check_in_id is required"), nil
	}

	checkIn, err := client.CheckIns.Get(ctx, projectID, checkInID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get check-in: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(checkIn)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleCreateCheckIn(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	scheduleType := req.GetString("schedule_type", "")
	if scheduleType == "" {
		return mcp.NewToolResultError("schedule_type is required"), nil
	}
	if scheduleType != "simple" && scheduleType != "cron" {
		return mcp.NewToolResultError("schedule_type must be 'simple' or 'cron'"), nil
	}

	params := hbapi.CheckInParams{
		Name:         name,
		Slug:         req.GetString("slug", ""),
		ScheduleType: scheduleType,
	}

	switch scheduleType {
	case "simple":
		reportPeriod := req.GetString("report_period", "")
		if reportPeriod == "" {
			return mcp.NewToolResultError("report_period is required for simple schedules"), nil
		}
		params.ReportPeriod = &reportPeriod
	case "cron":
		cronSchedule := req.GetString("cron_schedule", "")
		if cronSchedule == "" {
			return mcp.NewToolResultError("cron_schedule is required for cron schedules"), nil
		}
		params.CronSchedule = &cronSchedule
	}

	if gracePeriod := req.GetString("grace_period", ""); gracePeriod != "" {
		params.GracePeriod = &gracePeriod
	}
	if cronTimezone := req.GetString("cron_timezone", ""); cronTimezone != "" {
		params.CronTimezone = &cronTimezone
	}

	checkIn, err := client.CheckIns.Create(ctx, projectID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create check-in: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(checkIn)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateCheckIn(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	checkInID := req.GetString("check_in_id", "")
	if checkInID == "" {
		return mcp.NewToolResultError("check_in_id is required"), nil
	}

	// The API doesn't allow changing schedule_type after creation, so it is
	// deliberately not exposed here.
	params := hbapi.CheckInParams{
		Name: req.GetString("name", ""),
		Slug: req.GetString("slug", ""),
	}

	if reportPeriod := req.GetString("report_period", ""); reportPeriod != "" {
		params.ReportPeriod = &reportPeriod
	}
	if gracePeriod := req.GetString("grace_period", ""); gracePeriod != "" {
		params.GracePeriod = &gracePeriod
	}
	if cronSchedule := req.GetString("cron_schedule", ""); cronSchedule != "" {
		params.CronSchedule = &cronSchedule
	}
	if cronTimezone := req.GetString("cron_timezone", ""); cronTimezone != "" {
		params.CronTimezone = &cronTimezone
	}

	if err := client.CheckIns.Update(ctx, projectID, checkInID, params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update check-in: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Check-in %s successfully updated", checkInID)), nil
}

func handleDeleteCheckIn(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	checkInID := req.GetString("check_in_id", "")
	if checkInID == "" {
		return mcp.NewToolResultError("check_in_id is required"), nil
	}

	if err := client.CheckIns.Delete(ctx, projectID, checkInID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete check-in: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Check-in %s deleted successfully", checkInID)), nil
}
