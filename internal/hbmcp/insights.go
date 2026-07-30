package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterInsightsTools registers all insights-related MCP tools
func RegisterInsightsTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	// query_insights tool
	r.AddTool(
		mcp.NewTool("query_insights",
			mcp.WithTitleAnnotation("Query Insights"),
			mcp.WithDescription("Execute a BadgerQL query against Insights data. Requires reference topics: queries, badgerql (fetch via get_reference; skip topics still visible in your context). To visualize or share results, also fetch the charts topic."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to query insights for"),
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
			mcp.WithArray("stream_ids",
				mcp.WithStringItems(),
				mcp.Description("Optional list of stream IDs to restrict the query to specific Insights streams. Use list_streams to discover a project's stream IDs; pass the 'id' field (not the slug). Omit to query all streams. Passing only unrecognized IDs yields an error, not an empty result."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleQueryInsights(ctx, v3ClientFor(ctx), req)
		},
	)

}

func handleQueryInsights(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	query_ := apiv3.InsightsQuery{
		Query:     query,
		Ts:        req.GetString("ts", ""),
		Timezone:  req.GetString("timezone", ""),
		StreamIDs: req.GetStringSlice("stream_ids", nil),
	}

	// v3 rejects a bad query with a 422 rather than v2's inline error on a 200,
	// so a query error arrives here rather than in the response body.
	response, err := client.Insights.Query(ctx, projectID, query_)
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
