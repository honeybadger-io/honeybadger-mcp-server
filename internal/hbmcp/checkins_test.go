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

func TestHandleListCheckIns(t *testing.T) {
	mockResponse := `{
		"results": [
			{
				"id": "abc123",
				"name": "Nightly Backups",
				"slug": "nightly-backups",
				"state": "reporting",
				"schedule_type": "simple",
				"report_period": "1 day",
				"grace_period": "5 minutes",
				"cron_schedule": null,
				"cron_timezone": null,
				"reported_at": "2024-01-02T05:00:00Z",
				"expected_at": "2024-01-03T05:00:00Z",
				"missed_count": 0,
				"url": "https://api.honeybadger.io/v1/check_in/xyz789",
				"details_url": "https://app.honeybadger.io/projects/123/check_ins/abc123"
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/check_ins" {
			t.Errorf("expected path /v2/projects/123/check_ins, got %s", r.URL.Path)
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

	result, err := handleListCheckIns(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleListCheckIns() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "abc123") {
		t.Error("Result should contain check-in ID")
	}
	if !strings.Contains(resultText, "Nightly Backups") {
		t.Error("Result should contain check-in name")
	}
}

func TestHandleGetCheckIn(t *testing.T) {
	mockResponse := `{
		"id": "abc123",
		"name": "Nightly Backups",
		"slug": "nightly-backups",
		"state": "missing",
		"schedule_type": "cron",
		"report_period": null,
		"grace_period": "5 minutes",
		"cron_schedule": "0 5 * * *",
		"cron_timezone": "UTC",
		"reported_at": "2024-01-02T05:00:00Z",
		"expected_at": "2024-01-03T05:00:00Z",
		"missed_count": 2,
		"url": "https://api.honeybadger.io/v1/check_in/xyz789",
		"details_url": "https://app.honeybadger.io/projects/123/check_ins/abc123"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/check_ins/abc123" {
			t.Errorf("expected path /v2/projects/123/check_ins/abc123, got %s", r.URL.Path)
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
				"project_id":  123,
				"check_in_id": "abc123",
			},
		},
	}

	result, err := handleGetCheckIn(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetCheckIn() error = %v", err)
	}

	if result.IsError {
		t.Fatal("expected successful result, got error")
	}

	resultText := getResultText(result)

	var checkIn hbapi.CheckIn
	if err := json.Unmarshal([]byte(resultText), &checkIn); err != nil {
		t.Fatalf("Response should be valid JSON: %v", err)
	}

	if checkIn.ID != "abc123" {
		t.Errorf("expected ID abc123, got %s", checkIn.ID)
	}

	if checkIn.Name != "Nightly Backups" {
		t.Errorf("expected name 'Nightly Backups', got %s", checkIn.Name)
	}

	if checkIn.State != "missing" {
		t.Errorf("expected state 'missing', got %s", checkIn.State)
	}
}

func TestHandleCreateCheckIn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/check_ins" {
			t.Errorf("expected path /v2/projects/123/check_ins, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		checkIn, ok := body["check_in"].(map[string]interface{})
		if !ok {
			t.Fatal("expected check_in key in request body")
		}
		if checkIn["name"] != "Nightly Backups" {
			t.Errorf("expected name 'Nightly Backups', got %v", checkIn["name"])
		}
		if checkIn["schedule_type"] != "simple" {
			t.Errorf("expected schedule_type 'simple', got %v", checkIn["schedule_type"])
		}
		if checkIn["report_period"] != "1 day" {
			t.Errorf("expected report_period '1 day', got %v", checkIn["report_period"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": "new123",
			"name": "Nightly Backups",
			"slug": "nightly-backups",
			"state": "pending",
			"schedule_type": "simple",
			"report_period": "1 day",
			"grace_period": null,
			"cron_schedule": null,
			"cron_timezone": null,
			"reported_at": null,
			"expected_at": null,
			"missed_count": 0,
			"url": "https://api.honeybadger.io/v1/check_in/xyz789",
			"details_url": "https://app.honeybadger.io/projects/123/check_ins/new123"
		}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id":    123,
				"name":          "Nightly Backups",
				"slug":          "nightly-backups",
				"schedule_type": "simple",
				"report_period": "1 day",
			},
		},
	}

	result, err := handleCreateCheckIn(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateCheckIn() error = %v", err)
	}

	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "new123") {
		t.Error("Result should contain new check-in ID")
	}
}

