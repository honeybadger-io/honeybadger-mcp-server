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
			// widgets and default_ts are not advertised: v3's dashboard schema
			// cannot carry them, and a required parameter the handler always
			// refuses would make the tool unusable for a conforming caller. The
			// handler still rejects them if an older caller sends them anyway.
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
			// widgets and default_ts are not advertised: v3's dashboard schema
			// cannot carry them, and a required parameter the handler always
			// refuses would make the tool unusable for a conforming caller. The
			// handler still rejects them if an older caller sends them anyway.
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

// dashboardFieldsNotInV3 are dashboard fields v2 accepted that v3's write schema
// does not declare. A dashboard's widgets are the dashboard, so accepting them and
// dropping them would create an empty one.
var dashboardFieldsNotInV3 = []string{"widgets", "default_ts"}

// handleCreateDashboard creates a dashboard by name.
//
// v3's schema declares only name. That still produces a real, if empty, dashboard
// — unlike an alarm with no query — so this proceeds and refuses only a request
// that also carries widgets.
func handleCreateDashboard(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	// v2 called this title; v3's field is name.
	name := req.GetString("title", "")
	if name == "" {
		name = req.GetString("name", "")
	}
	if name == "" {
		return mcp.NewToolResultError("title is required"), nil
	}
	if msg := rejectUnsupported(req, dashboardFieldsNotInV3, "creating a dashboard",
		"v3's dashboard schema accepts only a name; add widgets in the Honeybadger UI"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	dashboard, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.Dashboard, error) {
			return client.Dashboards.Create(ctx, projectID, name, inAccount(accountID)...)
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

func handleUpdateDashboard(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	dashboardID := req.GetString("dashboard_id", "")
	if dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required"), nil
	}
	if msg := rejectUnsupported(req, dashboardFieldsNotInV3, "updating a dashboard",
		"v3's dashboard schema accepts only a name; edit widgets in the Honeybadger UI"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	name := req.GetString("title", "")
	if name == "" {
		name = req.GetString("name", "")
	}
	if name == "" {
		return mcp.NewToolResultError("title is required"), nil
	}

	dashboard, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.Dashboard, error) {
			return client.Dashboards.Update(ctx, projectID, dashboardID, name, inAccount(accountID)...)
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
