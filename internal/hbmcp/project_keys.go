package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

func RegisterProjectKeyTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	r.AddTool(
		mcp.NewTool("list_project_keys",
			mcp.WithTitleAnnotation("List Project Keys"),
			mcp.WithDescription("List ingestion keys for a Honeybadger project. Keys are the tokens notifiers use to send data — not API credentials."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to list keys for"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListProjectKeys(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("create_project_key",
			mcp.WithTitleAnnotation("Create Project Key"),
			mcp.WithDescription("Create a new ingestion key for a project. The key value is returned in the response."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the key in"),
			),
			mcp.WithString("label",
				mcp.Description("Optional human-readable name for the key"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateProjectKey(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("update_project_key",
			mcp.WithTitleAnnotation("Update Project Key"),
			mcp.WithDescription("Update a project key's label."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the key belongs to"),
			),
			mcp.WithString("key_id",
				mcp.Required(),
				mcp.Description("The ID of the key to update"),
			),
			mcp.WithString("label",
				mcp.Required(),
				mcp.Description("New label for the key"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateProjectKey(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("delete_project_key",
			mcp.WithTitleAnnotation("Delete Project Key"),
			mcp.WithDescription("Delete a project ingestion key. Notifiers using this key will no longer be able to send data."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the key belongs to"),
			),
			mcp.WithString("key_id",
				mcp.Required(),
				mcp.Description("The ID of the key to delete"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteProjectKey(ctx, v3ClientFor(ctx), req)
		},
	)
}

func handleListProjectKeys(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	keys, err := client.ProjectKeys.ListAll(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list project keys: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(keys)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleCreateProjectKey(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	var params apiv3.ProjectKeyParams
	if label := req.GetString("label", ""); label != "" {
		params.Label = &label
	}

	key, err := client.ProjectKeys.Create(ctx, projectID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create project key: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(key)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateProjectKey(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	keyID := req.GetString("key_id", "")
	if keyID == "" {
		return mcp.NewToolResultError("key_id is required"), nil
	}
	label := req.GetString("label", "")
	if label == "" {
		return mcp.NewToolResultError("label is required"), nil
	}

	params := apiv3.ProjectKeyParams{Label: &label}
	key, err := client.ProjectKeys.Update(ctx, projectID, keyID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update project key: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(key)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteProjectKey(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	keyID := req.GetString("key_id", "")
	if keyID == "" {
		return mcp.NewToolResultError("key_id is required"), nil
	}

	if err := client.ProjectKeys.Delete(ctx, projectID, keyID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete project key: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Project key %s deleted successfully", keyID)), nil
}
