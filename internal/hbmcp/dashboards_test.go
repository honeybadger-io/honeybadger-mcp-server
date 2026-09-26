package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func dashboardArgs(args map[string]interface{}) mcp.CallToolRequest { return mcpRequest(args) }

func TestHandleListDashboards(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/projects/Xk9mZp/dashboards"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{"data":[{"id":"d1","title":"Ops","project_id":"Xk9mZp","links":{"web":"https://app.honeybadger.io/projects/123/insights/dashboards/d1"}}],
		  "pagination":{"page":1,"per_page":25}}`)
	})

	result, err := handleListDashboards(context.Background(), client,
		dashboardArgs(map[string]interface{}{"project_id": "Xk9mZp"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if !strings.Contains(getResultText(result), "Ops") {
		t.Errorf("result = %q", getResultText(result))
	}
}

func TestHandleGetDashboard(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/projects/Xk9mZp/dashboards/d1"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{"data":{"id":"d1","title":"Ops","project_id":"Xk9mZp","links":{"web":"https://app.honeybadger.io/projects/123/insights/dashboards/d1"}}}`)
	})

	result, err := handleGetDashboard(context.Background(), client,
		dashboardArgs(map[string]interface{}{"project_id": "Xk9mZp", "dashboard_id": "d1"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
}

// title is the field on both sides now: the spec previously wrote it as name
// while reading it as title, and settled on title.
func TestHandleCreateDashboardSendsTitle(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusCreated, `{"data":{"id":"d1","title":"Ops","project_id":"Xk9mZp","links":{"web":"https://app.honeybadger.io/projects/123/insights/dashboards/d1"}}}`)
	})

	result, err := handleCreateDashboard(context.Background(), client,
		dashboardArgs(map[string]interface{}{"project_id": "Xk9mZp", "title": "Ops"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if body["title"] != "Ops" {
		t.Errorf("sent body = %v, want the title under title", body)
	}
}

func TestHandleCreateDashboardRequiresTitle(t *testing.T) {
	result, err := handleCreateDashboard(context.Background(), offlineV3Client(),
		dashboardArgs(map[string]interface{}{"project_id": "Xk9mZp"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "title is required") {
		t.Errorf("got %q", getResultText(result))
	}
}

func TestHandleUpdateDashboard(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusOK, `{"data":{"id":"d1","title":"Renamed","project_id":"Xk9mZp"}}`)
	})

	result, err := handleUpdateDashboard(context.Background(), client, dashboardArgs(map[string]interface{}{
		"project_id": "Xk9mZp", "dashboard_id": "d1", "title": "Renamed",
		// Update replaces rather than merges, so the widgets have to come along.
		"widgets": `[{"type":"errors"}]`,
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if body["title"] != "Renamed" {
		t.Errorf("sent body = %v", body)
	}
}

func TestHandleDeleteDashboard(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	result, err := handleDeleteDashboard(context.Background(), client,
		dashboardArgs(map[string]interface{}{"project_id": "Xk9mZp", "dashboard_id": "d1",
			"confirm": validConfirm("delete_dashboard", "Xk9mZp", "d1")}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
}

// Widgets are expressible now and travel through as raw JSON.
func TestHandleCreateDashboardSendsWidgets(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusCreated, `{"data":{"id":"d1","title":"Ops","project_id":"Xk9mZp","links":{"web":"https://app.honeybadger.io/projects/123/insights/dashboards/d1"}}}`)
	})

	result, err := handleCreateDashboard(context.Background(), client, dashboardArgs(map[string]interface{}{
		"project_id": "Xk9mZp",
		"title":      "Ops",
		"default_ts": "P1D",
		"widgets":    `[{"type":"errors","grid":{"x":0,"y":0,"w":6,"h":4}}]`,
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}

	if body["default_ts"] != "P1D" {
		t.Errorf("default_ts = %v", body["default_ts"])
	}
	widgets, ok := body["widgets"].([]any)
	if !ok || len(widgets) != 1 {
		t.Fatalf("widgets = %v", body["widgets"])
	}
	if widget, _ := widgets[0].(map[string]any); widget["type"] != "errors" {
		t.Errorf("widget = %v", widgets[0])
	}
}

// Malformed widget JSON is caught before a request is made.
func TestHandleCreateDashboardRejectsInvalidWidgetJSON(t *testing.T) {
	result, err := handleCreateDashboard(context.Background(), offlineV3Client(),
		dashboardArgs(map[string]interface{}{
			"project_id": "Xk9mZp", "title": "Ops", "widgets": "{not json",
		}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "valid JSON") {
		t.Errorf("got %q", getResultText(result))
	}
}

// Omitting widgets on update would clear the dashboard, since the API replaces
// rather than merges. The tool refuses instead of emptying it.
func TestHandleUpdateDashboardRequiresWidgets(t *testing.T) {
	result, err := handleUpdateDashboard(context.Background(), offlineV3Client(),
		dashboardArgs(map[string]interface{}{
			"project_id": "Xk9mZp", "dashboard_id": "d1", "title": "Renamed",
		}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError {
		t.Fatal("update without widgets was accepted; it would have cleared them")
	}
	if !strings.Contains(getResultText(result), "get_dashboard") {
		t.Errorf("error should say how to recover: %q", getResultText(result))
	}
}

// Dashboards carry their UI link in links.web, which reaches the tool output so
// the caller can hand it to the user.
func TestDashboardToolsCarryTheUILink(t *testing.T) {
	const link = `"links":{"web":"https://app.honeybadger.io/projects/123/insights/dashboards/d1"}`
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			v3JSON(w, http.StatusCreated, `{"data":{"id":"d1","title":"Ops","project_id":"Xk9mZp",`+link+`}}`)
		case r.URL.Path == "/v3/projects/Xk9mZp/dashboards":
			v3JSON(w, http.StatusOK, `{"data":[{"id":"d1","title":"Ops","project_id":"Xk9mZp",`+link+`}],"pagination":{"page":1,"per_page":25}}`)
		default:
			v3JSON(w, http.StatusOK, `{"data":{"id":"d1","title":"Ops","project_id":"Xk9mZp",`+link+`}}`)
		}
	})

	for name, call := range map[string]func() (*mcp.CallToolResult, error){
		"list": func() (*mcp.CallToolResult, error) {
			return handleListDashboards(context.Background(), client, dashboardArgs(map[string]interface{}{"project_id": "Xk9mZp"}))
		},
		"get": func() (*mcp.CallToolResult, error) {
			return handleGetDashboard(context.Background(), client, dashboardArgs(map[string]interface{}{"project_id": "Xk9mZp", "dashboard_id": "d1"}))
		},
		"create": func() (*mcp.CallToolResult, error) {
			return handleCreateDashboard(context.Background(), client, dashboardArgs(map[string]interface{}{"project_id": "Xk9mZp", "title": "Ops"}))
		},
	} {
		result, err := call()
		if err != nil || result.IsError {
			t.Fatalf("%s: %v %s", name, err, getResultText(result))
		}
		if !strings.Contains(getResultText(result), link) {
			t.Errorf("%s: result lacks the UI link: %s", name, getResultText(result))
		}
	}
}
