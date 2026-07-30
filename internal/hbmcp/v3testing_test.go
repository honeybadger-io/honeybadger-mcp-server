package hbmcp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// newV3TestClient starts a stub v3 API and returns a client pointed at it.
//
// Shared by the migrated tool tests. Two details matter and are easy to get
// wrong by hand: the response must declare a JSON content type, and v3 wraps
// every payload in a data envelope.
func newV3TestClient(t *testing.T, handler http.HandlerFunc) *apiv3.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return apiv3.NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_test")
}

// v3JSON writes a v3 response body with the content type the client requires.
func v3JSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	if status != 0 {
		w.WriteHeader(status)
	}
	_, _ = w.Write([]byte(body))
}

// offlineV3Client is for tests that never reach the network — argument
// validation, for instance. Its base URL is unroutable on purpose, so a test
// that accidentally issues a request fails rather than hitting production.
func offlineV3Client() *apiv3.Client {
	return apiv3.NewClient().WithBaseURL("http://127.0.0.1:1").WithBearerToken("hbt_test")
}

// mcpRequest builds a tool request from an argument map.
func mcpRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}
