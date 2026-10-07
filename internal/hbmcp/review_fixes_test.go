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
