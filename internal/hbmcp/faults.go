package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbapi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterFaultTools registers all fault-related MCP tools
func RegisterFaultTools(s *server.MCPServer, client *hbapi.Client) {
	// list_faults tool
	s.AddTool(
		mcp.NewTool("list_faults",
			mcp.WithDescription("Get a list of faults for a project with optional filtering and ordering"),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to get faults for"),
				mcp.Min(1),
			),
			mcp.WithString("q",
				mcp.Description("Search string to filter faults"),
			),
			mcp.WithString("created_after",
				mcp.Description("Filter faults created after this timestamp"),
			),
			mcp.WithString("occurred_after",
				mcp.Description("Filter faults that occurred after this timestamp"),
			),
			mcp.WithString("occurred_before",
				mcp.Description("Filter faults that occurred before this timestamp"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of faults to return (max 25)"),
				mcp.Min(1),
				mcp.Max(25),
			),
			mcp.WithString("order",
				mcp.Description("Order results by 'recent' or 'frequent'"),
				mcp.Enum("recent", "frequent"),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination"),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFaults(ctx, client, req)
		},
	)

	// get_fault tool
	s.AddTool(
		mcp.NewTool("get_fault",
			mcp.WithDescription("Get detailed information for a specific fault in a project"),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project containing the fault"),
				mcp.Min(1),
			),
			mcp.WithNumber("fault_id",
				mcp.Required(),
				mcp.Description("The ID of the fault to retrieve"),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetFault(ctx, client, req)
		},
	)

	// list_fault_notices tool
	s.AddTool(
		mcp.NewTool("list_fault_notices",
			mcp.WithDescription("Get a list of notices (individual error events) for a specific fault"),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project containing the fault"),
				mcp.Min(1),
			),
			mcp.WithNumber("fault_id",
				mcp.Required(),
				mcp.Description("The ID of the fault to get notices for"),
				mcp.Min(1),
			),
			mcp.WithString("created_after",
				mcp.Description("Filter notices created after this timestamp"),
			),
			mcp.WithString("created_before",
				mcp.Description("Filter notices created before this timestamp"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of notices to return (max 25)"),
				mcp.Min(1),
				mcp.Max(25),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFaultNotices(ctx, client, req)
		},
	)

	// list_fault_affected_users tool
	s.AddTool(
		mcp.NewTool("list_fault_affected_users",
			mcp.WithDescription("Get a list of users who were affected by a specific fault with occurrence counts"),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project containing the fault"),
				mcp.Min(1),
			),
			mcp.WithNumber("fault_id",
				mcp.Required(),
				mcp.Description("The ID of the fault to get affected users for"),
				mcp.Min(1),
			),
			mcp.WithString("q",
				mcp.Description("Search string to filter affected users"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFaultAffectedUsers(ctx, client, req)
		},
	)

	// get_fault_counts tool
	s.AddTool(
		mcp.NewTool("get_fault_counts",
			mcp.WithDescription("Get fault count statistics for a project with optional filtering"),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to get fault counts for"),
				mcp.Min(1),
			),
			mcp.WithString("q",
				mcp.Description("Search string to filter faults"),
			),
			mcp.WithString("created_after",
				mcp.Description("Filter faults created after this timestamp"),
			),
			mcp.WithString("occurred_after",
				mcp.Description("Filter faults that occurred after this timestamp"),
			),
			mcp.WithString("occurred_before",
				mcp.Description("Filter faults that occurred before this timestamp"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetFaultCounts(ctx, client, req)
		},
	)
}

func handleListFaults(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Since project_id is required, MCP will ensure it exists
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	// Build options struct
	options := hbapi.FaultListOptions{
		Q:              req.GetString("q", ""),
		CreatedAfter:   parseTimestamp(req.GetString("created_after", "")),
		OccurredAfter:  parseTimestamp(req.GetString("occurred_after", "")),
		OccurredBefore: parseTimestamp(req.GetString("occurred_before", "")),
		Limit:          req.GetInt("limit", 0),
		Order:          req.GetString("order", ""),
		Page:           req.GetInt("page", 0),
	}

	response, err := client.Faults.List(ctx, projectID, options)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list faults: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetFault(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	faultID := req.GetInt("fault_id", 0)
	if faultID == 0 {
		return mcp.NewToolResultError("fault_id is required"), nil
	}

	fault, err := client.Faults.Get(ctx, projectID, faultID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get fault: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(fault)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleListFaultNotices(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	faultID := req.GetInt("fault_id", 0)
	if faultID == 0 {
		return mcp.NewToolResultError("fault_id is required"), nil
	}

	// Build options struct
	options := hbapi.FaultListNoticesOptions{
		CreatedAfter:  parseTimestamp(req.GetString("created_after", "")),
		CreatedBefore: parseTimestamp(req.GetString("created_before", "")),
		Limit:         req.GetInt("limit", 0),
	}

	response, err := client.Faults.ListNotices(ctx, projectID, faultID, options)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list fault notices: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
func handleListFaultAffectedUsers(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	faultID := req.GetInt("fault_id", 0)
	if faultID == 0 {
		return mcp.NewToolResultError("fault_id is required"), nil
	}

	// Build options struct
	options := hbapi.FaultListAffectedUsersOptions{
		Q: req.GetString("q", ""),
	}

	users, err := client.Faults.ListAffectedUsers(ctx, projectID, faultID, options)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list fault affected users: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(users)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetFaultCounts(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Since project_id is required, MCP will ensure it exists
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	// Build options struct (reuse same filtering options as List)
	options := hbapi.FaultListOptions{
		Q:              req.GetString("q", ""),
		CreatedAfter:   parseTimestamp(req.GetString("created_after", "")),
		OccurredAfter:  parseTimestamp(req.GetString("occurred_after", "")),
		OccurredBefore: parseTimestamp(req.GetString("occurred_before", "")),
	}

	counts, err := client.Faults.GetCounts(ctx, projectID, options)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get fault counts: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(counts)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
