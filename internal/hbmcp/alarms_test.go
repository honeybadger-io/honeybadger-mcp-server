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
			"evaluation_period": "5m", "lookback_lag": "1m", "trigger_config": "{not json",
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

// The trigger changes in place too, with a fractional threshold intact.
func TestHandleUpdateAlarmChangesTrigger(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusOK, `{"data":{"id":"a1","name":"Spike"}}`)
	})

	result, err := handleUpdateAlarm(context.Background(), client, alarmArgs(map[string]interface{}{
		"project_id": "Xk9mZp", "alarm_id": "a1",
		"trigger_config": `{"type":"alert_result_count","config":{"operator":"gte","value":0.5}}`,
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	trigger, ok := body["trigger_config"].(map[string]any)
	if !ok || trigger["type"] != "alert_result_count" {
		t.Fatalf("trigger_config = %v", body["trigger_config"])
	}
	if config, ok := trigger["config"].(map[string]any); !ok || config["operator"] != "gte" || config["value"] != 0.5 {
		t.Errorf("trigger config = %v", trigger["config"])
	}
}

func TestHandleUpdateAlarmRejectsInvalidTriggerJSON(t *testing.T) {
	result, err := handleUpdateAlarm(context.Background(), noRequestClient(t),
		alarmArgs(map[string]interface{}{
			"project_id": "Xk9mZp", "alarm_id": "a1", "trigger_config": `{not json`,
		}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError {
		t.Errorf("expected an error for malformed trigger_config, got %s", getResultText(result))
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
		v3JSON(w, http.StatusOK, `{"data":[{"id":"t1","observer_root_id":"a1","observer_id":"v7","status":"alarm",
		    "created_at":"2026-09-26T00:00:00Z","evaluation_started_at":"2026-09-25T23:55:00Z","evaluation_result":91.5}],
		  "pagination":{"page":2,"per_page":25},
		  "links":{"self":"/x?page=2","next":"/x?page=3"}}`)
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
		Data []struct {
			Status           string  `json:"status"`
			EvaluationResult float64 `json:"evaluation_result"`
		} `json:"data"`
		Links struct {
			Next *string `json:"next"`
		} `json:"links"`
	}
	if err := json.Unmarshal([]byte(getResultText(result)), &got); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if len(got.Data) != 1 || got.Data[0].Status != "alarm" || got.Data[0].EvaluationResult != 91.5 {
		t.Errorf("data = %+v", got.Data)
	}
	// links.next is how the caller knows more history exists.
	if got.Links.Next == nil {
		t.Error("links.next was dropped from the response")
	}
}
