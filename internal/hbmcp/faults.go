package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterFaultTools registers all fault-related MCP tools
func RegisterFaultTools(r *toolRegistrar, clientFor ClientFactory) {
	// list_faults tool
	r.AddTool(
		mcp.NewTool("list_faults",
			mcp.WithTitleAnnotation("List Faults"),
			mcp.WithDescription("Get a list of faults for a project with optional filtering and ordering. Requires reference topic: errors (fetch via get_reference; skip if still visible in your context) for the fault/notice model and the q search syntax."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to get faults for"),
				mcp.Min(1),
			),
			mcp.WithString("q",
				mcp.Description("Search string to filter faults (see the errors reference topic for the search query syntax)"),
			),
			mcp.WithString("created_after",
				mcp.Description("Filter faults created after this timestamp"),
			),
			mcp.WithString("occurred_after",
				mcp.Description("Filter faults that occurred after this timestamp"),
			),
			mcp.WithString("occurred_before",
				mcp.Description("Filter faults that occurred before this timestamp"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of faults to return (max 25)"),
				mcp.Min(1),
				mcp.Max(25),
			),
			mcp.WithString("order",
				mcp.Description("Order results by 'recent' or 'frequent'"),
				mcp.Enum("recent", "frequent"),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination"),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFaults(ctx, clientFor(ctx), req)
		},
	)

	// get_fault tool
	r.AddTool(
		mcp.NewTool("get_fault",
			mcp.WithTitleAnnotation("Get Fault"),
			mcp.WithDescription("Get detailed information for a specific fault in a project"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project containing the fault"),
				mcp.Min(1),
			),
			mcp.WithNumber("fault_id",
				mcp.Required(),
				mcp.Description("The ID of the fault to retrieve"),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetFault(ctx, clientFor(ctx), req)
		},
	)

	// update_fault tool
	r.AddTool(
		mcp.NewTool("update_fault",
			mcp.WithTitleAnnotation("Update Fault"),
			mcp.WithDescription("Update a fault's resolved, ignored, assignee, or resolve-on-deploy state. Only the provided fields are changed. Setting resolved or ignored to true in the same request takes precedence over resolve_on_deploy."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project containing the fault"),
				mcp.Min(1),
			),
			mcp.WithNumber("fault_id",
				mcp.Required(),
				mcp.Description("The ID of the fault to update"),
				mcp.Min(1),
			),
			mcp.WithBoolean("resolved",
				mcp.Description("Whether the fault is resolved"),
			),
			mcp.WithBoolean("ignored",
				mcp.Description("Whether the fault is ignored"),
			),
			mcp.WithInteger("assignee_id",
				mcp.Description("Positive integer to assign that user; null to remove the current assignee; omit to leave unchanged"),
				mcp.Min(1),
				nullable,
			),
			mcp.WithBoolean("resolve_on_deploy",
				mcp.Description("Mark the fault to be resolved automatically on next deploy"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateFault(ctx, clientFor(ctx), req)
		},
	)

	// list_fault_notices tool
	r.AddTool(
		mcp.NewTool("list_fault_notices",
			mcp.WithTitleAnnotation("List Fault Notices"),
			mcp.WithDescription("Get a list of notices (individual error events) for a specific fault"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project containing the fault"),
				mcp.Min(1),
			),
			mcp.WithNumber("fault_id",
				mcp.Required(),
				mcp.Description("The ID of the fault to get notices for"),
				mcp.Min(1),
			),
			mcp.WithString("created_after",
				mcp.Description("Filter notices created after this timestamp"),
			),
			mcp.WithString("created_before",
				mcp.Description("Filter notices created before this timestamp"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of notices to return (max 25)"),
				mcp.Min(1),
				mcp.Max(25),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFaultNotices(ctx, clientFor(ctx), req)
		},
	)

	// list_fault_affected_users tool
	r.AddTool(
		mcp.NewTool("list_fault_affected_users",
			mcp.WithTitleAnnotation("List Fault Affected Users"),
			mcp.WithDescription("Get a list of users who were affected by a specific fault with occurrence counts"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project containing the fault"),
				mcp.Min(1),
			),
			mcp.WithNumber("fault_id",
				mcp.Required(),
				mcp.Description("The ID of the fault to get affected users for"),
				mcp.Min(1),
			),
			mcp.WithString("q",
				mcp.Description("Search string to filter affected users"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFaultAffectedUsers(ctx, clientFor(ctx), req)
		},
	)

	// get_fault_counts tool
	r.AddTool(
		mcp.NewTool("get_fault_counts",
			mcp.WithTitleAnnotation("Get Fault Counts"),
			mcp.WithDescription("Get fault count statistics for a project with optional filtering. Requires reference topic: errors (fetch via get_reference; skip if still visible in your context) for the q search syntax."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to get fault counts for"),
				mcp.Min(1),
			),
			mcp.WithString("q",
				mcp.Description("Search string to filter faults (see the errors reference topic for the search query syntax)"),
			),
			mcp.WithString("created_after",
				mcp.Description("Filter faults created after this timestamp"),
			),
			mcp.WithString("occurred_after",
				mcp.Description("Filter faults that occurred after this timestamp"),
			),
			mcp.WithString("occurred_before",
				mcp.Description("Filter faults that occurred before this timestamp"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetFaultCounts(ctx, clientFor(ctx), req)
		},
	)
}

func handleListFaults(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Since project_id is required, MCP will ensure it exists
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	// Build options struct
	options := hbapi.FaultListOptions{
		Q:              req.GetString("q", ""),
		CreatedAfter:   parseTimestampValue(req.GetString("created_after", "")),
		OccurredAfter:  parseTimestampValue(req.GetString("occurred_after", "")),
		OccurredBefore: parseTimestampValue(req.GetString("occurred_before", "")),
		Limit:          req.GetInt("limit", 0),
		Order:          req.GetString("order", ""),
		Page:           req.GetInt("page", 0),
	}

	response, err := client.Faults.List(ctx, projectID, options)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list faults: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetFault(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	faultID := req.GetInt("fault_id", 0)
	if faultID == 0 {
		return mcp.NewToolResultError("fault_id is required"), nil
	}

	fault, err := client.Faults.Get(ctx, projectID, faultID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get fault: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(fault)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateFault(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	projectID, ok := requireID(args, "project_id")
	if !ok {
		return mcp.NewToolResultError("project_id must be a positive integer"), nil
	}

	faultID, ok := requireID(args, "fault_id")
	if !ok {
		return mcp.NewToolResultError("fault_id must be a positive integer"), nil
	}

	// Only include fields that were explicitly provided, so unset fields are
	// omitted from the request instead of being reset. Values are validated
	// against the raw arguments because the typed getters silently coerce
	// invalid input (e.g. null to false, 1.5 to 1).
	params := hbapi.FaultUpdateParams{}

	for _, f := range []struct {
		name  string
		field **bool
	}{
		{"resolved", &params.Resolved},
		{"ignored", &params.Ignored},
		{"resolve_on_deploy", &params.ResolveOnDeploy},
	} {
		raw, ok := args[f.name]
		if !ok {
			continue
		}
		val, ok := raw.(bool)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("%s must be a boolean", f.name)), nil
		}
		*f.field = &val
	}

	if raw, ok := args["assignee_id"]; ok {
		switch v := raw.(type) {
		case nil:
			params.AssigneeID = hbapi.Null[int]() // explicit null unassigns the fault
		case float64:
			if v != float64(int(v)) || v < 1 {
				return mcp.NewToolResultError("assignee_id must be a positive integer or null"), nil
			}
			params.AssigneeID = hbapi.Value(int(v))
		case int: // arguments constructed in Go rather than decoded from JSON
			if v < 1 {
				return mcp.NewToolResultError("assignee_id must be a positive integer or null"), nil
			}
			params.AssigneeID = hbapi.Value(v)
		default:
			return mcp.NewToolResultError("assignee_id must be a positive integer or null"), nil
		}
	}

	if params.Resolved == nil && params.Ignored == nil && params.AssigneeID == nil && params.ResolveOnDeploy == nil {
		return mcp.NewToolResultError("at least one of resolved, ignored, assignee_id, or resolve_on_deploy is required"), nil
	}

	result, err := client.Faults.Update(ctx, projectID, faultID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update fault: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleListFaultNotices(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	faultID := req.GetInt("fault_id", 0)
	if faultID == 0 {
		return mcp.NewToolResultError("fault_id is required"), nil
	}

	// Build options struct
	options := hbapi.FaultListNoticesOptions{
		CreatedAfter:  parseTimestampValue(req.GetString("created_after", "")),
		CreatedBefore: parseTimestampValue(req.GetString("created_before", "")),
		Limit:         req.GetInt("limit", 0),
	}

	response, err := client.Faults.ListNotices(ctx, projectID, faultID, options)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list fault notices: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
func handleListFaultAffectedUsers(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	faultID := req.GetInt("fault_id", 0)
	if faultID == 0 {
		return mcp.NewToolResultError("fault_id is required"), nil
	}

	// Build options struct
	options := hbapi.FaultListAffectedUsersOptions{
		Q: req.GetString("q", ""),
	}

	users, err := client.Faults.ListAffectedUsers(ctx, projectID, faultID, options)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list fault affected users: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(users)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetFaultCounts(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Since project_id is required, MCP will ensure it exists
	projectID := req.GetInt("project_id", 0)
	if projectID == 0 {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	// Build options struct (reuse same filtering options as List)
	options := hbapi.FaultListOptions{
		Q:              req.GetString("q", ""),
		CreatedAfter:   parseTimestampValue(req.GetString("created_after", "")),
		OccurredAfter:  parseTimestampValue(req.GetString("occurred_after", "")),
		OccurredBefore: parseTimestampValue(req.GetString("occurred_before", "")),
	}

	counts, err := client.Faults.GetCounts(ctx, projectID, options)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get fault counts: %v", err)), nil
	}

	// Return JSON response
	jsonBytes, err := json.Marshal(counts)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
