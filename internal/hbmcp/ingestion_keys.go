package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oapi-codegen/nullable"
)

func RegisterIngestionKeyTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	r.AddTool(
		mcp.NewTool("list_ingestion_keys",
			mcp.WithTitleAnnotation("List Ingestion Keys"),
			mcp.WithDescription("List a project's Ingestion Keys: the hbp_ keys an app uses to send errors and events to Honeybadger, set as ingestion_key in the client's config (api_key in older clients). An Ingestion Key isn't a secret, since it ships in apps, and it can't call the Data API; that takes an API Token."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to list Ingestion Keys for"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListIngestionKeys(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("create_ingestion_key",
			mcp.WithTitleAnnotation("Create Ingestion Key"),
			mcp.WithDescription("Create a new Ingestion Key for a project. The key is returned in the response."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the Ingestion Key in"),
			),
			mcp.WithString("label",
				mcp.Description("Optional human-readable name for the key"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateIngestionKey(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("update_ingestion_key",
			mcp.WithTitleAnnotation("Update Ingestion Key"),
			mcp.WithDescription("Update an Ingestion Key's label."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the Ingestion Key belongs to"),
			),
			mcp.WithString("ingestion_key_id",
				mcp.Required(),
				mcp.Description("The ID of the Ingestion Key to update"),
			),
			mcp.WithString("label",
				mcp.Required(),
				mcp.Description("New label for the key"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateIngestionKey(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("delete_ingestion_key",
			mcp.WithTitleAnnotation("Delete Ingestion Key"),
			mcp.WithDescription("Delete an Ingestion Key. Apps sending with this key will no longer be able to send data."+confirmNote),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the Ingestion Key belongs to"),
			),
			mcp.WithString("ingestion_key_id",
				mcp.Required(),
				mcp.Description("The ID of the Ingestion Key to delete"),
			),
			withConfirmParam(),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteIngestionKey(ctx, v3ClientFor(ctx), req)
		},
	)
}

func handleListIngestionKeys(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	keys, err := client.IngestionKeys.ListAll(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list Ingestion Keys: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(keys)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleCreateIngestionKey(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if msg := refuseNonStrings(req, "label"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	params := apiv3.IngestionKeyParams{Label: setIfGiven(req, "label")}

	key, err := client.IngestionKeys.Create(ctx, projectID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create Ingestion Key: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(key)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateIngestionKey(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	keyID := req.GetString("ingestion_key_id", "")
	if keyID == "" {
		return mcp.NewToolResultError("ingestion_key_id is required"), nil
	}
	label := req.GetString("label", "")
	if label == "" {
		return mcp.NewToolResultError("label is required"), nil
	}

	params := apiv3.IngestionKeyParams{Label: nullable.NewNullableWithValue(label)}
	key, err := client.IngestionKeys.Update(ctx, projectID, keyID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update Ingestion Key: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(key)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteIngestionKey(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	keyID := req.GetString("ingestion_key_id", "")
	if keyID == "" {
		return mcp.NewToolResultError("ingestion_key_id is required"), nil
	}

	if !deletionConfirmed(ctx, req, "delete_ingestion_key", projectID, keyID) {
		summary, err := ingestionKeySummary(ctx, client, projectID, keyID)
		switch {
		case err == nil:
		case unreadable(err):
			summary = fmt.Sprintf("delete Ingestion Key %s from project %s; apps sending with it will be refused", keyID, projectID) + unreadableNote
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Failed to look up Ingestion Key: %v", err)), nil
		}
		return deletionPreview(ctx, req, "delete_ingestion_key", summary, projectID, keyID), nil
	}

	if err := client.IngestionKeys.Delete(ctx, projectID, keyID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete Ingestion Key: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Ingestion Key %s deleted successfully", keyID)), nil
}

// ingestionKeySummary describes a key for a deletion preview.
func ingestionKeySummary(ctx context.Context, client *apiv3.Client, projectID, keyID string) (string, error) {
	k, err := client.IngestionKeys.Get(ctx, projectID, keyID)
	if err != nil {
		return "", err
	}
	name := nullableString(k.Label)
	if name == "" {
		name = "unlabelled"
	}
	return fmt.Sprintf("delete the %s Ingestion Key (id %s) from project %s; apps sending with it will be refused", name, keyID, projectID), nil
}
