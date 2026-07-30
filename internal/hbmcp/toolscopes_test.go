package hbmcp

import (
	"testing"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// Every registered tool must appear in toolOperations. A tool added without an
// entry would be advertised to credentials that cannot use it, and its scope
// requirement would be invisible here.
func TestEveryToolDeclaresItsOperations(t *testing.T) {
	for _, tool := range registeredToolNames(t) {
		if _, ok := toolOperations[tool]; !ok {
			t.Errorf("tool %q has no toolOperations entry; add one (empty if it calls no API)", tool)
		}
	}
}

// Conversely, the map must not name tools that no longer exist.
func TestToolOperationsHasNoStaleEntries(t *testing.T) {
	registered := map[string]bool{}
	for _, name := range registeredToolNames(t) {
		registered[name] = true
	}
	for name := range toolOperations {
		if !registered[name] {
			t.Errorf("toolOperations names %q, which is not a registered tool", name)
		}
	}
}

// registeredToolNames returns the catalog the real server advertises, so these
// tests cannot drift from what production registers.
func registeredToolNames(t *testing.T) []string {
	t.Helper()
	_, catalog := NewServerWithCatalog(&config.Config{
		AuthToken:     "test-token",
		APIURL:        "https://app.honeybadger.io",
		LogLevel:      "info",
		ReadOnly:      false, // so the writing tools are included
		TransportMode: config.TransportStdio,
	}, "test")

	names := make([]string, 0, len(catalog))
	for _, tool := range catalog {
		names = append(names, tool.Name)
	}
	return names
}

// Every operationId named must exist in the generated map, or be deliberately
// scope-free. A typo would otherwise mean a tool requires no scope by accident.
func TestToolOperationsReferenceRealOperations(t *testing.T) {
	// getToken is the only operation the spec leaves scope-free.
	scopeFree := map[string]bool{"getToken": true}

	for tool, ops := range toolOperations {
		for _, op := range ops {
			if _, ok := apiv3.OperationScopes[op]; !ok && !scopeFree[op] {
				t.Errorf("tool %q names operation %q, which is not in apiv3.OperationScopes — typo?",
					tool, op)
			}
		}
	}
}

func TestToolRequiredScopes(t *testing.T) {
	for tool, want := range map[string]string{
		"list_faults":    "faults:read",
		"update_fault":   "faults:write",
		"query_insights": "insights:read",
		"create_project": "projects:create",
		"delete_alarm":   "alarms:write",
	} {
		got := toolRequiredScopes(tool)
		if len(got) != 1 || got[0] != want {
			t.Errorf("toolRequiredScopes(%q) = %v, want [%q]", tool, got, want)
		}
	}
}

// Only the tools that genuinely reach no API are scope-free. get_project_report
// is not one of them — it calls v2, so it needs a read scope like any other read.
func TestScopeFreeToolsAlwaysSurvive(t *testing.T) {
	tools := []mcp.Tool{{Name: "get_reference"}, {Name: "search_tools"}}
	kept := filterByScopes(tools, nil)
	if len(kept) != 2 {
		t.Errorf("kept %d of 2 scope-free tools with no scopes held", len(kept))
	}
}

// The tools awaiting a v3 endpoint still read data, so a credential with no read
// scope must not be offered them.
func TestPendingMigrationToolsStillRequireScopes(t *testing.T) {
	for _, tool := range []string{
		"get_fault_counts", "get_project_occurrence_counts",
		"get_project_report", "get_project_integrations",
	} {
		if len(toolRequiredScopes(tool)) == 0 {
			t.Errorf("%s requires no scope; it reads data and should need one", tool)
		}
	}

	kept := filterByScopes([]mcp.Tool{{Name: "get_fault_counts"}}, []string{"insights:read"})
	if len(kept) != 0 {
		t.Error("get_fault_counts was offered to a credential holding only insights:read")
	}
}

func TestFilterByScopesHidesUnusableTools(t *testing.T) {
	tools := []mcp.Tool{
		{Name: "list_faults"},    // faults:read
		{Name: "update_fault"},   // faults:write
		{Name: "query_insights"}, // insights:read
		{Name: "get_reference"},  // none
	}

	kept := filterByScopes(tools, []string{"faults:read"})

	names := map[string]bool{}
	for _, tool := range kept {
		names[tool.Name] = true
	}
	if !names["list_faults"] || !names["get_reference"] {
		t.Errorf("kept %v, want list_faults and get_reference", names)
	}
	if names["update_fault"] {
		t.Error("update_fault was offered to a credential holding only faults:read")
	}
	if names["query_insights"] {
		t.Error("query_insights was offered without insights:read")
	}
}

// An unmapped tool is kept rather than hidden: a missing map entry is a bug for
// the completeness test to catch, not a reason to make a tool disappear in
// production.
func TestFilterByScopesKeepsUnknownTools(t *testing.T) {
	kept := filterByScopes([]mcp.Tool{{Name: "brand_new_tool"}}, []string{"faults:read"})
	if len(kept) != 1 {
		t.Error("an unmapped tool was filtered out; it should be kept and flagged by tests")
	}
}
