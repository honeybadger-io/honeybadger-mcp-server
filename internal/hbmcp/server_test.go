package hbmcp

import (
	"testing"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		AuthToken: "test-token",
		APIURL:    "https://api.honeybadger.io/v2",
		LogLevel:  "info",
	}

	server := NewServer(cfg)
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

	server := NewServer(cfg)
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

	server := NewServer(cfg)
	if server == nil {
		t.Fatal("NewServer returned nil")
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
