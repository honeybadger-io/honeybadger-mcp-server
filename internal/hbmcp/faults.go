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
		CreatedAfter:   req.GetString("created_after", ""),
		OccurredAfter:  req.GetString("occurred_after", ""),
		OccurredBefore: req.GetString("occurred_before", ""),
		Limit:          req.GetInt("limit", 0),
		Order:          req.GetString("order", ""),
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
		CreatedAfter:  req.GetString("created_after", ""),
		CreatedBefore: req.GetString("created_before", ""),
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