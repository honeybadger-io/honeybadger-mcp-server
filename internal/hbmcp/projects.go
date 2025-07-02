package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbapi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterProjectTools registers all project-related MCP tools
func RegisterProjectTools(s *server.MCPServer, client *hbapi.Client) {
	// list_projects tool
	s.AddTool(
		mcp.NewTool("list_projects",
			mcp.WithDescription("List all Honeybadger projects"),
			mcp.WithString("account_id",
				mcp.Description("Optional account ID to filter projects by specific account"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListProjects(ctx, client, req)
		},
	)

	// get_project tool
	s.AddTool(
		mcp.NewTool("get_project",
			mcp.WithDescription("Get a single Honeybadger project by ID"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the project to retrieve"),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetProject(ctx, client, req)
		},
	)

	// create_project tool
	s.AddTool(
		mcp.NewTool("create_project",
			mcp.WithDescription("Create a new Honeybadger project"),
			mcp.WithString("account_id",
				mcp.Required(),
				mcp.Description("The account ID to associate the project with"),
				mcp.MinLength(1),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the new project"),
				mcp.MinLength(1),
				mcp.MaxLength(255),
			),
			mcp.WithBoolean("resolve_errors_on_deploy",
				mcp.Description("Whether all unresolved faults should be marked as resolved when a deploy is recorded"),
			),
			mcp.WithBoolean("disable_public_links",
				mcp.Description("Whether to allow fault details to be publicly shareable via a button on the fault detail page"),
			),
			mcp.WithString("user_url",
				mcp.Description("A URL format like 'http://example.com/admin/users/[user_id]' that will be displayed on the fault detail page"),
			),
			mcp.WithString("source_url",
				mcp.Description("A URL format like 'https://gitlab.com/username/reponame/blob/[sha]/[file]#L[line]' that is used to link lines in the backtrace to your git browser"),
			),
			mcp.WithNumber("purge_days",
				mcp.Description("The number of days to retain data (up to the max number of days available to your subscription plan)"),
				mcp.Min(1),
			),
			mcp.WithString("user_search_field",
				mcp.Description("A field such as 'context.user_email' that you provide in your error context"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateProject(ctx, client, req)
		},
	)

	// update_project tool
	s.AddTool(
		mcp.NewTool("update_project",
			mcp.WithDescription("Update an existing Honeybadger project"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the project to update"),
				mcp.Min(1),
			),
			mcp.WithString("name",
				mcp.Description("The name of the project"),
				mcp.MinLength(1),
				mcp.MaxLength(255),
			),
			mcp.WithBoolean("resolve_errors_on_deploy",
				mcp.Description("Whether all unresolved faults should be marked as resolved when a deploy is recorded"),
			),
			mcp.WithBoolean("disable_public_links",
				mcp.Description("Whether to allow fault details to be publicly shareable via a button on the fault detail page"),
			),
			mcp.WithString("user_url",
				mcp.Description("A URL format like 'http://example.com/admin/users/[user_id]' that will be displayed on the fault detail page"),
			),
			mcp.WithString("source_url",
				mcp.Description("A URL format like 'https://gitlab.com/username/reponame/blob/[sha]/[file]#L[line]' that is used to link lines in the backtrace to your git browser"),
			),
			mcp.WithNumber("purge_days",
				mcp.Description("The number of days to retain data (up to the max number of days available to your subscription plan)"),
				mcp.Min(1),
			),
			mcp.WithString("user_search_field",
				mcp.Description("A field such as 'context.user_email' that you provide in your error context"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateProject(ctx, client, req)
		},
	)

	// delete_project tool
	s.AddTool(
		mcp.NewTool("delete_project",
			mcp.WithDescription("Delete a Honeybadger project"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the project to delete"),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteProject(ctx, client, req)
		},
	)

	// get_project_occurrence_counts tool
	s.AddTool(
		mcp.NewTool("get_project_occurrence_counts",
			mcp.WithDescription("Get occurrence counts for all projects or a specific project"),
			mcp.WithNumber("project_id",
				mcp.Description("Optional project ID to get occurrence counts for a specific project"),
				mcp.Min(1),
			),
			mcp.WithString("period",
				mcp.Description("Time period for grouping data: 'hour', 'day', 'week', or 'month'. Defaults to 'hour'"),
				mcp.Enum("hour", "day", "week", "month"),
			),
			mcp.WithString("environment",
				mcp.Description("Optional environment name to filter results"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetProjectOccurrenceCounts(ctx, client, req)
		},
	)

	// get_project_integrations tool
	s.AddTool(
		mcp.NewTool("get_project_integrations",
			mcp.WithDescription("Get a list of integrations (channels) for a Honeybadger project"),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to get integrations for"),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetProjectIntegrations(ctx, client, req)
		},
	)

	// get_project_report tool
	s.AddTool(
		mcp.NewTool("get_project_report",
			mcp.WithDescription("Get report data for a Honeybadger project"),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to get report data for"),
				mcp.Min(1),
			),
			mcp.WithString("report",
				mcp.Required(),
				mcp.Description("The type of report to get: 'notices_by_class', 'notices_by_location', 'notices_by_user', or 'notices_per_day'"),
				mcp.Enum("notices_by_class", "notices_by_location", "notices_by_user", "notices_per_day"),
			),
			mcp.WithString("start",
				mcp.Description("Start date/time in ISO 8601 format for the beginning of the reporting period"),
			),
			mcp.WithString("stop",
				mcp.Description("Stop date/time in ISO 8601 format for the end of the reporting period"),
			),
			mcp.WithString("environment",
				mcp.Description("Optional environment name to filter results"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetProjectReport(ctx, client, req)
		},
	)
}

func handleListProjects(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract account_id parameter (optional)
	var response *hbapi.ProjectsResponse
	var err error

	accountID := req.GetString("account_id", "")
	if accountID != "" {
		response, err = client.Projects.ListByAccountID(ctx, accountID)
	} else {
		response, err = client.Projects.ListAll(ctx)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list projects: %v", err)), nil
	}

	// Include API tokens in response to allow LLM configuration

	// Return JSON response
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetProject(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetInt("id", 0)
	if id == 0 {
		return mcp.NewToolResultError("id is required"), nil
	}

	project, err := client.Projects.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get project: %v", err)), nil
	}

	// Include API tokens in response to allow LLM configuration

	// Return JSON response
	jsonBytes, err := json.Marshal(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleCreateProject(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := req.GetString("account_id", "")
	if accountID == "" {
		return mcp.NewToolResultError("account_id is required"), nil
	}

	// Build project request using typed getters
	projectReq := hbapi.ProjectRequest{
		Name: req.GetString("name", ""),
	}

	if projectReq.Name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	// Handle optional parameters
	if resolveErrors := req.GetString("resolve_errors_on_deploy", ""); resolveErrors != "" {
		val := req.GetBool("resolve_errors_on_deploy", false)
		projectReq.ResolveErrorsOnDeploy = &val
	}

	if disableLinks := req.GetString("disable_public_links", ""); disableLinks != "" {
		val := req.GetBool("disable_public_links", false)
		projectReq.DisablePublicLinks = &val
	}

	projectReq.UserURL = req.GetString("user_url", "")
	projectReq.SourceURL = req.GetString("source_url", "")

	if purgeDays := req.GetInt("purge_days", 0); purgeDays > 0 {
		projectReq.PurgeDays = &purgeDays
	}

	projectReq.UserSearchField = req.GetString("user_search_field", "")

	project, err := client.Projects.Create(ctx, accountID, projectReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create project: %v", err)), nil
	}

	// Include API tokens in response to allow LLM configuration

	// Return JSON response
	jsonBytes, err := json.Marshal(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateProject(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetInt("id", 0)
	if id == 0 {
		return mcp.NewToolResultError("id is required"), nil
	}

	// Build project request using typed getters - name not required for updates
	projectReq := hbapi.ProjectRequest{
		Name: req.GetString("name", ""),
	}

	// Handle optional parameters
	if resolveErrors := req.GetString("resolve_errors_on_deploy", ""); resolveErrors != "" {
		val := req.GetBool("resolve_errors_on_deploy", false)
		projectReq.ResolveErrorsOnDeploy = &val
	}

	if disableLinks := req.GetString("disable_public_links", ""); disableLinks != "" {
		val := req.GetBool("disable_public_links", false)
		projectReq.DisablePublicLinks = &val
	}

	projectReq.UserURL = req.GetString("user_url", "")
	projectReq.SourceURL = req.GetString("source_url", "")

	if purgeDays := req.GetInt("purge_days", 0); purgeDays > 0 {
		projectReq.PurgeDays = &purgeDays
	}

	projectReq.UserSearchField = req.GetString("user_search_field", "")

	result, err := client.Projects.Update(ctx, id, projectReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update project: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteProject(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetInt("id", 0)
	if id == 0 {
		return mcp.NewToolResultError("id is required"), nil
	}

	result, err := client.Projects.Delete(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete project: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetProjectOccurrenceCounts(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Build options struct using typed getters
	options := hbapi.ProjectGetOccurrenceCountsOptions{
		Period:      req.GetString("period", ""),
		Environment: req.GetString("environment", ""),
	}

	// Check if project_id is provided
	var result interface{}
	var err error

	projectID := req.GetInt("project_id", 0)
	if projectID > 0 {
		// Get occurrence counts for specific project
		result, err = client.Projects.GetOccurrenceCounts(ctx, projectID, options)
	} else {
		// Get occurrence counts for all projects
		result, err = client.Projects.GetAllOccurrenceCounts(ctx, options)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get occurrence counts: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetProjectIntegrations(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	integrations, err := client.Projects.GetIntegrations(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get project integrations: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(integrations)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetProjectReport(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	reportStr := req.GetString("report", "")
	if reportStr == "" {
		return mcp.NewToolResultError("report is required"), nil
	}

	// Convert report type - MCP enum constraint should handle validation
	var reportType hbapi.ProjectReportType
	switch reportStr {
	case "notices_by_class":
		reportType = hbapi.ProjectNoticesByClass
	case "notices_by_location":
		reportType = hbapi.ProjectNoticesByLocation
	case "notices_by_user":
		reportType = hbapi.ProjectNoticesByUser
	case "notices_per_day":
		reportType = hbapi.ProjectNoticesPerDay
	default:
		reportType = hbapi.ProjectReportType(reportStr) // Let the API handle unknown types
	}

	// Build options struct using typed getters
	options := hbapi.ProjectGetReportOptions{
		Start:       parseTimestamp(req.GetString("start", "")),
		Stop:        parseTimestamp(req.GetString("stop", "")),
		Environment: req.GetString("environment", ""),
	}

	report, err := client.Projects.GetReport(ctx, projectID, reportType, options)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get project report: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(report)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}


