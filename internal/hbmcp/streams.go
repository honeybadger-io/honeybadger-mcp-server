package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterStreamTools registers all stream-related MCP tools
func RegisterStreamTools(r *toolRegistrar, clientFor ClientFactory) {
	// list_streams tool
	r.AddTool(
		mcp.NewTool("list_streams",
			mcp.WithTitleAnnotation("List Streams"),
			mcp.WithDescription("List Insights data streams for a Honeybadger project. Streams partition Insights event data (e.g. default vs internal)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to list streams for"),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListStreams(ctx, clientFor(ctx), req)
		},
	)
}

func handleListStreams(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	streams, err := client.Streams.List(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list streams: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(streams)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
