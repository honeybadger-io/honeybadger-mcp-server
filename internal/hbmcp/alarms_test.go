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

func TestHandleListAlarms(t *testing.T) {
	mockResponse := `{
		"results": [
			{
				"id": "abc123",
				"name": "High Error Rate",
				"description": "Triggers when error count exceeds threshold",
				"state": "ok",
				"query": "filter event_type::str == \"notice\" | stats count()",
				"stream_ids": ["default"],
				"evaluation_period": "5m",
				"trigger_config": {"type": "alert_result_count", "config": {"operator": "gt", "value": 10}},
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-02T00:00:00Z",
				"url": "https://app.honeybadger.io/projects/123/insights/alarms/abc123",
				"project_id": 123
			}
		],
		"links": {"self": "", "next": "", "prev": ""}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms" {
			t.Errorf("expected path /v2/projects/123/alarms, got %s", r.URL.Path)
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

	result, err := handleListAlarms(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListAlarms() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "abc123") {
		t.Error("Result should contain alarm ID")
	}
	if !strings.Contains(resultText, "High Error Rate") {
		t.Error("Result should contain alarm name")
	}
}

func TestHandleGetAlarm(t *testing.T) {
	mockResponse := `{
		"id": "abc123",
		"name": "High Error Rate",
		"description": "Triggers when error count exceeds threshold",
		"state": "alarm",
		"query": "filter event_type::str == \"notice\" | stats count()",
		"stream_ids": ["default"],
		"evaluation_period": "5m",
		"trigger_config": {"type": "alert_result_count", "config": {"operator": "gt", "value": 10}},
		"last_checked_at": "2024-01-02T12:00:00Z",
		"next_check_at": "2024-01-02T12:05:00Z",
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-02T00:00:00Z",
		"url": "https://app.honeybadger.io/projects/123/insights/alarms/abc123",
		"project_id": 123
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms/abc123" {
			t.Errorf("expected path /v2/projects/123/alarms/abc123, got %s", r.URL.Path)
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
				"alarm_id":   "abc123",
			},
		},
	}

	result, err := handleGetAlarm(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetAlarm() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)

	var alarm hbapi.Alarm
	if err := json.Unmarshal([]byte(resultText), &alarm); err != nil {
		t.Fatalf("Response should be valid JSON: %v", err)
	}

	if alarm.ID != "abc123" {
		t.Errorf("expected ID abc123, got %s", alarm.ID)
	}

	if alarm.Name != "High Error Rate" {
		t.Errorf("expected name 'High Error Rate', got %s", alarm.Name)
	}

	if alarm.State != "alarm" {
		t.Errorf("expected state 'alarm', got %s", alarm.State)
	}
}

func TestHandleCreateAlarm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms" {
			t.Errorf("expected path /v2/projects/123/alarms, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		alarm, ok := body["alarm"].(map[string]interface{})
		if !ok {
			t.Fatal("expected alarm key in request body")
		}
		if alarm["name"] != "New Alarm" {
			t.Errorf("expected name 'New Alarm', got %v", alarm["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": "new123",
			"name": "New Alarm",
			"description": "",
			"state": "initial",
			"query": "stats count()",
			"stream_ids": ["default"],
			"evaluation_period": "5m",
			"trigger_config": {"type": "alert_result_count", "config": {"operator": "gt", "value": 5}},
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
			"url": "https://app.honeybadger.io/projects/123/insights/alarms/new123",
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
				"project_id":        123,
				"name":              "New Alarm",
				"query":             "stats count()",
				"evaluation_period": "5m",
				"lookback_lag":      "1m",
				"trigger_config":    `{"type": "alert_result_count", "config": {"operator": "gt", "value": 5}}`,
			},
		},
	}

	result, err := handleCreateAlarm(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateAlarm() error = %v", err)
	}

	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "new123") {
		t.Error("Result should contain new alarm ID")
	}
}

func TestHandleUpdateAlarm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms/abc123" {
			t.Errorf("expected path /v2/projects/123/alarms/abc123, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		alarm, ok := body["alarm"].(map[string]interface{})
		if !ok {
			t.Fatal("expected alarm key in request body")
		}
		if alarm["name"] != "Updated Alarm" {
			t.Errorf("expected name 'Updated Alarm', got %v", alarm["name"])
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
				"project_id":        123,
				"alarm_id":          "abc123",
				"name":              "Updated Alarm",
				"query":             "stats count()",
				"evaluation_period": "10m",
				"lookback_lag":      "1m",
				"trigger_config":    `{"type": "alert_result_count", "config": {"operator": "gt", "value": 20}}`,
			},
		},
	}

	result, err := handleUpdateAlarm(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleUpdateAlarm() error = %v", err)
	}

	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "successfully updated") {
		t.Error("Result should contain success message")
	}
}

func TestHandleDeleteAlarm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms/abc123" {
			t.Errorf("expected path /v2/projects/123/alarms/abc123, got %s", r.URL.Path)
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
				"project_id": 123,
				"alarm_id":   "abc123",
			},
		},
	}

	result, err := handleDeleteAlarm(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleDeleteAlarm() error = %v", err)
	}

	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "deleted successfully") {
		t.Error("Result should contain success message")
	}
}

func TestHandleGetAlarmHistory(t *testing.T) {
	mockResponse := `{
		"triggers": [
			{
				"id": "trigger1",
				"state": "alarm",
				"result": {"count": 15},
				"created_at": "2024-01-02T12:00:00Z"
			},
			{
				"id": "trigger2",
				"state": "ok",
				"result": {"count": 3},
				"created_at": "2024-01-02T11:55:00Z"
			}
		],
		"links": {"self": "", "next": "", "prev": ""}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms/abc123/history" {
			t.Errorf("expected path /v2/projects/123/alarms/abc123/history, got %s", r.URL.Path)
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
				"alarm_id":   "abc123",
			},
		},
	}

	result, err := handleGetAlarmHistory(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetAlarmHistory() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "trigger1") {
		t.Error("Result should contain trigger ID")
	}
	if !strings.Contains(resultText, "alarm") {
		t.Error("Result should contain trigger state")
	}
}

func TestHandleCreateAlarm_MissingProjectID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"name":              "Test",
				"query":             "stats count()",
				"evaluation_period": "5m",
				"lookback_lag":      "1m",
				"trigger_config":    `{}`,
			},
		},
	}

	result, err := handleCreateAlarm(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateAlarm() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing project ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "project_id is required") {
		t.Error("Error message should mention project_id is required")
	}
}

func TestHandleCreateAlarm_MissingName(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id":        123,
				"query":             "stats count()",
				"evaluation_period": "5m",
				"lookback_lag":      "1m",
				"trigger_config":    `{}`,
			},
		},
	}

	result, err := handleCreateAlarm(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateAlarm() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing name")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "name is required") {
		t.Error("Error message should mention name is required")
	}
}

func TestHandleCreateAlarm_InvalidTriggerConfigJSON(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id":        123,
				"name":              "Test",
				"query":             "stats count()",
				"evaluation_period": "5m",
				"lookback_lag":      "1m",
				"trigger_config":    "not valid json",
			},
		},
	}

	result, err := handleCreateAlarm(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateAlarm() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for invalid trigger_config JSON")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to parse trigger_config JSON") {
		t.Error("Error message should mention failed to parse trigger_config JSON")
	}
}
