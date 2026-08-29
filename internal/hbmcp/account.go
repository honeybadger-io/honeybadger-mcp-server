package hbmcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

// staleSchemaFields are parameters this server no longer advertises but an older
// client may still send, because MCP clients cache tool schemas until they
// reconnect.
//
// This is a safety net, not a description of the API. Each entry is something v3
// genuinely cannot do, so accepting it silently would report success for a change
// that never happened.
var staleSchemaFields = map[string][]string{
	"list_fault_notices": {"created_after", "created_before"},
}

// rejectStaleSchemaFields refuses a request carrying parameters this server used
// to advertise and can no longer honour.
func rejectStaleSchemaFields(tool string, req mcp.CallToolRequest) string {
	fields, ok := staleSchemaFields[tool]
	if !ok {
		return ""
	}
	return rejectUnsupported(req, fields, "this operation",
		"they are no longer accepted; reconnect to refresh the tool schemas")
}

// requireProjectAndFault reads the two ids every fault tool needs.
func requireProjectAndFault(req mcp.CallToolRequest) (projectID string, faultID int, errMsg string) {
	projectID = req.GetString("project_id", "")
	if projectID == "" {
		return "", 0, "project_id is required"
	}
	faultID = req.GetInt("fault_id", 0)
	if faultID == 0 {
		return "", 0, "fault_id is required"
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

// timeFilters reads the timestamp filters a fault listing or count accepts.
//
// Values are ISO 8601, as the tools have always taken them, and are dropped
// rather than erroring when unparseable — the API's own validation gives a better
// message than a guess here would.
func timeFilters(req mcp.CallToolRequest) []apiv3.Option {
	var opts []apiv3.Option
	for field, build := range map[string]func(time.Time) apiv3.ListAllOption{
		"created_after":   apiv3.CreatedAfter,
		"occurred_after":  apiv3.OccurredAfter,
		"occurred_before": apiv3.OccurredBefore,
	} {
		raw := req.GetString(field, "")
		if raw == "" {
			continue
		}
		if at, err := time.Parse(time.RFC3339, raw); err == nil {
			opts = append(opts, build(at))
		}
	}
	return opts
}
