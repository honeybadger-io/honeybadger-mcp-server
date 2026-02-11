package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestServerStartup verifies the server starts correctly
func TestServerStartup(t *testing.T) {
	server, err := StartTestServer(t, "test-token")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Server should be running
	if server.cmd.Process == nil {
		t.Fatal("Server process is nil")
	}
}

// TestListTools verifies the tools/list method works correctly in non-read-only mode
func TestListTools(t *testing.T) {
	server, err := StartTestServer(t, "test-token")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	tools, err := server.ListTools()
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	expectedToolCount := 26 // create_alarm, create_dashboard, create_project, delete_alarm, delete_dashboard, delete_project, get_alarm, get_alarm_history, get_dashboard, get_fault, get_fault_counts, get_insights_reference, get_project, get_project_integrations, get_project_occurrence_counts, get_project_report, list_alarms, list_dashboards, list_fault_affected_users, list_fault_notices, list_faults, list_projects, query_insights, update_alarm, update_dashboard, update_project
	if len(tools) != expectedToolCount {
		t.Errorf("Expected %d tools, got %d", expectedToolCount, len(tools))
	}

	// Track which tools we find
	var foundTools []string
	var updateProjectTool map[string]interface{}

	for _, tool := range tools {
		if toolMap, ok := tool.(map[string]interface{}); ok {
			if name, ok := toolMap["name"].(string); ok {
				foundTools = append(foundTools, name)
				t.Logf("Found tool: %s", name)

				if name == "update_project" {
					updateProjectTool = toolMap
				}
			}
		}
	}

	// Verify all expected tools are present
	expectedTools := []string{"create_alarm", "create_dashboard", "create_project", "delete_alarm", "delete_dashboard", "delete_project", "get_alarm", "get_alarm_history", "get_dashboard", "get_fault", "get_fault_counts", "get_insights_reference", "get_project", "get_project_integrations", "get_project_occurrence_counts", "get_project_report", "list_alarms", "list_dashboards", "list_fault_affected_users", "list_fault_notices", "list_faults", "list_projects", "query_insights", "update_alarm", "update_dashboard", "update_project"}
	for _, expectedTool := range expectedTools {
		found := false
		for _, foundTool := range foundTools {
			if foundTool == expectedTool {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool %s not found. Found tools: %v", expectedTool, foundTools)
		}
	}

	if updateProjectTool == nil {
		t.Fatal("Update project tool not found")
	}

	// Verify update_project tool properties
	if desc, ok := updateProjectTool["description"].(string); !ok || desc != "Update an existing Honeybadger project" {
		t.Errorf("Unexpected update_project tool description: %v", updateProjectTool["description"])
	}
}

// TestListToolsReadOnly verifies the tools/list method works correctly in read-only mode
func TestListToolsReadOnly(t *testing.T) {
	server, err := StartTestServerWithReadOnly(t, "test-token", true)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	tools, err := server.ListTools()
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	expectedToolCount := 17 // get_alarm, get_alarm_history, get_dashboard, get_fault, get_fault_counts, get_insights_reference, get_project, get_project_integrations, get_project_occurrence_counts, get_project_report, list_alarms, list_dashboards, list_fault_affected_users, list_fault_notices, list_faults, list_projects, query_insights
	if len(tools) != expectedToolCount {
		t.Errorf("Expected %d tools in read-only mode, got %d", expectedToolCount, len(tools))
	}

	// Track which tools we find
	var foundTools []string

	for _, tool := range tools {
		if toolMap, ok := tool.(map[string]interface{}); ok {
			if name, ok := toolMap["name"].(string); ok {
				foundTools = append(foundTools, name)
				t.Logf("Found tool: %s", name)
			}
		}
	}

	// Verify only read-only tools are present
	expectedReadOnlyTools := []string{"get_alarm", "get_alarm_history", "get_dashboard", "get_fault", "get_fault_counts", "get_insights_reference", "get_project", "get_project_integrations", "get_project_occurrence_counts", "get_project_report", "list_alarms", "list_dashboards", "list_fault_affected_users", "list_fault_notices", "list_faults", "list_projects", "query_insights"}
	for _, expectedTool := range expectedReadOnlyTools {
		found := false
		for _, foundTool := range foundTools {
			if foundTool == expectedTool {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected read-only tool %s not found. Found tools: %v", expectedTool, foundTools)
		}
	}

	// Verify destructive tools are NOT present
	destructiveTools := []string{"create_alarm", "create_dashboard", "create_project", "delete_alarm", "delete_dashboard", "delete_project", "update_alarm", "update_dashboard", "update_project"}
	for _, destructiveTool := range destructiveTools {
		for _, foundTool := range foundTools {
			if foundTool == destructiveTool {
				t.Errorf("Destructive tool %s should not be present in read-only mode", destructiveTool)
			}
		}
	}
}

// TestInvalidToolCall verifies error handling for invalid tool calls
func TestInvalidToolCall(t *testing.T) {
	server, err := StartTestServer(t, "test-token")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Call non-existent tool
	_, err = server.CallTool("nonexistent", nil)
	if err == nil {
		t.Fatal("Expected error for non-existent tool")
	}

	if !strings.Contains(err.Error(), "tool error") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestMalformedMessage verifies the server handles malformed messages gracefully
func TestMalformedMessage(t *testing.T) {
	server, err := StartTestServer(t, "test-token")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Send a message with missing required fields
	msg := MCPMessage{
		JSONRPC: "2.0",
		Method:  "tools/call",
		ID:      999,
		// Missing params
	}

	resp, err := server.SendMessage(msg)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("Expected error response for malformed message")
	}
}

// TestProjectToolIntegration verifies project tools work correctly
func TestProjectToolIntegration(t *testing.T) {
	server, err := StartTestServer(t, "test-token")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Test update_project tool with valid parameters
	args := map[string]interface{}{
		"id":   123,
		"name": "Updated Test Project",
	}

	result, err := server.CallTool("update_project", args)
	if err != nil {
		t.Fatalf("Failed to call update_project tool: %v", err)
	}

	// Verify result structure is correct (basic validation)
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Unexpected result type: %T", result)
	}

	// Check for content array (MCP standard response format)
	content, ok := resultMap["content"].([]interface{})
	if !ok {
		t.Fatalf("No content in result: %+v", resultMap)
	}

	if len(content) == 0 {
		t.Fatal("Empty content array")
	}

	// Verify basic response structure
	contentItem, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Unexpected content item type: %T", content[0])
	}

	if contentType, ok := contentItem["type"].(string); !ok || contentType != "text" {
		t.Errorf("Unexpected content type: %v", contentItem["type"])
	}

	// For e2e testing, we just verify we get a valid response structure
	// The actual API behavior is tested in unit tests
	if _, ok := contentItem["text"].(string); !ok {
		t.Fatal("No text content in response")
	}
}

