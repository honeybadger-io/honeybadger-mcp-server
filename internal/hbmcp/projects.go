package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oapi-codegen/nullable"
)

// RegisterProjectTools registers all project-related MCP tools.
func RegisterProjectTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	// list_projects tool
	r.AddTool(
		mcp.NewTool("list_projects",
			mcp.WithTitleAnnotation("List Projects"),
			mcp.WithDescription("List all Honeybadger projects (returns summary info; use get_project for full details). A project can have several Ingestion Keys; list them with list_ingestion_keys."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("name",
				mcp.Description("Optional exact project name; returns only the project with that name"),
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
			mcp.WithDescription("Get a single Honeybadger project by ID. Its Ingestion Keys aren't included; list them with list_ingestion_keys."),
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
				mcp.Description("A new name for the project"),
				mcp.MinLength(1),
				mcp.MaxLength(255),
			),
			mcp.WithBoolean("resolve_errors_on_deploy",
				mcp.Description("Whether all unresolved faults should be marked as resolved when a deploy is recorded"),
			),
			mcp.WithBoolean("disable_public_links",
				mcp.Description("Whether to allow fault details to be publicly shareable via a button on the fault detail page"),
			),
			mcp.WithString("user_url", acceptsNull,
				mcp.Description("A URL format like 'http://example.com/admin/users/[user_id]' that will be displayed on the fault detail page. An empty string or null clears it."),
			),
			mcp.WithString("source_url", acceptsNull,
				mcp.Description("A URL format like 'https://gitlab.com/username/reponame/blob/[sha]/[file]#L[line]' that is used to link lines in the backtrace to your git browser. An empty string or null clears it."),
			),
			mcp.WithNumber("purge_days",
				mcp.Description("The number of days to retain data (up to the max number of days available to your subscription plan)"),
				mcp.Min(1),
			),
			mcp.WithString("user_search_field", acceptsNull,
				mcp.Description("A field such as 'context.user_email' that you provide in your error context. An empty string or null clears it."),
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
			mcp.WithDescription("Delete a Honeybadger project and all of its data."+confirmNote),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the project to delete"),
			),
			withConfirmParam(),
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
				mcp.Description("Window to report over: 'hour' (61 one-minute buckets), 'day' (25 hourly), 'week' (8 daily), or 'month' (31 daily). Defaults to 'hour'"),
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
	Active               bool       `json:"active"`
	CreatedAt            time.Time  `json:"created_at"`
	LastNoticeAt         *time.Time `json:"last_notice_at"`
	FaultCount           int        `json:"fault_count"`
	UnresolvedFaultCount int        `json:"unresolved_fault_count"`
}

// projectSummaryResponse wraps the summaries. ListAll has already walked every
// page, so there is no pagination to report.
type projectSummaryResponse struct {
	Results []projectSummary `json:"results"`
}

func handleListProjects(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if msg := rejectStaleSchemaFields("list_projects", req); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	var opts []apiv3.ListAllOption
	if name := req.GetString("name", ""); name != "" {
		opts = append(opts, apiv3.Named(name))
	}
	projects, err := client.Projects.ListAll(ctx, opts...)
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
			CreatedAt:            p.CreatedAt,
			FaultCount:           p.FaultCount,
			UnresolvedFaultCount: p.UnresolvedFaultCount,
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
func projectParamsFrom(req mcp.CallToolRequest) (apiv3.ProjectParams, string) {
	args := req.GetArguments()
	var params apiv3.ProjectParams

	// A name is never blank, so an empty one is treated as absent.
	if name := req.GetString("name", ""); name != "" {
		params.Name = &name
	}

	// Presence rather than emptiness: an empty string is how a caller clears a
	// setting, so it's sent as null rather than dropped.
	params.UserUrl = setOrClear(req, "user_url")
	params.SourceUrl = setOrClear(req, "source_url")
	params.UserSearchField = setOrClear(req, "user_search_field")
	params.Language = setOrClear(req, "language")
	for field, target := range map[string]*nullable.Nullable[bool]{
		"resolve_errors_on_deploy": &params.ResolveErrorsOnDeploy,
		"disable_public_links":     &params.DisablePublicLinks,
	} {
		if v, ok := args[field].(bool); ok {
			*target = nullable.NewNullableWithValue(v)
		}
	}
	days, given, problem := positiveIntArg(args, "purge_days")
	if problem != "" {
		return params, problem
	}
	if given {
		params.PurgeDays = nullable.NewNullableWithValue(days)
	}
	return params, ""
}

func handleCreateProject(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if msg := rejectStaleSchemaFields("create_project", req); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	p, problem := projectParamsFrom(req)
	if problem != "" {
		return mcp.NewToolResultError(problem), nil
	}
	params := apiv3.ProjectCreateParams{
		Name:                  name,
		UserUrl:               p.UserUrl,
		SourceUrl:             p.SourceUrl,
		UserSearchField:       p.UserSearchField,
		Language:              p.Language,
		ResolveErrorsOnDeploy: p.ResolveErrorsOnDeploy,
		DisablePublicLinks:    p.DisablePublicLinks,
		PurgeDays:             p.PurgeDays,
	}

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
	params, problem := projectParamsFrom(req)
	if problem != "" {
		return mcp.NewToolResultError(problem), nil
	}
	if changesNothing(params) {
		return mcp.NewToolResultError("provide at least one field to change"), nil
	}

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

	if !deletionConfirmed(ctx, req, "delete_project", id) {
		project, err := client.Projects.Get(ctx, id)
		var summary string
		switch {
		case err == nil:
			summary = fmt.Sprintf("delete project %q (id %s) and all of its data", project.Name, id)
		case unreadable(err):
			summary = fmt.Sprintf("delete project %s and all of its data", id) + unreadableNote
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Failed to look up project: %v", err)), nil
		}
		return deletionPreview(ctx, req, "delete_project", summary, id), nil
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

	// Omitting the project reports across every project the credential reaches,
	// returning a series per project rather than one object.
	if msg := rejectStaleSchemaFields("get_project_occurrence_counts", req); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
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
