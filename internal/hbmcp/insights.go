package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterInsightsTools registers all insights-related MCP tools
func RegisterInsightsTools(s *server.MCPServer, client *hbapi.Client) {
	// query_insights tool
	s.AddTool(
		mcp.NewTool("query_insights",
			mcp.WithDescription("Execute a BadgerQL query against Insights data"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to query insights for"),
				mcp.Min(1),
			),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("BadgerQL query string to execute against your Insights data"),
			),
			mcp.WithString("ts",
				mcp.Description("Time range - shortcuts like 'today', 'week', or ISO 8601 duration (e.g., 'PT3H'). Defaults to PT3H."),
			),
			mcp.WithString("timezone",
				mcp.Description("IANA timezone identifier (e.g., 'America/New_York') for timestamp interpretation"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleQueryInsights(ctx, client, req)
		},
	)
}

func handleQueryInsights(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	// Build request struct
	request := hbapi.InsightsQueryRequest{
		Query:    query,
		Ts:       req.GetString("ts", ""),
		Timezone: req.GetString("timezone", ""),
	}

	response, err := client.Insights.Query(ctx, projectID, request)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to query insights: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
