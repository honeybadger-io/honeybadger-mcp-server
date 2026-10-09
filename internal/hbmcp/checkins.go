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

	r.AddTool(
		mcp.NewTool("list_check_in_events",
			mcp.WithTitleAnnotation("List Check-In Events"),
			mcp.WithDescription("List a check-in's history, newest first: each time it reported, went missing or was paused. Use it to explain a check_in_missing or check_in_reporting event. To page back, pass the previous response's next_created_before as created_before."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("The ID of the project the check-in belongs to")),
			mcp.WithString("check_in_id", mcp.Required(), mcp.Description("The ID of the check-in")),
			timeSeriesLimit(),
			createdBeforeParam(),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListCheckInEvents(ctx, v3ClientFor(ctx), req)
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
				mcp.Description("The name of the check-in. Optional: an unnamed check-in shows its ID."),
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
				mcp.Description("A new name for the check-in"),
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
			mcp.WithDescription("Delete a check-in. This also deletes the check-in's reporting history."+confirmNote),
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
			withConfirmParam(),
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

// checkInParamsFrom reads a check-in write out of a request. Create and update
// take the same fields. Empty arguments are left out, so "" means "not given";
// these tools don't clear a setting.
func checkInParamsFrom(req mcp.CallToolRequest) apiv3.CheckInUpdateParams {
	params := apiv3.CheckInUpdateParams{
		Name:         setIfGiven(req, "name"),
		ReportPeriod: setIfGiven(req, "report_period"),
		GracePeriod:  setIfGiven(req, "grace_period"),
		CronSchedule: setIfGiven(req, "cron_schedule"),
		CronTimezone: setIfGiven(req, "cron_timezone"),
		Slug:         setIfGiven(req, "slug"),
	}
	if st := req.GetString("schedule_type", ""); st != "" {
		scheduleType := apiv3.CheckInScheduleType(st)
		params.ScheduleType = &scheduleType
	}
	return params
}

func handleCreateCheckIn(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	if msg := refuseNonStrings(req, "name", "report_period", "grace_period", "cron_schedule", "cron_timezone", "slug", "schedule_type"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	params := checkInParamsFrom(req)
	if params.ScheduleType != nil && *params.ScheduleType == apiv3.ScheduleCron && !params.CronSchedule.IsSpecified() {
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

	if msg := refuseNonStrings(req, "name", "report_period", "grace_period", "cron_schedule", "cron_timezone", "slug", "schedule_type"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	params := checkInParamsFrom(req)
	if changesNothing(params) {
		return mcp.NewToolResultError("provide at least one field to change"), nil
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

	if !deletionConfirmed(ctx, req, "delete_check_in", projectID, checkInID) {
		checkIn, err := client.CheckIns.Get(ctx, projectID, checkInID)
		var summary string
		switch {
		case err == nil && nullableString(checkIn.Name) == "":
			// An unnamed check-in shows its ID, so there's no name to quote.
			summary = fmt.Sprintf("delete the unnamed check-in %s from project %s, along with its reporting history", checkInID, projectID)
		case err == nil:
			summary = fmt.Sprintf("delete check-in %q (id %s) from project %s, along with its reporting history", nullableString(checkIn.Name), checkInID, projectID)
		case unreadable(err):
			summary = fmt.Sprintf("delete check-in %s from project %s, along with its reporting history", checkInID, projectID) + unreadableNote
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Failed to look up check-in: %v", err)), nil
		}
		return deletionPreview(ctx, req, "delete_check_in", summary, projectID, checkInID), nil
	}

	if err := client.CheckIns.Delete(ctx, projectID, checkInID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete check-in: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Check-in %s deleted successfully", checkInID)), nil
}

func handleListCheckInEvents(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	checkInID := req.GetString("check_in_id", "")
	if checkInID == "" {
		return mcp.NewToolResultError("check_in_id is required"), nil
	}
	opts, msg := olderThanOptions(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	events, err := client.CheckIns.ListEvents(ctx, projectID, checkInID, opts...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list check-in events: %v", err)), nil
	}
	return jsonResult(timestampPageOf(events))
}
