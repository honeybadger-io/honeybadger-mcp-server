package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestHandleListProjects(t *testing.T) {
	mockResponse := `{
		"results": [
			{"id": 1, "name": "Project 1", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "secret123", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []},
			{"id": 2, "name": "Project 2", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "secret456", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 2, "email": "user2@example.com", "name": "User 2"}, "sites": [], "teams": [], "users": []}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects" {
			t.Errorf("expected path /v2/projects, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	result, err := handleListProjects(context.Background(), client, map[string]interface{}{})
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid API token"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("invalid-token")

	result, err := handleListProjects(context.Background(), client, map[string]interface{}{})
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
	mockResponse := `{
		"results": [
			{"id": 1, "name": "Project 1", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "secret123", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		expectedPath := "/v2/projects?account_id=12345"
		if r.URL.Path+"?"+r.URL.RawQuery != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path+"?"+r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	// Test with account_id parameter
	args := map[string]interface{}{
		"account_id": "12345",
	}

	result, err := handleListProjects(context.Background(), client, args)
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
	mockResponse := `{"id": 123, "name": "Test Project", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "secret123", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123" {
			t.Errorf("expected path /v2/projects/123, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"id": "123",
	}

	result, err := handleGetProject(context.Background(), client, args)
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
	client := hbapi.NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	args := map[string]interface{}{}

	result, err := handleGetProject(context.Background(), client, args)
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
	client := hbapi.NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"id": "",
	}

	result, err := handleGetProject(context.Background(), client, args)
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
	mockResponse := `{"id": 456, "name": "New Project", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "secret789", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects" {
			t.Errorf("expected path /v2/projects, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		project, ok := body["project"].(map[string]interface{})
		if !ok {
			t.Fatal("expected project object in request body")
		}

		if project["name"] != "New Project" {
			t.Errorf("expected project name 'New Project', got %v", project["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"name": "New Project",
	}

	result, err := handleCreateProject(context.Background(), client, args)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"error": "Name has already been taken"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"name": "Duplicate Name",
	}

	result, err := handleCreateProject(context.Background(), client, args)
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
	mockResponse := `{"id": 123, "name": "Updated Project", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "secret123", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123" {
			t.Errorf("expected path /v2/projects/123, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		project, ok := body["project"].(map[string]interface{})
		if !ok {
			t.Fatal("expected project object in request body")
		}

		if project["name"] != "Updated Project" {
			t.Errorf("expected project name 'Updated Project', got %v", project["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"id": "123",
		"updates": map[string]interface{}{
			"name": "Updated Project",
		},
	}

	result, err := handleUpdateProject(context.Background(), client, args)
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
	client := hbapi.NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"id": "123",
	}

	result, err := handleUpdateProject(context.Background(), client, args)
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
	client := hbapi.NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"id":      "123",
		"updates": map[string]interface{}{},
	}

	result, err := handleUpdateProject(context.Background(), client, args)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123" {
			t.Errorf("expected path /v2/projects/123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"id": "123",
	}

	result, err := handleDeleteProject(context.Background(), client, args)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Project not found"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"id": "nonexistent",
	}

	result, err := handleDeleteProject(context.Background(), client, args)
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

func TestSanitizeProject(t *testing.T) {
	project := &hbapi.Project{
		ID:        123,
		Name:      "Test Project",
		Active:    true,
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Token:     "secret123",
		Owner:     hbapi.Account{ID: 1, Email: "user@example.com", Name: "User 1"},
	}

	sanitizeProject(project)

	// Check that token field is removed
	if project.Token != "" {
		t.Error("token field should be cleared")
	}

	// Check that non-sensitive fields remain
	if project.ID != 123 {
		t.Error("project id should remain")
	}
	if project.Name != "Test Project" {
		t.Error("project name should remain")
	}
	if project.Owner.Email != "user@example.com" {
		t.Error("owner fields should remain")
	}
}
