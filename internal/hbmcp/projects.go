package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbapi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// APIClient interface for testing
type APIClient interface {
	ListProjects(accountID string) ([]map[string]interface{}, error)
	GetProject(id string) (map[string]interface{}, error)
	CreateProject(name string) (map[string]interface{}, error)
	UpdateProject(id string, updates map[string]interface{}) (map[string]interface{}, error)
	DeleteProject(id string) error
}

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
			return handleListProjects(client, args)
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
			return handleGetProject(client, args)
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
			return handleCreateProject(client, args)
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
			return handleUpdateProject(client, args)
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
			return handleDeleteProject(client, args)
		},
	)
}

func handleListProjects(client APIClient, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Extract account_id parameter (optional)
	accountID := ""
	if value, exists := args["account_id"]; exists {
		if str, ok := value.(string); ok {
			accountID = str
		}
	}

	projects, err := client.ListProjects(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list projects: %v", err)), nil
	}

	// Sanitize the response to remove API tokens
	for _, project := range projects {
		sanitizeProjectData(project)
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(projects)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetProject(client APIClient, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := validateStringParam(args, "id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := client.GetProject(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get project: %v", err)), nil
	}

	// Sanitize the response to remove API tokens
	sanitizeProjectData(project)

	// Return JSON response
	jsonBytes, err := json.Marshal(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleCreateProject(client APIClient, args map[string]interface{}) (*mcp.CallToolResult, error) {
	name, err := validateStringParam(args, "name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := client.CreateProject(name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create project: %v", err)), nil
	}

	// Sanitize the response to remove API tokens
	sanitizeProjectData(project)

	// Return JSON response
	jsonBytes, err := json.Marshal(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateProject(client APIClient, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := validateStringParam(args, "id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	updates, err := validateObjectParam(args, "updates")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := client.UpdateProject(id, updates)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update project: %v", err)), nil
	}

	// Sanitize the response to remove API tokens
	sanitizeProjectData(project)

	// Return JSON response
	jsonBytes, err := json.Marshal(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteProject(client APIClient, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := validateStringParam(args, "id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	err = client.DeleteProject(id)
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

// Sanitization function to remove sensitive data like API tokens

func sanitizeProjectData(project map[string]interface{}) {
	// Remove token field
	delete(project, "token")
}
