package hbmcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestToolRegistrar_AddTool(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	r := newToolRegistrar(s)

	r.AddTool(
		mcp.NewTool("test_tool",
			mcp.WithDescription("A test tool"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("ok"), nil
		},
	)

	if len(r.catalog) != 1 {
		t.Fatalf("expected 1 tool in catalog, got %d", len(r.catalog))
	}

	info := r.catalog[0]
	if info.Name != "test_tool" {
		t.Errorf("expected name 'test_tool', got %q", info.Name)
	}
	if info.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", info.Description)
	}
	if !info.ReadOnly {
		t.Error("expected ReadOnly to be true")
	}
}

func TestToolRegistrar_AddTool_NonReadOnly(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	r := newToolRegistrar(s)

	r.AddTool(
		mcp.NewTool("write_tool",
			mcp.WithDescription("A write tool"),
			mcp.WithReadOnlyHintAnnotation(false),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("ok"), nil
		},
	)

	if len(r.catalog) != 1 {
		t.Fatalf("expected 1 tool in catalog, got %d", len(r.catalog))
	}

	if r.catalog[0].ReadOnly {
		t.Error("expected ReadOnly to be false")
	}
}

func TestToolRegistrar_MultipleTools(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	r := newToolRegistrar(s)

	r.AddTool(
		mcp.NewTool("tool_a", mcp.WithDescription("Tool A"), mcp.WithReadOnlyHintAnnotation(true)),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("ok"), nil
		},
	)
	r.AddTool(
		mcp.NewTool("tool_b", mcp.WithDescription("Tool B"), mcp.WithReadOnlyHintAnnotation(false)),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("ok"), nil
		},
	)

	if len(r.catalog) != 2 {
		t.Fatalf("expected 2 tools in catalog, got %d", len(r.catalog))
	}

	if r.catalog[0].Name != "tool_a" {
		t.Errorf("expected first tool 'tool_a', got %q", r.catalog[0].Name)
	}
	if r.catalog[1].Name != "tool_b" {
		t.Errorf("expected second tool 'tool_b', got %q", r.catalog[1].Name)
	}
}

func TestSearchCatalog(t *testing.T) {
	catalog := []ToolInfo{
		{Name: "list_projects", Description: "List all Honeybadger projects", ReadOnly: true},
		{Name: "get_project", Description: "Get a single Honeybadger project by ID", ReadOnly: true},
		{Name: "create_project", Description: "Create a new Honeybadger project", ReadOnly: false},
		{Name: "list_faults", Description: "Get a list of faults for a project", ReadOnly: true},
		{Name: "query_insights", Description: "Execute a BadgerQL query against Insights data", ReadOnly: true},
	}

	tests := []struct {
		name     string
		query    string
		expected int
		names    []string
	}{
		{
			name:     "match by name",
			query:    "list",
			expected: 2,
			names:    []string{"list_projects", "list_faults"},
		},
		{
			name:     "match by description",
			query:    "BadgerQL",
			expected: 1,
			names:    []string{"query_insights"},
		},
		{
			name:     "case insensitive",
			query:    "HONEYBADGER",
			expected: 3,
			names:    []string{"list_projects", "get_project", "create_project"},
		},
		{
			name:     "partial name match",
			query:    "project",
			expected: 4,
			names:    []string{"list_projects", "get_project", "create_project", "list_faults"},
		},
		{
			name:     "no matches",
			query:    "nonexistent",
			expected: 0,
		},
		{
			name:     "match by description keyword",
			query:    "faults",
			expected: 1,
			names:    []string{"list_faults"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := searchCatalog(catalog, tt.query)
			if len(results) != tt.expected {
				t.Errorf("searchCatalog(%q) returned %d results, expected %d", tt.query, len(results), tt.expected)
			}

			if tt.names != nil {
				nameMap := make(map[string]bool)
				for _, r := range results {
					nameMap[r.Name] = true
				}
				for _, expected := range tt.names {
					if !nameMap[expected] {
						t.Errorf("expected tool %q in results", expected)
					}
				}
			}
		})
	}
}

