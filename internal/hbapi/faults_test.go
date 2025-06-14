package hbapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
		]
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
		w.Write([]byte(mockFaults))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	faults, err := client.Faults.List(context.Background(), 123, FaultListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(faults) != 2 {
		t.Errorf("expected 2 faults, got %d", len(faults))
	}

	if faults[0].ID != 1 {
		t.Errorf("expected first fault ID 1, got %d", faults[0].ID)
	}

	if faults[0].Message != "undefined method 'foo' for nil:NilClass" {
		t.Errorf("expected first fault message 'undefined method 'foo' for nil:NilClass', got %s", faults[0].Message)
	}

	if faults[1].Resolved != true {
		t.Errorf("expected second fault to be resolved, got %v", faults[1].Resolved)
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
		w.Write([]byte(`{"results": []}`))
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

func TestFaultsList_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid API token"}`))
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
		w.Write([]byte(mockFault))
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
		w.Write([]byte(`{"error": "Fault not found"}`))
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
		w.Write([]byte(`{"error": "Project not found"}`))
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
