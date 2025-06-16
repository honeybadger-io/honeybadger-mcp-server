package hbapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		_, _ = w.Write([]byte(mockProjects))
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
		_, _ = w.Write([]byte(`{"error": "Invalid API token"}`))
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
		_, _ = w.Write([]byte(mockProjects))
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
		_, _ = w.Write([]byte(mockProject))
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
		_, _ = w.Write([]byte(`{"error": "Project not found"}`))
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
		expectedPath := "/v2/projects?account_id=K7xmQqN"
		if r.URL.Path+"?"+r.URL.RawQuery != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path+"?"+r.URL.RawQuery)
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
		_, _ = w.Write([]byte(mockProject))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := ProjectRequest{
		Name: "New Project",
	}

	project, err := client.Projects.Create(context.Background(), "K7xmQqN", req)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	if project.Name != "New Project" {
		t.Errorf("expected project name 'New Project', got %v", project.Name)
	}
}

func TestCreateProject_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error": "Name has already been taken"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := ProjectRequest{
		Name: "Duplicate Name",
	}

	_, err := client.Projects.Create(context.Background(), "K7xmQqN", req)
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

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := ProjectRequest{
		Name: "Updated Project",
	}

	result, err := client.Projects.Update(context.Background(), 123, req)
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	if !result.Success {
		t.Errorf("expected success to be true, got %v", result.Success)
	}

	expectedMessage := "Project 123 was successfully updated"
	if result.Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, result.Message)
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
		expectedPath := "/v2/projects?account_id=K7xmQqN"
		if r.URL.Path+"?"+r.URL.RawQuery != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path+"?"+r.URL.RawQuery)
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
		_, _ = w.Write([]byte(mockProject))
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

	project, err := client.Projects.Create(context.Background(), "K7xmQqN", req)
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

		// Update API returns empty body on success
		w.WriteHeader(http.StatusOK)
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

	result, err := client.Projects.Update(context.Background(), 123, req)
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	if !result.Success {
		t.Errorf("expected success to be true, got %v", result.Success)
	}

	expectedMessage := "Project 123 was successfully updated"
	if result.Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, result.Message)
	}
}

func TestCreateProject_PartialFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v2/projects?account_id=K7xmQqN"
		if r.URL.Path+"?"+r.URL.RawQuery != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path+"?"+r.URL.RawQuery)
		}
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
		_, _ = w.Write([]byte(mockProject))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := ProjectRequest{
		Name: "Partial Project",
		// Only name specified, other fields should be omitted
	}

	project, err := client.Projects.Create(context.Background(), "K7xmQqN", req)
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

	result, err := client.Projects.Delete(context.Background(), 123)
	if err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}

	if !result.Success {
		t.Errorf("expected success to be true, got %v", result.Success)
	}

	expectedMessage := "Project 123 deleted successfully"
	if result.Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, result.Message)
	}
}

func TestDeleteProject_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Project not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	result, err := client.Projects.Delete(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result on error, got %v", result)
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 404 {
		t.Errorf("expected status code 404, got %d", apiErr.StatusCode)
	}
}

