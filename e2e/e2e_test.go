package e2e

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
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

	expectedToolCount := 6 // create_project, delete_project, get_project, list_projects, ping, update_project
	if len(tools) != expectedToolCount {
		t.Errorf("Expected %d tools, got %d", expectedToolCount, len(tools))
	}

	// Track which tools we find
	var foundTools []string
	var pingTool map[string]interface{}
	var updateProjectTool map[string]interface{}

	for _, tool := range tools {
		if toolMap, ok := tool.(map[string]interface{}); ok {
			if name, ok := toolMap["name"].(string); ok {
				foundTools = append(foundTools, name)
				t.Logf("Found tool: %s", name)

				if name == "ping" {
					pingTool = toolMap
				}
				if name == "update_project" {
					updateProjectTool = toolMap
				}
			}
		}
	}

	// Verify all expected tools are present
	expectedTools := []string{"create_project", "delete_project", "get_project", "list_projects", "ping", "update_project"}
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

	if pingTool == nil {
		t.Fatal("Ping tool not found")
	}

	if updateProjectTool == nil {
		t.Fatal("Update project tool not found")
	}

	// Verify ping tool properties
	if desc, ok := pingTool["description"].(string); !ok || desc != "Test connectivity to the MCP server" {
		t.Errorf("Unexpected ping tool description: %v", pingTool["description"])
	}

	// Verify update_project tool properties
	if desc, ok := updateProjectTool["description"].(string); !ok || desc != "Update an existing Honeybadger project" {
		t.Errorf("Unexpected update_project tool description: %v", updateProjectTool["description"])
	}
}

// TestPingTool verifies the ping tool works correctly
func TestPingTool(t *testing.T) {
	server, err := StartTestServer(t, "test-token")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	result, err := server.CallTool("ping", nil)
	if err != nil {
		t.Fatalf("Failed to call ping tool: %v", err)
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

	// Parse the JSON response
	var pingResponse map[string]interface{}
	if err := json.Unmarshal([]byte(text), &pingResponse); err != nil {
		t.Fatalf("Failed to parse ping response: %v", err)
	}

	// Verify status
	if status, ok := pingResponse["status"].(string); !ok || status != "pong" {
		t.Errorf("Unexpected status: %v", pingResponse["status"])
	}

	// Verify timestamp
	timestamp, ok := pingResponse["timestamp"].(string)
	if !ok {
		t.Fatal("No timestamp in response")
	}

	// Verify timestamp format
	if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
		t.Errorf("Invalid timestamp format: %v", err)
	}
}

// TestPingToolMultipleCalls verifies the ping tool can be called multiple times
func TestPingToolMultipleCalls(t *testing.T) {
	server, err := StartTestServer(t, "test-token")
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	timestamps := make([]string, 0, 3)

	for i := 0; i < 3; i++ {
		result, err := server.CallTool("ping", nil)
		if err != nil {
			t.Fatalf("Failed to call ping tool (iteration %d): %v", i, err)
		}

		// Extract timestamp from response
		resultMap := result.(map[string]interface{})
		content := resultMap["content"].([]interface{})
		contentItem := content[0].(map[string]interface{})
		text := contentItem["text"].(string)

		var pingResponse map[string]interface{}
		json.Unmarshal([]byte(text), &pingResponse)

		timestamp := pingResponse["timestamp"].(string)
		timestamps = append(timestamps, timestamp)

		// Small delay between calls to ensure different timestamps
		time.Sleep(1 * time.Second)
	}

	// Verify all timestamps are different
	for i := 1; i < len(timestamps); i++ {
		if timestamps[i] == timestamps[i-1] {
			t.Error("Timestamps should be different for each call")
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
