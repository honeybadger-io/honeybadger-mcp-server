package hbmcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbapi"
	"github.com/mark3labs/mcp-go/mcp"
)

// Helper function to get text from MCP result
func getResultText(result *mcp.CallToolResult) string {
	if len(result.Content) > 0 {
		// Check if it's a TextContent type
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return textContent.Text
		}
		// If that doesn't work, try converting to string directly
		return fmt.Sprintf("%v", result.Content[0])
	}
	return ""
}

// Mock API client for testing
type mockAPIClient struct {
	listProjectsResult json.RawMessage
	listProjectsError  error
	getProjectResult   json.RawMessage
	getProjectError    error
	createProjectResult json.RawMessage
	createProjectError  error
	updateProjectResult json.RawMessage
	updateProjectError  error
	deleteProjectError  error
}

func (m *mockAPIClient) ListProjects(accountID string) (json.RawMessage, error) {
	return m.listProjectsResult, m.listProjectsError
}

func (m *mockAPIClient) GetProject(id string) (json.RawMessage, error) {
	return m.getProjectResult, m.getProjectError
}

func (m *mockAPIClient) CreateProject(name string) (json.RawMessage, error) {
	return m.createProjectResult, m.createProjectError
}

func (m *mockAPIClient) UpdateProject(id string, updates map[string]interface{}) (json.RawMessage, error) {
	return m.updateProjectResult, m.updateProjectError
}

func (m *mockAPIClient) DeleteProject(id string) error {
	return m.deleteProjectError
}