func TestSearchCatalog_MultiWord(t *testing.T) {
	catalog := []ToolInfo{
		{Name: "list_projects", Description: "List all Honeybadger projects", ReadOnly: true},
		{Name: "list_faults", Description: "Get a list of faults for a project", ReadOnly: true},
		{Name: "create_check_in", Description: "Create a check-in for a project", ReadOnly: false},
		{Name: "query_insights", Description: "Execute a BadgerQL query against Insights data", ReadOnly: true},
		{Name: "get_fault_counts", Description: "Return occurrence totals over a time range", ReadOnly: true},
		{Name: "list_streams", Description: "List Insights streams, including read-only internal ones", ReadOnly: true},
	}

	tests := []struct {
		name  string
		query string
		names []string
	}{
		{
			name:  "words match underscored name",
			query: "list faults",
			names: []string{"list_faults"},
		},
		{
			name:  "word order does not matter",
			query: "faults list",
			names: []string{"list_faults"},
		},
		{
			name:  "all words must match",
			query: "list dashboards",
			names: []string{},
		},
		{
			// "counts" appears only in the name, "occurrence" only in the description.
			name:  "words may span name and description",
			query: "counts occurrence",
			names: []string{"get_fault_counts"},
		},
		{
			name:  "empty query matches nothing",
			query: "",
			names: []string{},
		},
		{
			name:  "whitespace-only query matches nothing",
			query: "   ",
			names: []string{},
		},
		{
			name:  "separator-only query matches nothing",
			query: "_",
			names: []string{},
		},
		{
			// "read-only" appears only in list_streams' description, and
			// neither word appears in its name.
			name:  "hyphenated description matches spaced query",
			query: "read only",
			names: []string{"list_streams"},
		},
		{
			name:  "underscored query matches underscored name",
			query: "list_faults",
			names: []string{"list_faults"},
		},
		{
			name:  "hyphenated query matches underscored name",
			query: "list-faults",
			names: []string{"list_faults"},
		},
		{
			name:  "hyphenated query matches hyphenated description",
			query: "read-only",
			names: []string{"list_streams"},
		},
		{
			name:  "extra whitespace is ignored",
			query: "  list   faults  ",
			names: []string{"list_faults"},
		},
		{
			name:  "single word behaves as before",
			query: "project",
			names: []string{"list_projects", "list_faults", "create_check_in"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := searchCatalog(catalog, tt.query)
			if len(results) != len(tt.names) {
				t.Errorf("searchCatalog(%q) returned %d results, expected %d", tt.query, len(results), len(tt.names))
			}

			got := make(map[string]bool, len(results))
			for _, r := range results {
				got[r.Name] = true
			}
			for _, want := range tt.names {
				if !got[want] {
					t.Errorf("expected tool %q in results for query %q", want, tt.query)
				}
			}
		})
	}
}

func TestSearchCatalog_MultiWordAgainstRealCatalog(t *testing.T) {
	_, catalog := NewServerWithCatalog(&config.Config{TransportMode: config.TransportStdio}, "test")

	cases := map[string]string{
		"list faults":  "list_faults",
		"create alarm": "create_alarm",
		"check in":     "list_check_ins",
	}

	for query, want := range cases {
		results := searchCatalog(catalog, query)
		found := false
		for _, r := range results {
			if r.Name == want {
				found = true
			}
		}
		if !found {
			t.Errorf("searchCatalog(%q) did not return %q (got %d results)", query, want, len(results))
		}
	}
}

func TestSearchCatalog_PreservesReadOnlyInfo(t *testing.T) {
	catalog := []ToolInfo{
		{Name: "list_projects", Description: "List all projects", ReadOnly: true},
		{Name: "create_project", Description: "Create a new project", ReadOnly: false},
	}

	results := searchCatalog(catalog, "project")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Name == "list_projects" && !r.ReadOnly {
			t.Error("list_projects should be read-only")
		}
		if r.Name == "create_project" && r.ReadOnly {
			t.Error("create_project should not be read-only")
		}
	}
}

func TestSearchCatalog_EmptyCatalog(t *testing.T) {
	results := searchCatalog(nil, "anything")
	if len(results) != 0 {
		t.Errorf("expected 0 results from empty catalog, got %d", len(results))
	}
}

