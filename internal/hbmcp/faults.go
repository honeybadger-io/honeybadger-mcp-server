package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterFaultTools registers all fault-related MCP tools
// RegisterFaultTools registers the fault tools.
//
// All but get_fault_counts run on v3; that one has no v3 endpoint yet and keeps
// the v2 client along with its numeric ids.
func RegisterFaultTools(r *toolRegistrar, clientFor ClientFactory, v3ClientFor V3ClientFactory) {
	// list_faults tool
	r.AddTool(
		mcp.NewTool("list_faults",
			mcp.WithTitleAnnotation("List Faults"),
			mcp.WithDescription("Get a list of faults for a project with optional filtering and ordering. Requires reference topic: errors (fetch via get_reference; skip if still visible in your context) for the fault/notice model and the q search syntax."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to get faults for"),
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
			return handleListFaults(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_fault tool
	r.AddTool(
		mcp.NewTool("get_fault",
			mcp.WithTitleAnnotation("Get Fault"),
			mcp.WithDescription("Get detailed information for a specific fault in a project"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project containing the fault"),
			),
			mcp.WithString("fault_id",
				mcp.Required(),
				mcp.Description("The ID of the fault to retrieve"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetFault(ctx, v3ClientFor(ctx), req)
		},
	)

	// update_fault tool
	r.AddTool(
		mcp.NewTool("update_fault",
			mcp.WithTitleAnnotation("Update Fault"),
			mcp.WithDescription("Resolve, unresolve, ignore, or unignore a fault. Only the provided fields are changed. Assigning a fault and resolve-on-deploy are not yet available."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project containing the fault"),
			),
			mcp.WithString("fault_id",
				mcp.Required(),
				mcp.Description("The ID of the fault to update"),
			),
			mcp.WithBoolean("resolved",
				mcp.Description("Whether the fault is resolved"),
			),
			mcp.WithBoolean("ignored",
				mcp.Description("Whether the fault is ignored"),
			),
			// assignee_id and resolve_on_deploy are intentionally gone: the v3
			// assign endpoint does not specify its request body, and there is no
			// resolve-on-deploy endpoint at all. Advertising parameters that
			// cannot be sent would be worse than dropping them.
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateFault(ctx, v3ClientFor(ctx), req)
		},
	)

	// list_fault_notices tool
	r.AddTool(
		mcp.NewTool("list_fault_notices",
			mcp.WithTitleAnnotation("List Fault Notices"),
			mcp.WithDescription("Get a list of notices (individual error events) for a specific fault"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project containing the fault"),
			),
			mcp.WithString("fault_id",
				mcp.Required(),
				mcp.Description("The ID of the fault to get notices for"),
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
			return handleListFaultNotices(ctx, v3ClientFor(ctx), req)
		},
	)

	// list_fault_affected_users tool
	r.AddTool(
		mcp.NewTool("list_fault_affected_users",
			mcp.WithTitleAnnotation("List Fault Affected Users"),
			mcp.WithDescription("Get a list of users who were affected by a specific fault with occurrence counts"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project containing the fault"),
			),
			mcp.WithString("fault_id",
				mcp.Required(),
				mcp.Description("The ID of the fault to get affected users for"),
			),
			mcp.WithString("q",
				mcp.Description("Search string to filter affected users"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFaultAffectedUsers(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_fault_counts tool
	r.AddTool(
		mcp.NewTool("get_fault_counts",
			mcp.WithTitleAnnotation("Get Fault Counts"),
			mcp.WithDescription("Get fault count statistics for a project with optional filtering. Requires reference topic: errors (fetch via get_reference; skip if still visible in your context) for the q search syntax. NOTE: this tool still runs on the v2 API, which has no v3 equivalent yet, so it needs the legacy numeric project id — not the opaque id list_projects returns. If you do not already have that numeric id, this tool cannot be used."),
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

// faultFiltersNotInV3 are list_faults parameters v2 accepted that v3 has no
// equivalent for. v3's listFaults takes only page, per_page, and q.
var faultFiltersNotInV3 = []string{"created_after", "occurred_after", "occurred_before", "order"}

func handleListFaults(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	if msg := rejectUnsupported(req, faultFiltersNotInV3,
		"filtering faults", "express the filter in q instead, where the search syntax supports it"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	// limit maps to per_page: both cap how many faults come back in one call.
	opts := []apiv3.Option{}
	if q := req.GetString("q", ""); q != "" {
		opts = append(opts, apiv3.Search(q))
	}
	if page, perPage := req.GetInt("page", 0), req.GetInt("limit", 0); page > 0 || perPage > 0 {
		opts = append(opts, apiv3.Page(max(page, 1), perPage))
	}

	response, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.ListResponse[apiv3.Fault], error) {
			return client.Faults.List(ctx, projectID, append(opts, inAccount(accountID)...)...)
		})
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

func handleGetFault(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, faultID, msg := requireProjectAndFault(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	fault, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.Fault, error) {
			return client.Faults.Get(ctx, projectID, faultID, inAccount(accountID)...)
		})
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

// handleUpdateFault maps the tool's resolved and ignored flags onto v3's discrete
// endpoints.
//
// v2 took one mutable PUT carrying every field. v3 replaced it with an endpoint
// per action, which is why this reads as a sequence rather than a single call.
// Each endpoint takes a list of fault ids; this tool changes one at a time.
//
// The response is a summary of what changed rather than the updated fault: the
// endpoints answer 204, and re-fetching the record to return it would cost an
// extra request the caller may not want. Use get_fault for the new state.
func handleUpdateFault(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, faultID, msg := requireProjectAndFault(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	if msg := rejectUnsupported(req, []string{"assignee_id", "resolve_on_deploy"},
		"updating a fault",
		"the v3 assign endpoint does not specify its request body, and there is no "+
			"resolve-on-deploy endpoint; change these in the Honeybadger UI"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	args := req.GetArguments()
	resolved, hasResolved, err := optionalBool(args, "resolved")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	ignored, hasIgnored, err := optionalBool(args, "ignored")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if !hasResolved && !hasIgnored {
		return mcp.NewToolResultError("at least one of resolved or ignored is required"), nil
	}

	applied := map[string]any{"project_id": projectID, "fault_id": faultID}
	ids := []string{faultID}

	_, err = withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (any, error) {
			opts := inAccount(accountID)
			if hasResolved {
				action := client.Faults.Resolve
				if !resolved {
					action = client.Faults.Unresolve
				}
				if err := action(ctx, projectID, ids, opts...); err != nil {
					return nil, err
				}
				applied["resolved"] = resolved
			}
			if hasIgnored {
				action := client.Faults.Ignore
				if !ignored {
					action = client.Faults.Unignore
				}
				if err := action(ctx, projectID, ids, opts...); err != nil {
					return nil, err
				}
				applied["ignored"] = ignored
			}
			return nil, nil
		})
	if err != nil {
		// Each flag is its own request in v3, so the first can succeed and the
		// second fail. Saying only "failed" would leave the caller believing
		// nothing changed when something did.
		if len(applied) > 2 { // more than the two ids means a change landed
			partial, marshalErr := json.Marshal(applied)
			if marshalErr == nil {
				return mcp.NewToolResultError(fmt.Sprintf(
					"Failed to update fault: %v. These changes were already applied and remain "+
						"in effect: %s", err, partial)), nil
			}
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update fault: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(applied)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleListFaultNotices(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, faultID, msg := requireProjectAndFault(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	// Notices are cursor-paginated in v3, so the timestamp filters have no
	// equivalent — paging walks links rather than naming a time.
	if msg := rejectUnsupported(req, []string{"created_after", "created_before"},
		"listing notices", "notices are paged by cursor in v3; omit these and page instead"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	opts := []apiv3.Option{}
	if limit := req.GetInt("limit", 0); limit > 0 {
		opts = append(opts, apiv3.Limit(limit))
	}

	response, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (*apiv3.ListResponse[apiv3.Notice], error) {
			return client.Faults.ListNotices(ctx, projectID, faultID, append(opts, inAccount(accountID)...)...)
		})
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
func handleListFaultAffectedUsers(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, faultID, msg := requireProjectAndFault(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	// v3's affected-users endpoint takes no search parameter.
	if msg := rejectUnsupported(req, []string{"q"},
		"listing affected users", "v3 returns the full set; filter the result yourself"); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	users, err := withAccount(ctx, client, req.GetString("account_id", ""),
		func(accountID string) (map[string]any, error) {
			return client.Faults.AffectedUsers(ctx, projectID, faultID, inAccount(accountID)...)
		})
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
