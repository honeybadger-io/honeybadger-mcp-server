package httptransport

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbmcp"
)

//go:embed landing.html.tmpl
var landingTemplate string

type LandingData struct {
	MCPURL string
	// Region-specific app origin for the sign-in link (the authorization
	// server: app.honeybadger.io in US, eu-app.honeybadger.io in EU).
	AppURL  string
	Version string
	Tools   []hbmcp.ToolInfo
}

// Tool descriptions are written for LLM consumption — after the first
// sentence they carry agent guidance ("IMPORTANT: call X first…") that
// reads as noise to people. Show only the human-meaningful summary.
func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1]
	}
	return s
}

// NewLandingHandler serves a static informational page at "/" describing the
// MCP endpoint, client setup, and the tool catalog. The page is rendered once
// at construction so template problems fail at boot, not per-request.
func NewLandingHandler(data LandingData) (http.Handler, error) {
	tmpl, err := template.New("landing").Parse(landingTemplate)
	if err != nil {
		return nil, err
	}
	tools := make([]hbmcp.ToolInfo, len(data.Tools))
	for i, t := range data.Tools {
		t.Description = firstSentence(t.Description)
		tools[i] = t
	}
	data.Tools = tools
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	page := buf.Bytes()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mounted at "/", this handler receives every path no other mux
		// pattern claims; only the root itself is a real page.
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(page)))
		_, _ = w.Write(page)
	}), nil
}