func TestHandleListProjects(t *testing.T) {
	mockProjects := json.RawMessage(`[
		{"id": "1", "name": "Project 1", "token": "secret123"},
		{"id": "2", "name": "Project 2", "token": "secret456"}
	]`)

	client := &mockAPIClient{
		listProjectsResult: mockProjects,
	}

	result, err := handleListProjects(client, map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleListProjects() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that tokens are sanitized
	resultText := getResultText(result)
	if strings.Contains(resultText, "secret123") {
		t.Error("Token should be sanitized from response")
	}

	// Check that project data is still present
	if !strings.Contains(resultText, "Project 1") {
		t.Error("Project name should be present in response")
	}
}

func TestHandleListProjects_Error(t *testing.T) {
	client := &mockAPIClient{
		listProjectsError: &hbapi.APIError{StatusCode: 401, Message: "Unauthorized"},
	}

	result, err := handleListProjects(client, map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleListProjects() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to list projects") {
		t.Error("Error message should contain 'Failed to list projects'")
	}
}

func TestHandleListProjects_WithAccountID(t *testing.T) {
	mockProjects := json.RawMessage(`[
		{"id": "1", "name": "Project 1", "token": "secret123"},
		{"id": "2", "name": "Project 2", "token": "secret456"}
	]`)

	client := &mockAPIClient{
		listProjectsResult: mockProjects,
	}

	// Test with account_id parameter
	args := map[string]interface{}{
		"account_id": "12345",
	}

	result, err := handleListProjects(client, args)
	if err != nil {
		t.Fatalf("handleListProjects() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that tokens are sanitized
	resultText := getResultText(result)
	if strings.Contains(resultText, "secret123") {
		t.Error("Token should be sanitized from response")
	}

	// Check that project data is still present
	if !strings.Contains(resultText, "Project 1") {
		t.Error("Project name should be present in response")
	}
}

func TestHandleGetProject(t *testing.T) {
	mockProject := json.RawMessage(`{"id": "123", "name": "Test Project", "token": "secret123"}`)

	client := &mockAPIClient{
		getProjectResult: mockProject,
	}

	args := map[string]interface{}{
		"id": "123",
	}

	result, err := handleGetProject(client, args)
	if err != nil {
		t.Fatalf("handleGetProject() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that token is sanitized
	if strings.Contains(getResultText(result), "secret123") {
		t.Error("Token should be sanitized from response")
	}

	// Check that project data is still present
	if !strings.Contains(getResultText(result), "Test Project") {
		t.Error("Project name should be present in response")
	}
}

func TestHandleGetProject_MissingID(t *testing.T) {
	client := &mockAPIClient{}

	args := map[string]interface{}{}

	result, err := handleGetProject(client, args)
	if err != nil {
		t.Fatalf("handleGetProject() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing ID")
	}

	if !strings.Contains(getResultText(result), "required parameter 'id' is missing") {
		t.Error("Error message should indicate missing ID parameter")
	}
}

func TestHandleGetProject_EmptyID(t *testing.T) {
	client := &mockAPIClient{}

	args := map[string]interface{}{
		"id": "",
	}

	result, err := handleGetProject(client, args)
	if err != nil {
		t.Fatalf("handleGetProject() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for empty ID")
	}

	if !strings.Contains(getResultText(result), "parameter 'id' cannot be empty") {
		t.Error("Error message should indicate empty ID parameter")
	}
}

func TestHandleCreateProject(t *testing.T) {
	mockProject := json.RawMessage(`{"id": "456", "name": "New Project", "token": "secret789"}`)

	client := &mockAPIClient{
		createProjectResult: mockProject,
	}

	args := map[string]interface{}{
		"name": "New Project",
	}

	result, err := handleCreateProject(client, args)
	if err != nil {
		t.Fatalf("handleCreateProject() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that API key is sanitized
	if strings.Contains(getResultText(result), "secret789") {
		t.Error("API key should be sanitized from response")
	}

	// Check that project data is still present
	if !strings.Contains(getResultText(result), "New Project") {
		t.Error("Project name should be present in response")
	}
}

func TestHandleCreateProject_ValidationError(t *testing.T) {
	client := &mockAPIClient{
		createProjectError: &hbapi.APIError{StatusCode: 422, Message: "Name has already been taken"},
	}

	args := map[string]interface{}{
		"name": "Duplicate Name",
	}

	result, err := handleCreateProject(client, args)
	if err != nil {
		t.Fatalf("handleCreateProject() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for validation error")
	}

	if !strings.Contains(getResultText(result), "Failed to create project") {
		t.Error("Error message should contain 'Failed to create project'")
	}
}

func TestHandleUpdateProject(t *testing.T) {
	mockProject := json.RawMessage(`{"id": "123", "name": "Updated Project", "token": "secret123"}`)

	client := &mockAPIClient{
		updateProjectResult: mockProject,
	}

	args := map[string]interface{}{
		"id": "123",
		"updates": map[string]interface{}{
			"name": "Updated Project",
		},
	}

	result, err := handleUpdateProject(client, args)
	if err != nil {
		t.Fatalf("handleUpdateProject() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that token is sanitized
	if strings.Contains(getResultText(result), "secret123") {
		t.Error("Token should be sanitized from response")
	}

	// Check that project data is still present
	if !strings.Contains(getResultText(result), "Updated Project") {
		t.Error("Updated project name should be present in response")
	}
}

func TestHandleUpdateProject_MissingUpdates(t *testing.T) {
	client := &mockAPIClient{}

	args := map[string]interface{}{
		"id": "123",
	}

	result, err := handleUpdateProject(client, args)
	if err != nil {
		t.Fatalf("handleUpdateProject() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing updates")
	}

	if !strings.Contains(getResultText(result), "required parameter 'updates' is missing") {
		t.Error("Error message should indicate missing updates parameter")
	}
}

func TestHandleUpdateProject_EmptyUpdates(t *testing.T) {
	client := &mockAPIClient{}

	args := map[string]interface{}{
		"id":      "123",
		"updates": map[string]interface{}{},
	}

	result, err := handleUpdateProject(client, args)
	if err != nil {
		t.Fatalf("handleUpdateProject() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for empty updates")
	}

	if !strings.Contains(getResultText(result), "parameter 'updates' cannot be empty") {
		t.Error("Error message should indicate empty updates parameter")
	}
}

func TestHandleDeleteProject(t *testing.T) {
	client := &mockAPIClient{}

	args := map[string]interface{}{
		"id": "123",
	}

	result, err := handleDeleteProject(client, args)
	if err != nil {
		t.Fatalf("handleDeleteProject() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check success message
	if !strings.Contains(getResultText(result), "deleted successfully") {
		t.Error("Success message should indicate project was deleted")
	}

	// Verify JSON structure
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(getResultText(result)), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("Response should include success: true")
	}
}

func TestHandleDeleteProject_Error(t *testing.T) {
	client := &mockAPIClient{
		deleteProjectError: &hbapi.APIError{StatusCode: 404, Message: "Project not found"},
	}

	args := map[string]interface{}{
		"id": "nonexistent",
	}

	result, err := handleDeleteProject(client, args)
	if err != nil {
		t.Fatalf("handleDeleteProject() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}

	if !strings.Contains(getResultText(result), "Failed to delete project") {
		t.Error("Error message should contain 'Failed to delete project'")
	}
}

func TestValidateStringParam(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		paramName string
		expected  string
		wantError bool
	}{
		{
			name:      "valid string parameter",
			args:      map[string]interface{}{"test": "value"},
			paramName: "test",
			expected:  "value",
			wantError: false,
		},
		{
			name:      "missing parameter",
			args:      map[string]interface{}{},
			paramName: "test",
			wantError: true,
		},
		{
			name:      "empty string parameter",
			args:      map[string]interface{}{"test": ""},
			paramName: "test",
			wantError: true,
		},
		{
			name:      "non-string parameter",
			args:      map[string]interface{}{"test": 123},
			paramName: "test",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateStringParam(tt.args, tt.paramName)

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			if !tt.wantError && result != tt.expected {
				t.Errorf("expected result '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestValidateObjectParam(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		paramName string
		wantError bool
	}{
		{
			name:      "valid object parameter",
			args:      map[string]interface{}{"test": map[string]interface{}{"key": "value"}},
			paramName: "test",
			wantError: false,
		},
		{
			name:      "missing parameter",
			args:      map[string]interface{}{},
			paramName: "test",
			wantError: true,
		},
		{
			name:      "empty object parameter",
			args:      map[string]interface{}{"test": map[string]interface{}{}},
			paramName: "test",
			wantError: true,
		},
		{
			name:      "non-object parameter",
			args:      map[string]interface{}{"test": "string"},
			paramName: "test",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateObjectParam(tt.args, tt.paramName)

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestSanitizeProjects(t *testing.T) {
	input := json.RawMessage(`[
		{"id": "1", "name": "Project 1", "token": "secret123"},
		{"id": "2", "name": "Project 2", "token": "secret456"}
	]`)

	result := sanitizeProjects(input)

	resultStr := string(result)

	// Check that token fields are removed
	if strings.Contains(resultStr, "secret123") {
		t.Error("token should be removed")
	}
	if strings.Contains(resultStr, "secret456") {
		t.Error("token should be removed")
	}

	// Check that non-sensitive fields remain
	if !strings.Contains(resultStr, "Project 1") {
		t.Error("project name should remain")
	}
	if !strings.Contains(resultStr, "Project 2") {
		t.Error("project name should remain")
	}
}

func TestSanitizeProject(t *testing.T) {
	input := json.RawMessage(`{"id": "123", "name": "Test Project", "token": "secret123"}`)

	result := sanitizeProject(input)

	resultStr := string(result)

	// Check that token field is removed
	if strings.Contains(resultStr, "secret123") {
		t.Error("token should be removed")
	}

	// Check that non-sensitive fields remain
	if !strings.Contains(resultStr, "Test Project") {
		t.Error("project name should remain")
	}
	if !strings.Contains(resultStr, "123") {
		t.Error("project id should remain")
	}
}

