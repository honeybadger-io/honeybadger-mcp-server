package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func alarmArgs(args map[string]interface{}) mcp.CallToolRequest { return mcpRequest(args) }

func TestHandleListAlarms(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/alarms"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		// Alarms are unpaginated: one call is the whole collection.
		if r.URL.RawQuery != "" {
			t.Errorf("query = %q, want none", r.URL.RawQuery)
		}
		v3JSON(w, http.StatusOK, `{"data":[
			{"id":"a1","name":"Error spike","project_id":"Xk9mZp"},
			{"id":"a2","name":"Latency","project_id":"Xk9mZp"}
		]}`)
	})

	result, err := handleListAlarms(context.Background(), client,
		alarmArgs(map[string]interface{}{"project_id": "Xk9mZp"}))
	if err != nil {
		t.Fatalf("handleListAlarms() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if !strings.Contains(getResultText(result), "Error spike") {
		t.Errorf("result = %q", getResultText(result))
	}
}

func TestHandleListAlarms_MissingProjectID(t *testing.T) {
	result, err := handleListAlarms(context.Background(), offlineV3Client(), alarmArgs(nil))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "project_id is required") {
		t.Errorf("got %q", getResultText(result))
	}
}

func TestHandleGetAlarm(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/alarms/a1"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{"data":{"id":"a1","name":"Error spike","project_id":"Xk9mZp"}}`)
	})

	result, err := handleGetAlarm(context.Background(), client,
		alarmArgs(map[string]interface{}{"project_id": "Xk9mZp", "alarm_id": "a1"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
}

func TestHandleGetAlarm_MissingAlarmID(t *testing.T) {
	result, err := handleGetAlarm(context.Background(), offlineV3Client(),
		alarmArgs(map[string]interface{}{"project_id": "Xk9mZp"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "alarm_id is required") {
		t.Errorf("got %q", getResultText(result))
	}
}

// v3's alarm create schema accepts only a name, so an alarm made through it
// could never fire. Refusing beats leaving a broken alarm in the account.
func TestHandleCreateAlarmRefusesUntilV3CanExpressOne(t *testing.T) {
	result, err := handleCreateAlarm(context.Background(), offlineV3Client(),
		alarmArgs(map[string]interface{}{
			"project_id":        "Xk9mZp",
			"name":              "Error spike",
			"query":             "count() > 100",
			"evaluation_period": "5m",
			"trigger_config":    `{"threshold":100}`,
			"lookback_lag":      "1m",
		}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError {
		t.Fatal("create was allowed; it would have made an alarm that never fires")
	}
	text := getResultText(result)
	for _, want := range []string{"not available on the v3 API yet", "never fires"} {
		if !strings.Contains(text, want) {
			t.Errorf("error should explain why: %q", text)
		}
	}
}

// Renaming is a genuinely useful subset of update, so it is allowed.
func TestHandleUpdateAlarmRenames(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/alarms/a1"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusOK, `{"data":{"id":"a1","name":"Renamed","project_id":"Xk9mZp"}}`)
	})

	result, err := handleUpdateAlarm(context.Background(), client, alarmArgs(map[string]interface{}{
		"project_id": "Xk9mZp", "alarm_id": "a1", "name": "Renamed",
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if body["name"] != "Renamed" {
		t.Errorf("sent body = %v", body)
	}
}

// Touching an alarm's configuration is refused rather than applying only the
// name and reporting success.
func TestHandleUpdateAlarmRejectsConfigChanges(t *testing.T) {
	for field, value := range map[string]interface{}{
		"query":             "count() > 5",
		"evaluation_period": "10m",
		"trigger_config":    `{"threshold":5}`,
		"lookback_lag":      "2m",
		"stream_ids":        `["s1"]`,
		"description":       "watch this",
	} {
		result, err := handleUpdateAlarm(context.Background(), offlineV3Client(),
			alarmArgs(map[string]interface{}{
				"project_id": "Xk9mZp", "alarm_id": "a1", "name": "N", field: value,
			}))
		if err != nil {
			t.Fatalf("%s: error = %v", field, err)
		}
		if !result.IsError {
			t.Errorf("%s: accepted; only the name would have been applied", field)
			continue
		}
		if !strings.Contains(getResultText(result), field) {
			t.Errorf("%s: error does not name it: %q", field, getResultText(result))
		}
	}
}

func TestHandleDeleteAlarm(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	result, err := handleDeleteAlarm(context.Background(), client,
		alarmArgs(map[string]interface{}{"project_id": "Xk9mZp", "alarm_id": "a1"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if !strings.Contains(getResultText(result), "a1") {
		t.Errorf("result should name what was deleted: %q", getResultText(result))
	}
}

func TestHandleGetAlarmHistory(t *testing.T) {
	var query string
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/alarms/a1/history"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		query = r.URL.RawQuery
		// Rows come from the query service, so they stay untyped.
		v3JSON(w, http.StatusOK, `{"data":[{"state":"triggered","value":91.5}],
		  "pagination":{"page":2,"total_pages":2}}`)
	})

	result, err := handleGetAlarmHistory(context.Background(), client, alarmArgs(map[string]interface{}{
		"project_id": "Xk9mZp", "alarm_id": "a1", "page": 2,
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if !strings.Contains(query, "page=2") {
		t.Errorf("query = %q, want page=2", query)
	}
	// This endpoint takes no per_page.
	if strings.Contains(query, "per_page") {
		t.Errorf("query = %q, must not send per_page", query)
	}
	if !strings.Contains(getResultText(result), "triggered") {
		t.Errorf("result = %q", getResultText(result))
	}
}
