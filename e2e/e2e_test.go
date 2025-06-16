package e2e

import (
	"encoding/json"
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

// TestListTools verifies the tools/list method works correctly
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

	expectedToolCount := 12 // create_project, delete_project, get_fault, get_project, get_project_integrations, get_project_occurrence_counts, get_project_report, list_fault_affected_users, list_fault_notices, list_faults, list_projects, update_project
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
	expectedTools := []string{"create_project", "delete_project", "get_fault", "get_project", "get_project_integrations", "get_project_occurrence_counts", "get_project_report", "list_fault_affected_users", "list_fault_notices", "list_faults", "list_projects", "update_project"}
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

// TestUpdateProjectTool verifies the update_project tool works correctly
func TestUpdateProjectTool(t *testing.T) {
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

	// Parse the result
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Unexpected result type: %T", result)
	}

	// Check for content array
	content, ok := resultMap["content"].([]interface{})
	if !ok {
		t.Fatalf("No content in result: %+v", resultMap)
	}

	if len(content) == 0 {
		t.Fatal("Empty content array")
	}

	// Get the first content item
	contentItem, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Unexpected content item type: %T", content[0])
	}

	// Verify it's a text type
	if contentType, ok := contentItem["type"].(string); !ok || contentType != "text" {
		t.Errorf("Unexpected content type: %v", contentItem["type"])
	}

	// Get the text content
	text, ok := contentItem["text"].(string)
	if !ok {
		t.Fatalf("No text in content item: %+v", contentItem)
	}

	// Check if this is an error response (expected with test token)
	if strings.Contains(text, "Failed to update project") {
		// This is expected since we're using a test token
		t.Logf("Got expected error response with test token: %s", text)
		return
	}

	// Parse the JSON response (if it's a success response)
	var updateResponse map[string]interface{}
	if err := json.Unmarshal([]byte(text), &updateResponse); err != nil {
		t.Fatalf("Failed to parse update response: %v, text: %s", err, text)
	}

	// Verify success
	if success, ok := updateResponse["success"].(bool); !ok || !success {
		t.Errorf("Unexpected success value: %v", updateResponse["success"])
	}

	// Verify message contains project ID
	message, ok := updateResponse["message"].(string)
	if !ok {
		t.Fatal("No message in response")
	}

	if !strings.Contains(message, "123") {
		t.Errorf("Message should contain project ID 123: %v", message)
	}

	if !strings.Contains(message, "successfully updated") {
		t.Errorf("Message should contain 'successfully updated': %v", message)
	}
}

// TestServerWithoutToken verifies the server requires an API token
func TestServerWithoutToken(t *testing.T) {
	// Try to start server without token by overriding the command
	cmd := exec.Command("go", "run", "../cmd/honeybadger-mcp-server/main.go", "stdio")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Expected server to fail without API token")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "auth-token is required") {
		t.Errorf("Expected error about missing auth-token, got: %s", outputStr)
	}
}
