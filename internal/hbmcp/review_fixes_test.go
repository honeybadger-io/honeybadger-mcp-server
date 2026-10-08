package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
)

// toolSchemas lists every registered tool's input schema properties.
func toolSchemas(t *testing.T) map[string]map[string]map[string]any {
	t.Helper()
	s := NewServer(&config.Config{AuthToken: "t", APIURL: "https://api.honeybadger.io/v2",
		LogLevel: "info", TransportMode: config.TransportStdio}, "test")
	raw, err := json.Marshal(s.HandleMessage(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)))
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				InputSchema struct {
					Properties map[string]map[string]any `json:"properties"`
				} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]map[string]any{}
	for _, tool := range parsed.Result.Tools {
		out[tool.Name] = tool.InputSchema.Properties
	}
	return out
}

func allowsNull(prop map[string]any) bool {
	types, _ := prop["type"].([]any)
	typeOK := false
	for _, ty := range types {
		typeOK = typeOK || ty == "null"
	}
	enum, hasEnum := prop["enum"].([]any)
	if !hasEnum {
		return typeOK
	}
	for _, v := range enum {
		if v == nil {
			return typeOK
		}
	}
	return false
}

// Where a tool promises null, its schema has to accept it, enums included, or a
// schema-checking client refuses the call before it's sent.
func TestNullablePromisesAreInTheSchema(t *testing.T) {
	schemas := toolSchemas(t)
	for tool, params := range map[string][]string{
		"update_site":        {"match_type", "request_method", "match", "locations", "frequency", "request_headers"},
		"update_integration": {"events", "filters", "site_ids", "check_in_ids", "rate"},
		"update_project":     {"user_url", "source_url", "user_search_field"},
	} {
		for _, param := range params {
			prop, ok := schemas[tool][param]
			if !ok {
				t.Errorf("%s has no %s parameter", tool, param)
				continue
			}
			if !allowsNull(prop) {
				t.Errorf("%s.%s doesn't accept null: %v", tool, param, prop)
			}
		}
	}
}

func TestUpdateSiteWithNothingToChange(t *testing.T) {
	result, err := handleUpdateSite(context.Background(), noRequestClient(t), mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "site_id": siteUUID,
	}))
	if err != nil || !result.IsError || !strings.Contains(getResultText(result), "at least one setting") {
		t.Errorf("result = %s, %v", getResultText(result), err)
	}
}

// Leaving site_ids out on create is what makes an integration follow every site,
// so it must not be sent; [] (none) must be.
func TestCreateIntegrationLeavesScopeOutUnlessGiven(t *testing.T) {
	client, body := bodyCapture(t, `{"data":{"id":"i1","project_id":"Xk9mZp","type":"WebHook","active":true,"links":{"web":"https://app/x"}}}`)
	result, err := handleCreateIntegration(context.Background(), client, integrationArgs(map[string]any{
		"project_id": "Xk9mZp", "type": "WebHook", "config": `{"url":"https://example.com/hook"}`, "check_in_ids": []any{},
	}))
	mustSucceed(t, result, err)
	if _, present := (*body)["site_ids"]; present {
		t.Errorf("site_ids = %v, want it left out", (*body)["site_ids"])
	}
	if v, ok := (*body)["check_in_ids"].([]any); !ok || len(v) != 0 {
		t.Errorf("check_in_ids = %v, want []", (*body)["check_in_ids"])
	}
}

// null clears a project setting just as an empty string does.
func TestUpdateProjectClearsASettingWithNull(t *testing.T) {
	client, body := bodyCapture(t, `{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"App","active":true}}`)
	result, err := handleUpdateProject(context.Background(), client, projectArgs(map[string]any{
		"id": "Xk9mZp", "source_url": nil,
	}))
	mustSucceed(t, result, err)
	if v, present := (*body)["source_url"]; !present || v != nil {
		t.Errorf("source_url = %v (present %v), want an explicit null", v, present)
	}
}

