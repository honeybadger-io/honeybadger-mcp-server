package hbmcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleListStreams(t *testing.T) {
	mockResponse := `{
		"results": [
			{
				"id": "abc123def456",
				"name": "Default",
				"slug": "default",
				"internal": false,
				"project_id": 123,
				"created_at": "2024-01-01T00:00:00Z"
			},
			{
				"id": "789ghi012jkl",
				"name": "Internal",
				"slug": "internal",
				"internal": true,
				"project_id": 123,
				"created_at": "2024-01-01T00:00:00Z"
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/streams" {
			t.Errorf("expected path /v2/projects/123/streams, got %s", r.URL.Path)
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

	result, err := handleListStreams(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListStreams() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "abc123def456") {
		t.Error("Result should contain stream ID")
	}
	if !strings.Contains(resultText, "Default") {
		t.Error("Result should contain stream name")
	}
	if !strings.Contains(resultText, "internal") {
		t.Error("Result should contain internal flag")
	}
}

func TestHandleListStreamsMissingProjectID(t *testing.T) {
	client := hbapi.NewClient().WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handleListStreams(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListStreams() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing project_id")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "project_id is required") {
		t.Errorf("expected 'project_id is required' error, got: %s", resultText)
	}
}

func TestHandleListStreamsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors": "Not found"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 999,
			},
		},
	}

	result, err := handleListStreams(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListStreams() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for API failure")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to list streams") {
		t.Errorf("expected 'Failed to list streams' error, got: %s", resultText)
	}
}
