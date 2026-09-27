package hbmcp

import (
	"context"
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
