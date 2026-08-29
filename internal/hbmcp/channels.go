package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

func RegisterIntegrationTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	r.AddTool(
		mcp.NewTool("get_integration",
			mcp.WithTitleAnnotation("Get Integration"),
			mcp.WithDescription("Get a single notification integration by ID. To interpret integration fields, fetch reference topic: integrations (via get_reference)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the integration belongs to"),
			),
			mcp.WithString("integration_id",
				mcp.Required(),
				mcp.Description("The ID of the integration to retrieve"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetIntegration(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("create_integration",
			mcp.WithTitleAnnotation("Create Integration"),
			mcp.WithDescription("Create a notification integration for a project. IMPORTANT: Requires reference topic: integrations — fetch via get_reference first (skip if still visible in your context) for the list of API-creatable types, required config fields per type, and event names."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the integration in"),
			),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("Integration type: WebHook, PagerDutyV2, Email, etc. Must be an API-creatable type."),
			),
			mcp.WithString("config",
				mcp.Description("JSON object of integration settings. Common fields: events (array of event names), active (bool), rate, threshold, notification_limit, site_ids, check_in_ids, environments, excluded_environments, included_environments. Type-specific fields vary (e.g. url for WebHook, integration_key for PagerDutyV2)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateIntegration(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("update_integration",
			mcp.WithTitleAnnotation("Update Integration"),
			mcp.WithDescription("Update a notification integration's settings. Only provided fields are changed. IMPORTANT: Requires reference topic: integrations — fetch via get_reference first (skip if still visible in your context) for config fields and event names."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the integration belongs to"),
			),
			mcp.WithString("integration_id",
				mcp.Required(),
				mcp.Description("The ID of the integration to update"),
			),
			mcp.WithString("config",
				mcp.Required(),
				mcp.Description("JSON object of integration settings to update. Same fields as create_integration config, minus type (which cannot be changed)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateIntegration(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("delete_integration",
			mcp.WithTitleAnnotation("Delete Integration"),
			mcp.WithDescription("Delete a notification integration. Also deletes all associated tickets and configurations."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the integration belongs to"),
			),
			mcp.WithString("integration_id",
				mcp.Required(),
				mcp.Description("The ID of the integration to delete"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteIntegration(ctx, v3ClientFor(ctx), req)
		},
	)
}

func handleGetIntegration(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	integrationID := req.GetString("integration_id", "")
	if integrationID == "" {
		return mcp.NewToolResultError("integration_id is required"), nil
	}

	integration, err := client.Integrations.Get(ctx, projectID, integrationID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get integration: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(integration)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleCreateIntegration(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	integrationType := req.GetString("type", "")
	if integrationType == "" {
		return mcp.NewToolResultError("type is required"), nil
	}

	var params apiv3.IntegrationParams
	if raw := req.GetString("config", ""); raw != "" {
		if err := json.Unmarshal([]byte(raw), &params); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse config JSON: %v", err)), nil
		}
	}
	params.Type = &integrationType

	integration, err := client.Integrations.Create(ctx, projectID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create integration: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(integration)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateIntegration(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	integrationID := req.GetString("integration_id", "")
	if integrationID == "" {
		return mcp.NewToolResultError("integration_id is required"), nil
	}

	raw := req.GetString("config", "")
	if raw == "" {
		return mcp.NewToolResultError("config is required"), nil
	}

	var params apiv3.IntegrationParams
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse config JSON: %v", err)), nil
	}

	integration, err := client.Integrations.Update(ctx, projectID, integrationID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update integration: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(integration)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteIntegration(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	integrationID := req.GetString("integration_id", "")
	if integrationID == "" {
		return mcp.NewToolResultError("integration_id is required"), nil
	}

	if err := client.Integrations.Delete(ctx, projectID, integrationID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete integration: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Integration %s deleted successfully", integrationID)), nil
}
