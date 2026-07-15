package httptransport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbmcp"
)

func testLandingHandler(t *testing.T) http.Handler {
	t.Helper()
	h, err := NewLandingHandler(LandingData{
		MCPURL:  "https://eu-mcp.honeybadger.io/mcp",
		AppURL:  "https://eu-app.honeybadger.io",
		Version: "1.2.3",
		Tools: []hbmcp.ToolInfo{
			{Name: "list_projects", Description: "List all projects", ReadOnly: true},
			{Name: "create_project", Description: "Create a new project", ReadOnly: false},
			{Name: "create_alarm", Description: "Create a new Insights alarm. IMPORTANT: Call get_reference first for full alarm documentation.", ReadOnly: false},
		},
	})
	if err != nil {
		t.Fatalf("NewLandingHandler: %v", err)
	}
	return h
}

func TestLandingHandler_ServesRootHTML(t *testing.T) {
	h := testLandingHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"https://eu-mcp.honeybadger.io/mcp",
		`href="https://eu-app.honeybadger.io"`,
		"list_projects",
		"List all projects",
		"create_project",
		"docs.honeybadger.io",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}

	// Agent-facing guidance after the first sentence must not reach the page.
	if !strings.Contains(body, "Create a new Insights alarm.") {
		t.Error("body missing first sentence of create_alarm description")
	}
	if strings.Contains(body, "IMPORTANT") {
		t.Error("body leaks agent guidance past the first sentence")
	}
}

func TestLandingHandler_NotFoundForOtherPaths(t *testing.T) {
	h := testLandingHandler(t)

	for _, path := range []string{"/nope", "/index.html", "/mcp/extra"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want %d", path, rec.Code, http.StatusNotFound)
		}
	}
}

func TestLandingHandler_MethodNotAllowed(t *testing.T) {
	h := testLandingHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST / status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
