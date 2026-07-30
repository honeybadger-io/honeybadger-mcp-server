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

	// No v3 endpoint exists for these yet. Left empty rather than omitted, so the
	// completeness test still covers them and they are not filtered out on a
	// scope nobody can hold. See openapi/README.md in api-go.
	"get_fault_counts":              {},
	"get_project_occurrence_counts": {},
	"get_project_report":            {},
	"get_project_integrations":      {},
}

// toolRequiredScopes returns the scopes a tool needs, derived from the spec.
func toolRequiredScopes(tool string) []string {
	ops, known := toolOperations[tool]
	if !known {
		return nil
	}
	var scopes []string
	for _, op := range ops {
		if scope, ok := apiv3.OperationScopes[op]; ok && !contains(scopes, scope) {
			scopes = append(scopes, scope)
		}
	}
	return scopes
}

// filterByScopes drops tools the credential cannot use.
//
// Only called when the credential's scopes are actually known. A tool needs at
// least one of its scopes to be offered: update_fault maps to several endpoints
// that share a scope, and holding it is enough to make the tool useful.
//
// A tool absent from toolOperations is kept rather than hidden. A tool added
// without a map entry should still work; the completeness test is what catches
// the omission, not a silently vanishing tool in production.
func filterByScopes(tools []mcp.Tool, held []string) []mcp.Tool {
	kept := make([]mcp.Tool, 0, len(tools))
	for _, tool := range tools {
		required := toolRequiredScopes(tool.Name)
		if len(required) == 0 || holdsAny(held, required) {
			kept = append(kept, tool)
		}
	}
	return kept
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
