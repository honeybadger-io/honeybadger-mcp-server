package hbapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFaultsList(t *testing.T) {
	mockFaults := `{
		"results": [
			{
				"id": 1,
				"action": "index",
				"assignee": null,
				"comments_count": 0,
				"component": "HomeController",
				"created_at": "2024-01-01T00:00:00Z",
				"environment": "production",
				"ignored": false,
				"klass": "NoMethodError",
				"last_notice_at": "2024-01-02T00:00:00Z",
				"message": "undefined method 'foo' for nil:NilClass",
				"notices_count": 10,
				"project_id": 123,
				"resolved": false,
				"tags": ["urgent", "production"],
				"url": "https://app.honeybadger.io/projects/123/faults/1"
			},
			{
				"id": 2,
				"action": "show",
				"assignee": {"id": 1, "email": "user@example.com", "name": "User 1"},
				"comments_count": 2,
				"component": "UserController",
				"created_at": "2024-01-01T00:00:00Z",
				"environment": "staging",
				"ignored": false,
				"klass": "ArgumentError",
				"last_notice_at": "2024-01-03T00:00:00Z",
				"message": "wrong number of arguments",
				"notices_count": 5,
				"project_id": 123,
				"resolved": true,
				"tags": [],
				"url": "https://app.honeybadger.io/projects/123/faults/2"
			}
		],
		"links": {
			"self": "https://app.honeybadger.io/v2/projects/123/faults"
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/faults" {
			t.Errorf("expected path /v2/projects/123/faults, got %s", r.URL.Path)
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
		_, _ = w.Write([]byte(mockFaults))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	response, err := client.Faults.List(context.Background(), 123, FaultListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(response.Results) != 2 {
		t.Errorf("expected 2 faults, got %d", len(response.Results))
	}

	if response.Results[0].ID != 1 {
		t.Errorf("expected first fault ID 1, got %d", response.Results[0].ID)
	}

	if response.Results[0].Message != "undefined method 'foo' for nil:NilClass" {
		t.Errorf("expected first fault message 'undefined method 'foo' for nil:NilClass', got %s", response.Results[0].Message)
	}

	if response.Results[1].Resolved != true {
		t.Errorf("expected second fault to be resolved, got %v", response.Results[1].Resolved)
	}

	// Verify that notices_count_in_range is nil when not provided
	if response.Results[0].NoticesCountInRange != nil {
		t.Errorf("expected notices_count_in_range to be nil when not provided, got %v", *response.Results[0].NoticesCountInRange)
	}
}

func TestFaultsList_WithNoticesCountInRange(t *testing.T) {
	mockFaults := `{
		"results": [
			{
				"id": 1,
				"action": "index",
				"assignee": null,
				"comments_count": 0,
				"component": "HomeController",
				"created_at": "2024-01-01T00:00:00Z",
				"environment": "production",
				"ignored": false,
				"klass": "NoMethodError",
				"last_notice_at": "2024-01-02T00:00:00Z",
				"message": "undefined method 'foo' for nil:NilClass",
				"notices_count": 100,
				"notices_count_in_range": 15,
				"project_id": 123,
				"resolved": false,
				"tags": ["urgent", "production"],
				"url": "https://app.honeybadger.io/projects/123/faults/1"
			}
		],
		"links": {}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockFaults))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	response, err := client.Faults.List(context.Background(), 123, FaultListOptions{
		Q: "search query", // Simulate a search that would trigger notices_count_in_range
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(response.Results) != 1 {
		t.Errorf("expected 1 fault, got %d", len(response.Results))
	}

	fault := response.Results[0]

	// Verify normal notices_count
	if fault.NoticesCount != 100 {
		t.Errorf("expected notices_count to be 100, got %d", fault.NoticesCount)
	}

	// Verify notices_count_in_range is properly parsed
	if fault.NoticesCountInRange == nil {
		t.Errorf("expected notices_count_in_range to be present, got nil")
	} else if *fault.NoticesCountInRange != 15 {
		t.Errorf("expected notices_count_in_range to be 15, got %d", *fault.NoticesCountInRange)
	}
}

func TestFaultsList_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("q") != "NoMethodError" {
			t.Errorf("expected q=NoMethodError, got %s", query.Get("q"))
		}
		if query.Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", query.Get("limit"))
		}
		if query.Get("order") != "recent" {
			t.Errorf("expected order=recent, got %s", query.Get("order"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	options := FaultListOptions{
		Q:     "NoMethodError",
		Limit: 10,
		Order: "recent",
	}

	_, err := client.Faults.List(context.Background(), 123, options)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
}

func TestFaultsList_WithPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("page") != "3" {
			t.Errorf("expected page=3, got %s", query.Get("page"))
		}
		if query.Get("limit") != "25" {
			t.Errorf("expected limit=25, got %s", query.Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [],
			"links": {
				"prev": "https://app.honeybadger.io/v2/projects/123/faults?page=2",
				"next": "https://app.honeybadger.io/v2/projects/123/faults?page=4"
			}
		}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	options := FaultListOptions{
		Page:  3,
		Limit: 25,
	}

	response, err := client.Faults.List(context.Background(), 123, options)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// Verify pagination links are present
	if response.Links == nil {
		t.Error("expected links to be present in response")
	}
}

func TestFaultsList_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors": "Invalid API token"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("invalid-token")

	_, err := client.Faults.List(context.Background(), 123, FaultListOptions{})
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

func TestGetFault(t *testing.T) {
	mockFault := `{
		"id": 456,
		"action": "create",
		"assignee": {"id": 1, "email": "user@example.com", "name": "User 1"},
		"comments_count": 3,
		"component": "PostController",
		"created_at": "2024-01-01T00:00:00Z",
		"environment": "production",
		"ignored": false,
		"klass": "ActiveRecord::RecordNotFound",
		"last_notice_at": "2024-01-02T00:00:00Z",
		"message": "Couldn't find Post with 'id'=999",
		"notices_count": 25,
		"project_id": 123,
		"resolved": false,
		"tags": ["database", "production"],
		"url": "https://app.honeybadger.io/projects/123/faults/456"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/faults/456" {
			t.Errorf("expected path /v2/projects/123/faults/456, got %s", r.URL.Path)
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
		_, _ = w.Write([]byte(mockFault))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	fault, err := client.Faults.Get(context.Background(), 123, 456)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if fault.ID != 456 {
		t.Errorf("expected fault ID 456, got %d", fault.ID)
	}

	if fault.Message != "Couldn't find Post with 'id'=999" {
		t.Errorf("expected fault message 'Couldn't find Post with 'id'=999', got %s", fault.Message)
	}

	if fault.Klass != "ActiveRecord::RecordNotFound" {
		t.Errorf("expected fault class 'ActiveRecord::RecordNotFound', got %s", fault.Klass)
	}

	if fault.NoticesCount != 25 {
		t.Errorf("expected notices count 25, got %d", fault.NoticesCount)
	}
}

func TestGetFault_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors": "Fault not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Faults.Get(context.Background(), 123, 999)
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

func TestGetFault_ProjectNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors": "Project not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Faults.Get(context.Background(), 999, 456)
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

func TestListNotices(t *testing.T) {
	mockNotices := `{
		"results": [
			{
				"id": "notice-uuid-1",
				"created_at": "2024-01-01T10:00:00Z",
				"fault_id": 456,
				"message": "Couldn't find Post with 'id'=999",
				"url": "https://app.honeybadger.io/projects/123/faults/456/notices/notice-uuid-1",
				"environment": {
					"environment_name": "production",
					"hostname": "web-01.example.com",
					"project_root": {"path": "/app"}
				},
				"cookies": {"session_id": "abc123"},
				"web_environment": {"HTTP_HOST": "example.com"},
				"request": {
					"action": "show",
					"component": "PostsController",
					"url": "https://example.com/posts/999",
					"context": {"user_id": 42},
					"params": {"id": "999"},
					"session": {"user_id": 42},
					"user": {"id": 42, "email": "user@example.com"}
				},
				"backtrace": [
					{"number": "1", "file": "/app/models/post.rb", "method": "find"},
					{"number": "2", "file": "/app/controllers/posts_controller.rb", "method": "show"}
				],
				"application_trace": [
					{"number": "1", "file": "/app/models/post.rb", "method": "find"}
				]
			},
			{
				"id": "notice-uuid-2",
				"created_at": "2024-01-01T11:00:00Z",
				"fault_id": 456,
				"message": "Another occurrence of the same error",
				"url": "https://app.honeybadger.io/projects/123/faults/456/notices/notice-uuid-2",
				"environment": {
					"environment_name": "production",
					"hostname": "web-02.example.com",
					"project_root": {"path": "/app"}
				},
				"cookies": {},
				"web_environment": {"HTTP_HOST": "example.com"},
				"request": {
					"action": "show",
					"component": "PostsController",
					"url": "https://example.com/posts/888",
					"context": {},
					"params": {"id": "888"},
					"session": {},
					"user": {}
				},
				"backtrace": [],
				"application_trace": []
			}
		],
		"links": {
			"self": "https://app.honeybadger.io/v2/projects/123/faults/456/notices"
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/faults/456/notices" {
			t.Errorf("expected path /v2/projects/123/faults/456/notices, got %s", r.URL.Path)
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
		_, _ = w.Write([]byte(mockNotices))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	response, err := client.Faults.ListNotices(context.Background(), 123, 456, FaultListNoticesOptions{})
	if err != nil {
		t.Fatalf("ListNotices() error = %v", err)
	}

	if len(response.Results) != 2 {
		t.Errorf("expected 2 notices, got %d", len(response.Results))
	}

	if response.Results[0].ID != "notice-uuid-1" {
		t.Errorf("expected first notice ID 'notice-uuid-1', got %s", response.Results[0].ID)
	}

	if response.Results[0].FaultID != 456 {
		t.Errorf("expected first notice fault ID 456, got %d", response.Results[0].FaultID)
	}

	if response.Results[0].Message != "Couldn't find Post with 'id'=999" {
		t.Errorf("expected first notice message 'Couldn't find Post with 'id'=999', got %s", response.Results[0].Message)
	}

	if response.Results[0].Environment.EnvironmentName != "production" {
		t.Errorf("expected environment name 'production', got %s", response.Results[0].Environment.EnvironmentName)
	}

	if response.Results[0].Request.Action == nil || *response.Results[0].Request.Action != "show" {
		var action string
		if response.Results[0].Request.Action != nil {
			action = *response.Results[0].Request.Action
		}
		t.Errorf("expected request action 'show', got %s", action)
	}

	if len(response.Results[0].Backtrace) != 2 {
		t.Errorf("expected 2 backtrace entries, got %d", len(response.Results[0].Backtrace))
	}
}

func TestListNotices_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("created_after") != "2024-01-01T00:00:00Z" {
			t.Errorf("expected created_after=2024-01-01T00:00:00Z, got %s", query.Get("created_after"))
		}
		if query.Get("created_before") != "2024-01-02T00:00:00Z" {
			t.Errorf("expected created_before=2024-01-02T00:00:00Z, got %s", query.Get("created_before"))
		}
		if query.Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", query.Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	options := FaultListNoticesOptions{
		CreatedAfter:  &time.Time{},
		CreatedBefore: &time.Time{},
		Limit:         10,
	}
	// Parse the time strings
	createdAfter, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	createdBefore, _ := time.Parse(time.RFC3339, "2024-01-02T00:00:00Z")
	options.CreatedAfter = &createdAfter
	options.CreatedBefore = &createdBefore

	_, err := client.Faults.ListNotices(context.Background(), 123, 456, options)
	if err != nil {
		t.Fatalf("ListNotices() error = %v", err)
	}
}

func TestListNotices_FaultNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors": "Fault not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Faults.ListNotices(context.Background(), 123, 999, FaultListNoticesOptions{})
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

func TestListNotices_ProjectNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors": "Project not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Faults.ListNotices(context.Background(), 999, 456, FaultListNoticesOptions{})
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

func TestListAffectedUsers(t *testing.T) {
	mockUsers := `[
		{
			"user": "user1@example.com",
			"count": 15
		},
		{
			"user": "user2@example.com",
			"count": 8
		}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/faults/456/affected_users" {
			t.Errorf("expected path /v2/projects/123/faults/456/affected_users, got %s", r.URL.Path)
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
		_, _ = w.Write([]byte(mockUsers))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	users, err := client.Faults.ListAffectedUsers(context.Background(), 123, 456, FaultListAffectedUsersOptions{})
	if err != nil {
		t.Fatalf("ListAffectedUsers() error = %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 affected users, got %d", len(users))
	}

	if users[0].User != "user1@example.com" {
		t.Errorf("expected first user 'user1@example.com', got %s", users[0].User)
	}

	if users[0].Count != 15 {
		t.Errorf("expected first user count 15, got %d", users[0].Count)
	}

	if users[1].User != "user2@example.com" {
		t.Errorf("expected second user 'user2@example.com', got %s", users[1].User)
	}

	if users[1].Count != 8 {
		t.Errorf("expected second user count 8, got %d", users[1].Count)
	}
}

func TestListAffectedUsers_WithSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("q") != "user1" {
			t.Errorf("expected q=user1, got %s", query.Get("q"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	options := FaultListAffectedUsersOptions{
		Q: "user1",
	}

	_, err := client.Faults.ListAffectedUsers(context.Background(), 123, 456, options)
	if err != nil {
		t.Fatalf("ListAffectedUsers() error = %v", err)
	}
}

func TestListAffectedUsers_FaultNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors": "Fault not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Faults.ListAffectedUsers(context.Background(), 123, 999, FaultListAffectedUsersOptions{})
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

func TestListAffectedUsers_ProjectNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors": "Project not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Faults.ListAffectedUsers(context.Background(), 999, 456, FaultListAffectedUsersOptions{})
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