// TestFaultToolIntegration verifies fault tools work correctly
func TestFaultToolIntegration(t *testing.T) {
	server, err := StartTestServer(t, "test-token")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Test list_faults tool with valid parameters
	args := map[string]interface{}{
		"project_id": 123,
		"limit":      5,
	}

	result, err := server.CallTool("list_faults", args)
	if err != nil {
		t.Fatalf("Failed to call list_faults tool: %v", err)
	}

	// Verify result structure is correct (basic validation)
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Unexpected result type: %T", result)
	}

	// Check for content array (MCP standard response format)
	content, ok := resultMap["content"].([]interface{})
	if !ok {
		t.Fatalf("No content in result: %+v", resultMap)
	}

	if len(content) == 0 {
		t.Fatal("Empty content array")
	}

	// Verify basic response structure
	contentItem, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Unexpected content item type: %T", content[0])
	}

	if contentType, ok := contentItem["type"].(string); !ok || contentType != "text" {
		t.Errorf("Unexpected content type: %v", contentItem["type"])
	}

	// For e2e testing, we just verify we get a valid response structure
	if _, ok := contentItem["text"].(string); !ok {
		t.Fatal("No text content in response")
	}
}

// TestInsightsToolIntegration verifies insights tools work correctly
func TestInsightsToolIntegration(t *testing.T) {
	server, err := StartTestServer(t, "test-token")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Test query_insights tool with valid parameters
	args := map[string]interface{}{
		"project_id": 123,
		"query":      "stats count()",
		"ts":         "today",
	}

	result, err := server.CallTool("query_insights", args)
	if err != nil {
		t.Fatalf("Failed to call query_insights tool: %v", err)
	}

	// Verify result structure is correct (basic validation)
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Unexpected result type: %T", result)
	}

	// Check for content array (MCP standard response format)
	content, ok := resultMap["content"].([]interface{})
	if !ok {
		t.Fatalf("No content in result: %+v", resultMap)
	}

	if len(content) == 0 {
		t.Fatal("Empty content array")
	}

	// Verify basic response structure
	contentItem, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Unexpected content item type: %T", content[0])
	}

	if contentType, ok := contentItem["type"].(string); !ok || contentType != "text" {
		t.Errorf("Unexpected content type: %v", contentItem["type"])
	}

	// For e2e testing, we just verify we get a valid response structure
	if _, ok := contentItem["text"].(string); !ok {
		t.Fatal("No text content in response")
	}
}

// TestServerEdgeCases verifies the server handles edge cases correctly
func TestServerEdgeCases(t *testing.T) {
	t.Run("ServerWithoutToken", func(t *testing.T) {
		// Try to start server without token by overriding the command
		cmd := exec.Command("go", "run", "../cmd/honeybadger-mcp-server/main.go", "stdio")

		// Clear only the auth token from environment while preserving other vars
		env := os.Environ()
		filteredEnv := make([]string, 0, len(env))
		for _, e := range env {
			if !strings.HasPrefix(e, "HONEYBADGER_PERSONAL_AUTH_TOKEN=") {
				filteredEnv = append(filteredEnv, e)
			}
		}
		cmd.Env = filteredEnv

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("Expected server to fail without API token")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "auth-token is required") {
			t.Errorf("Expected error about missing auth-token, got: %s", outputStr)
		}
	})

	t.Run("ServerShutdownGracefully", func(t *testing.T) {
		server, err := StartTestServer(t, "test-token")
		if err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}

		// Verify server is running
		if server.cmd.Process == nil {
			t.Fatal("Server process is nil")
		}

		// Stop server and verify it shuts down without hanging
		server.Stop()

		// Process should be stopped
		if server.cmd.ProcessState == nil {
			t.Error("Process state should be set after stopping")
		}
	})

	t.Run("InvalidToolArguments", func(t *testing.T) {
		server, err := StartTestServer(t, "test-token")
		if err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}
		defer server.Stop()

		// Call valid tool with invalid arguments
		args := map[string]interface{}{
			"invalid_arg": "invalid_value",
		}

		_, err = server.CallTool("list_projects", args)
		// Should not crash, may succeed or return validation error
		// Either outcome is acceptable for e2e testing
		if err != nil {
			t.Logf("Got expected validation error: %v", err)
		}
	})
}
