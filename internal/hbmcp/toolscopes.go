package hbmcp

import (
	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// toolOperations maps each API-backed tool to the v3 operations it calls.
//
// The scope each operation requires comes from apiv3.OperationScopes, which is
// generated from the spec — so this map records only which endpoints a tool
// touches, never what they permit. That split matters: the permission half
// cannot drift from the API, and this half is checked for completeness by
// TestEveryToolDeclaresItsOperations.
//
// A tool listed with no operations reaches no API and needs no scope.
var toolOperations = map[string][]string{
	// Reachable without any credential scope.
	"get_reference": {},
	"search_tools":  {},

	"list_projects":  {"listProjects"},
	"get_project":    {"getProject"},
	"create_project": {"createProject"},
	"update_project": {"updateProject"},
	"delete_project": {"deleteProject"},

	"list_faults":               {"listFaults"},
	"get_fault":                 {"getFault"},
	"list_fault_notices":        {"listNotices"},
	"list_fault_affected_users": {"listFaultAffectedUsers"},

	// update_fault covers several discrete v3 endpoints, since v3 replaced v2's
	// mutable PUT with one endpoint per action. They share a scope, but listing
	// them all keeps the map honest about what the tool reaches.
	"update_fault": {"resolveFaults", "unresolveFaults", "ignoreFaults", "unignoreFaults"},

	"list_check_ins":  {"listCheckIns"},
	"get_check_in":    {"getCheckIn"},
	"create_check_in": {"createCheckIn"},
	"update_check_in": {"updateCheckIn"},
	"delete_check_in": {"deleteCheckIn"},

	"list_dashboards":  {"listDashboards"},
	"get_dashboard":    {"getDashboard"},
	"create_dashboard": {"createDashboard"},
	"update_dashboard": {"updateDashboard"},
	"delete_dashboard": {"deleteDashboard"},

	"list_alarms":       {"listAlarms"},
	"get_alarm":         {"getAlarm"},
	"get_alarm_history": {"listAlarmHistory"},
	"create_alarm":      {"createAlarm"},
	"update_alarm":      {"updateAlarm"},
	"delete_alarm":      {"deleteAlarm"},

	"query_insights": {"runInsightsQuery"},
	"list_streams":   {"listStreams"},

	// These four still run on the v2 client because v3 has no equivalent endpoint.
	//
	// They are mapped to the v3 operation that will replace each one, so scope
	// filtering already treats them as what they are — reads of faults and
	// projects — rather than as tools needing nothing. A credential holding no
	// read scope should not be offered them just because their migration is
	// pending.
	"get_fault_counts":              {"getFaultSummary"},
	"get_project_occurrence_counts": {"getProjectOccurrences", "listAccountOccurrences"},
	"get_project_integrations":      {"listChannels"},

	// The last tool without a v3 endpoint. Mapped to the closest read that does
	// exist, so a credential holding no project read scope is not offered it.
	"get_project_report": {"getProjectStats"},
}

// toolRequiredScopes returns the scopes a tool needs, derived from the spec.
//
// resolved reports whether every operation the tool names was found in the
// generated map. It is not the same question as whether any scopes came back: a
// tool that reaches no API resolves cleanly to no scopes, while a tool naming an
// operation the generator dropped resolves to nothing knowable. Collapsing those
// two into an empty slice is what made a truncated scope map widen access
// instead of narrowing it.
func toolRequiredScopes(tool string) (scopes []string, resolved bool) {
	ops, known := toolOperations[tool]
	if !known {
		return nil, false
	}
	for _, op := range ops {
		scope, ok := apiv3.OperationScopes[op]
		if !ok {
			return nil, false
		}
		if !contains(scopes, scope) {
			scopes = append(scopes, scope)
		}
	}
	return scopes, true
}

// filterByScopes drops tools the credential cannot use.
//
// Only called when the credential's scopes are actually known. A tool needs at
// least one of its scopes to be offered: update_fault maps to several endpoints
// that share a scope, and holding it is enough to make the tool useful.
//
// Two kinds of "we don't know" are handled differently, because they fail in
// opposite directions:
//
//   - The tool is absent from toolOperations. That map is hand-written and
//     directly guarded by TestEveryToolDeclaresItsOperations, so an omission is a
//     new tool nobody mapped yet. Offered, because hiding it would make a working
//     tool vanish for everyone.
//   - The tool names an operation the generated scope map does not describe. That
//     map is regenerated from the spec and can lose an entry without anyone
//     writing a line of code. Hidden, because an operation that claims to need a
//     scope and cannot say which is not something to offer on a guess.
//
// The second case should be unreachable — TestEveryMappedOperationHasAScope fails
// the build first — and exists as a backstop, not a condition to rely on.
func filterByScopes(tools []mcp.Tool, held []string) []mcp.Tool {
	kept := make([]mcp.Tool, 0, len(tools))
	for _, tool := range tools {
		if offerTool(tool.Name, held) {
			kept = append(kept, tool)
		}
	}
	return kept
}

// offerTool reports whether a credential holding these scopes may see the tool.
func offerTool(name string, held []string) bool {
	if _, mapped := toolOperations[name]; !mapped {
		return true
	}
	required, resolved := toolRequiredScopes(name)
	if !resolved {
		return false
	}
	return len(required) == 0 || holdsAny(held, required)
}

func holdsAny(held, required []string) bool {
	for _, r := range required {
		if contains(held, r) {
			return true
		}
	}
	return false
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// filterCatalogByScopes is filterByScopes for the searchable catalog, which
// carries ToolInfo rather than mcp.Tool. Both must apply the same rule: search
// would otherwise surface tools tools/list had hidden.
func filterCatalogByScopes(catalog []ToolInfo, held []string) []ToolInfo {
	kept := make([]ToolInfo, 0, len(catalog))
	for _, tool := range catalog {
		if offerTool(tool.Name, held) {
			kept = append(kept, tool)
		}
	}
	return kept
}
