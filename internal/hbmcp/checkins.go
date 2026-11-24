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
		generatedTool("list_check_ins"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListCheckIns(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_check_in tool
	r.AddTool(
		generatedTool("get_check_in"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetCheckIn(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("list_check_in_events"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListCheckInEvents(ctx, v3ClientFor(ctx), req)
		},
	)

	// create_check_in tool
	r.AddTool(
		generatedTool("create_check_in"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateCheckIn(ctx, v3ClientFor(ctx), req)
		},
	)

	// update_check_in tool
	r.AddTool(
		generatedTool("update_check_in"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateCheckIn(ctx, v3ClientFor(ctx), req)
		},
	)

	// delete_check_in tool
	r.AddTool(
		generatedTool("delete_check_in"),
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

func handleCreateCheckIn(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	var params apiv3.CheckInCreateParams
	if msg := bodyFrom(req, &params, "project_id"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
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

	var params apiv3.CheckInUpdateParams
	if msg := bodyFrom(req, &params, "project_id", "check_in_id"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
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
