package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterDashboardTools registers all dashboard-related MCP tools
// RegisterDashboardTools registers the dashboard tools, all on v3.
func RegisterDashboardTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	// list_dashboards tool
	r.AddTool(
		mcp.NewTool("list_dashboards",
			mcp.WithTitleAnnotation("List Dashboards"),
			mcp.WithDescription("List all Insights dashboards for a Honeybadger project"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to list dashboards for"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListDashboards(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_dashboard tool
	r.AddTool(
		mcp.NewTool("get_dashboard",
			mcp.WithTitleAnnotation("Get Dashboard"),
			mcp.WithDescription("Get a single Insights dashboard by ID"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the dashboard belongs to"),
			),
			mcp.WithString("dashboard_id",
				mcp.Required(),
				mcp.Description("The ID of the dashboard to retrieve"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetDashboard(ctx, v3ClientFor(ctx), req)
		},
	)

	// create_dashboard tool
	r.AddTool(
		mcp.NewTool("create_dashboard",
			mcp.WithTitleAnnotation("Create Dashboard"),
			mcp.WithDescription("Create a new Insights dashboard for a Honeybadger project. IMPORTANT: Requires reference topics: dashboards, charts, queries, badgerql — fetch via get_reference first (skip topics still visible in your context). Verify each widget's query returns the expected results via query_insights before creating the dashboard."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the dashboard in"),
			),
			mcp.WithString("title",
				mcp.Required(),
				mcp.Description("The title of the dashboard"),
			),
			mcp.WithString("widgets",
				mcp.Description("JSON array of widget objects. The dashboards reference topic has the full schema and examples. Each widget needs a type (insights_vis, alarms, errors, deployments, checkins, uptime) and optionally grid, presentation and config."),
			),
			mcp.WithString("default_ts",
				mcp.Description("Default time range for the dashboard. ISO 8601 duration (e.g. P1D, PT3H) or a keyword (today, yesterday, week, month)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateDashboard(ctx, v3ClientFor(ctx), req)
		},
	)

	// update_dashboard tool
	r.AddTool(
		mcp.NewTool("update_dashboard",
			mcp.WithTitleAnnotation("Update Dashboard"),
			mcp.WithDescription("Update an existing Insights dashboard. Only the parameters given are changed, so a rename needs only title. IMPORTANT: Requires reference topics: dashboards, charts, queries, badgerql — fetch via get_reference first (skip topics still visible in your context)."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the dashboard belongs to"),
			),
			mcp.WithString("dashboard_id",
				mcp.Required(),
				mcp.Description("The ID of the dashboard to update"),
			),
			mcp.WithString("title",
				mcp.Description("A new title for the dashboard"),
			),
			mcp.WithString("widgets",
				mcp.Description("JSON array of widget objects that replaces the dashboard's widgets: a widget left out is removed, and a widget keeps its identity only if it's sent with the id get_dashboard returned. Omit to leave the widgets as they are. The dashboards reference topic has the schema."),
			),
			mcp.WithString("default_ts",
				mcp.Description("Default time range for the dashboard. ISO 8601 duration (e.g. P1D, PT3H) or a keyword (today, yesterday, week, month). Pass an empty string to clear it."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateDashboard(ctx, v3ClientFor(ctx), req)
		},
	)

	// delete_dashboard tool
	r.AddTool(
		mcp.NewTool("delete_dashboard",
			mcp.WithTitleAnnotation("Delete Dashboard"),
			mcp.WithDescription("Delete an Insights dashboard."+confirmNote),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the dashboard belongs to"),
			),
			mcp.WithString("dashboard_id",
				mcp.Required(),
				mcp.Description("The ID of the dashboard to delete"),
			),
			withConfirmParam(),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteDashboard(ctx, v3ClientFor(ctx), req)
		},
	)
}

func handleListDashboards(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	response, err := client.Dashboards.ListAll(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list dashboards: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetDashboard(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
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

// dashboardWidgets reads the widgets argument, a JSON array as the tools have
// always taken it. Unknown keys on a widget are refused rather than dropped: the
// API would refuse them too, and dropping one would change what was asked for.
func dashboardWidgets(req mcp.CallToolRequest) (*[]apiv3.DashboardWidget, string) {
	raw, problem := jsonTextArg(req, "widgets")
	if problem != "" || raw == "" {
		return nil, problem
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	var widgets []apiv3.DashboardWidget
	if err := dec.Decode(&widgets); err != nil {
		return nil, fmt.Sprintf("widgets must be a JSON array of widget objects: %v", err)
	}
	return &widgets, ""
}

// optionalString points at a string argument when it was given, even empty.
func optionalString(req mcp.CallToolRequest, name string) *string {
	if v, ok := req.GetArguments()[name].(string); ok {
		return &v
	}
	return nil
}

func handleCreateDashboard(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if msg := refuseNonStrings(req, "title", "name", "default_ts"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	// v2 called this title and so does v3 now, but accept name too since the tool
	// has advertised both.
	title := req.GetString("title", "")
	if title == "" {
		title = req.GetString("name", "")
	}
	if title == "" {
		return mcp.NewToolResultError("title is required"), nil
	}
	widgets, msg := dashboardWidgets(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	params := apiv3.DashboardCreateParams{Title: title, Widgets: widgets, DefaultTs: setIfGiven(req, "default_ts")}

	dashboard, err := client.Dashboards.Create(ctx, projectID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create dashboard: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(dashboard)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleUpdateDashboard changes whichever of a dashboard's title, time range and
// widgets the caller supplies; the rest keep their values.
func handleUpdateDashboard(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if msg := refuseNonStrings(req, "title", "default_ts"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	dashboardID := req.GetString("dashboard_id", "")
	if dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required"), nil
	}
	widgets, msg := dashboardWidgets(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	params := apiv3.DashboardUpdateParams{
		Title:     optionalString(req, "title"),
		DefaultTs: setOrClear(req, "default_ts"),
		Widgets:   widgets,
	}
	if changesNothing(params) {
		return mcp.NewToolResultError("provide at least one of title, default_ts or widgets"), nil
	}

	dashboard, err := client.Dashboards.Update(ctx, projectID, dashboardID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update dashboard: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(dashboard)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteDashboard(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	dashboardID := req.GetString("dashboard_id", "")
	if dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required"), nil
	}

	if !deletionConfirmed(ctx, req, "delete_dashboard", projectID, dashboardID) {
		dashboard, err := client.Dashboards.Get(ctx, projectID, dashboardID)
		var summary string
		switch {
		case err == nil:
			summary = fmt.Sprintf("delete dashboard %q (id %s) from project %s", dashboard.Title, dashboardID, projectID)
		case unreadable(err):
			summary = fmt.Sprintf("delete dashboard %s from project %s", dashboardID, projectID) + unreadableNote
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Failed to look up dashboard: %v", err)), nil
		}
		return deletionPreview(ctx, req, "delete_dashboard", summary, projectID, dashboardID), nil
	}

	err := client.Dashboards.Delete(ctx, projectID, dashboardID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete dashboard: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(map[string]any{"deleted": true, "dashboard_id": dashboardID})
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