// With two bad settings the error always names the same one, with that one's
// own expected type.
func TestIntegrationSettingErrorsAreStable(t *testing.T) {
	for range 20 {
		result, err := handleUpdateIntegration(context.Background(), noRequestClient(t), integrationArgs(map[string]any{
			"project_id": "Xk9mZp", "integration_id": "i1", "threshold": "lots", "events": "occurred",
		}))
		if err != nil || getResultText(result) != "events must be a list of strings" {
			t.Fatalf("got %q, want the error for events", getResultText(result))
		}
	}
}

func TestPagingInputsAreRefusedRatherThanDropped(t *testing.T) {
	for name, args := range map[string]map[string]any{
		"fractional limit":     {"limit": 2.5},
		"created_before a URL": {"created_before": "/v3/projects/Xk9mZp/sites/x/outages?created_before=1704153600.5"},
	} {
		args["project_id"], args["site_id"] = "Xk9mZp", siteUUID
		result, err := handleListSiteOutages(context.Background(), noRequestClient(t), mcpRequest(args))
		if err != nil || !result.IsError {
			t.Errorf("%s: result = %s, %v; want it refused", name, getResultText(result), err)
		}
	}
}

func TestDeleteSitePreviewForAnUnnamedSite(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s; the preview should only read", r.Method)
		}
		v3JSON(w, http.StatusOK, `{"data":{"id":"`+siteUUID+`","project_id":"Xk9mZp","name":"","url":"https://example.com"}}`)
	})
	result, err := handleDeleteSite(context.Background(), client, mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "site_id": siteUUID,
	}))
	if err != nil || !strings.Contains(getResultText(result), "delete the unnamed site https://example.com") {
		t.Errorf("preview = %s, %v", getResultText(result), err)
	}
}

// On create, stream_ids null means every current stream, the same as leaving
// it out; the create input doesn't accept null.
func TestCreateAlarmTreatsNullStreamsAsOmitted(t *testing.T) {
	client, body := bodyCapture(t, `{"data":{"id":"a1","name":"Spike"}}`)
	result, err := handleCreateAlarm(context.Background(), client, alarmArgs(map[string]any{
		"project_id": "Xk9mZp", "name": "Spike", "query": "q", "evaluation_period": "5m", "lookback_lag": "1m",
		"trigger_config": `{"type":"alert_result_count","config":{"operator":"gt","value":1}}`, "stream_ids": "null",
	}))
	mustSucceed(t, result, err)
	if _, present := (*body)["stream_ids"]; present {
		t.Errorf("stream_ids = %v, want it left out", (*body)["stream_ids"])
	}
}

// purge_days arrives as a number or a numeric string; anything else is refused
// rather than dropped while the rest of the update succeeds.
func TestProjectPurgeDays(t *testing.T) {
	for _, sent := range []any{30.0, "30"} {
		client, body := bodyCapture(t, `{"data":{"id":"Xk9mZp","name":"New"}}`)
		result, err := handleUpdateProject(context.Background(), client, mcpRequest(map[string]any{
			"id": "Xk9mZp", "name": "New", "purge_days": sent,
		}))
		mustSucceed(t, result, err)
		if got := (*body)["purge_days"]; got != 30.0 {
			t.Errorf("sent %#v: purge_days = %#v, want 30", sent, got)
		}
	}
	for _, sent := range []any{0.0, -5.0, 2.5, "thirty", true} {
		result, err := handleUpdateProject(context.Background(), offlineV3Client(), mcpRequest(map[string]any{
			"id": "Xk9mZp", "name": "New", "purge_days": sent,
		}))
		if err != nil || !result.IsError {
			t.Errorf("sent %#v: result = %s, want an error", sent, getResultText(result))
		}
	}
}

// list_streams no longer filters; a cached schema's query is refused, not ignored.
func TestListStreamsRefusesQuery(t *testing.T) {
	result, err := handleListStreams(context.Background(), offlineV3Client(), mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "query": "prod",
	}))
	if err != nil || !result.IsError || !strings.Contains(getResultText(result), "query") {
		t.Errorf("result = %s, want query refused", getResultText(result))
	}
}

