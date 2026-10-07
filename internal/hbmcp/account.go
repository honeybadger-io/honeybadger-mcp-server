package hbmcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// staleSchemaFields are parameters this server no longer advertises but an older
// client may still send, because MCP clients cache tool schemas until they
// reconnect.
//
// This is a safety net, not a description of the API. Each entry is something v3
// genuinely cannot do, so accepting it silently would report success for a change
// that never happened.
var staleSchemaFields = map[string][]string{
	"list_fault_notices": {"created_after", "created_before"},
	// v3 resolves the account from the credential and takes no account id, so a
	// project "created in account B" would silently land in the credential's.
	"list_projects":                 {"account_id"},
	"create_project":                {"account_id"},
	"get_project_occurrence_counts": {"account_id"},
	// filter_events/filter_queries were replaced by filters, a list of
	// event/query pairs; all_sites/all_check_ins by site_ids/check_in_ids null.
	"create_integration": {"filter_events", "filter_queries", "all_sites", "all_check_ins"},
	"update_integration": {"filter_events", "filter_queries", "all_sites", "all_check_ins"},
}

// rejectStaleSchemaFields refuses a request carrying parameters this server used
// to advertise and can no longer honour.
func rejectStaleSchemaFields(tool string, req mcp.CallToolRequest) string {
	fields, ok := staleSchemaFields[tool]
	if !ok {
		return ""
	}
	return rejectUnsupported(req, fields, "using this tool",
		"They are no longer accepted; reconnect to refresh the tool schemas")
}

// requireProjectAndFault reads the two ids every fault tool needs.
func requireProjectAndFault(req mcp.CallToolRequest) (projectID, faultID, errMsg string) {
	projectID = req.GetString("project_id", "")
	if projectID == "" {
		return "", "", "project_id is required"
	}
	faultID, ok := requireFaultID(req.GetArguments(), "fault_id")
	if !ok {
		return "", "", "fault_id is required: the fault's ID, as list_faults and get_fault return it"
	}
	return projectID, faultID, ""
}

// optionalBool reads a boolean argument that may be absent.
//
// Read from the raw arguments rather than through the typed getter, which coerces
// invalid input — null becomes false — and would turn a malformed request into a
// silent state change.
func optionalBool(args map[string]any, name string) (value, present bool, err error) {
	raw, ok := args[name]
	if !ok {
		return false, false, nil
	}
	v, ok := raw.(bool)
	if !ok {
		return false, false, fmt.Errorf("%s must be a boolean", name)
	}
	return v, true, nil
}

// rejectUnsupported refuses a request carrying parameters v3 cannot express.
//
// Silently ignoring them is the one option not on the table: a caller who filters
// or sets something and gets a success back has been told the wrong thing.
func rejectUnsupported(req mcp.CallToolRequest, fields []string, action, advice string) string {
	args := req.GetArguments()
	var present []string
	for _, field := range fields {
		if _, ok := args[field]; ok {
			present = append(present, field)
		}
	}
	if len(present) == 0 {
		return ""
	}
	return fmt.Sprintf("The v3 API does not support %s when %s, so they would be ignored. %s.",
		strings.Join(present, ", "), action, advice)
}

// timeFilters reads the timestamp filters a fault listing or count accepts: an
// RFC 3339 timestamp, or a bare date meaning midnight UTC.
//
// A value it can't parse is refused by name. Dropping it would never reach the
// API's validation — it would just run the query unfiltered and return more than
// was asked for.
func timeFilters(req mcp.CallToolRequest) ([]apiv3.Option, string) {
	var opts []apiv3.Option
	for _, f := range []struct {
		field string
		build func(time.Time) apiv3.ListAllOption
	}{
		{"created_after", apiv3.CreatedAfter},
		{"occurred_after", apiv3.OccurredAfter},
		{"occurred_before", apiv3.OccurredBefore},
	} {
		raw := req.GetString(f.field, "")
		if raw == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			at, err = time.Parse(time.DateOnly, raw)
		}
		if err != nil {
			return nil, fmt.Sprintf("%s must be an RFC 3339 timestamp like 2026-01-02T15:04:05Z or a date like 2026-01-02; got %q", f.field, raw)
		}
		opts = append(opts, f.build(at))
	}
	return opts, ""
}
