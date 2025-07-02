package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handleListProjects(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListProjects() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that tokens are now included in response
	resultText := getResultText(result)
	if !strings.Contains(resultText, "secret123") {
		t.Error("Token should be included in response")
	}

	// Check that project data is still present
	if !strings.Contains(resultText, "Project 1") {
		t.Error("Project name should be present in response")
	}
}

func TestHandleListProjects_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors": "Invalid API token"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("invalid-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handleListProjects(context.Background(), client, req)
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
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	// Test with account_id parameter
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"account_id": "K7xmQqN",
			},
		},
	}

	result, err := handleListProjects(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListProjects() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that tokens are now included in response
	resultText := getResultText(result)
	if !strings.Contains(resultText, "secret123") {
		t.Error("Token should be included in response")
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
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"id": 123,
			},
		},
	}

	result, err := handleGetProject(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetProject() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that token is now included in response
	if !strings.Contains(getResultText(result), "secret123") {
		t.Error("Token should be included in response")
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

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handleGetProject(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetProject() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing ID")
	}

	if !strings.Contains(getResultText(result), "id is required") {
		t.Error("Error message should indicate missing ID parameter")
	}
}

func TestHandleGetProject_InvalidID(t *testing.T) {
	client := hbapi.NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"id": 0,
			},
		},
	}

	result, err := handleGetProject(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetProject() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for invalid ID")
	}

	if !strings.Contains(getResultText(result), "id is required") {
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
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"account_id": "K7xmQqN",
				"name":       "New Project",
			},
		},
	}

	result, err := handleCreateProject(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateProject() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that API key is now included in response
	if !strings.Contains(getResultText(result), "secret789") {
		t.Error("API key should be included in response")
	}

	// Check that project data is still present
	if !strings.Contains(getResultText(result), "New Project") {
		t.Error("Project name should be present in response")
	}
}

func TestHandleCreateProject_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v2/projects?account_id=K7xmQqN"
		if r.URL.Path+"?"+r.URL.RawQuery != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path+"?"+r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors": "Name has already been taken"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"account_id": "K7xmQqN",
				"name":       "Duplicate Name",
			},
		},
	}

	result, err := handleCreateProject(context.Background(), client, req)
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

func TestHandleCreateProject_MissingAccountID(t *testing.T) {
	client := hbapi.NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"name": "Test Project",
			},
		},
	}

	result, err := handleCreateProject(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateProject() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing account_id")
	}

	if !strings.Contains(getResultText(result), "account_id is required") {
		t.Error("Error message should indicate missing account_id parameter")
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

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"id":   123,
				"name": "Updated Project",
			},
		},
	}

	result, err := handleUpdateProject(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleUpdateProject() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that success message is present
	resultText := getResultText(result)
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(resultText), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if message, ok := response["message"].(string); !ok || !strings.Contains(message, "successfully updated") {
		t.Error("Success message should be present in response")
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("Response should include success: true")
	}
}

func TestHandleUpdateProject_MissingID(t *testing.T) {
	client := hbapi.NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"name": "Updated Project",
			},
		},
	}

	result, err := handleUpdateProject(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleUpdateProject() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}

	if !strings.Contains(getResultText(result), "id is required") {
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

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"id": 123,
				// No fields to update - should still work (sends empty struct)
			},
		},
	}

	result, err := handleUpdateProject(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleUpdateProject() error = %v", err)
	}

	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	// Check that success message is present
	resultText := getResultText(result)
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(resultText), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if message, ok := response["message"].(string); !ok || !strings.Contains(message, "successfully updated") {
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

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"id": 123,
			},
		},
	}

	result, err := handleDeleteProject(context.Background(), client, req)
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
		_, _ = w.Write([]byte(`{"errors": "Project not found"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"id": 999,
			},
		},
	}

	result, err := handleDeleteProject(context.Background(), client, req)
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


func TestHandleGetProjectReport(t *testing.T) {
	mockResponse := `[["RuntimeError", 8347], ["SocketError", 4651]]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/reports/notices_by_class" {
			t.Errorf("expected path /v2/projects/123/reports/notices_by_class, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": float64(123),
				"report":     "notices_by_class",
			},
		},
	}

	result, err := handleGetProjectReport(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetProjectReport() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}
	resultText := getResultText(result)
	if !strings.Contains(resultText, "RuntimeError") {
		t.Error("Result should contain RuntimeError")
	}
}

func TestHandleGetProjectReport_InvalidReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors": "Invalid report type"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": float64(123),
				"report":     "invalid_report_type",
			},
		},
	}

	result, err := handleGetProjectReport(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetProjectReport() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid report type")
	}
	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to get project report") {
		t.Error("Error message should contain 'Failed to get project report'")
	}
}

func TestHandleGetProjectReport_WithOptions(t *testing.T) {
	mockResponse := `[["inquiries#create", 2904]]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/projects/456/reports/notices_by_location" {
			t.Errorf("expected path /v2/projects/456/reports/notices_by_location, got %s", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("start") != "2023-01-01T00:00:00Z" {
			t.Errorf("expected start=2023-01-01T00:00:00Z, got %s", query.Get("start"))
		}
		if query.Get("stop") != "2023-01-31T23:59:59Z" {
			t.Errorf("expected stop=2023-01-31T23:59:59Z, got %s", query.Get("stop"))
		}
		if query.Get("environment") != "production" {
			t.Errorf("expected environment=production, got %s", query.Get("environment"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id":  float64(456),
				"report":      "notices_by_location",
				"start":       "2023-01-01T00:00:00Z",
				"stop":        "2023-01-31T23:59:59Z",
				"environment": "production",
			},
		},
	}

	result, err := handleGetProjectReport(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetProjectReport() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}
}
