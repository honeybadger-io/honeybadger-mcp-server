package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/honeybadger-io/api-go/apiv2"
	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterProjectTools registers all project-related MCP tools.
func RegisterProjectTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
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
				mcp.Required(),
				mcp.Description("The project's name. Required even when changing something else: the v3 API's update takes the same body as create, so the current name must be sent."),
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
			mcp.WithDescription("Get occurrence counts over time, for one project or across every project the credential can reach."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Description("Optional project ID. Omit to report across every project the credential can reach."),
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
			return handleGetProjectOccurrenceCounts(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_project_integrations tool
	r.AddTool(
		mcp.NewTool("get_project_integrations",
			mcp.WithTitleAnnotation("Get Project Integrations"),
			mcp.WithDescription("List notification integrations for a Honeybadger project. To interpret integration types and config fields, fetch reference topic: integrations (via get_reference)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to get integrations for"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetProjectIntegrations(ctx, v3ClientFor(ctx), req)
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
	Links   apiv2.PaginationLinks `json:"links"`
}

func handleListProjects(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projects, err := client.Projects.ListAll(ctx)
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

	project, err := client.Projects.Get(ctx, id)
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

// projectParamsFrom reads the writable project fields out of a request.
//
// Only fields the caller actually sent are set, so an update leaves the rest
// alone. Booleans come from the raw arguments because the typed getter cannot
// distinguish false from absent, and false is a real value here — it is how a
// caller turns a setting off.
func projectParamsFrom(req mcp.CallToolRequest, name string) apiv3.ProjectParams {
	args := req.GetArguments()
	params := apiv3.ProjectParams{Name: name}

	// Presence rather than emptiness: an empty string is how a caller clears a
	// setting, so dropping it would make these fields impossible to unset.
	for field, target := range map[string]**string{
		"user_url":          &params.UserUrl,
		"source_url":        &params.SourceUrl,
		"user_search_field": &params.UserSearchField,
		"language":          &params.Language,
	} {
		if v, ok := args[field].(string); ok {
			value := v
			*target = &value
		}
	}
	for field, target := range map[string]**bool{
		"resolve_errors_on_deploy": &params.ResolveErrorsOnDeploy,
		"disable_public_links":     &params.DisablePublicLinks,
	} {
		if v, ok := args[field].(bool); ok {
			value := v
			*target = &value
		}
	}
	if v, ok := args["purge_days"].(float64); ok && v > 0 {
		days := int(v)
		params.PurgeDays = &days
	}
	return params
}

func handleCreateProject(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	params := projectParamsFrom(req, name)

	project, err := client.Projects.Create(ctx, params)
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
	// The API's update body is the same schema as create, with name required, so a
	// caller changing only another field still has to supply the current name.
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError(
			"name is required: the v3 API's project update takes the same body as create, " +
				"so the current name must be sent even when changing something else"), nil
	}
	params := projectParamsFrom(req, name)

	result, err := client.Projects.Update(ctx, id, params)
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

	err := client.Projects.Delete(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete project: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(map[string]any{"deleted": true, "id": id})
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetProjectOccurrenceCounts(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	o := apiv3.OccurrenceOptions{
		Period:      apiv3.OccurrencePeriod(req.GetString("period", "")),
		Environment: req.GetString("environment", ""),
	}
	projectID := req.GetString("project_id", "")

	// Omitting the project reports across the whole account, which is what v2's
	// all-projects variant did — though it is account-scoped rather than global,
	// and returns a series per project rather than one object.
	o.AccountID = req.GetString("account_id", "")
	var (
		counts any
		err    error
	)
	if projectID == "" {
		counts, err = client.Projects.AccountOccurrences(ctx, o)
	} else {
		counts, err = client.Projects.Occurrences(ctx, projectID, o)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get occurrence counts: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(counts)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetProjectIntegrations(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	integrations, err := client.Integrations.ListAll(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get project integrations: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(integrations)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

