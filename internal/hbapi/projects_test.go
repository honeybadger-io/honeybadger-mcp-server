package hbapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListProjects(t *testing.T) {
	mockProjects := `[
		{"id": "1", "name": "Project 1", "api_key": "abc123"},
		{"id": "2", "name": "Project 2", "api_key": "def456"}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects" {
			t.Errorf("expected path /v2/projects, got %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test-token" {
			t.Errorf("expected X-API-Key test-token, got %s", r.Header.Get("X-API-Key"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockProjects))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}

	var projects []map[string]interface{}
	if err := json.Unmarshal(result, &projects); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}

	if projects[0]["name"] != "Project 1" {
		t.Errorf("expected first project name 'Project 1', got %v", projects[0]["name"])
	}
}

func TestListProjects_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid API token"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "invalid-token")
	_, err := client.ListProjects()
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

func TestGetProject(t *testing.T) {
	mockProject := `{"id": "123", "name": "Test Project", "api_key": "abc123"}`

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

	client := NewClient(server.URL, "test-token")
	result, err := client.GetProject("123")
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}

	var project map[string]interface{}
	if err := json.Unmarshal(result, &project); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if project["id"] != "123" {
		t.Errorf("expected project id '123', got %v", project["id"])
	}

	if project["name"] != "Test Project" {
		t.Errorf("expected project name 'Test Project', got %v", project["name"])
	}
}

func TestGetProject_EmptyID(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	_, err := client.GetProject("")
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

	client := NewClient(server.URL, "test-token")
	_, err := client.GetProject("nonexistent")
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
	mockProject := `{"id": "456", "name": "New Project", "api_key": "xyz789"}`

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

	client := NewClient(server.URL, "test-token")
	result, err := client.CreateProject("New Project")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	var project map[string]interface{}
	if err := json.Unmarshal(result, &project); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if project["name"] != "New Project" {
		t.Errorf("expected project name 'New Project', got %v", project["name"])
	}
}

func TestCreateProject_EmptyName(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	_, err := client.CreateProject("")
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

	client := NewClient(server.URL, "test-token")
	_, err := client.CreateProject("Duplicate Name")
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
	mockProject := `{"id": "123", "name": "Updated Project", "api_key": "abc123"}`

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

	client := NewClient(server.URL, "test-token")
	updates := map[string]interface{}{
		"name": "Updated Project",
	}

	result, err := client.UpdateProject("123", updates)
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	var project map[string]interface{}
	if err := json.Unmarshal(result, &project); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if project["name"] != "Updated Project" {
		t.Errorf("expected project name 'Updated Project', got %v", project["name"])
	}
}

func TestUpdateProject_EmptyID(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	updates := map[string]interface{}{"name": "Test"}
	_, err := client.UpdateProject("", updates)
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}

	if !strings.Contains(err.Error(), "project ID cannot be empty") {
		t.Errorf("expected empty ID error, got %s", err.Error())
	}
}

func TestUpdateProject_EmptyUpdates(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")

	tests := []struct {
		name    string
		updates map[string]interface{}
	}{
		{"nil updates", nil},
		{"empty updates", map[string]interface{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UpdateProject("123", tt.updates)
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

	client := NewClient(server.URL, "test-token")
	err := client.DeleteProject("123")
	if err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
}

func TestDeleteProject_EmptyID(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	err := client.DeleteProject("")
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

	client := NewClient(server.URL, "test-token")
	err := client.DeleteProject("nonexistent")
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