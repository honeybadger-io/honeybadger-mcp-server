package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterDashboardTools registers all dashboard-related MCP tools
// RegisterDashboardTools registers the dashboard tools, all on v3.
func RegisterDashboardTools(r *toolRegistrar, clientFor ClientFactory, v3ClientFor V3ClientFactory) {
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
			mcp.WithDescription("Update an existing Insights dashboard. IMPORTANT: Requires reference topics: dashboards, charts, queries, badgerql — fetch via get_reference first (skip topics still visible in your context)."),
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
			return handleUpdateDashboard(ctx, v3ClientFor(ctx), req)
		},
	)

	// delete_dashboard tool
	r.AddTool(
		mcp.NewTool("delete_dashboard",
			mcp.WithTitleAnnotation("Delete Dashboard"),
			mcp.WithDescription("Delete an Insights dashboard"),
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

	response, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) ([]apiv3.Dashboard, error) {
			return client.Dashboards.ListAll(ctx, projectID, listAllInAccount(accountID)...)
		})
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

	dashboard, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.Dashboard, error) {
			return client.Dashboards.Get(ctx, projectID, dashboardID, inAccount(accountID)...)
		})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get dashboard: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(dashboard)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// dashboardParamsFrom reads a dashboard write out of a request.
//
// Widgets arrive as a JSON string, as they did on v2, and travel through as raw
// JSON — the generated widget type is a nested anonymous struct no caller could
// build, so passing the array untouched is what makes widgets usable at all.
func dashboardParamsFrom(req mcp.CallToolRequest) (apiv3.DashboardParams, string) {
	// v2 called this title and so does v3 now, but accept name too since the tool
	// has advertised both.
	title := req.GetString("title", "")
	if title == "" {
		title = req.GetString("name", "")
	}
	if title == "" {
		return apiv3.DashboardParams{}, "title is required"
	}

	params := apiv3.DashboardParams{Title: title, DefaultTs: req.GetString("default_ts", "")}
	if raw := req.GetString("widgets", ""); raw != "" {
		if !json.Valid([]byte(raw)) {
			return apiv3.DashboardParams{}, "widgets must be valid JSON"
		}
		params.Widgets = json.RawMessage(raw)
	}
	return params, ""
}

func handleCreateDashboard(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	params, msg := dashboardParamsFrom(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	dashboard, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.Dashboard, error) {
			return client.Dashboards.Create(ctx, projectID, params, inAccount(accountID)...)
		})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create dashboard: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(dashboard)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleUpdateDashboard replaces a dashboard's title, time range and widgets.
//
// Note this is a replacement rather than a merge: the update body is the same
// schema as create, so omitting widgets clears them. The tool therefore requires
// widgets to be sent explicitly when changing anything else, rather than silently
// emptying a dashboard.
func handleUpdateDashboard(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	dashboardID := req.GetString("dashboard_id", "")
	if dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required"), nil
	}
	params, msg := dashboardParamsFrom(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	if len(params.Widgets) == 0 {
		return mcp.NewToolResultError(
			"widgets is required on update: the v3 API replaces the dashboard rather than " +
				"merging, so omitting widgets would clear the ones it has. Read the dashboard " +
				"first with get_dashboard and send its widgets back, changed or unchanged."), nil
	}

	dashboard, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.Dashboard, error) {
			return client.Dashboards.Update(ctx, projectID, dashboardID, params, inAccount(accountID)...)
		})
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

	_, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (any, error) {
			return nil, client.Dashboards.Delete(ctx, projectID, dashboardID, inAccount(accountID)...)
		})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete dashboard: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(map[string]any{"deleted": true, "dashboard_id": dashboardID})
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
