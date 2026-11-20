package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

func RegisterIngestionKeyTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	r.AddTool(
		generatedTool("list_ingestion_keys"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListIngestionKeys(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("create_ingestion_key"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateIngestionKey(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("update_ingestion_key"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateIngestionKey(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("delete_ingestion_key"),
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
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	var params apiv3.IngestionKeyParams
	if msg := bodyFrom(req, &params, "project_id"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

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
	var params apiv3.IngestionKeyParams
	if msg := bodyFrom(req, &params, "project_id", "ingestion_key_id"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	if changesNothing(params) {
		return mcp.NewToolResultError("provide label: the new label, or null to clear it"), nil
	}
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
