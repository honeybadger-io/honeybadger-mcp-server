package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterDashboardTools registers all dashboard-related MCP tools
func RegisterDashboardTools(s *server.MCPServer, client *hbapi.Client) {
	// list_dashboards tool
	s.AddTool(
		mcp.NewTool("list_dashboards",
			mcp.WithDescription("List all Insights dashboards for a Honeybadger project"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to list dashboards for"),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListDashboards(ctx, client, req)
		},
	)

	// get_dashboard tool
	s.AddTool(
		mcp.NewTool("get_dashboard",
			mcp.WithDescription("Get a single Insights dashboard by ID"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the dashboard belongs to"),
				mcp.Min(1),
			),
			mcp.WithString("dashboard_id",
				mcp.Required(),
				mcp.Description("The ID of the dashboard to retrieve"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetDashboard(ctx, client, req)
		},
	)

	// create_dashboard tool
	s.AddTool(
		mcp.NewTool("create_dashboard",
			mcp.WithDescription("Create a new Insights dashboard for a Honeybadger project"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the dashboard in"),
				mcp.Min(1),
			),
			mcp.WithString("title",
				mcp.Required(),
				mcp.Description("The title of the dashboard"),
			),
			mcp.WithString("widgets",
				mcp.Required(),
				mcp.Description("JSON array of widget objects. Call get_insights_reference for full widget schema and examples. Each widget needs: type (insights_vis, alarms, errors, deployments, checkins, uptime), and optionally: grid ({x,y,w,h}), presentation ({title, subtitle}), config (type-specific settings). For insights_vis widgets, config should include query (BadgerQL string) and vis ({view, chart_config})."),
			),
			mcp.WithString("default_ts",
				mcp.Description("Default time range for the dashboard. ISO 8601 duration (e.g., P1D, PT3H) or keyword (today, yesterday, week, month)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateDashboard(ctx, client, req)
		},
	)

	// update_dashboard tool
	s.AddTool(
		mcp.NewTool("update_dashboard",
			mcp.WithDescription("Update an existing Insights dashboard"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the dashboard belongs to"),
				mcp.Min(1),
			),
			mcp.WithString("dashboard_id",
				mcp.Required(),
				mcp.Description("The ID of the dashboard to update"),
			),
			mcp.WithString("title",
				mcp.Required(),
				mcp.Description("The title of the dashboard"),
			),
			mcp.WithString("widgets",
				mcp.Required(),
				mcp.Description("JSON array of widget objects. Call get_insights_reference for full widget schema and examples. Each widget needs: type (insights_vis, alarms, errors, deployments, checkins, uptime), and optionally: grid ({x,y,w,h}), presentation ({title, subtitle}), config (type-specific settings). For insights_vis widgets, config should include query (BadgerQL string) and vis ({view, chart_config})."),
			),
			mcp.WithString("default_ts",
				mcp.Description("Default time range for the dashboard. ISO 8601 duration (e.g., P1D, PT3H) or keyword (today, yesterday, week, month)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateDashboard(ctx, client, req)
		},
	)

	// delete_dashboard tool
	s.AddTool(
		mcp.NewTool("delete_dashboard",
			mcp.WithDescription("Delete an Insights dashboard"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the dashboard belongs to"),
				mcp.Min(1),
			),
			mcp.WithString("dashboard_id",
				mcp.Required(),
				mcp.Description("The ID of the dashboard to delete"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteDashboard(ctx, client, req)
		},
	)
}

func handleListDashboards(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	response, err := client.Dashboards.List(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list dashboards: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetDashboard(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	dashboardID := req.GetString("dashboard_id", "")
	if dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required"), nil
	}

	dashboard, err := client.Dashboards.Get(ctx, projectID, dashboardID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get dashboard: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(dashboard)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleCreateDashboard(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	title := req.GetString("title", "")
	if title == "" {
		return mcp.NewToolResultError("title is required"), nil
	}

	widgetsJSON := req.GetString("widgets", "")
	if widgetsJSON == "" {
		return mcp.NewToolResultError("widgets is required"), nil
	}

	var widgets []map[string]interface{}
	if err := json.Unmarshal([]byte(widgetsJSON), &widgets); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse widgets JSON: %v", err)), nil
	}

	dashboardReq := hbapi.DashboardRequest{
		Title:     title,
		DefaultTs: req.GetString("default_ts", ""),
		Widgets:   widgets,
	}

	dashboard, err := client.Dashboards.Create(ctx, projectID, dashboardReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create dashboard: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(dashboard)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateDashboard(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	dashboardID := req.GetString("dashboard_id", "")
	if dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required"), nil
	}

	title := req.GetString("title", "")
	if title == "" {
		return mcp.NewToolResultError("title is required"), nil
	}

	widgetsJSON := req.GetString("widgets", "")
	if widgetsJSON == "" {
		return mcp.NewToolResultError("widgets is required"), nil
	}

	var widgets []map[string]interface{}
	if err := json.Unmarshal([]byte(widgetsJSON), &widgets); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse widgets JSON: %v", err)), nil
	}

	dashboardReq := hbapi.DashboardRequest{
		Title:     title,
		DefaultTs: req.GetString("default_ts", ""),
		Widgets:   widgets,
	}

	result, err := client.Dashboards.Update(ctx, projectID, dashboardID, dashboardReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update dashboard: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteDashboard(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	dashboardID := req.GetString("dashboard_id", "")
	if dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required"), nil
	}

	result, err := client.Dashboards.Delete(ctx, projectID, dashboardID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete dashboard: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
