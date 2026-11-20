package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterProjectTools registers all project-related MCP tools.
func RegisterProjectTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	// list_projects tool
	r.AddTool(
		generatedTool("list_projects"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListProjects(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_project tool
	r.AddTool(
		generatedTool("get_project"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetProject(ctx, v3ClientFor(ctx), req)
		},
	)

	// create_project tool
	r.AddTool(
		generatedTool("create_project"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateProject(ctx, v3ClientFor(ctx), req)
		},
	)

	// update_project tool
	r.AddTool(
		generatedTool("update_project"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateProject(ctx, v3ClientFor(ctx), req)
		},
	)

	// delete_project tool
	r.AddTool(
		generatedTool("delete_project"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteProject(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_project_occurrence_counts tool
	r.AddTool(
		generatedTool("get_project_occurrence_counts"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetProjectOccurrenceCounts(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_project_integrations tool
	r.AddTool(
		generatedTool("get_project_integrations"),
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
	id := req.GetString("project_id", "")
	if id == "" {
		return mcp.NewToolResultError("project_id is required"), nil
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

func handleCreateProject(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	var params apiv3.ProjectCreateParams
	if problem := bodyFrom(req, &params); problem != "" {
		return mcp.NewToolResultError(problem), nil
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
	id := req.GetString("project_id", "")
	if id == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	var params apiv3.ProjectParams
	if problem := bodyFrom(req, &params, "project_id"); problem != "" {
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
	id := req.GetString("project_id", "")
	if id == "" {
		return mcp.NewToolResultError("project_id is required"), nil
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
