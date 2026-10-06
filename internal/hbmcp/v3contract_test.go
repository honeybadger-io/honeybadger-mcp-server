package hbmcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// bodyCapture records the JSON body of the one request a tool makes.
func bodyCapture(t *testing.T, response string) (*apiv3.Client, *map[string]any) {
	t.Helper()
	body := map[string]any{}
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("request body %q: %v", raw, err)
		}
		v3JSON(w, http.StatusOK, response)
	})
	return client, &body
}

func mustSucceed(t *testing.T, result *mcp.CallToolResult, err error) {
	t.Helper()
	if err != nil || result.IsError {
		t.Fatalf("result = %s, %v", getResultText(result), err)
	}
}

// null resets an alarm to every current stream, so it must reach the API as
// null rather than being dropped or sent as [].
func TestUpdateAlarmResetsStreamsWithNull(t *testing.T) {
	client, body := bodyCapture(t, `{"data":{"id":"a1","name":"Spike"}}`)
	result, err := handleUpdateAlarm(context.Background(), client, alarmArgs(map[string]any{
		"project_id": "Xk9mZp", "alarm_id": "a1", "stream_ids": "null",
	}))
	mustSucceed(t, result, err)
	if v, present := (*body)["stream_ids"]; !present || v != nil {
		t.Errorf("stream_ids = %v (present %v), want an explicit null", v, present)
	}
}

// An empty description clears it, sent as null; an absent one is left alone.
func TestUpdateAlarmClearsDescriptionWithEmptyString(t *testing.T) {
	client, body := bodyCapture(t, `{"data":{"id":"a1","name":"Spike"}}`)
	result, err := handleUpdateAlarm(context.Background(), client, alarmArgs(map[string]any{
		"project_id": "Xk9mZp", "alarm_id": "a1", "description": "",
	}))
	mustSucceed(t, result, err)
	if v, present := (*body)["description"]; !present || v != nil {
		t.Errorf("description = %v (present %v), want an explicit null", v, present)
	}
	if _, present := (*body)["name"]; present {
		t.Error("name was sent though it wasn't given")
	}
}

func TestCreateAlarmRequiresWindowAndTrigger(t *testing.T) {
	base := map[string]any{"project_id": "Xk9mZp", "name": "Spike", "query": "q",
		"evaluation_period": "5m", "lookback_lag": "1m",
		"trigger_config": `{"type":"alert_result_count","config":{"operator":"gt","value":1}}`}
	for _, missing := range []string{"evaluation_period", "lookback_lag", "trigger_config"} {
		args := map[string]any{}
		for k, v := range base {
			if k != missing {
				args[k] = v
			}
		}
		result, err := handleCreateAlarm(context.Background(), noRequestClient(t), alarmArgs(args))
		if err != nil || !result.IsError {
			t.Errorf("without %s: result = %s, %v; want it refused", missing, getResultText(result), err)
		}
	}
}

// Filters are event/query pairs, and all_sites follows every site.
func TestUpdateIntegrationSendsFiltersAndAllSites(t *testing.T) {
	client, body := bodyCapture(t, `{"data":{"id":"i1","project_id":"Xk9mZp","type":"WebHook","active":true,"links":{"web":"https://app/x"}}}`)
	result, err := handleUpdateIntegration(context.Background(), client, integrationArgs(map[string]any{
		"project_id": "Xk9mZp", "integration_id": "i1", "all_sites": true,
		"filters": []any{map[string]any{"event": "occurred", "query": "environment:production"}},
	}))
	mustSucceed(t, result, err)
	if (*body)["all_sites"] != true {
		t.Errorf("all_sites = %v, want true", (*body)["all_sites"])
	}
	filters, ok := (*body)["filters"].([]any)
	if !ok || len(filters) != 1 || filters[0].(map[string]any)["event"] != "occurred" {
		t.Errorf("filters = %v", (*body)["filters"])
	}
}

// The parameters filters replaced are refused, not silently dropped.
func TestIntegrationToolsRefuseTheOldFilterParameters(t *testing.T) {
	result, err := handleUpdateIntegration(context.Background(), noRequestClient(t), integrationArgs(map[string]any{
		"project_id": "Xk9mZp", "integration_id": "i1",
		"filter_events": []any{"occurred"}, "filter_queries": []any{"environment:production"},
	}))
	if err != nil || !result.IsError {
		t.Errorf("result = %s, %v; want it refused", getResultText(result), err)
	}
}

// An empty string clears a project setting, sent as null.
func TestUpdateProjectClearsASettingWithEmptyString(t *testing.T) {
	client, body := bodyCapture(t, `{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"App","active":true}}`)
	result, err := handleUpdateProject(context.Background(), client, projectArgs(map[string]any{
		"id": "Xk9mZp", "user_url": "",
	}))
	mustSucceed(t, result, err)
	if v, present := (*body)["user_url"]; !present || v != nil {
		t.Errorf("user_url = %v (present %v), want an explicit null", v, present)
	}
}

// A check-in needs no name: an unnamed one shows its ID.
func TestCreateCheckInWithoutAName(t *testing.T) {
	client, body := bodyCapture(t, `{"data":{"id":"c1","project_id":"Xk9mZp"}}`)
	result, err := handleCreateCheckIn(context.Background(), client, checkInArgs(map[string]any{
		"project_id": "Xk9mZp", "schedule_type": "simple", "report_period": "1 day",
	}))
	mustSucceed(t, result, err)
	if _, present := (*body)["name"]; present {
		t.Errorf("name = %v, want it left out", (*body)["name"])
	}
}
