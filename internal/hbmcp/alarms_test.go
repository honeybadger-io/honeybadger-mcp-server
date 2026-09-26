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
		if want := "/v3/projects/Xk9mZp/alarms"; r.URL.Path != want {
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
		if want := "/v3/projects/Xk9mZp/alarms/a1"; r.URL.Path != want {
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

// An alarm can be created with the query and trigger that make it fire.
func TestHandleCreateAlarm(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusCreated, `{"data":{"id":"a1","name":"Error spike",
			"links":{"web":"https://app.honeybadger.io/projects/123/insights/alarms/a1"}}}`)
	})

	result, err := handleCreateAlarm(context.Background(), client, alarmArgs(map[string]interface{}{
		"project_id":        "Xk9mZp",
		"name":              "Error spike",
		"query":             "count() > 100",
		"evaluation_period": "5m",
		"lookback_lag":      "1m",
		"trigger_config":    `{"type":"threshold","config":{"operator":">","value":100}}`,
		"stream_ids":        `["str_1"]`,
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}

	if body["query"] != "count() > 100" {
		t.Errorf("query = %v", body["query"])
	}
	trigger, ok := body["trigger_config"].(map[string]any)
	if !ok || trigger["type"] != "threshold" {
		t.Fatalf("trigger_config = %v", body["trigger_config"])
	}
	streams, ok := body["stream_ids"].([]any)
	if !ok || len(streams) != 1 {
		t.Errorf("stream_ids = %v", body["stream_ids"])
	}
	// The UI link rides along in links.web, so the caller can hand it to the user.
	if !strings.Contains(getResultText(result), `"links":{"web":"https://app.honeybadger.io/projects/123/insights/alarms/a1"}`) {
		t.Errorf("result should carry the alarm's UI link, got %s", getResultText(result))
	}
}

// A query is what makes an alarm fire, so it is required rather than optional.
func TestHandleCreateAlarmRequiresQuery(t *testing.T) {
	result, err := handleCreateAlarm(context.Background(), offlineV3Client(),
		alarmArgs(map[string]interface{}{"project_id": "Xk9mZp", "name": "Spike"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "query is required") {
		t.Errorf("got %q", getResultText(result))
	}
}

// Malformed trigger JSON is caught before a request is made.
func TestHandleCreateAlarmRejectsInvalidTriggerJSON(t *testing.T) {
	result, err := handleCreateAlarm(context.Background(), offlineV3Client(),
		alarmArgs(map[string]interface{}{
			"project_id": "Xk9mZp", "name": "Spike", "query": "count() > 1",
			"trigger_config": "{not json",
		}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "trigger_config") {
		t.Errorf("got %q", getResultText(result))
	}
}

// Renaming is a genuinely useful subset of update, so it is allowed.
func TestHandleUpdateAlarmRenames(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/projects/Xk9mZp/alarms/a1"; r.URL.Path != want {
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

// An alarm's query, window and streams change in place, keeping its history.
func TestHandleUpdateAlarmChangesBehaviour(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusOK, `{"data":{"id":"a1","name":"Spike"}}`)
	})

	result, err := handleUpdateAlarm(context.Background(), client, alarmArgs(map[string]interface{}{
		"project_id": "Xk9mZp", "alarm_id": "a1",
		"query": "count() > 5", "evaluation_period": "10m", "stream_ids": `["s1"]`,
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if body["query"] != "count() > 5" || body["evaluation_period"] != "10m" {
		t.Errorf("body = %v", body)
	}
	if ids, ok := body["stream_ids"].([]any); !ok || len(ids) != 1 || ids[0] != "s1" {
		t.Errorf("stream_ids = %v", body["stream_ids"])
	}
	for _, absent := range []string{"name", "description", "lookback_lag", "trigger_config"} {
		if _, present := body[absent]; present {
			t.Errorf("%s was sent though it was not supplied", absent)
		}
	}
}

// The trigger has no typed update field, so a request touching it is refused
// rather than applying the other fields and reporting success.
func TestHandleUpdateAlarmRefusesTrigger(t *testing.T) {
	result, err := handleUpdateAlarm(context.Background(), noRequestClient(t),
		alarmArgs(map[string]interface{}{
			"project_id": "Xk9mZp", "alarm_id": "a1", "name": "N", "trigger_config": `{"type":"alert_result_count"}`,
		}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "trigger_config") {
		t.Errorf("expected trigger_config to be refused by name, got %s", getResultText(result))
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
		alarmArgs(map[string]interface{}{"project_id": "Xk9mZp", "alarm_id": "a1",
			"confirm": validConfirm("delete_alarm", "Xk9mZp", "a1")}))
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
		if want := "/v3/projects/Xk9mZp/alarms/a1/history"; r.URL.Path != want {
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
	var got struct {
		Results    []map[string]any `json:"results"`
		Page       int              `json:"page"`
		TotalPages int              `json:"total_pages"`
	}
	if err := json.Unmarshal([]byte(getResultText(result)), &got); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if len(got.Results) != 1 || got.Results[0]["state"] != "triggered" {
		t.Errorf("results = %v", got.Results)
	}
	// total_pages is the only way to know whether more history exists.
	if got.Page != 2 || got.TotalPages != 2 {
		t.Errorf("page %d of %d, want 2 of 2", got.Page, got.TotalPages)
	}
}
