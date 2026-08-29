package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterCheckInTools registers all check-in-related MCP tools
// RegisterCheckInTools registers the check-in tools, all on v3.
func RegisterCheckInTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	// list_check_ins tool
	r.AddTool(
		mcp.NewTool("list_check_ins",
			mcp.WithTitleAnnotation("List Check-Ins"),
			mcp.WithDescription("List check-ins (cron/scheduled task monitoring) for a Honeybadger project. Returns every check-in, following pagination. To interpret check-in state and schedule fields, fetch reference topic: checkins (via get_reference)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to list check-ins for"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListCheckIns(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_check_in tool
	r.AddTool(
		mcp.NewTool("get_check_in",
			mcp.WithTitleAnnotation("Get Check-In"),
			mcp.WithDescription("Get a single check-in by ID. To interpret check-in state and schedule fields, fetch reference topic: checkins (via get_reference)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the check-in belongs to"),
			),
			mcp.WithString("check_in_id",
				mcp.Required(),
				mcp.Description("The ID of the check-in to retrieve"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetCheckIn(ctx, v3ClientFor(ctx), req)
		},
	)

	// create_check_in tool
	r.AddTool(
		mcp.NewTool("create_check_in",
			mcp.WithTitleAnnotation("Create Check-In"),
			mcp.WithDescription("Create a new check-in for a Honeybadger project. Check-ins monitor cron jobs and scheduled tasks by alerting when an expected report doesn't arrive. IMPORTANT: Requires reference topic: checkins — fetch via get_reference first (skip if still visible in your context) for schedule types, the required field per type, plan gating (cron needs the Business plan), and the timezone format."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the check-in in"),
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
				mcp.Description("Optional timezone for the cron schedule. Rails/ActiveSupport zone name (e.g. 'Eastern Time (US & Canada)', 'London'), NOT an IANA identifier like 'America/New_York'. Defaults to UTC."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateCheckIn(ctx, v3ClientFor(ctx), req)
		},
	)

	// update_check_in tool
	r.AddTool(
		mcp.NewTool("update_check_in",
			mcp.WithTitleAnnotation("Update Check-In"),
			mcp.WithDescription("Update an existing check-in. Only the provided fields are changed; fields cannot be cleared once set. The schedule type cannot be changed after creation. IMPORTANT: Requires reference topic: checkins — fetch via get_reference first (skip if still visible in your context) for schedule fields, plan gating, and the timezone format."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the check-in belongs to"),
			),
			mcp.WithString("check_in_id",
				mcp.Required(),
				mcp.Description("The ID of the check-in to update"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The check-in's name. Required even when changing something else: the v3 API's update takes the same body as create, so the current name must be sent."),
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
				mcp.Description("Timezone for the cron schedule. Rails/ActiveSupport zone name (e.g. 'Eastern Time (US & Canada)', 'London'), NOT an IANA identifier like 'America/New_York'."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateCheckIn(ctx, v3ClientFor(ctx), req)
		},
	)

	// delete_check_in tool
	r.AddTool(
		mcp.NewTool("delete_check_in",
			mcp.WithTitleAnnotation("Delete Check-In"),
			mcp.WithDescription("Delete a check-in. This also deletes the check-in's reporting history."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the check-in belongs to"),
			),
			mcp.WithString("check_in_id",
				mcp.Required(),
				mcp.Description("The ID of the check-in to delete"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteCheckIn(ctx, v3ClientFor(ctx), req)
		},
	)
}

func handleListCheckIns(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	checkIns, err := client.CheckIns.ListAll(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list check-ins: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(checkIns)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetCheckIn(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
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

// checkInParamsFrom reads a check-in write out of a request.
//
// The cron fields exist in v3 now, so a cron check-in is expressible.
func checkInParamsFrom(req mcp.CallToolRequest) apiv3.CheckInParams {
	return apiv3.CheckInParams{
		Name:         req.GetString("name", ""),
		ScheduleType: req.GetString("schedule_type", ""),
		ReportPeriod: req.GetString("report_period", ""),
		GracePeriod:  req.GetString("grace_period", ""),
		CronSchedule: req.GetString("cron_schedule", ""),
		CronTimezone: req.GetString("cron_timezone", ""),
		Slug:         req.GetString("slug", ""),
	}
}

func handleCreateCheckIn(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	params := checkInParamsFrom(req)
	if params.Name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	if params.ScheduleType == "cron" && params.CronSchedule == "" {
		return mcp.NewToolResultError("cron_schedule is required when schedule_type is cron"), nil
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

func handleUpdateCheckIn(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	checkInID := req.GetString("check_in_id", "")
	if checkInID == "" {
		return mcp.NewToolResultError("check_in_id is required"), nil
	}

	// The update body is the same schema as create, with name required, so the
	// current name must be sent even when changing something else.
	params := checkInParamsFrom(req)
	if params.Name == "" {
		return mcp.NewToolResultError(
			"name is required: the v3 API's check-in update takes the same body as create, " +
				"so the current name must be sent even when changing something else"), nil
	}

	checkIn, err := client.CheckIns.Update(ctx, projectID, checkInID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update check-in: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(checkIn)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteCheckIn(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
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
