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
		if want := "/v3/accounts/me/projects/Xk9mZp/dashboards"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{"data":[{"id":"d1","title":"Ops","project_id":"Xk9mZp"}],
		  "pagination":{"page":1,"per_page":25,"total_count":1,"total_pages":1}}`)
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
		if want := "/v3/accounts/me/projects/Xk9mZp/dashboards/d1"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{"data":{"id":"d1","title":"Ops","project_id":"Xk9mZp"}}`)
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

// The resource is read as title but written as name, so the tool accepts title —
// which is what v2 used and what a read returns — and sends name.
func TestHandleCreateDashboardMapsTitleToName(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusCreated, `{"data":{"id":"d1","title":"Ops","project_id":"Xk9mZp"}}`)
	})

	result, err := handleCreateDashboard(context.Background(), client,
		dashboardArgs(map[string]interface{}{"project_id": "Xk9mZp", "title": "Ops"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if body["name"] != "Ops" {
		t.Errorf("sent body = %v, want the title under name", body)
	}
}

// A dashboard's widgets are the dashboard; v3 cannot send them, so a request
// carrying them is refused rather than silently creating an empty one.
func TestHandleCreateDashboardRejectsWidgets(t *testing.T) {
	for field, value := range map[string]interface{}{
		"widgets":    `[{"type":"chart"}]`,
		"default_ts": "1h",
	} {
		result, err := handleCreateDashboard(context.Background(), offlineV3Client(),
			dashboardArgs(map[string]interface{}{
				"project_id": "Xk9mZp", "title": "Ops", field: value,
			}))
		if err != nil {
			t.Fatalf("%s: error = %v", field, err)
		}
		if !result.IsError {
			t.Errorf("%s: accepted; an empty dashboard would have been created", field)
			continue
		}
		if !strings.Contains(getResultText(result), field) {
			t.Errorf("%s: error does not name it: %q", field, getResultText(result))
		}
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

func TestHandleDeleteDashboard(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	result, err := handleDeleteDashboard(context.Background(), client,
		dashboardArgs(map[string]interface{}{"project_id": "Xk9mZp", "dashboard_id": "d1"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
}
