package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// A project can hold several integrations of one type, so the preview names the
// label as well as the type.
func TestDeleteIntegrationPreviewNamesTheLabel(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want only the preview lookup", r.Method)
		}
		v3JSON(w, http.StatusOK, `{"data":{"id":"int1","project_id":"Xk9mZp","type":"WebHook","active":true,
		  "events":[],"site_ids":[],"check_in_ids":[],"excluded_environments":[],"alarm_alert_ids":[],"alarm_ok_ids":[],
		  "config":{"label":"Deploy hook","url":"https://example.com/hook"}}}`)
	})

	result, err := handleDeleteIntegration(context.Background(), client, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"project_id": "Xk9mZp", "integration_id": "int1"}},
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if text := getResultText(result); !strings.Contains(text, `WebHook integration "Deploy hook"`) {
		t.Errorf("preview = %q, want it to name the label", text)
	}
}

func integrationArgs(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

// The shared settings go beside config and the type's settings inside it, the
// shape the API takes.
func TestCreateIntegrationSendsTheAPIShape(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusCreated, `{"data":{"id":"i1","project_id":"Xk9mZp","type":"WebHook","active":true,"links":{"web":"https://app/x"}}}`)
	})

	result, err := handleCreateIntegration(context.Background(), client, integrationArgs(map[string]any{
		"project_id": "Xk9mZp", "type": "WebHook",
		"events":       []any{"occurred", "resolved"},
		"check_in_ids": []any{},
		"threshold":    float64(10),
		"config":       `{"url":"https://example.com/hook","label":"Deploys"}`,
	}))
	if err != nil || result.IsError {
		t.Fatalf("create = %s, %v", getResultText(result), err)
	}
	if body["type"] != "WebHook" || body["threshold"] != float64(10) {
		t.Errorf("body = %v", body)
	}
	// An explicit empty list is a setting (check-in notifications off), not an omission.
	if ids, ok := body["check_in_ids"].([]any); !ok || len(ids) != 0 {
		t.Errorf("check_in_ids = %v, want []", body["check_in_ids"])
	}
	config, _ := body["config"].(map[string]any)
	if config["url"] != "https://example.com/hook" {
		t.Errorf("config = %v", body["config"])
	}
	if _, flat := body["url"]; flat {
		t.Error("url was sent outside config")
	}
}

// The old flat shape put shared settings inside config; the API now refuses
// them there, so the tool says where they go instead.
func TestCreateIntegrationRefusesSharedSettingsInsideConfig(t *testing.T) {
	result, err := handleCreateIntegration(context.Background(), noRequestClient(t), integrationArgs(map[string]any{
		"project_id": "Xk9mZp", "type": "WebHook",
		"config": `{"url":"https://example.com/hook","events":["occurred"]}`,
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "events is its own parameter") {
		t.Errorf("got %s", getResultText(result))
	}
}

// Values the API types strictly are checked before sending, and the refusal is
// in the tool's terms rather than Go's.
func TestIntegrationSettingsAreTypeChecked(t *testing.T) {
	for name, tc := range map[string]struct {
		args map[string]any
		want string
	}{
		"fractional threshold": {map[string]any{"threshold": 2.5}, "threshold must be a whole number"},
		"active as a string":   {map[string]any{"active": "yes"}, "active must be true or false"},
		"events not a list":    {map[string]any{"events": "occurred"}, "events must be a list of strings"},
		"filters not a list":   {map[string]any{"filters": "occurred"}, `filters must be a list of {"event"`},
		"site id not a uuid":   {map[string]any{"site_ids": []any{"not-a-uuid"}}, "site_ids must be site IDs"},
	} {
		t.Run(name, func(t *testing.T) {
			tc.args["project_id"], tc.args["integration_id"] = "Xk9mZp", "i1"
			result, err := handleUpdateIntegration(context.Background(), noRequestClient(t), integrationArgs(tc.args))
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if !result.IsError || !strings.Contains(getResultText(result), tc.want) {
				t.Errorf("got %q, want it to say %q", getResultText(result), tc.want)
			}
		})
	}
}

func TestUpdateIntegrationNeedsSomethingToChange(t *testing.T) {
	result, err := handleUpdateIntegration(context.Background(), noRequestClient(t), integrationArgs(map[string]any{
		"project_id": "Xk9mZp", "integration_id": "i1",
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError {
		t.Errorf("expected a refusal, got %s", getResultText(result))
	}
}
