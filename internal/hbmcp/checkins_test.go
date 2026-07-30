package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func checkInArgs(args map[string]interface{}) mcp.CallToolRequest { return mcpRequest(args) }

func TestHandleListCheckIns(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/check_ins"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{"data":[{"id":"c1","name":"Nightly","slug":"nightly"}],
		  "pagination":{"page":1,"per_page":25,"total_count":1,"total_pages":1}}`)
	})

	result, err := handleListCheckIns(context.Background(), client,
		checkInArgs(map[string]interface{}{"project_id": "Xk9mZp"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if !strings.Contains(getResultText(result), "Nightly") {
		t.Errorf("result = %q", getResultText(result))
	}
}

func TestHandleGetCheckIn(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/check_ins/c1"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{"data":{"id":"c1","name":"Nightly"}}`)
	})

	result, err := handleGetCheckIn(context.Background(), client,
		checkInArgs(map[string]interface{}{"project_id": "Xk9mZp", "check_in_id": "c1"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
}

func TestHandleCheckInMissingIDs(t *testing.T) {
	for name, call := range map[string]func() (*mcp.CallToolResult, error){
		"get": func() (*mcp.CallToolResult, error) {
			return handleGetCheckIn(context.Background(), offlineV3Client(), checkInArgs(nil))
		},
		"delete": func() (*mcp.CallToolResult, error) {
			return handleDeleteCheckIn(context.Background(), offlineV3Client(), checkInArgs(nil))
		},
	} {
		result, err := call()
		if err != nil {
			t.Fatalf("%s: error = %v", name, err)
		}
		if !result.IsError || !strings.Contains(getResultText(result), "project_id is required") {
			t.Errorf("%s: got %q", name, getResultText(result))
		}
	}
}

func TestHandleCreateCheckIn(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusCreated, `{"data":{"id":"c1","name":"Nightly"}}`)
	})

	result, err := handleCreateCheckIn(context.Background(), client, checkInArgs(map[string]interface{}{
		"project_id":    "Xk9mZp",
		"name":          "Nightly",
		"schedule_type": "simple",
		"report_period": "1d",
		"grace_period":  "1h",
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	for key, want := range map[string]any{
		"name": "Nightly", "schedule_type": "simple", "report_period": "1d", "grace_period": "1h",
	} {
		if body[key] != want {
			t.Errorf("%s = %v, want %v", key, body[key], want)
		}
	}
}

// An update sends only what it was given, so unset fields keep their values.
func TestHandleUpdateCheckInOmitsUnsetFields(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusOK, `{"data":{"id":"c1","name":"Nightly"}}`)
	})

	result, err := handleUpdateCheckIn(context.Background(), client, checkInArgs(map[string]interface{}{
		// name comes along because the update body requires it.
		"project_id": "Xk9mZp", "check_in_id": "c1", "name": "Nightly", "grace_period": "10m",
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if body["grace_period"] != "10m" {
		t.Errorf("grace_period = %v", body["grace_period"])
	}
	if body["name"] != "Nightly" {
		t.Errorf("name = %v; the update body requires it", body["name"])
	}
	for _, absent := range []string{"schedule_type", "report_period", "cron_schedule"} {
		if _, present := body[absent]; present {
			t.Errorf("%q was sent unset; it would blank the field", absent)
		}
	}
}

func TestHandleDeleteCheckIn(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	result, err := handleDeleteCheckIn(context.Background(), client,
		checkInArgs(map[string]interface{}{"project_id": "Xk9mZp", "check_in_id": "c1"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if !strings.Contains(getResultText(result), "c1") {
		t.Errorf("result should name what was deleted: %q", getResultText(result))
	}
}

// A simple check-in is unaffected.
func TestHandleCreateCheckInAllowsSimpleSchedule(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusCreated, `{"data":{"id":"c1","name":"Nightly"}}`)
	})

	result, err := handleCreateCheckIn(context.Background(), client, checkInArgs(map[string]interface{}{
		"project_id": "Xk9mZp", "name": "Nightly", "schedule_type": "simple", "report_period": "1d",
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("a simple check-in was refused: %s", getResultText(result))
	}
}

// Cron check-ins are expressible now: the schema carries the schedule and its
// timezone.
func TestHandleCreateCheckInSendsCronSchedule(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusCreated, `{"data":{"id":"c1","name":"Nightly"}}`)
	})

	result, err := handleCreateCheckIn(context.Background(), client, checkInArgs(map[string]interface{}{
		"project_id":    "Xk9mZp",
		"name":          "Nightly",
		"schedule_type": "cron",
		"cron_schedule": "0 3 * * *",
		// A Rails zone name, not an IANA identifier — the API rejects the latter.
		"cron_timezone": "Central Time (US & Canada)",
		"slug":          "nightly",
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}

	for key, want := range map[string]any{
		"schedule_type": "cron",
		"cron_schedule": "0 3 * * *",
		"cron_timezone": "Central Time (US & Canada)",
		"slug":          "nightly",
	} {
		if body[key] != want {
			t.Errorf("%s = %v, want %v", key, body[key], want)
		}
	}
}

// A cron schedule_type without the schedule is caught here rather than by the API.
func TestHandleCreateCheckInRequiresCronSchedule(t *testing.T) {
	result, err := handleCreateCheckIn(context.Background(), offlineV3Client(),
		checkInArgs(map[string]interface{}{
			"project_id": "Xk9mZp", "name": "Nightly", "schedule_type": "cron",
		}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "cron_schedule is required") {
		t.Errorf("got %q", getResultText(result))
	}
}

// The update body is the same schema as create, with name required.
func TestHandleUpdateCheckInRequiresName(t *testing.T) {
	result, err := handleUpdateCheckIn(context.Background(), offlineV3Client(),
		checkInArgs(map[string]interface{}{
			"project_id": "Xk9mZp", "check_in_id": "c1", "grace_period": "5m",
		}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "name is required") {
		t.Errorf("got %q", getResultText(result))
	}
}
