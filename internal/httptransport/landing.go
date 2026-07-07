package httptransport

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"
	"strconv"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbmcp"
)

//go:embed landing.html.tmpl
var landingTemplate string

type LandingData struct {
	MCPURL  string
	Version string
	Tools   []hbmcp.ToolInfo
}

// NewLandingHandler serves a static informational page at "/" describing the
// MCP endpoint, client setup, and the tool catalog. The page is rendered once
// at construction so template problems fail at boot, not per-request.
func NewLandingHandler(data LandingData) (http.Handler, error) {
	tmpl, err := template.New("landing").Parse(landingTemplate)
	if err != nil {
		return nil, err
	}
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
