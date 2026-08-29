package hbmcp

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleListStreams(t *testing.T) {
	mockResponse := `{
		"data": [
			{
				"id": "abc123def456",
				"name": "Default",
				"slug": "default",
				"internal": false,
				"project_id": "Xk9mZp",
				"created_at": "2024-01-01T00:00:00Z"
			},
			{
				"id": "789ghi012jkl",
				"name": "Internal",
				"slug": "internal",
				"internal": true,
				"project_id": "Xk9mZp",
				"created_at": "2024-01-01T00:00:00Z"
			}
		],
		"pagination": {"page": 1, "per_page": 25, "total_count": 2, "total_pages": 1}
	}`

	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if want := "/v3/projects/Xk9mZp/streams"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		v3JSON(w, http.StatusOK, mockResponse)
	})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": "Xk9mZp",
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
	if !strings.Contains(resultText, `"internal":true`) {
		t.Error("Result should contain the internal boolean flag set to true")
	}
}

func TestHandleListStreamsMissingProjectID(t *testing.T) {
	client := offlineV3Client()

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
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusNotFound,
			`{"error":{"code":"not_found","message":"Resource not found"}}`)
	})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": "nope",
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