func TestHandleUpdateCheckIn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/check_ins/abc123" {
			t.Errorf("expected path /v2/projects/123/check_ins/abc123, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		checkIn, ok := body["check_in"].(map[string]interface{})
		if !ok {
			t.Fatal("expected check_in key in request body")
		}
		if checkIn["name"] != "Hourly Backups" {
			t.Errorf("expected name 'Hourly Backups', got %v", checkIn["name"])
		}
		if checkIn["report_period"] != "1 hour" {
			t.Errorf("expected report_period '1 hour', got %v", checkIn["report_period"])
		}
		if _, present := checkIn["cron_schedule"]; present {
			t.Error("cron_schedule should be omitted when not provided")
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
				"project_id":    123,
				"check_in_id":   "abc123",
				"name":          "Hourly Backups",
				"report_period": "1 hour",
			},
		},
	}

	result, err := handleUpdateCheckIn(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleUpdateCheckIn() error = %v", err)
	}

	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "successfully updated") {
		t.Error("Result should contain success message")
	}
}

func TestHandleDeleteCheckIn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/check_ins/abc123" {
			t.Errorf("expected path /v2/projects/123/check_ins/abc123, got %s", r.URL.Path)
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
				"project_id":  123,
				"check_in_id": "abc123",
			},
		},
	}

	result, err := handleDeleteCheckIn(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleDeleteCheckIn() error = %v", err)
	}

	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "deleted successfully") {
		t.Error("Result should contain success message")
	}
}

func TestHandleCreateCheckIn_MissingProjectID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"name":          "Test",
				"schedule_type": "simple",
				"report_period": "1 day",
			},
		},
	}

	result, err := handleCreateCheckIn(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateCheckIn() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing project ID")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "project_id is required") {
		t.Error("Error message should mention project_id is required")
	}
}

func TestHandleCreateCheckIn_MissingName(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id":    123,
				"schedule_type": "simple",
				"report_period": "1 day",
			},
		},
	}

	result, err := handleCreateCheckIn(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateCheckIn() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing name")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "name is required") {
		t.Error("Error message should mention name is required")
	}
}

func TestHandleCreateCheckIn_InvalidScheduleType(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id":    123,
				"name":          "Test",
				"schedule_type": "hourly",
			},
		},
	}

	result, err := handleCreateCheckIn(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateCheckIn() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for invalid schedule_type")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "schedule_type must be 'simple' or 'cron'") {
		t.Error("Error message should mention valid schedule types")
	}
}

func TestHandleCreateCheckIn_SimpleMissingReportPeriod(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id":    123,
				"name":          "Test",
				"schedule_type": "simple",
			},
		},
	}

	result, err := handleCreateCheckIn(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateCheckIn() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing report_period")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "report_period is required for simple schedules") {
		t.Error("Error message should mention report_period is required for simple schedules")
	}
}

func TestHandleCreateCheckIn_CronMissingCronSchedule(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id":    123,
				"name":          "Test",
				"schedule_type": "cron",
			},
		},
	}

	result, err := handleCreateCheckIn(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleCreateCheckIn() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing cron_schedule")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "cron_schedule is required for cron schedules") {
		t.Error("Error message should mention cron_schedule is required for cron schedules")
	}
}

func TestHandleUpdateCheckIn_MissingCheckInID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
				"name":       "Test",
			},
		},
	}

	result, err := handleUpdateCheckIn(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleUpdateCheckIn() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing check_in_id")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "check_in_id is required") {
		t.Error("Error message should mention check_in_id is required")
	}
}

func TestHandleDeleteCheckIn_MissingCheckInID(t *testing.T) {
	client := hbapi.NewClient()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": 123,
			},
		},
	}

	result, err := handleDeleteCheckIn(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleDeleteCheckIn() error = %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error result for missing check_in_id")
	}

	resultText := getResultText(result)
	if !strings.Contains(resultText, "check_in_id is required") {
		t.Error("Error message should mention check_in_id is required")
	}
}
