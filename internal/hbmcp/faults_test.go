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
		]
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
		w.Write([]byte(mockResponse))
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

	// Verify the response can be unmarshaled as a fault array
	var faults []hbapi.Fault
	if err := json.Unmarshal([]byte(resultText), &faults); err != nil {
		t.Errorf("Response should be valid JSON array of faults: %v", err)
	}

	if len(faults) != 1 {
		t.Errorf("expected 1 fault, got %d", len(faults))
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
		w.Write([]byte(`{"results": []}`))
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
		w.Write([]byte(`{"error": "Invalid API token"}`))
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


