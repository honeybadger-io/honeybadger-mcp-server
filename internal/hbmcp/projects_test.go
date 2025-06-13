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
		expectedPath := "/v2/projects?account_id=K7xmQqN"
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
		"account_id": "K7xmQqN",
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
		"id": 123,
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

func TestHandleGetProject_InvalidID(t *testing.T) {
	client := hbapi.NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"id": 0,
	}

	result, err := handleGetProject(context.Background(), client, args)
	if err != nil {
		t.Fatalf("handleGetProject() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for invalid ID")
	}

	if !strings.Contains(getResultText(result), "parameter 'id' must be positive") {
		t.Error("Error message should indicate invalid ID parameter")
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

		// Update API returns empty body on success
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"id":   123,
		"name": "Updated Project",
	}

	result, err := handleUpdateProject(context.Background(), client, args)
	if err != nil {
		t.Fatalf("handleUpdateProject() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that success message is present
	resultText := getResultText(result)
	if !strings.Contains(resultText, "successfully updated") {
		t.Error("Success message should be present in response")
	}

	if !strings.Contains(resultText, "123") {
		t.Error("Project ID should be present in success message")
	}
}

func TestHandleUpdateProject_MissingID(t *testing.T) {
	client := hbapi.NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"name": "Updated Project",
	}

	result, err := handleUpdateProject(context.Background(), client, args)
	if err != nil {
		t.Fatalf("handleUpdateProject() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}

	if !strings.Contains(getResultText(result), "required parameter 'id' is missing") {
		t.Errorf("expected missing id error, got %s", getResultText(result))
	}
}

func TestHandleUpdateProject_NoFieldsToUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123" {
			t.Errorf("expected path /v2/projects/123, got %s", r.URL.Path)
		}
		// Update API returns empty body on success
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	args := map[string]interface{}{
		"id": 123,
		// No fields to update - should still work (sends empty struct)
	}

	result, err := handleUpdateProject(context.Background(), client, args)
	if err != nil {
		t.Fatalf("handleUpdateProject() error = %v", err)
	}

	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	// Check that success message is present
	resultText := getResultText(result)
	if !strings.Contains(resultText, "successfully updated") {
		t.Error("Success message should be present in response")
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
		"id": 123,
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
		"id": 999,
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

func TestValidateIntParam(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		paramName string
		expected  int
		wantError bool
	}{
		{
			name:      "valid int parameter",
			args:      map[string]interface{}{"test": 123},
			paramName: "test",
			expected:  123,
			wantError: false,
		},
		{
			name:      "valid float64 parameter (whole number)",
			args:      map[string]interface{}{"test": 123.0},
			paramName: "test",
			expected:  123,
			wantError: false,
		},
		{
			name:      "missing parameter",
			args:      map[string]interface{}{},
			paramName: "test",
			wantError: true,
		},
		{
			name:      "zero parameter",
			args:      map[string]interface{}{"test": 0},
			paramName: "test",
			wantError: true,
		},
		{
			name:      "negative parameter",
			args:      map[string]interface{}{"test": -1},
			paramName: "test",
			wantError: true,
		},
		{
			name:      "float64 parameter (decimal)",
			args:      map[string]interface{}{"test": 123.5},
			paramName: "test",
			wantError: true,
		},
		{
			name:      "string parameter",
			args:      map[string]interface{}{"test": "123"},
			paramName: "test",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateIntParam(tt.args, tt.paramName)

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			if !tt.wantError && result != tt.expected {
				t.Errorf("expected result %d, got %d", tt.expected, result)
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

func TestArgsToProjectRequest(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]interface{}
		requireName bool
		expected    hbapi.ProjectRequest
		wantError   bool
	}{
		{
			name: "all fields valid",
			args: map[string]interface{}{
				"name":                     "Test Project",
				"resolve_errors_on_deploy": true,
				"disable_public_links":     false,
				"user_url":                 "https://example.com/users/[user_id]",
				"source_url":               "https://github.com/user/repo/blob/[sha]/[file]#L[line]",
				"purge_days":               30,
				"user_search_field":        "context.user_email",
			},
			requireName: true,
			expected: hbapi.ProjectRequest{
				Name:                  "Test Project",
				ResolveErrorsOnDeploy: func() *bool { b := true; return &b }(),
				DisablePublicLinks:    func() *bool { b := false; return &b }(),
				UserURL:               "https://example.com/users/[user_id]",
				SourceURL:             "https://github.com/user/repo/blob/[sha]/[file]#L[line]",
				PurgeDays:             func() *int { i := 30; return &i }(),
				UserSearchField:       "context.user_email",
			},
			wantError: false,
		},
		{
			name: "partial fields",
			args: map[string]interface{}{
				"name": "Partial Project",
			},
			requireName: false,
			expected: hbapi.ProjectRequest{
				Name: "Partial Project",
			},
			wantError: false,
		},
		{
			name: "float64 purge_days",
			args: map[string]interface{}{
				"purge_days": float64(90),
			},
			requireName: false,
			expected: hbapi.ProjectRequest{
				PurgeDays: func() *int { i := 90; return &i }(),
			},
			wantError: false,
		},
		{
			name:        "missing required name",
			args:        map[string]interface{}{},
			requireName: true,
			wantError:   true,
		},
		{
			name:        "missing name but not required",
			args:        map[string]interface{}{},
			requireName: false,
			expected:    hbapi.ProjectRequest{},
			wantError:   false,
		},
		{
			name: "invalid name type",
			args: map[string]interface{}{
				"name": 123,
			},
			requireName: true,
			wantError:   true,
		},
		{
			name: "invalid resolve_errors_on_deploy type",
			args: map[string]interface{}{
				"resolve_errors_on_deploy": "true",
			},
			requireName: false,
			wantError:   true,
		},
		{
			name: "invalid purge_days type",
			args: map[string]interface{}{
				"purge_days": "30",
			},
			requireName: false,
			wantError:   true,
		},
		{
			name: "invalid purge_days decimal",
			args: map[string]interface{}{
				"purge_days": 30.5,
			},
			requireName: false,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := argsToProjectRequest(tt.args, tt.requireName)

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
				return
			}

			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got %v", err)
				return
			}

			if tt.wantError {
				return // Don't check result if we expected an error
			}

			// Compare the results
			if result.Name != tt.expected.Name {
				t.Errorf("expected name %q, got %q", tt.expected.Name, result.Name)
			}

			if result.UserURL != tt.expected.UserURL {
				t.Errorf("expected user_url %q, got %q", tt.expected.UserURL, result.UserURL)
			}

			if result.SourceURL != tt.expected.SourceURL {
				t.Errorf("expected source_url %q, got %q", tt.expected.SourceURL, result.SourceURL)
			}

			if result.UserSearchField != tt.expected.UserSearchField {
				t.Errorf("expected user_search_field %q, got %q", tt.expected.UserSearchField, result.UserSearchField)
			}

			// Compare pointer fields
			if (result.ResolveErrorsOnDeploy == nil) != (tt.expected.ResolveErrorsOnDeploy == nil) {
				t.Errorf("resolve_errors_on_deploy pointer mismatch")
			} else if result.ResolveErrorsOnDeploy != nil && tt.expected.ResolveErrorsOnDeploy != nil {
				if *result.ResolveErrorsOnDeploy != *tt.expected.ResolveErrorsOnDeploy {
					t.Errorf("expected resolve_errors_on_deploy %v, got %v", *tt.expected.ResolveErrorsOnDeploy, *result.ResolveErrorsOnDeploy)
				}
			}

			if (result.DisablePublicLinks == nil) != (tt.expected.DisablePublicLinks == nil) {
				t.Errorf("disable_public_links pointer mismatch")
			} else if result.DisablePublicLinks != nil && tt.expected.DisablePublicLinks != nil {
				if *result.DisablePublicLinks != *tt.expected.DisablePublicLinks {
					t.Errorf("expected disable_public_links %v, got %v", *tt.expected.DisablePublicLinks, *result.DisablePublicLinks)
				}
			}

			if (result.PurgeDays == nil) != (tt.expected.PurgeDays == nil) {
				t.Errorf("purge_days pointer mismatch")
			} else if result.PurgeDays != nil && tt.expected.PurgeDays != nil {
				if *result.PurgeDays != *tt.expected.PurgeDays {
					t.Errorf("expected purge_days %v, got %v", *tt.expected.PurgeDays, *result.PurgeDays)
				}
			}
		})
	}
}
