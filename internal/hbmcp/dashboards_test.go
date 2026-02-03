package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleListDashboards(t *testing.T) {
	mockResponse := `{
		"results": [
			{
				"id": "abc123",
				"title": "Project Overview",
				"widgets": [{"id": "w1", "type": "errors"}],
				"is_default": true,
				"shared": true,
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-02T00:00:00Z",
				"project_id": 123
			}
		],
		"links": {"self": "", "next": "", "prev": ""}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/dashboards" {
			t.Errorf("expected path /v2/projects/123/dashboards, got %s", r.URL.Path)
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

	result, err := handleListDashboards(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListDashboards() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "abc123") {
		t.Error("Result should contain dashboard ID")
	}
	if !strings.Contains(resultText, "Project Overview") {
		t.Error("Result should contain dashboard title")
	}
}

func TestHandleGetDashboard(t *testing.T) {
	mockResponse := `{
		"id": "abc123",
		"title": "Project Overview",
		"widgets": [{"id": "w1", "type": "errors"}],
		"is_default": true,
		"shared": true,
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-02T00:00:00Z",
		"project_id": 123
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/dashboards/abc123" {
			t.Errorf("expected path /v2/projects/123/dashboards/abc123, got %s", r.URL.Path)
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
				"project_id":   123,
				"dashboard_id": "abc123",
			},
		},
	}

	result, err := handleGetDashboard(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetDashboard() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)

	var dashboard hbapi.Dashboard
	if err := json.Unmarshal([]byte(resultText), &dashboard); err != nil {
		t.Fatalf("Response should be valid JSON: %v", err)
	}

	if dashboard.ID != "abc123" {
		t.Errorf("expected ID abc123, got %s", dashboard.ID)
	}

	if dashboard.Title != "Project Overview" {
		t.Errorf("expected title 'Project Overview', got %s", dashboard.Title)
	}
}

func TestHandleCreateDashboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/dashboards" {
			t.Errorf("expected path /v2/projects/123/dashboards, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		dashboard, ok := body["dashboard"].(map[string]interface{})
		if !ok {
			t.Fatal("expected dashboard key in request body")
		}
		if dashboard["title"] != "My Dashboard" {
			t.Errorf("expected title 'My Dashboard', got %v", dashboard["title"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": "new123",
			"title": "My Dashboard",
			"widgets": [{"id": "w1", "type": "insights_vis"}],
			"is_default": false,
			"shared": true,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
			"project_id": 123
		}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
				"title":      "My Dashboard",
				"widgets":    `[{"type": "insights_vis", "config": {"query": "stats count()"}}]`,
			},
		},
	}

	result, err := handleCreateDashboard(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateDashboard() error = %v", err)
	}

	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "new123") {
		t.Error("Result should contain new dashboard ID")
	}
}

func TestHandleUpdateDashboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/dashboards/abc123" {
			t.Errorf("expected path /v2/projects/123/dashboards/abc123, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		dashboard, ok := body["dashboard"].(map[string]interface{})
		if !ok {
			t.Fatal("expected dashboard key in request body")
		}
		if dashboard["title"] != "Updated Title" {
			t.Errorf("expected title 'Updated Title', got %v", dashboard["title"])
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
				"project_id":   123,
				"dashboard_id": "abc123",
				"title":        "Updated Title",
				"widgets":      `[]`,
			},
		},
	}

	result, err := handleUpdateDashboard(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleUpdateDashboard() error = %v", err)
	}

	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "successfully updated") {
		t.Error("Result should contain success message")
	}
}

func TestHandleDeleteDashboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/dashboards/abc123" {
			t.Errorf("expected path /v2/projects/123/dashboards/abc123, got %s", r.URL.Path)
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
				"project_id":   123,
				"dashboard_id": "abc123",
			},
		},
	}

	result, err := handleDeleteDashboard(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleDeleteDashboard() error = %v", err)
	}

	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "deleted successfully") {
		t.Error("Result should contain success message")
	}
}

func TestHandleCreateDashboard_MissingProjectID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"title":   "Test",
				"widgets": "[]",
			},
		},
	}

	result, err := handleCreateDashboard(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateDashboard() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing project ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "project_id is required") {
		t.Error("Error message should mention project_id is required")
	}
}

func TestHandleCreateDashboard_MissingTitle(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
				"widgets":    "[]",
			},
		},
	}

	result, err := handleCreateDashboard(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateDashboard() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing title")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "title is required") {
		t.Error("Error message should mention title is required")
	}
}

func TestHandleCreateDashboard_InvalidWidgetsJSON(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
				"title":      "Test",
				"widgets":    "not valid json",
			},
		},
	}

	result, err := handleCreateDashboard(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateDashboard() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for invalid widgets JSON")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to parse widgets JSON") {
		t.Error("Error message should mention failed to parse widgets JSON")
	}
}