func TestRegisterSearchTool(t *testing.T) {
	catalog := []ToolInfo{
		{Name: "list_projects", Description: "List all Honeybadger projects", ReadOnly: true},
		{Name: "create_project", Description: "Create a new Honeybadger project", ReadOnly: false},
	}

	s := server.NewMCPServer("test", "1.0.0")
	registerSearchTool(s, catalog, &config.Config{ReadOnly: false, TransportMode: config.TransportStdio})

	// Verify search_tools is registered by calling it through HandleMessage
	callMsg := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_tools","arguments":{"query":"list"}}}`
	resp := s.HandleMessage(context.Background(), []byte(callMsg))

	respBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}
	respStr := string(respBytes)

	if strings.Contains(respStr, "tool search_tools not found") {
		t.Error("search_tools tool was not registered")
	}
	if !strings.Contains(respStr, "list_projects") {
		t.Errorf("expected search results to contain 'list_projects', got: %s", respStr)
	}
}

func TestRegisterSearchTool_NoMatches(t *testing.T) {
	catalog := []ToolInfo{
		{Name: "list_projects", Description: "List all Honeybadger projects", ReadOnly: true},
	}

	s := server.NewMCPServer("test", "1.0.0")
	registerSearchTool(s, catalog, &config.Config{ReadOnly: false, TransportMode: config.TransportStdio})

	callMsg := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_tools","arguments":{"query":"nonexistent"}}}`
	resp := s.HandleMessage(context.Background(), []byte(callMsg))

	respBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}
	respStr := string(respBytes)

	if !strings.Contains(respStr, "No tools found matching the query.") {
		t.Errorf("expected no-matches message, got: %s", respStr)
	}
}

func TestRegisterSearchTool_ReadOnlyMode(t *testing.T) {
	catalog := []ToolInfo{
		{Name: "list_projects", Description: "List all Honeybadger projects", ReadOnly: true},
		{Name: "create_project", Description: "Create a new Honeybadger project", ReadOnly: false},
		{Name: "delete_project", Description: "Delete a Honeybadger project", ReadOnly: false},
	}

	s := server.NewMCPServer("test", "1.0.0")
	registerSearchTool(s, catalog, &config.Config{ReadOnly: true, TransportMode: config.TransportStdio})

	// Search for "project" - should only return read-only tools
	callMsg := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_tools","arguments":{"query":"project"}}}`
	resp := s.HandleMessage(context.Background(), []byte(callMsg))

	respBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}
	respStr := string(respBytes)

	if !strings.Contains(respStr, "list_projects") {
		t.Error("expected read-only tool 'list_projects' in results")
	}
	if strings.Contains(respStr, "create_project") {
		t.Error("destructive tool 'create_project' should not appear in read-only mode")
	}
	if strings.Contains(respStr, "delete_project") {
		t.Error("destructive tool 'delete_project' should not appear in read-only mode")
	}
}

func TestRegisterSearchTool_BlankQueries(t *testing.T) {
	catalog := []ToolInfo{
		{Name: "list_projects", Description: "List all Honeybadger projects", ReadOnly: true},
	}

	queries := map[string]string{
		"empty":           "",
		"whitespace only": "   ",
		"separator only":  "_",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			s := server.NewMCPServer("test", "1.0.0")
			registerSearchTool(s, catalog, &config.Config{ReadOnly: false, TransportMode: config.TransportStdio})

			args, err := json.Marshal(map[string]string{"query": query})
			if err != nil {
				t.Fatalf("failed to marshal arguments: %v", err)
			}
			callMsg := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_tools","arguments":` + string(args) + `}}`
			resp := s.HandleMessage(context.Background(), []byte(callMsg))

			respBytes, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("failed to marshal response: %v", err)
			}
			respStr := string(respBytes)

			if !strings.Contains(respStr, "query is required") {
				t.Errorf("expected 'query is required' for query %q, got: %s", query, respStr)
			}
		})
	}
}

func TestFilterReadOnlyCatalog(t *testing.T) {
	catalog := []ToolInfo{
		{Name: "list_projects", Description: "List all projects", ReadOnly: true},
		{Name: "create_project", Description: "Create a project", ReadOnly: false},
		{Name: "get_project", Description: "Get a project", ReadOnly: true},
		{Name: "delete_project", Description: "Delete a project", ReadOnly: false},
	}

	filtered := filterReadOnlyCatalog(catalog)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 read-only tools, got %d", len(filtered))
	}

	for _, t2 := range filtered {
		if !t2.ReadOnly {
			t.Errorf("non-read-only tool %q in filtered results", t2.Name)
		}
	}
}
