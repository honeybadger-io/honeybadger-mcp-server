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
		expectedPath := "/v2/projects?account_id=K7xmQqN"
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

	projects, err := client.Projects.ListByAccountID(context.Background(), "K7xmQqN")
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

	project, err := client.Projects.Get(context.Background(), 123)
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

func TestGetProject_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Project not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Projects.Get(context.Background(), 999)
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

	req := ProjectRequest{
		Name: "New Project",
	}

	project, err := client.Projects.Create(context.Background(), req)
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

	req := ProjectRequest{
		Name: "",
	}

	_, err := client.Projects.Create(context.Background(), req)
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

	req := ProjectRequest{
		Name: "Duplicate Name",
	}

	_, err := client.Projects.Create(context.Background(), req)
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

	req := ProjectRequest{
		Name: "Updated Project",
	}

	project, err := client.Projects.Update(context.Background(), 123, req)
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	if project.Name != "Updated Project" {
		t.Errorf("expected project name 'Updated Project', got %v", project.Name)
	}
}

func TestCreateProject_WithAllFields(t *testing.T) {
	resolveErrorsOnDeploy := true
	disablePublicLinks := false
	purgeDays := 90

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		project, ok := body["project"].(map[string]interface{})
		if !ok {
			t.Fatal("expected project object in request body")
		}

		// Verify all fields are present
		if project["name"] != "Full Featured Project" {
			t.Errorf("expected name 'Full Featured Project', got %v", project["name"])
		}
		if project["resolve_errors_on_deploy"] != true {
			t.Errorf("expected resolve_errors_on_deploy true, got %v", project["resolve_errors_on_deploy"])
		}
		if project["disable_public_links"] != false {
			t.Errorf("expected disable_public_links false, got %v", project["disable_public_links"])
		}

		if project["user_url"] != "https://example.com/users/[user_id]" {
			t.Errorf("expected user_url, got %v", project["user_url"])
		}
		if project["source_url"] != "https://github.com/user/repo/blob/[sha]/[file]#L[line]" {
			t.Errorf("expected source_url, got %v", project["source_url"])
		}
		if project["purge_days"] != float64(90) {
			t.Errorf("expected purge_days 90, got %v", project["purge_days"])
		}
		if project["user_search_field"] != "context.user_email" {
			t.Errorf("expected user_search_field, got %v", project["user_search_field"])
		}

		mockProject := `{"id": 789, "name": "Full Featured Project", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "full123", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(mockProject))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := ProjectRequest{
		Name:                  "Full Featured Project",
		ResolveErrorsOnDeploy: &resolveErrorsOnDeploy,
		DisablePublicLinks:    &disablePublicLinks,
		UserURL:               "https://example.com/users/[user_id]",
		SourceURL:             "https://github.com/user/repo/blob/[sha]/[file]#L[line]",
		PurgeDays:             &purgeDays,
		UserSearchField:       "context.user_email",
	}

	project, err := client.Projects.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	if project.Name != "Full Featured Project" {
		t.Errorf("expected project name 'Full Featured Project', got %v", project.Name)
	}
}

func TestUpdateProject_WithAllFields(t *testing.T) {
	resolveErrorsOnDeploy := false
	disablePublicLinks := true
	purgeDays := 30

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

		// Verify all fields are present
		if project["name"] != "Updated Full Project" {
			t.Errorf("expected name 'Updated Full Project', got %v", project["name"])
		}
		if project["resolve_errors_on_deploy"] != false {
			t.Errorf("expected resolve_errors_on_deploy false, got %v", project["resolve_errors_on_deploy"])
		}
		if project["disable_public_links"] != true {
			t.Errorf("expected disable_public_links true, got %v", project["disable_public_links"])
		}

		if project["purge_days"] != float64(30) {
			t.Errorf("expected purge_days 30, got %v", project["purge_days"])
		}

		mockProject := `{"id": 123, "name": "Updated Full Project", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "abc123", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockProject))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := ProjectRequest{
		Name:                  "Updated Full Project",
		ResolveErrorsOnDeploy: &resolveErrorsOnDeploy,
		DisablePublicLinks:    &disablePublicLinks,
		UserURL:               "https://example.com/admin/users/[user_id]",
		SourceURL:             "https://gitlab.com/user/repo/-/blob/[sha]/[file]#L[line]",
		PurgeDays:             &purgeDays,
		UserSearchField:       "context.user_id",
	}

	project, err := client.Projects.Update(context.Background(), 123, req)
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	if project.Name != "Updated Full Project" {
		t.Errorf("expected project name 'Updated Full Project', got %v", project.Name)
	}
}

func TestCreateProject_PartialFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		project, ok := body["project"].(map[string]interface{})
		if !ok {
			t.Fatal("expected project object in request body")
		}

		// Verify only specified fields are present (omitempty should exclude others)
		if project["name"] != "Partial Project" {
			t.Errorf("expected name 'Partial Project', got %v", project["name"])
		}

		// These fields should not be present due to omitempty
		if _, exists := project["resolve_errors_on_deploy"]; exists {
			t.Error("resolve_errors_on_deploy should not be present")
		}
		if _, exists := project["disable_public_links"]; exists {
			t.Error("disable_public_links should not be present")
		}
		if _, exists := project["user_url"]; exists {
			t.Error("user_url should not be present")
		}

		mockProject := `{"id": 456, "name": "Partial Project", "active": true, "created_at": "2024-01-01T00:00:00Z", "token": "partial123", "fault_count": 0, "unresolved_fault_count": 0, "environments": [], "owner": {"id": 1, "email": "user@example.com", "name": "User 1"}, "sites": [], "teams": [], "users": []}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(mockProject))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := ProjectRequest{
		Name: "Partial Project",
		// Only name specified, other fields should be omitted
	}

	project, err := client.Projects.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	if project.Name != "Partial Project" {
		t.Errorf("expected project name 'Partial Project', got %v", project.Name)
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

	err := client.Projects.Delete(context.Background(), 123)
	if err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
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

	err := client.Projects.Delete(context.Background(), 999)
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
