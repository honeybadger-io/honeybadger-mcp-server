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
		generatedTool("query_insights"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleQueryInsights(ctx, v3ClientFor(ctx), req)
		},
	)

}

func handleQueryInsights(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if msg := refuseNonStrings(req, "ts", "timezone"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	// The alarm tools take stream_ids as a JSON string, and an agent checking an
	// alarm's query here will send it the same way; dropping it would run the
	// query over every stream instead.
	streamIDs, _, _, problem := streamIDsArg(req)
	if problem != "" {
		return mcp.NewToolResultError(problem), nil
	}
	query_ := apiv3.InsightsQuery{
		Query:     query,
		Ts:        req.GetString("ts", ""),
		Timezone:  req.GetString("timezone", ""),
		StreamIDs: streamIDs,
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
