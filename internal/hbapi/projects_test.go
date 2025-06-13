package hbapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListProjects(t *testing.T) {
	mockProjects := `{
		"results": [
			{"id": 1, "name": "Project 1", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "abc123", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []},
			{"id": 2, "name": "Project 2", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "def456", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 2, "email": "user2@example.com", "name": "User 2"}, "sites": [], "teams": [], "users": []}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects" {
			t.Errorf("expected path /v2/projects, got %s", r.URL.Path)
		}
		// Check Basic Auth
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic Auth to be set")
		}
		if username != "test-token" {
			t.Errorf("expected Basic Auth username test-token, got %s", username)
		}
		if password != "" {
			t.Errorf("expected Basic Auth password to be empty, got %s", password)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockProjects))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	projects, err := client.Projects.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}

	if projects[0].Name != "Project 1" {
		t.Errorf("expected first project name 'Project 1', got %v", projects[0].Name)
	}
}

func TestListProjects_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid API token"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("invalid-token")

	_, err := client.Projects.ListAll(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 401 {
		t.Errorf("expected status code 401, got %d", apiErr.StatusCode)
	}
}

func TestListProjects_WithAccountID(t *testing.T) {
	mockProjects := `{
		"results": [
			{"id": 1, "name": "Project 1", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "abc123", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []},
			{"id": 2, "name": "Project 2", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "def456", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 2, "email": "user2@example.com", "name": "User 2"}, "sites": [], "teams": [], "users": []}
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
		// Check Basic Auth
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic Auth to be set")
		}
		if username != "test-token" {
			t.Errorf("expected Basic Auth username test-token, got %s", username)
		}
		if password != "" {
			t.Errorf("expected Basic Auth password to be empty, got %s", password)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockProjects))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	projects, err := client.Projects.ListByAccountID(context.Background(), 12345)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}

	if projects[0].Name != "Project 1" {
		t.Errorf("expected first project name 'Project 1', got %v", projects[0].Name)
	}
}

func TestGetProject(t *testing.T) {
	mockProject := `{"id": 123, "name": "Test Project", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "abc123", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123" {
			t.Errorf("expected path /v2/projects/123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockProject))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	project, err := client.Projects.Get(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}

	if project.ID != 123 {
		t.Errorf("expected project id 123, got %v", project.ID)
	}

	if project.Name != "Test Project" {
		t.Errorf("expected project name 'Test Project', got %v", project.Name)
	}
}

func TestGetProject_EmptyID(t *testing.T) {
	client := NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	_, err := client.Projects.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}

	if !strings.Contains(err.Error(), "project ID cannot be empty") {
		t.Errorf("expected empty ID error, got %s", err.Error())
	}
}

func TestGetProject_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Project not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Projects.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 404 {
		t.Errorf("expected status code 404, got %d", apiErr.StatusCode)
	}
}

func TestCreateProject(t *testing.T) {
	mockProject := `{"id": 456, "name": "New Project", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "xyz789", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []}`

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
		w.Write([]byte(mockProject))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	project, err := client.Projects.Create(context.Background(), "New Project")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	if project.Name != "New Project" {
		t.Errorf("expected project name 'New Project', got %v", project.Name)
	}
}

func TestCreateProject_EmptyName(t *testing.T) {
	client := NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	_, err := client.Projects.Create(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}

	if !strings.Contains(err.Error(), "project name cannot be empty") {
		t.Errorf("expected empty name error, got %s", err.Error())
	}
}

func TestCreateProject_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"error": "Name has already been taken"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Projects.Create(context.Background(), "Duplicate Name")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 422 {
		t.Errorf("expected status code 422, got %d", apiErr.StatusCode)
	}
}

func TestUpdateProject(t *testing.T) {
	mockProject := `{"id": 123, "name": "Updated Project", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "abc123", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []}`

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
		w.Write([]byte(mockProject))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	updates := map[string]interface{}{
		"name": "Updated Project",
	}

	project, err := client.Projects.Update(context.Background(), "123", updates)
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	if project.Name != "Updated Project" {
		t.Errorf("expected project name 'Updated Project', got %v", project.Name)
	}
}

func TestUpdateProject_EmptyID(t *testing.T) {
	client := NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	updates := map[string]interface{}{"name": "Test"}
	_, err := client.Projects.Update(context.Background(), "", updates)
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}

	if !strings.Contains(err.Error(), "project ID cannot be empty") {
		t.Errorf("expected empty ID error, got %s", err.Error())
	}
}

func TestUpdateProject_EmptyUpdates(t *testing.T) {
	client := NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	tests := []struct {
		name    string
		updates map[string]interface{}
	}{
		{"nil updates", nil},
		{"empty updates", map[string]interface{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Projects.Update(context.Background(), "123", tt.updates)
			if err == nil {
				t.Fatal("expected error for empty updates, got nil")
			}

			if !strings.Contains(err.Error(), "updates cannot be empty") {
				t.Errorf("expected empty updates error, got %s", err.Error())
			}
		})
	}
}

func TestDeleteProject(t *testing.T) {
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

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	err := client.Projects.Delete(context.Background(), "123")
	if err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
}

func TestDeleteProject_EmptyID(t *testing.T) {
	client := NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	err := client.Projects.Delete(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}

	if !strings.Contains(err.Error(), "project ID cannot be empty") {
		t.Errorf("expected empty ID error, got %s", err.Error())
	}
}

func TestDeleteProject_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Project not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	err := client.Projects.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 404 {
		t.Errorf("expected status code 404, got %d", apiErr.StatusCode)
	}
}
