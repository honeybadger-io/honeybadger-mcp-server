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

func TestHandleQueryInsights(t *testing.T) {
	mockResponse := `{
		"results": [
			{"ts": "2024-01-01T00:00:00Z", "count": 10, "name": "web"},
			{"ts": "2024-01-01T01:00:00Z", "count": 15, "name": "api"}
		],
		"meta": {
			"query": "stats count() by event_type::str",
			"fields": ["ts", "count", "name"],
			"schema": [
				{"name": "ts", "type": "DateTime"},
				{"name": "count", "type": "UInt64"},
				{"name": "name", "type": "String"}
			],
			"rows": 2,
			"total_rows": 2,
			"start_at": "2024-01-01T00:00:00Z",
			"end_at": "2024-01-01T03:00:00Z"
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/insights/queries" {
			t.Errorf("expected path /v2/projects/123/insights/queries, got %s", r.URL.Path)
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
				"query":      "stats count() by event_type::str",
			},
		},
	}

	result, err := handleQueryInsights(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleQueryInsights() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	// Check that insights data is present
	resultText := getResultText(result)
	if !strings.Contains(resultText, "web") {
		t.Error("Result data should be present in response")
	}

	// Verify the response can be unmarshaled as an insights query response
	var response hbapi.InsightsQueryResponse
	if err := json.Unmarshal([]byte(resultText), &response); err != nil {
		t.Errorf("Response should be valid JSON insights query response: %v", err)
	}

	if len(response.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(response.Results))
	}

	if response.Meta.Query != "stats count() by event_type::str" {
		t.Errorf("expected query in meta, got %s", response.Meta.Query)
	}

	if response.Meta.Rows != 2 {
		t.Errorf("expected 2 rows, got %d", response.Meta.Rows)
	}
}

func TestHandleQueryInsights_WithAllOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		// Verify the request body contains the expected fields
		var reqBody hbapi.InsightsQueryRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if reqBody.Query != "fields @ts, message::str" {
			t.Errorf("expected query 'fields @ts, message::str', got %s", reqBody.Query)
		}
		if reqBody.Ts != "week" {
			t.Errorf("expected ts 'week', got %s", reqBody.Ts)
		}
		if reqBody.Timezone != "America/New_York" {
			t.Errorf("expected timezone 'America/New_York', got %s", reqBody.Timezone)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [],
			"meta": {
				"query": "fields @ts, message::str",
				"fields": [],
				"schema": [],
				"rows": 0,
				"total_rows": 0,
				"start_at": "2024-01-01T00:00:00Z",
				"end_at": "2024-01-07T00:00:00Z"
			}
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
				"query":      "fields @ts, message::str",
				"ts":         "week",
				"timezone":   "America/New_York",
			},
		},
	}

	result, err := handleQueryInsights(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleQueryInsights() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}
}

func TestHandleQueryInsights_MissingProjectID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"query": "stats count()",
			},
		},
	}

	result, err := handleQueryInsights(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleQueryInsights() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing project ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "project_id is required") {
		t.Error("Error message should mention project_id is required")
	}
}

func TestHandleQueryInsights_MissingQuery(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
			},
		},
	}

	result, err := handleQueryInsights(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleQueryInsights() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing query")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "query is required") {
		t.Error("Error message should mention query is required")
	}
}

func TestHandleQueryInsights_Error(t *testing.T) {
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
			Arguments: map[string]interface{}{
				"project_id": 123,
				"query":      "stats count()",
			},
		},
	}

	result, err := handleQueryInsights(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleQueryInsights() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to query insights") {
		t.Error("Error message should contain 'Failed to query insights'")
	}
}

func TestHandleQueryInsights_InvalidQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors": "Invalid query syntax"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
				"query":      "INVALID QUERY",
			},
		},
	}

	result, err := handleQueryInsights(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleQueryInsights() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for invalid query")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to query insights") {
		t.Error("Error message should contain 'Failed to query insights'")
	}
}

func TestHandleQueryInsights_ProjectNotFound(t *testing.T) {
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
				"project_id": 999,
				"query":      "stats count()",
			},
		},
	}

	result, err := handleQueryInsights(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleQueryInsights() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for project not found")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to query insights") {
		t.Error("Error message should contain 'Failed to query insights'")
	}
}
