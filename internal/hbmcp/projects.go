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
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, ok := req.Params.Arguments.(map[string]interface{})
			if !ok {
				args = make(map[string]interface{})
			}
			return handleListProjects(ctx, client, args)
		},
	)

	// get_project tool
	s.AddTool(
		mcp.NewTool("get_project",
			mcp.WithDescription("Get a single Honeybadger project by ID"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, ok := req.Params.Arguments.(map[string]interface{})
			if !ok {
				return mcp.NewToolResultError("Invalid arguments"), nil
			}
			return handleGetProject(ctx, client, args)
		},
	)

	// create_project tool
	s.AddTool(
		mcp.NewTool("create_project",
			mcp.WithDescription("Create a new Honeybadger project"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, ok := req.Params.Arguments.(map[string]interface{})
			if !ok {
				return mcp.NewToolResultError("Invalid arguments"), nil
			}
			return handleCreateProject(ctx, client, args)
		},
	)

	// update_project tool
	s.AddTool(
		mcp.NewTool("update_project",
			mcp.WithDescription("Update an existing Honeybadger project"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, ok := req.Params.Arguments.(map[string]interface{})
			if !ok {
				return mcp.NewToolResultError("Invalid arguments"), nil
			}
			return handleUpdateProject(ctx, client, args)
		},
	)

	// delete_project tool
	s.AddTool(
		mcp.NewTool("delete_project",
			mcp.WithDescription("Delete a Honeybadger project"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, ok := req.Params.Arguments.(map[string]interface{})
			if !ok {
				return mcp.NewToolResultError("Invalid arguments"), nil
			}
			return handleDeleteProject(ctx, client, args)
		},
	)
}

func handleListProjects(ctx context.Context, client *hbapi.Client, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Extract account_id parameter (optional)
	var projects []hbapi.Project
	var err error

	if value, exists := args["account_id"]; exists {
		if accountID, ok := value.(int); ok {
			projects, err = client.Projects.ListByAccountID(ctx, accountID)
		} else {
			return mcp.NewToolResultError("account_id must be an integer"), nil
		}
	} else {
		projects, err = client.Projects.ListAll(ctx)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list projects: %v", err)), nil
	}

	// Sanitize the response to remove API tokens
	for i := range projects {
		sanitizeProject(&projects[i])
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(projects)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetProject(ctx context.Context, client *hbapi.Client, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := validateStringParam(args, "id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := client.Projects.Get(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get project: %v", err)), nil
	}

	// Sanitize the response to remove API tokens
	sanitizeProject(project)

	// Return JSON response
	jsonBytes, err := json.Marshal(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleCreateProject(ctx context.Context, client *hbapi.Client, args map[string]interface{}) (*mcp.CallToolResult, error) {
	name, err := validateStringParam(args, "name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := client.Projects.Create(ctx, name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create project: %v", err)), nil
	}

	// Sanitize the response to remove API tokens
	sanitizeProject(project)

	// Return JSON response
	jsonBytes, err := json.Marshal(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateProject(ctx context.Context, client *hbapi.Client, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := validateStringParam(args, "id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	updates, err := validateObjectParam(args, "updates")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := client.Projects.Update(ctx, id, updates)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update project: %v", err)), nil
	}

	// Sanitize the response to remove API tokens
	sanitizeProject(project)

	// Return JSON response
	jsonBytes, err := json.Marshal(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteProject(ctx context.Context, client *hbapi.Client, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := validateStringParam(args, "id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	err = client.Projects.Delete(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete project: %v", err)), nil
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Project %s deleted successfully", id),
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Helper functions for parameter validation
func validateStringParam(args map[string]interface{}, paramName string) (string, error) {
	value, exists := args[paramName]
	if !exists {
		return "", fmt.Errorf("required parameter '%s' is missing", paramName)
	}

	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("parameter '%s' must be a string", paramName)
	}

	if str == "" {
		return "", fmt.Errorf("parameter '%s' cannot be empty", paramName)
	}

	return str, nil
}

func validateObjectParam(args map[string]interface{}, paramName string) (map[string]interface{}, error) {
	value, exists := args[paramName]
	if !exists {
		return nil, fmt.Errorf("required parameter '%s' is missing", paramName)
	}

	obj, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parameter '%s' must be an object", paramName)
	}

	if len(obj) == 0 {
		return nil, fmt.Errorf("parameter '%s' cannot be empty", paramName)
	}

	return obj, nil
}

// Sanitization functions to remove sensitive data like API tokens

func sanitizeProject(project *hbapi.Project) {
	// Remove token field
	project.Token = ""
}