func TestGetAllOccurrenceCounts(t *testing.T) {
	mockResponse := `{
		"123": [
			[1510963200, 1440],
			[1511049600, 1441]
		],
		"456": [
			[1510963200, 500],
			[1511049600, 600]
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/occurrences" {
			t.Errorf("expected path /v2/projects/occurrences, got %s", r.URL.Path)
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
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	counts, err := client.Projects.GetAllOccurrenceCounts(context.Background(), ProjectGetOccurrenceCountsOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(counts) != 2 {
		t.Errorf("expected 2 projects, got %d", len(counts))
	}

	if project123, exists := counts["123"]; exists {
		if len(project123) != 2 {
			t.Errorf("expected 2 data points for project 123, got %d", len(project123))
		}
		if project123[0][0] != 1510963200 || project123[0][1] != 1440 {
			t.Errorf("unexpected first data point for project 123: %v", project123[0])
		}
	} else {
		t.Error("expected project 123 in response")
	}

	if project456, exists := counts["456"]; exists {
		if len(project456) != 2 {
			t.Errorf("expected 2 data points for project 456, got %d", len(project456))
		}
		if project456[0][0] != 1510963200 || project456[0][1] != 500 {
			t.Errorf("unexpected first data point for project 456: %v", project456[0])
		}
	} else {
		t.Error("expected project 456 in response")
	}
}

func TestGetAllOccurrenceCounts_WithOptions(t *testing.T) {
	mockResponse := `{
		"123": [
			[1510963200, 1440],
			[1511049600, 1441]
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/occurrences" {
			t.Errorf("expected path /v2/projects/occurrences, got %s", r.URL.Path)
		}

		query := r.URL.Query()
		if query.Get("period") != "day" {
			t.Errorf("expected period=day, got %s", query.Get("period"))
		}
		if query.Get("environment") != "production" {
			t.Errorf("expected environment=production, got %s", query.Get("environment"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	options := ProjectGetOccurrenceCountsOptions{
		Period:      "day",
		Environment: "production",
	}

	counts, err := client.Projects.GetAllOccurrenceCounts(context.Background(), options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(counts) != 1 {
		t.Errorf("expected 1 project, got %d", len(counts))
	}
}

func TestGetAllOccurrenceCounts_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "Invalid API token"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("invalid-token")

	_, err := client.Projects.GetAllOccurrenceCounts(context.Background(), ProjectGetOccurrenceCountsOptions{})
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

func TestGetOccurrenceCounts(t *testing.T) {
	mockResponse := `[
		[1510963200, 1440],
		[1511049600, 1441],
		[1511136000, 1442]
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/occurrences" {
			t.Errorf("expected path /v2/projects/123/occurrences, got %s", r.URL.Path)
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
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	counts, err := client.Projects.GetOccurrenceCounts(context.Background(), 123, ProjectGetOccurrenceCountsOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(counts) != 3 {
		t.Errorf("expected 3 data points, got %d", len(counts))
	}

	if counts[0][0] != 1510963200 || counts[0][1] != 1440 {
		t.Errorf("unexpected first data point: %v", counts[0])
	}

	if counts[1][0] != 1511049600 || counts[1][1] != 1441 {
		t.Errorf("unexpected second data point: %v", counts[1])
	}

	if counts[2][0] != 1511136000 || counts[2][1] != 1442 {
		t.Errorf("unexpected third data point: %v", counts[2])
	}
}

func TestGetOccurrenceCounts_WithOptions(t *testing.T) {
	mockResponse := `[
		[1510963200, 500],
		[1511049600, 600]
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/456/occurrences" {
			t.Errorf("expected path /v2/projects/456/occurrences, got %s", r.URL.Path)
		}

		query := r.URL.Query()
		if query.Get("period") != "week" {
			t.Errorf("expected period=week, got %s", query.Get("period"))
		}
		if query.Get("environment") != "staging" {
			t.Errorf("expected environment=staging, got %s", query.Get("environment"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	options := ProjectGetOccurrenceCountsOptions{
		Period:      "week",
		Environment: "staging",
	}

	counts, err := client.Projects.GetOccurrenceCounts(context.Background(), 456, options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(counts) != 2 {
		t.Errorf("expected 2 data points, got %d", len(counts))
	}

	if counts[0][0] != 1510963200 || counts[0][1] != 500 {
		t.Errorf("unexpected first data point: %v", counts[0])
	}
}

func TestGetOccurrenceCounts_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Project not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Projects.GetOccurrenceCounts(context.Background(), 999, ProjectGetOccurrenceCountsOptions{})
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

func TestGetOccurrenceCounts_OnlyPeriodOption(t *testing.T) {
	mockResponse := `[
		[1510963200, 100]
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/projects/123/occurrences" {
			t.Errorf("expected path /v2/projects/123/occurrences, got %s", r.URL.Path)
		}

		expectedQuery := "period=month"
		if r.URL.RawQuery != expectedQuery {
			t.Errorf("expected query %s, got %s", expectedQuery, r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	options := ProjectGetOccurrenceCountsOptions{
		Period: "month",
		// Environment intentionally left empty
	}

	counts, err := client.Projects.GetOccurrenceCounts(context.Background(), 123, options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(counts) != 1 {
		t.Errorf("expected 1 data point, got %d", len(counts))
	}
}

func TestGetOccurrenceCounts_OnlyEnvironmentOption(t *testing.T) {
	mockResponse := `[
		[1510963200, 200]
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/projects/789/occurrences" {
			t.Errorf("expected path /v2/projects/789/occurrences, got %s", r.URL.Path)
		}

		expectedQuery := "environment=development"
		if r.URL.RawQuery != expectedQuery {
			t.Errorf("expected query %s, got %s", expectedQuery, r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	options := ProjectGetOccurrenceCountsOptions{
		// Period intentionally left empty
		Environment: "development",
	}

	counts, err := client.Projects.GetOccurrenceCounts(context.Background(), 789, options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(counts) != 1 {
		t.Errorf("expected 1 data point, got %d", len(counts))
	}
}

func TestGetIntegrations(t *testing.T) {
	mockResponse := `[{"id": 9693, "active": false, "events": ["occurred"], "site_ids": [], "options": {"url": "http://test.com"}, "excluded_environments": [], "filters": [], "type": "WebHook"}]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/integrations" {
			t.Errorf("expected path /v2/projects/123/integrations, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	integrations, err := client.Projects.GetIntegrations(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(integrations) != 1 {
		t.Errorf("expected 1 integration, got %d", len(integrations))
	}
	if integrations[0].ID != 9693 {
		t.Errorf("expected integration ID 9693, got %d", integrations[0].ID)
	}
}

func TestGetIntegrations_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Project not found"}`))
	}))
	defer server.Close()

	client := NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	_, err := client.Projects.GetIntegrations(context.Background(), 999)
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

func TestGetReport_ProjectNoticesByClass(t *testing.T) {
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

	client := NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	report, err := client.Projects.GetReport(context.Background(), 123, ProjectNoticesByClass, ProjectGetReportOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report) != 2 {
		t.Errorf("expected 2 report entries, got %d", len(report))
	}
	if report[0][0] != "RuntimeError" || report[0][1] != float64(8347) {
		t.Errorf("unexpected first report entry: %v", report[0])
	}
}

func TestGetReport_ProjectNoticesByLocation(t *testing.T) {
	mockResponse := `[["inquiries#create", 2904], ["members#details", 862]]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/projects/456/reports/notices_by_location" {
			t.Errorf("expected path /v2/projects/456/reports/notices_by_location, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	report, err := client.Projects.GetReport(context.Background(), 456, ProjectNoticesByLocation, ProjectGetReportOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report) != 2 {
		t.Errorf("expected 2 report entries, got %d", len(report))
	}
}

func TestGetReport_WithOptions(t *testing.T) {
	mockResponse := `[["RuntimeError", 100]]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/projects/789/reports/notices_by_class" {
			t.Errorf("expected path /v2/projects/789/reports/notices_by_class, got %s", r.URL.Path)
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

	client := NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	options := ProjectGetReportOptions{
		Start:       "2023-01-01T00:00:00Z",
		Stop:        "2023-01-31T23:59:59Z",
		Environment: "production",
	}
	_, err := client.Projects.GetReport(context.Background(), 789, ProjectNoticesByClass, options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetReport_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Project not found"}`))
	}))
	defer server.Close()

	client := NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	_, err := client.Projects.GetReport(context.Background(), 999, ProjectNoticesByClass, ProjectGetReportOptions{})
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

func TestGetReport_ProjectNoticesPerDay(t *testing.T) {
	mockResponse := `[["2023-01-24T00:00:00.000000+00:00", 3161], ["2023-01-25T00:00:00.000000+00:00", 2620]]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/projects/100/reports/notices_per_day" {
			t.Errorf("expected path /v2/projects/100/reports/notices_per_day, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	report, err := client.Projects.GetReport(context.Background(), 100, ProjectNoticesPerDay, ProjectGetReportOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report) != 2 {
		t.Errorf("expected 2 report entries, got %d", len(report))
	}
}
