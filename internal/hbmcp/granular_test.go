package hbmcp

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/server"
)

// Granular (scope version 2) OAuth tokens carry permission names such as
// "faults:write" instead of read/write. When introspection is unavailable the
// JWT is all this server knows, and for these tokens it is exact.

func oauthCtx(claims *Claims) context.Context {
	ctx := WithCredentialKind(context.Background(), KindOAuth)
	return WithClaims(ctx, claims)
}

var httpConfig = &config.Config{TransportMode: config.TransportHTTP}

func TestEffectiveReadOnlyGranularTokenWithWritePermission(t *testing.T) {
	for _, scopes := range [][]string{
		{"faults:read", "faults:write"},
		{"faults:write"},
		{"projects:create"},
	} {
		ctx := oauthCtx(&Claims{ScopeVersion: 2, Scopes: scopes})
		if EffectiveReadOnly(ctx, httpConfig) {
			t.Errorf("version 2 token %v lost its write tools without introspection", scopes)
		}
	}
}

func TestEffectiveReadOnlyGranularTokenWithoutWritePermission(t *testing.T) {
	ctx := oauthCtx(&Claims{ScopeVersion: 2, Scopes: []string{"faults:read", "insights:read"}})
	if !EffectiveReadOnly(ctx, httpConfig) {
		t.Error("a read-only version 2 token was offered the write tools")
	}
}

// Tokens minted before the scope_version claim existed are told apart by their
// scope names.
func TestEffectiveReadOnlyGranularTokenWithoutVersionClaim(t *testing.T) {
	if EffectiveReadOnly(oauthCtx(&Claims{Scopes: []string{"faults:write"}}), httpConfig) {
		t.Error("an unversioned granular token with faults:write lost its write tools")
	}
	if !EffectiveReadOnly(oauthCtx(&Claims{Scopes: []string{"faults:read"}}), httpConfig) {
		t.Error("an unversioned granular token with only faults:read was offered the write tools")
	}
}

// Version 1 tokens behave exactly as before: only "write" grants writing. A
// stray granular name on one was never honored by the API, so it must not
// unlock the write tools here either.
func TestEffectiveReadOnlyVersion1TokenUnchanged(t *testing.T) {
	for _, tc := range []struct {
		scopes []string
		want   bool
	}{
		{[]string{"read", "write"}, false},
		{[]string{"read"}, true},
		{[]string{"read", "faults:write"}, true},
		{nil, true},
	} {
		ctx := oauthCtx(&Claims{ScopeVersion: 1, Scopes: tc.scopes})
		if got := EffectiveReadOnly(ctx, httpConfig); got != tc.want {
			t.Errorf("version 1 token %v: EffectiveReadOnly = %v, want %v", tc.scopes, got, tc.want)
		}
	}
}

// Introspection still wins over the JWT when it is available.
func TestEffectiveReadOnlyIntrospectionOutranksGranularClaims(t *testing.T) {
	ctx := oauthCtx(&Claims{ScopeVersion: 2, Scopes: []string{"faults:write"}})
	ctx = WithTokenInfo(ctx, &apiv3.TokenInfo{Scopes: []string{"faults:read"}})
	if !EffectiveReadOnly(ctx, httpConfig) {
		t.Error("introspection said read-only, but the JWT's write permission was used")
	}
}

func listToolNames(t *testing.T, s *server.MCPServer, ctx context.Context) []string {
	t.Helper()
	var list struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	rpc(t, s, ctx, "tools/list", map[string]any{}, &list)
	names := make([]string, 0, len(list.Result.Tools))
	for _, tool := range list.Result.Tools {
		names = append(names, tool.Name)
	}
	return names
}

func newHTTPServer(t *testing.T) *server.MCPServer {
	t.Helper()
	return NewServer(&config.Config{
		APIURL:        "http://127.0.0.1:0",
		LogLevel:      "error",
		TransportMode: config.TransportHTTP,
	}, "test")
}

