package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/honeybadger-io/api-go/apiv3"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleQueryInsights(t *testing.T) {
	// v3 passes the query service's result through under data, rather than
	// splitting it into results and meta as v2 did.
	mockResponse := `{
		"data": {
			"results": [
				{"ts": "2024-01-01T00:00:00Z", "count": 10, "name": "web"},
				{"ts": "2024-01-01T01:00:00Z", "count": 15, "name": "api"}
			],
			"fields": ["ts", "count", "name"],
			"rows": 2,
			"total_rows": 2
		},
		"meta": {"request_id": "req_1"}
	}`

	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v3/projects/Xk9mZp/insights/queries" {
			t.Errorf("expected path /v3/projects/Xk9mZp/insights/queries, got %s", r.URL.Path)
		}
		v3JSON(w, http.StatusOK, mockResponse)
	})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": "Xk9mZp",
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

	// v3 hands the query service's payload through untouched, so the tool output
	// carries whatever shape that service produced rather than a fixed one.
	var response apiv3.InsightsResult
	if err := json.Unmarshal([]byte(resultText), &response); err != nil {
		t.Errorf("Response should be valid JSON insights query response: %v", err)
	}

	rows, ok := response.Data["results"].([]any)
	if !ok || len(rows) != 2 {
		t.Errorf("expected 2 results, got %v", response.Data["results"])
	}
	if response.Data["rows"] != float64(2) {
		t.Errorf("expected rows 2, got %v", response.Data["rows"])
	}
	if response.RequestID != "req_1" {
		t.Errorf("expected request id to survive, got %q", response.RequestID)
	}
}

func TestHandleQueryInsights_WithAllOptions(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		// Decoded against the wire contract rather than a client type, since the
		// point is what actually leaves the process.
		var reqBody struct {
			Query     string   `json:"query"`
			Ts        string   `json:"ts"`
			Timezone  string   `json:"timezone"`
			StreamIDs []string `json:"stream_ids"`
		}
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
		if len(reqBody.StreamIDs) != 2 || reqBody.StreamIDs[0] != "Oh3Y3WdMFvde" || reqBody.StreamIDs[1] != "MuHadpB4C9G4" {
			t.Errorf("expected stream_ids [Oh3Y3WdMFvde MuHadpB4C9G4], got %v", reqBody.StreamIDs)
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
	})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": "Xk9mZp",
				"query":      "fields @ts, message::str",
				"ts":         "week",
				"timezone":   "America/New_York",
				"stream_ids": []interface{}{"Oh3Y3WdMFvde", "MuHadpB4C9G4"},
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
	client := offlineV3Client()

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
	client := offlineV3Client()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": "Xk9mZp",
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

func TestHandleQueryInsights_QueryRejected(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// v2 reported query failures inline on a 200. v3 answers 422, and the
		// query service's own message survives in the error envelope because it
		// is more specific than anything the client could substitute.
		v3JSON(w, http.StatusUnprocessableEntity, `{
			"error": {"code": "validation_error", "message": "query timed out"},
			"meta": {"request_id": "req_bad"}
		}`)
	})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": "Xk9mZp",
				"query":      "stats count()",
			},
		},
	}

	result, err := handleQueryInsights(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleQueryInsights() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for a rejected query")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "query timed out") {
		t.Error("Error message should contain 'query timed out'")
	}
}

func TestHandleQueryInsights_Error(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusUnauthorized, `{"error":{"code":"unauthorized","message":"Invalid API token"}}`)
	})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": "Xk9mZp",
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
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusUnprocessableEntity, `{"error":{"code":"validation_error","message":"Invalid query syntax"}}`)
	})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": "Xk9mZp",
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
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusNotFound, `{"errors": "Project not found"}`)
	})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": "nope",
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
