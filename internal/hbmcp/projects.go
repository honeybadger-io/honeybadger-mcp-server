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
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the project to retrieve"),
				mcp.Min(1),
			),
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
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the project to delete"),
				mcp.Min(1),
			),
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
		if accountID, ok := value.(string); ok {
			projects, err = client.Projects.ListByAccountID(ctx, accountID)
		} else {
			return mcp.NewToolResultError("account_id must be a string"), nil
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
	id, err := validateIntParam(args, "id")
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
	accountID, err := validateStringParam(args, "account_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req, err := argsToProjectRequest(args, true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := client.Projects.Create(ctx, accountID, req)
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
	id, err := validateIntParam(args, "id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req, err := argsToProjectRequest(args, false)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := client.Projects.Update(ctx, id, req)
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

func handleDeleteProject(ctx context.Context, client *hbapi.Client, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := validateIntParam(args, "id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
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

func validateIntParam(args map[string]interface{}, paramName string) (int, error) {
	value, exists := args[paramName]
	if !exists {
		return 0, fmt.Errorf("required parameter '%s' is missing", paramName)
	}

	// Handle both int and float64 (JSON numbers are parsed as float64)
	switch v := value.(type) {
	case int:
		if v <= 0 {
			return 0, fmt.Errorf("parameter '%s' must be positive", paramName)
		}
		return v, nil
	case float64:
		if v != float64(int(v)) {
			return 0, fmt.Errorf("parameter '%s' must be an integer", paramName)
		}
		intVal := int(v)
		if intVal <= 0 {
			return 0, fmt.Errorf("parameter '%s' must be positive", paramName)
		}
		return intVal, nil
	default:
		return 0, fmt.Errorf("parameter '%s' must be an integer", paramName)
	}
}

// argsToProjectRequest converts MCP arguments to a ProjectRequest struct
func argsToProjectRequest(args map[string]interface{}, requireName bool) (hbapi.ProjectRequest, error) {
	var req hbapi.ProjectRequest

	if name, exists := args["name"]; exists {
		if str, ok := name.(string); ok {
			req.Name = str
		} else {
			return req, fmt.Errorf("name must be a string")
		}
	} else if requireName {
		return req, fmt.Errorf("name is required")
	}

	if resolveErrors, exists := args["resolve_errors_on_deploy"]; exists {
		if b, ok := resolveErrors.(bool); ok {
			req.ResolveErrorsOnDeploy = &b
		} else {
			return req, fmt.Errorf("resolve_errors_on_deploy must be a boolean")
		}
	}

	if disableLinks, exists := args["disable_public_links"]; exists {
		if b, ok := disableLinks.(bool); ok {
			req.DisablePublicLinks = &b
		} else {
			return req, fmt.Errorf("disable_public_links must be a boolean")
		}
	}

	if userURL, exists := args["user_url"]; exists {
		if str, ok := userURL.(string); ok {
			req.UserURL = str
		} else {
			return req, fmt.Errorf("user_url must be a string")
		}
	}

	if sourceURL, exists := args["source_url"]; exists {
		if str, ok := sourceURL.(string); ok {
			req.SourceURL = str
		} else {
			return req, fmt.Errorf("source_url must be a string")
		}
	}

	if purgeDays, exists := args["purge_days"]; exists {
		switch v := purgeDays.(type) {
		case int:
			req.PurgeDays = &v
		case float64:
			if v != float64(int(v)) {
				return req, fmt.Errorf("purge_days must be an integer")
			}
			intVal := int(v)
			req.PurgeDays = &intVal
		default:
			return req, fmt.Errorf("purge_days must be an integer")
		}
	}

	if userSearchField, exists := args["user_search_field"]; exists {
		if str, ok := userSearchField.(string); ok {
			req.UserSearchField = str
		} else {
			return req, fmt.Errorf("user_search_field must be a string")
		}
	}

	return req, nil
}

// Sanitization functions to remove sensitive data like API tokens

func sanitizeProject(project *hbapi.Project) {
	// Remove token field
	project.Token = ""
}