// A tool needing faults:write is offered only to a token holding it, judged
// from the JWT when introspection is unavailable.
func TestToolsListGranularTokenWithoutIntrospection(t *testing.T) {
	s := newHTTPServer(t)

	t.Run("faults:write", func(t *testing.T) {
		names := listToolNames(t, s, oauthCtx(&Claims{ScopeVersion: 2, Scopes: []string{"faults:read", "faults:write"}}))
		for _, want := range []string{"list_faults", "get_fault", "update_fault", "get_reference", "search_tools"} {
			if !slices.Contains(names, want) {
				t.Errorf("%s missing from %v", want, names)
			}
		}
		for _, hidden := range []string{"create_project", "list_dashboards", "create_dashboard", "create_check_in", "query_insights"} {
			if slices.Contains(names, hidden) {
				t.Errorf("%s offered to a token without its permission", hidden)
			}
		}
	})

	t.Run("faults:read only", func(t *testing.T) {
		names := listToolNames(t, s, oauthCtx(&Claims{ScopeVersion: 2, Scopes: []string{"faults:read"}}))
		if !slices.Contains(names, "list_faults") {
			t.Errorf("list_faults missing from %v", names)
		}
		if slices.Contains(names, "update_fault") {
			t.Error("update_fault offered to a token without faults:write")
		}
	})

	t.Run("faults:write without faults:read", func(t *testing.T) {
		// Granular permissions don't imply each other.
		names := listToolNames(t, s, oauthCtx(&Claims{ScopeVersion: 2, Scopes: []string{"faults:write"}}))
		if !slices.Contains(names, "update_fault") {
			t.Errorf("update_fault missing from %v", names)
		}
		if slices.Contains(names, "list_faults") {
			t.Error("list_faults offered to a token without faults:read")
		}
	})
}

// Version 1 tokens see what they saw before: everything with write, the
// read-only tools with read.
func TestToolsListVersion1TokenUnchanged(t *testing.T) {
	s := newHTTPServer(t)
	all := listToolNames(t, s, WithClaims(context.Background(), &Claims{Scopes: []string{"read", "write"}}))

	for _, version := range []int{0, 1} {
		got := listToolNames(t, s, oauthCtx(&Claims{ScopeVersion: version, Scopes: []string{"read", "write"}}))
		if !slices.Equal(got, all) {
			t.Errorf("version %d read/write token: tools changed\n got %v\nwant %v", version, got, all)
		}
		readOnly := listToolNames(t, s, oauthCtx(&Claims{ScopeVersion: version, Scopes: []string{"read"}}))
		for _, want := range []string{"list_faults", "list_dashboards", "list_projects", "query_insights"} {
			if !slices.Contains(readOnly, want) {
				t.Errorf("version %d read token lost %s", version, want)
			}
		}
		if slices.Contains(readOnly, "update_fault") {
			t.Errorf("version %d read token was offered update_fault", version)
		}
	}
}

// search_tools must apply the same rule as tools/list.
func TestSearchToolsGranularTokenWithoutIntrospection(t *testing.T) {
	s := newHTTPServer(t)
	ctx := oauthCtx(&Claims{ScopeVersion: 2, Scopes: []string{"faults:read", "faults:write"}})

	text, isErr := callTool(t, s, ctx, "search_tools", map[string]any{"query": "dashboard"})
	if isErr {
		t.Fatalf("search_tools failed: %s", text)
	}
	for _, hidden := range []string{"list_dashboards", "get_dashboard", "create_dashboard"} {
		if slices.Contains(strings.Split(text, "\n"), "Name: "+hidden) {
			t.Errorf("search surfaced %s to a token without dashboards permissions", hidden)
		}
	}

	text, _ = callTool(t, s, ctx, "search_tools", map[string]any{"query": "fault"})
	if !slices.Contains(strings.Split(text, "\n"), "Name: update_fault") {
		t.Errorf("search hid update_fault from a token holding faults:write:\n%s", text)
	}
}
