package hbmcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestAllToolsHaveTitleAndAnnotations enforces the tool-authoring contract for
// the Anthropic Connectors Directory: every tool exposed over tools/list must
// carry a human-readable title annotation and an explicit readOnlyHint. This is
// a guard against forgetting mcp.WithTitleAnnotation (or the hint) on a new
// tool — a missing title is an automatic directory-review rejection.
func TestAllToolsHaveTitleAndAnnotations(t *testing.T) {
	cfg := &config.Config{
		AuthToken:     "test-token",
		APIURL:        "https://api.honeybadger.io/v2",
		LogLevel:      "info",
		ReadOnly:      false, // non-read-only so write tools are listed too
		TransportMode: config.TransportStdio,
	}

	s := NewServer(cfg, "test")

	listMsg := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	resp := s.HandleMessage(context.Background(), []byte(listMsg))

	respBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal tools/list response: %v", err)
	}

	var parsed struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Annotations struct {
					Title           string `json:"title"`
					ReadOnlyHint    *bool  `json:"readOnlyHint"`
					DestructiveHint *bool  `json:"destructiveHint"`
				} `json:"annotations"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal tools/list response: %v", err)
	}

	if len(parsed.Result.Tools) == 0 {
		t.Fatal("tools/list returned no tools")
	}

	for _, tool := range parsed.Result.Tools {
		if tool.Annotations.Title == "" {
			t.Errorf("tool %q is missing a title annotation (add mcp.WithTitleAnnotation)", tool.Name)
		}
		if tool.Annotations.ReadOnlyHint == nil {
			t.Errorf("tool %q is missing a readOnlyHint annotation", tool.Name)
		}
		if tool.Annotations.DestructiveHint == nil {
			t.Errorf("tool %q is missing a destructiveHint annotation", tool.Name)
		}
	}
}

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		AuthToken: "test-token",
		APIURL:    "https://api.honeybadger.io/v2",
		LogLevel:  "info",
	}

	server := NewServer(cfg, "test")
	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	// Basic test to ensure server is created
	// Note: mcp-go doesn't expose tool listing in the public API
	// We'll test functionality through manual testing
}

func TestNewServer_ReadOnlyMode(t *testing.T) {
	cfg := &config.Config{
		AuthToken: "test-token",
		APIURL:    "https://api.honeybadger.io/v2",
		LogLevel:  "info",
		ReadOnly:  true,
	}

	server := NewServer(cfg, "test")
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestNewServer_NonReadOnlyMode(t *testing.T) {
	cfg := &config.Config{
		AuthToken: "test-token",
		APIURL:    "https://api.honeybadger.io/v2",
		LogLevel:  "info",
		ReadOnly:  false,
	}

	server := NewServer(cfg, "test")
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestNewServerWithCatalog(t *testing.T) {
	cfg := &config.Config{
		AuthToken: "test-token",
		APIURL:    "https://api.honeybadger.io/v2",
		LogLevel:  "info",
	}

	server, catalog := NewServerWithCatalog(cfg, "test")
	if server == nil {
		t.Fatal("NewServerWithCatalog returned nil server")
	}
	if len(catalog) == 0 {
		t.Fatal("NewServerWithCatalog returned empty catalog")
	}

	byName := make(map[string]ToolInfo, len(catalog))
	for _, tool := range catalog {
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
		byName[tool.Name] = tool
	}

	cases := []struct {
		name     string
		readOnly bool
	}{
		{"list_projects", true},
		{"create_project", false},
		{"search_tools", true},
	}
	for _, c := range cases {
		tool, ok := byName[c.name]
		if !ok {
			t.Errorf("catalog missing tool %q", c.name)
			continue
		}
		if tool.ReadOnly != c.readOnly {
			t.Errorf("tool %q ReadOnly = %v, want %v", c.name, tool.ReadOnly, c.readOnly)
		}
	}
}

func TestEffectiveReadOnly(t *testing.T) {
	withClaims := func(scopes ...string) context.Context {
		return WithClaims(context.Background(), &Claims{Scopes: scopes})
	}

	cases := []struct {
		name string
		ctx  context.Context
		cfg  *config.Config
		want bool
	}{
		{
			name: "stdio + read-only flag → read-only",
			ctx:  context.Background(),
			cfg:  &config.Config{TransportMode: config.TransportStdio, ReadOnly: true},
			want: true,
		},
		{
			name: "stdio + no read-only flag → not read-only (PAT trusts the operator)",
			ctx:  context.Background(),
			cfg:  &config.Config{TransportMode: config.TransportStdio, ReadOnly: false},
			want: false,
		},
		{
			name: "http + read-only startup flag ignored (scope is authoritative)",
			ctx:  withClaims("read", "write"),
			cfg:  &config.Config{TransportMode: config.TransportHTTP, ReadOnly: true},
			want: false,
		},
		{
			name: "http + write scope → not read-only",
			ctx:  withClaims("read", "write"),
			cfg:  &config.Config{TransportMode: config.TransportHTTP, ReadOnly: false},
			want: false,
		},
		{
			name: "http + read-only scope → read-only",
			ctx:  withClaims("read"),
			cfg:  &config.Config{TransportMode: config.TransportHTTP, ReadOnly: false},
			want: true,
		},
		{
			name: "http + no scopes → read-only",
			ctx:  withClaims(),
			cfg:  &config.Config{TransportMode: config.TransportHTTP, ReadOnly: false},
			want: true,
		},
		{
			name: "http + missing claims (middleware bypass) → fail closed to read-only",
			ctx:  context.Background(),
			cfg:  &config.Config{TransportMode: config.TransportHTTP, ReadOnly: false},
			want: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := EffectiveReadOnly(c.ctx, c.cfg); got != c.want {
				t.Errorf("EffectiveReadOnly = %v, want %v", got, c.want)
			}
		})
	}
}

func TestFilterReadOnlyTools(t *testing.T) {
	tests := []struct {
		name     string
		tools    []mcp.Tool
		expected int
	}{
		{
			name:     "empty tools list",
			tools:    []mcp.Tool{},
			expected: 0,
		},
		{
			name: "only read-only tools",
			tools: []mcp.Tool{
				mcp.NewTool("read_tool_1", mcp.WithReadOnlyHintAnnotation(true)),
				mcp.NewTool("read_tool_2", mcp.WithReadOnlyHintAnnotation(true)),
			},
			expected: 2,
		},
		{
			name: "only non-readonly tools",
			tools: []mcp.Tool{
				mcp.NewTool("non_readonly_tool_1", mcp.WithReadOnlyHintAnnotation(false)),
				mcp.NewTool("non_readonly_tool_2", mcp.WithReadOnlyHintAnnotation(false)),
			},
			expected: 0,
		},
		{
			name: "mixed tools",
			tools: []mcp.Tool{
				mcp.NewTool("read_tool", mcp.WithReadOnlyHintAnnotation(true)),
				mcp.NewTool("non_readonly_tool", mcp.WithReadOnlyHintAnnotation(false)),
				mcp.NewTool("another_read_tool", mcp.WithReadOnlyHintAnnotation(true)),
			},
			expected: 2,
		},
		{
			name: "tools without read-only hint",
			tools: []mcp.Tool{
				mcp.NewTool("tool_without_hint"),
				mcp.NewTool("read_tool", mcp.WithReadOnlyHintAnnotation(true)),
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterReadOnlyTools(tt.tools)
			if len(result) != tt.expected {
				t.Errorf("filterReadOnlyTools() returned %d tools, expected %d", len(result), tt.expected)
			}

			for _, tool := range result {
				if tool.Annotations.ReadOnlyHint == nil || !*tool.Annotations.ReadOnlyHint {
					t.Errorf("filterReadOnlyTools() returned non-read-only tool: %s", tool.Name)
				}
			}
		})
	}
}

func TestFilterReadOnlyTools_SpecificTools(t *testing.T) {
	tools := []mcp.Tool{
		// Read-only tools (should be included)
		mcp.NewTool("list_projects", mcp.WithReadOnlyHintAnnotation(true)),
		mcp.NewTool("get_project", mcp.WithReadOnlyHintAnnotation(true)),
		mcp.NewTool("list_faults", mcp.WithReadOnlyHintAnnotation(true)),
		// Non-readonly tools (should be filtered out)
		mcp.NewTool("create_project", mcp.WithReadOnlyHintAnnotation(false)),
		mcp.NewTool("update_project", mcp.WithReadOnlyHintAnnotation(false)),
		mcp.NewTool("delete_project", mcp.WithReadOnlyHintAnnotation(false)),
	}

	result := filterReadOnlyTools(tools)

	if len(result) != 3 {
		t.Errorf("Expected 3 read-only tools, got %d", len(result))
	}

	expectedTools := map[string]bool{
		"list_projects": false,
		"get_project":   false,
		"list_faults":   false,
	}

	for _, tool := range result {
		if _, exists := expectedTools[tool.Name]; !exists {
			t.Errorf("Unexpected tool in result: %s", tool.Name)
		}
		expectedTools[tool.Name] = true
	}

	// Verify all expected tools were found
	for toolName, found := range expectedTools {
		if !found {
			t.Errorf("Expected tool not found in result: %s", toolName)
		}
	}

	// Verify no non-readonly tools are included
	nonReadonlyTools := []string{"create_project", "update_project", "delete_project"}
	for _, nonReadonlyTool := range nonReadonlyTools {
		for _, resultTool := range result {
			if resultTool.Name == nonReadonlyTool {
				t.Errorf("Non-readonly tool should not be in result: %s", nonReadonlyTool)
			}
		}
	}
}