// name and url can't be reset, so null for either is refused rather than
// dropped while the rest of the update goes through.
func TestUpdateSiteRefusesNullNameAndURL(t *testing.T) {
	for _, field := range []string{"name", "url"} {
		result, err := handleUpdateSite(context.Background(), offlineV3Client(), mcpRequest(map[string]any{
			"project_id": "Xk9mZp", "site_id": "9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47", field: nil, "frequency": 1.0,
		}))
		if err != nil || !result.IsError || !strings.Contains(getResultText(result), field) {
			t.Errorf("%s: null: result = %s, want it refused", field, getResultText(result))
		}
	}
}

// Parameters documented as a string holding JSON also take the JSON value
// itself; reading only strings dropped it while the call reported success.
func TestJSONParametersTakeNativeValues(t *testing.T) {
	ctx := context.Background()

	client, body := bodyCapture(t, `{"data":{"id":"d1","title":"Ops","widgets":[]}}`)
	result, err := handleUpdateDashboard(ctx, client, mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "dashboard_id": "d1", "widgets": []any{map[string]any{"type": "errors"}},
	}))
	mustSucceed(t, result, err)
	if w, _ := (*body)["widgets"].([]any); len(w) != 1 {
		t.Errorf("update_dashboard widgets = %v, want the one widget sent", (*body)["widgets"])
	}

	client, body = bodyCapture(t, `{"data":{"id":"a1","name":"Spike"}}`)
	result, err = handleUpdateAlarm(ctx, client, alarmArgs(map[string]any{
		"project_id": "Xk9mZp", "alarm_id": "a1",
		"trigger_config": map[string]any{"type": "alert_result_count", "config": map[string]any{"operator": "gt", "value": 10.0}},
	}))
	mustSucceed(t, result, err)
	if _, sent := (*body)["trigger_config"]; !sent {
		t.Error("update_alarm trigger_config object was dropped")
	}

	client, body = bodyCapture(t, `{"data":{"id":"i1","type":"WebHook","active":true}}`)
	result, err = handleUpdateIntegration(ctx, client, mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "integration_id": "i1", "config": map[string]any{"url": "https://new.example.com"},
	}))
	mustSucceed(t, result, err)
	if config, _ := (*body)["config"].(map[string]any); config["url"] != "https://new.example.com" {
		t.Errorf("update_integration config = %v, want the url sent", (*body)["config"])
	}

	result, err = handleUpdateAlarm(ctx, offlineV3Client(), alarmArgs(map[string]any{
		"project_id": "Xk9mZp", "alarm_id": "a1", "name": "x", "trigger_config": 42.0,
	}))
	if err != nil || !result.IsError {
		t.Errorf("trigger_config 42 was accepted: %s", getResultText(result))
	}
}

// query_insights takes stream_ids as the alarm tools do, as a JSON string too,
// rather than running over every stream.
func TestQueryInsightsTakesStreamIDsAsAString(t *testing.T) {
	client, body := bodyCapture(t, `{"data":{"results":[],"fields":[],"meta":{}}}`)
	_, _ = handleQueryInsights(context.Background(), client, mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "query": "fields @ts", "stream_ids": `["s1"]`,
	}))
	if ids, _ := (*body)["stream_ids"].([]any); len(ids) != 1 || ids[0] != "s1" {
		t.Errorf("stream_ids = %v, want [s1]", (*body)["stream_ids"])
	}
}

// Project flags that aren't booleans are refused rather than dropped.
func TestUpdateProjectRefusesNonBooleanFlags(t *testing.T) {
	result, err := handleUpdateProject(context.Background(), offlineV3Client(), mcpRequest(map[string]any{
		"id": "Xk9mZp", "name": "X", "disable_public_links": "true",
	}))
	if err != nil || !result.IsError || !strings.Contains(getResultText(result), "disable_public_links") {
		t.Errorf("result = %s, want disable_public_links refused", getResultText(result))
	}
}
