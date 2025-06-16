package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbapi"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleListFaults(t *testing.T) {
	mockResponse := `{
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
				"project_id": 123,
			},
		},
	}

	result, err := handleListFaults(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaults() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that fault data is present
	resultText := getResultText(result)
	if !strings.Contains(resultText, "NoMethodError") {
		t.Error("Fault message should be present in response")
	}

	// Verify the response can be unmarshaled as a fault list response
	var response hbapi.FaultListResponse
	if err := json.Unmarshal([]byte(resultText), &response); err != nil {
		t.Errorf("Response should be valid JSON fault list response: %v", err)
	}

	if len(response.Results) != 1 {
		t.Errorf("expected 1 fault, got %d", len(response.Results))
	}
}

func TestHandleListFaults_WithOptions(t *testing.T) {
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

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
				"q":          "NoMethodError",
				"limit":      10,
				"order":      "recent",
			},
		},
	}

	result, err := handleListFaults(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaults() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}
}

func TestHandleListFaults_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "Invalid API token"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("invalid-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
			},
		},
	}

	result, err := handleListFaults(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaults() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to list faults") {
		t.Error("Error message should contain 'Failed to list faults'")
	}
}

func TestHandleListFaults_MissingProjectID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handleListFaults(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaults() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing project ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "project_id is required") {
		t.Error("Error message should mention project_id is required")
	}
}

func TestHandleGetFault(t *testing.T) {
	mockResponse := `{
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
				"project_id": 123,
				"fault_id":   456,
			},
		},
	}

	result, err := handleGetFault(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetFault() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that fault data is present
	resultText := getResultText(result)
	if !strings.Contains(resultText, "ActiveRecord::RecordNotFound") {
		t.Error("Fault class should be present in response")
	}

	// Verify the response can be unmarshaled as a fault
	var fault hbapi.Fault
	if err := json.Unmarshal([]byte(resultText), &fault); err != nil {
		t.Errorf("Response should be valid JSON fault: %v", err)
	}

	if fault.ID != 456 {
		t.Errorf("expected fault ID 456, got %d", fault.ID)
	}

	if fault.Message != "Couldn't find Post with 'id'=999" {
		t.Errorf("expected fault message 'Couldn't find Post with 'id'=999', got %s", fault.Message)
	}
}

func TestHandleGetFault_MissingProjectID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"fault_id": 456,
			},
		},
	}

	result, err := handleGetFault(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetFault() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing project ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "project_id is required") {
		t.Error("Error message should mention project_id is required")
	}
}

func TestHandleGetFault_MissingFaultID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
			},
		},
	}

	result, err := handleGetFault(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetFault() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing fault ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "fault_id is required") {
		t.Error("Error message should mention fault_id is required")
	}
}

func TestHandleGetFault_InvalidProjectID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 0,
				"fault_id":   456,
			},
		},
	}

	result, err := handleGetFault(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetFault() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for invalid project ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "project_id is required") {
		t.Error("Error message should mention project_id is required")
	}
}

func TestHandleGetFault_InvalidFaultID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
				"fault_id":   0,
			},
		},
	}

	result, err := handleGetFault(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetFault() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for invalid fault ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "fault_id is required") {
		t.Error("Error message should mention fault_id is required")
	}
}

func TestHandleGetFault_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Fault not found"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
				"fault_id":   999,
			},
		},
	}

	result, err := handleGetFault(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetFault() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to get fault") {
		t.Error("Error message should contain 'Failed to get fault'")
	}
}

func TestHandleListFaultNotices(t *testing.T) {
	mockResponse := `{
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
					"project_root": "/app"
				},
				"environment_name": "production",
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
				"project_id": 123,
				"fault_id":   456,
			},
		},
	}

	result, err := handleListFaultNotices(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaultNotices() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that notice data is present
	resultText := getResultText(result)
	if !strings.Contains(resultText, "notice-uuid-1") {
		t.Error("Notice ID should be present in response")
	}

	// Verify the response can be unmarshaled as a notice response
	var response hbapi.FaultNoticesResponse
	if err := json.Unmarshal([]byte(resultText), &response); err != nil {
		t.Errorf("Response should be valid JSON fault notices response: %v", err)
	}

	if len(response.Results) != 1 {
		t.Errorf("expected 1 notice, got %d", len(response.Results))
	}

	if response.Results[0].ID != "notice-uuid-1" {
		t.Errorf("expected notice ID 'notice-uuid-1', got %s", response.Results[0].ID)
	}

	if response.Results[0].FaultID != 456 {
		t.Errorf("expected fault ID 456, got %d", response.Results[0].FaultID)
	}
}

func TestHandleListFaultNotices_WithOptions(t *testing.T) {
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

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id":     123,
				"fault_id":       456,
				"created_after":  "2024-01-01T00:00:00Z",
				"created_before": "2024-01-02T00:00:00Z",
				"limit":          10,
			},
		},
	}

	result, err := handleListFaultNotices(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaultNotices() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}
}

func TestHandleListFaultNotices_MissingProjectID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"fault_id": 456,
			},
		},
	}

	result, err := handleListFaultNotices(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaultNotices() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing project ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "project_id is required") {
		t.Error("Error message should mention project_id is required")
	}
}

func TestHandleListFaultNotices_MissingFaultID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
			},
		},
	}

	result, err := handleListFaultNotices(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaultNotices() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing fault ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "fault_id is required") {
		t.Error("Error message should mention fault_id is required")
	}
}

func TestHandleListFaultNotices_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Fault not found"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
				"fault_id":   999,
			},
		},
	}

	result, err := handleListFaultNotices(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaultNotices() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to list fault notices") {
		t.Error("Error message should contain 'Failed to list fault notices'")
	}
}

func TestHandleListFaultAffectedUsers(t *testing.T) {
	mockResponse := `[
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
				"project_id": 123,
				"fault_id":   456,
			},
		},
	}

	result, err := handleListFaultAffectedUsers(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaultAffectedUsers() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that user data is present
	resultText := getResultText(result)
	if !strings.Contains(resultText, "user1@example.com") {
		t.Error("User email should be present in response")
	}

	// Verify the response can be unmarshaled as an affected users array
	var users []hbapi.FaultAffectedUser
	if err := json.Unmarshal([]byte(resultText), &users); err != nil {
		t.Errorf("Response should be valid JSON array of affected users: %v", err)
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
}

func TestHandleListFaultAffectedUsers_WithSearch(t *testing.T) {
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

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
				"fault_id":   456,
				"q":          "user1",
			},
		},
	}

	result, err := handleListFaultAffectedUsers(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaultAffectedUsers() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}
}

func TestHandleListFaultAffectedUsers_MissingProjectID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"fault_id": 456,
			},
		},
	}

	result, err := handleListFaultAffectedUsers(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaultAffectedUsers() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing project ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "project_id is required") {
		t.Error("Error message should mention project_id is required")
	}
}

func TestHandleListFaultAffectedUsers_MissingFaultID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
			},
		},
	}

	result, err := handleListFaultAffectedUsers(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaultAffectedUsers() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing fault ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "fault_id is required") {
		t.Error("Error message should mention fault_id is required")
	}
}

func TestHandleListFaultAffectedUsers_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Fault not found"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
				"fault_id":   999,
			},
		},
	}

	result, err := handleListFaultAffectedUsers(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListFaultAffectedUsers() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to list fault affected users") {
		t.Error("Error message should contain 'Failed to list fault affected users'")
	}
}
