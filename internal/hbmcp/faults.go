package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterFaultTools registers all fault-related MCP tools
// RegisterFaultTools registers the fault tools, all on v3.
func RegisterFaultTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	// list_faults tool
	r.AddTool(
		generatedTool("list_faults"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFaults(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_fault tool
	r.AddTool(
		generatedTool("get_fault"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetFault(ctx, v3ClientFor(ctx), req)
		},
	)

	// update_fault tool
	r.AddTool(
		mcp.NewTool("update_fault",
			mcp.WithTitleAnnotation("Update Fault"),
			mcp.WithDescription("Resolve, unresolve, ignore, or unignore a fault, assign it to a "+
				"project member, snooze or unsnooze its notifications, or set it to resolve on "+
				"the next deploy. Only the provided "+
				"fields are changed. Each field is a separate request, so a later one can fail "+
				"after an earlier one has already taken effect; the result names what was applied."),
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
			// Nullable on purpose: a user's public ID assigns, null unassigns
			// through the DELETE side of the same path, and omitting it changes
			// nothing. The handler reads it from the raw arguments because the typed
			// getter cannot tell null from absent.
			mcp.WithString("assignee_id",
				mcp.Description("Public ID of a project member to assign the fault to. "+
					"Send null to unassign. A user who is not a member of the project is "+
					"rejected rather than silently unassigning."),
				acceptsNull,
			),
			// Nullable like assignee_id: a duration snoozes, null unsnoozes, and
			// omitting it changes nothing.
			mcp.WithString("snooze",
				mcp.Description("Stop the fault's notifications for an hour, a day, or a week. "+
					"Send null to end a snooze. Notices are still recorded, and a resolved "+
					"fault that recurs is still unresolved; only notifications stop. Replaces "+
					"any existing snooze. Cannot be combined with snooze_occurrences."),
				mcp.Enum("hour", "day", "week"),
				acceptsNull,
			),
			mcp.WithNumber("snooze_occurrences",
				mcp.Description("Stop the fault's notifications until this many more "+
					"occurrences arrive: 10, 100, or 1000. With 10, the eleventh notifies. "+
					"Otherwise like snooze. Cannot be combined with snooze."),
				// mcp.Enum takes only strings, so the numeric enum is set by hand;
				// without it the schema would advertise any number.
				func(schema map[string]any) { schema["enum"] = []any{10, 100, 1000} },
			),
			mcp.WithBoolean("resolve_on_deploy",
				mcp.Description("Resolve this fault the next time a deploy is recorded. "+
					"Stored as a pending resolution rather than a change to the fault, so it "+
					"cannot be combined with resolved or ignored — either of those clears it. "+
					"Send false to cancel a pending resolution."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateFault(ctx, v3ClientFor(ctx), req)
		},
	)

	// list_fault_notices tool
	r.AddTool(
		generatedTool("list_fault_notices"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFaultNotices(ctx, v3ClientFor(ctx), req)
		},
	)

	// list_fault_affected_users tool
	r.AddTool(
		generatedTool("list_fault_affected_users"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFaultAffectedUsers(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_fault_counts tool
	r.AddTool(
		generatedTool("get_fault_counts"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetFaultCounts(ctx, v3ClientFor(ctx), req)
		},
	)
}

func handleListFaults(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	opts := []apiv3.Option{}
	if q := req.GetString("q", ""); q != "" {
		opts = append(opts, apiv3.Search(q))
	}
	if order := req.GetString("order", ""); order != "" {
		opts = append(opts, apiv3.OrderBy(order))
	}
	times, msg := timeFilters(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	opts = append(opts, times...)
	if page, perPage := req.GetInt("page", 0), req.GetInt("per_page", 0); page > 0 || perPage > 0 {
		opts = append(opts, apiv3.Page(max(page, 1), perPage))
	}

	response, err := client.Faults.List(ctx, projectID, opts...)
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

// handleUpdateFault maps the tool's resolved and ignored flags onto v3's discrete
// endpoints.
//
// v2 took one mutable PUT carrying every field. v3 replaced it with an endpoint
// per action, which is why this reads as a sequence rather than a single call.
// Each endpoint takes a list of fault ids; this tool changes one at a time.
//
// The response is a summary of what changed rather than the updated fault:
// re-fetching the record to return it would cost an extra request the caller may
// not want. Use get_fault for the new state.
func handleUpdateFault(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, faultID, msg := requireProjectAndFault(req)
	if msg != "" {
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
	// assignee_id is nullable: an explicit null unassigns, a value assigns, and
	// omitting it changes nothing. Read from the raw arguments because the typed
	// getter cannot distinguish null from absent.
	assignee, hasAssignee := args["assignee_id"]
	if hasAssignee {
		if _, isString := assignee.(string); !isString && assignee != nil {
			return mcp.NewToolResultError("assignee_id must be a user's public ID string or null"), nil
		}
	}
	snooze, hasSnooze := args["snooze"]
	if hasSnooze && snooze != nil {
		if d, ok := snooze.(string); !ok || !apiv3.PauseDuration(d).Valid() {
			return mcp.NewToolResultError("snooze must be hour, day, week, or null"), nil
		}
	}
	snoozeCount, hasSnoozeCount := args["snooze_occurrences"]
	if hasSnoozeCount {
		n, ok := snoozeCount.(float64)
		if !ok || n != float64(int(n)) || !apiv3.SnoozeCount(int(n)).Valid() {
			return mcp.NewToolResultError("snooze_occurrences must be 10, 100, or 1000"), nil
		}
	}
	if hasSnooze && hasSnoozeCount {
		return mcp.NewToolResultError(
			"snooze and snooze_occurrences cannot be combined — a fault has one snooze, " +
				"by time or by count. Send one of them."), nil
	}
	onDeploy, hasOnDeploy, err := optionalBool(args, "resolve_on_deploy")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// Resolving or ignoring a fault clears a pending resolution, so only that
	// combination is contradictory — asking to resolve now and on the next deploy.
	// Turning either state off, or cancelling a pending resolution alongside any
	// state change, is coherent and runs in the order given.
	if hasOnDeploy && onDeploy && ((hasResolved && resolved) || (hasIgnored && ignored)) {
		return mcp.NewToolResultError(
			"resolve_on_deploy cannot be combined with resolved:true or ignored:true — both " +
				"clear any pending resolution, so the fault would end up resolved or ignored " +
				"now rather than on the next deploy. Send resolve_on_deploy in its own call."), nil
	}
	if !hasResolved && !hasIgnored && !hasAssignee && !hasSnooze && !hasSnoozeCount && !hasOnDeploy {
		return mcp.NewToolResultError(
			"at least one of resolved, ignored, assignee_id, snooze, snooze_occurrences, or " +
				"resolve_on_deploy is required"), nil
	}

	applied := map[string]any{"project_id": projectID, "fault_id": faultID}

	if hasResolved {
		action := client.Faults.Resolve
		if !resolved {
			action = client.Faults.Unresolve
		}
		err = applyStateChange(ctx, client, action, projectID, faultID, applied, "resolved", resolved)
	}
	if err == nil && hasIgnored {
		action := client.Faults.Ignore
		if !ignored {
			action = client.Faults.Unignore
		}
		err = applyStateChange(ctx, client, action, projectID, faultID, applied, "ignored", ignored)
	}
	if err == nil && hasAssignee {
		var fault *apiv3.Fault
		if id, ok := assignee.(string); ok && id != "" {
			fault, err = client.Faults.Assign(ctx, projectID, faultID, id)
		} else {
			// Null or empty unassigns, through its own endpoint.
			fault, err = client.Faults.Unassign(ctx, projectID, faultID)
		}
		if err == nil {
			// Report the assignee the API stored, not the one requested.
			applied["assignee_id"] = nil
			if a, getErr := fault.Assignee.Get(); getErr == nil && a.Id != nil {
				applied["assignee_id"] = *a.Id
			}
		}
	}
	if err == nil && (hasSnooze || hasSnoozeCount) {
		var fault *apiv3.Fault
		switch {
		case hasSnoozeCount:
			n := apiv3.SnoozeCount(int(snoozeCount.(float64)))
			fault, err = client.Faults.SnoozeOccurrences(ctx, projectID, faultID, n)
			applied["snooze_occurrences"] = int(n)
		case snooze != nil:
			d := apiv3.PauseDuration(snooze.(string))
			fault, err = client.Faults.Snooze(ctx, projectID, faultID, d)
			applied["snooze"] = string(d)
		default:
			fault, err = client.Faults.Unsnooze(ctx, projectID, faultID)
			applied["snooze"] = nil
		}
		if err != nil {
			// Not applied: drop the requested value so a partial-failure report
			// does not list it among the completed steps.
			delete(applied, "snooze")
			delete(applied, "snooze_occurrences")
		} else {
			// Report the state the API returned, not the one requested.
			applied["snoozed"] = fault.Snoozed
		}
	}
	if err == nil && hasOnDeploy {
		// Not a fault column but a pending resolution, so it goes through the
		// fault update endpoint rather than having one of its own.
		var fault *apiv3.Fault
		fault, err = client.Faults.Update(ctx, projectID, faultID,
			apiv3.FaultParams{ResolveOnDeploy: &onDeploy})
		if err == nil {
			// The request succeeds even when the value was discarded: a fault that
			// is already resolved or ignored has no pending resolution to store, and
			// an inactive project omits the field entirely. Report the echo rather
			// than the request, so the tool never claims a change that did not
			// happen.
			applied["resolve_on_deploy"] = fault.ResolveOnDeploy
			if fault.ResolveOnDeploy == nil || *fault.ResolveOnDeploy != onDeploy {
				applied["resolve_on_deploy_note"] = "The API did not store this value. A " +
					"fault that is already resolved or ignored has no pending resolution to " +
					"set, and an inactive project does not report the field."
			}
		}
	}
	if err != nil {
		// Each flag is its own request in v3, so the first can succeed and the
		// second fail. Saying only "failed" would leave the caller believing
		// nothing changed when something did.
		//
		// Worded as steps completed rather than changes applied: a completed step
		// can have changed nothing (already resolved, or unverifiable without read
		// access), and its note says so.
		if len(applied) > 2 { // more than the two ids means a step completed
			partial, marshalErr := json.Marshal(applied)
			if marshalErr == nil {
				return mcp.NewToolResultError(fmt.Sprintf(
					"Failed to update fault: %v. These steps completed before the failure, and "+
						"their results stand (see any notes): %s", err, partial)), nil
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

// bulkAction is one of the four bulk state changes.
type bulkAction func(context.Context, string, apiv3.FaultSelection) (*apiv3.FaultBulkResult, error)

// applyStateChange runs a bulk state change on one fault and records it in
// applied, but only if it can be shown to have landed.
//
// The bulk endpoints succeed with a count of zero both when the fault was
// already in that state and when the id names no fault in the project, so a
// zero is resolved by fetching the fault: a missing or merged fault is an error,
// and an existing one was simply already there.
func applyStateChange(ctx context.Context, client *apiv3.Client, action bulkAction, projectID string, faultID string, applied map[string]any, field string, value bool) error {
	result, err := action(ctx, projectID, apiv3.SelectFaults(faultID))
	if err != nil {
		return err
	}
	if result.Count == 0 {
		fault, err := client.Faults.Get(ctx, projectID, faultID)
		if unreadable(err) {
			// A write-only credential can make the change but not read the fault
			// back, so it cannot tell "already there" from "no such fault here".
			// Say so rather than claim either.
			applied[field+"_note"] = fmt.Sprintf("Nothing changed: the fault was already %s, or is "+
				"not in this project. This credential cannot read faults, so it cannot tell which.",
				stateName(field, value))
			return nil
		}
		if err != nil {
			return err
		}
		// Report what the fault is, not what was asked: a new occurrence can reopen
		// it between the change and this read.
		current := fault.Resolved
		if field == "ignored" {
			current = fault.Ignored
		}
		if current == nil || *current != value {
			return fmt.Errorf("the change to %s did not take: the fault is not %s now (it may have "+
				"changed between requests); check it with get_fault", field, stateName(field, value))
		}
		applied[field+"_note"] = fmt.Sprintf("The fault was already %s; nothing changed.", stateName(field, value))
	}
	applied[field] = value
	return nil
}

func stateName(field string, value bool) string {
	if value {
		return field
	}
	return "un" + field
}

func handleListFaultNotices(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, faultID, msg := requireProjectAndFault(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	// Notices are cursor-paginated in v3, so the old timestamp filters have no
	// equivalent; a client with a cached schema may still send them.
	if msg := rejectStaleSchemaFields("list_fault_notices", req); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	opts := []apiv3.Option{}
	if limit := req.GetInt("limit", 0); limit > 0 {
		opts = append(opts, apiv3.Limit(limit))
	}
	if cursor := req.GetString("before", ""); cursor != "" {
		opts = append(opts, apiv3.Before(cursor))
	}
	if cursor := req.GetString("after", ""); cursor != "" {
		opts = append(opts, apiv3.After(cursor))
	}
	breadcrumbs, _, err := optionalBool(req.GetArguments(), "breadcrumbs")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if breadcrumbs {
		opts = append(opts, apiv3.IncludeBreadcrumbs())
	}

	response, err := client.Faults.ListNotices(ctx, projectID, faultID, opts...)
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
	var opts []apiv3.Option
	if q := req.GetString("q", ""); q != "" {
		opts = append(opts, apiv3.Search(q))
	}

	users, err := client.Faults.AffectedUsers(ctx, projectID, faultID, opts...)
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

func handleGetFaultCounts(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	var opts []apiv3.Option
	if q := req.GetString("q", ""); q != "" {
		opts = append(opts, apiv3.Search(q))
	}
	times, msg := timeFilters(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	opts = append(opts, times...)

	counts, err := client.Faults.Summary(ctx, projectID, opts...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get fault counts: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(counts)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}
