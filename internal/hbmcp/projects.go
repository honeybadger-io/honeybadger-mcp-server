package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterProjectTools registers all project-related MCP tools.
//
// Most run on v3. Three still take the v2 client because v3 has no equivalent
// endpoint yet — occurrence counts, integrations, and reports. They keep numeric
// project ids for that reason, while the migrated tools take v3's opaque string
// ids; the inconsistency is visible on purpose rather than papered over with a
// conversion that could not work.
func RegisterProjectTools(r *toolRegistrar, clientFor ClientFactory, v3ClientFor V3ClientFactory) {
	// list_projects tool
	r.AddTool(
		mcp.NewTool("list_projects",
			mcp.WithTitleAnnotation("List Projects"),
			mcp.WithDescription("List all Honeybadger projects (returns summary info; use get_project for full details)"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("account_id",
				mcp.Description("Optional account ID to filter projects by specific account"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListProjects(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_project tool
	r.AddTool(
		mcp.NewTool("get_project",
			mcp.WithTitleAnnotation("Get Project"),
			mcp.WithDescription("Get a single Honeybadger project by ID"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the project to retrieve"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetProject(ctx, v3ClientFor(ctx), req)
		},
	)

	// create_project tool
	r.AddTool(
		mcp.NewTool("create_project",
			mcp.WithTitleAnnotation("Create Project"),
			mcp.WithDescription("Create a new Honeybadger project"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("account_id",
				mcp.Description("The account ID to associate the project with. If omitted, the project is created in the first account your auth token has access to."),
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
			return handleCreateProject(ctx, v3ClientFor(ctx), req)
		},
	)

	// update_project tool
	r.AddTool(
		mcp.NewTool("update_project",
			mcp.WithTitleAnnotation("Update Project"),
			mcp.WithDescription("Update an existing Honeybadger project"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the project to update"),
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
			return handleUpdateProject(ctx, v3ClientFor(ctx), req)
		},
	)

	// delete_project tool
	r.AddTool(
		mcp.NewTool("delete_project",
			mcp.WithTitleAnnotation("Delete Project"),
			mcp.WithDescription("Delete a Honeybadger project"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the project to delete"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteProject(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_project_occurrence_counts tool
	r.AddTool(
		mcp.NewTool("get_project_occurrence_counts",
			mcp.WithTitleAnnotation("Get Project Occurrence Counts"),
			mcp.WithDescription("Get occurrence counts for all projects or a specific project"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
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
			return handleGetProjectOccurrenceCounts(ctx, clientFor(ctx), req)
		},
	)

	// get_project_integrations tool
	r.AddTool(
		mcp.NewTool("get_project_integrations",
			mcp.WithTitleAnnotation("Get Project Integrations"),
			mcp.WithDescription("Get a list of integrations (channels) for a Honeybadger project"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to get integrations for"),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetProjectIntegrations(ctx, clientFor(ctx), req)
		},
	)

	// get_project_report tool
	r.AddTool(
		mcp.NewTool("get_project_report",
			mcp.WithTitleAnnotation("Get Project Report"),
			mcp.WithDescription("Get report data for a Honeybadger project"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
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
			return handleGetProjectReport(ctx, clientFor(ctx), req)
		},
	)
}

// projectSummary is a lightweight representation of a project for list results.
// It omits large nested arrays (sites, teams, users, environments) that can
// cause the response to exceed MCP token limits. Use get_project for full details.
type projectSummary struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Token                string     `json:"token"`
	Active               bool       `json:"active"`
	CreatedAt            time.Time  `json:"created_at"`
	LastNoticeAt         *time.Time `json:"last_notice_at"`
	FaultCount           int        `json:"fault_count"`
	UnresolvedFaultCount int        `json:"unresolved_fault_count"`
}

// projectSummaryResponse wraps summary results with pagination links,
// preserving the same envelope shape as the upstream API response.
type projectSummaryResponse struct {
	Results []projectSummary      `json:"results"`
	Links   hbapi.PaginationLinks `json:"links"`
}

func handleListProjects(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// account_id is optional: omitted, v3 resolves the account from the
	// credential. It is only needed for a credential covering several accounts.
	projects, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) ([]apiv3.Project, error) {
			return client.Projects.ListAll(ctx, listAllInAccount(accountID)...)
		})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list projects: %v", err)), nil
	}

	// Map to lightweight summaries to reduce token usage.
	// Full project details are available via get_project.
	summaries := make([]projectSummary, len(projects))
	for i, p := range projects {
		summaries[i] = projectSummary{
			ID:                   p.Id,
			Name:                 p.Name,
			Active:               p.Active,
			FaultCount:           derefInt(p.FaultCount),
			UnresolvedFaultCount: derefInt(p.UnresolvedFaultCount),
		}
		if p.Token != nil {
			summaries[i].Token = *p.Token
		}
		if p.CreatedAt != nil {
			summaries[i].CreatedAt = *p.CreatedAt
		}
		// A nullable field distinguishes "never" from "not reported"; the summary
		// only needs the value when there is one.
		if last, err := p.LastNoticeAt.Get(); err == nil {
			summaries[i].LastNoticeAt = &last
		}
	}

	jsonBytes, err := json.Marshal(projectSummaryResponse{Results: summaries})
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetProject(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	project, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.Project, error) {
			return client.Projects.Get(ctx, id, inAccount(accountID)...)
		})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get project: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleCreateProject(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	if msg := rejectUnsupportedProjectSettings(req); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	project, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.Project, error) {
			return client.Projects.Create(ctx, name, inAccount(accountID)...)
		})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create project: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateProject(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	if msg := rejectUnsupportedProjectSettings(req); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	result, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.Project, error) {
			return client.Projects.Update(ctx, id, name, inAccount(accountID)...)
		})
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

func handleDeleteProject(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	_, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (any, error) {
			return nil, client.Projects.Delete(ctx, id, inAccount(accountID)...)
		})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete project: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(map[string]any{"deleted": true, "id": id})
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
