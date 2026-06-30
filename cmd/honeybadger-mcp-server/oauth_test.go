package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizePublicURL(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"empty stays empty", "", "", false},
		{"plain origin", "https://mcp.honeybadger.io", "https://mcp.honeybadger.io", false},
		{"http origin with port", "http://localhost:9090", "http://localhost:9090", false},
		{"trailing slash stripped", "https://mcp.honeybadger.io/", "https://mcp.honeybadger.io", false},
		{"path rejected", "https://mcp.honeybadger.io/mcp", "", true},
		{"deep path rejected", "https://mcp.honeybadger.io/a/b", "", true},
		{"missing scheme", "mcp.honeybadger.io", "", true},
		{"missing host", "https://", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := normalizePublicURL(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestNormalizeEndpointPath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "/mcp"},
		{"mcp", "/mcp"},
		{"/mcp", "/mcp"},
		{"/mcp/", "/mcp"},
		{"mcp/", "/mcp"},
		{"/", "/"},
		{"/v1/mcp", "/v1/mcp"},
		{"/v1/mcp/", "/v1/mcp"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := normalizeEndpointPath(c.in); got != c.want {
				t.Errorf("normalizeEndpointPath(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestPRMHandler(t *testing.T) {
	h := prmHandler("https://host/mcp", []string{"https://auth.example"})

	t.Run("GET returns metadata", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/.well-known/oauth-protected-resource", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		var got struct {
			Resource             string   `json:"resource"`
			AuthorizationServers []string `json:"authorization_servers"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("body unmarshal: %v", err)
		}
		if got.Resource != "https://host/mcp" {
			t.Errorf("resource = %q", got.Resource)
		}
		if len(got.AuthorizationServers) != 1 || got.AuthorizationServers[0] != "https://auth.example" {
			t.Errorf("authorization_servers = %v", got.AuthorizationServers)
		}
	})

	t.Run("POST rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/.well-known/oauth-protected-resource", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
		if got := rec.Header().Get("Allow"); got != "GET" {
			t.Errorf("Allow = %q, want GET", got)
		}
	})
}

func TestChallengeMiddleware(t *testing.T) {
	prmURL := "https://host/.well-known/oauth-protected-resource"
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := challengeMiddleware(prmURL, next)

	cases := []struct {
		name     string
		header   string
		wantPass bool
	}{
		{"no header → 401", "", false},
		{"empty bearer → 401", "Bearer ", false},
		{"bearer with whitespace only → 401", "Bearer    ", false},
		{"non-bearer scheme → 401", "Basic xxx", false},
		{"valid bearer → passes", "Bearer jwt-abc", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest("POST", "/mcp", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			mw.ServeHTTP(rec, req)

			if c.wantPass {
				if !called {
					t.Fatal("next handler not called")
				}
				if rec.Code != http.StatusOK {
					t.Errorf("status = %d, want 200", rec.Code)
				}
				return
			}
			if called {
				t.Fatal("next handler called for unauthenticated request")
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
			want := `Bearer resource_metadata="` + prmURL + `"`
			if got := rec.Header().Get("WWW-Authenticate"); got != want {
				t.Errorf("WWW-Authenticate = %q, want %q", got, want)
			}
		})
	}
}

func TestBearerFromRequest(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"", ""},
		{"Bearer abc", "abc"},
		{"Bearer   abc  ", "abc"},
		{"Bearer ", ""},
		{"Basic abc", ""},
		{"bearer abc", ""}, // case-sensitive by design — RFC 6750 is case-insensitive but real clients capitalize.
	}
	for _, c := range cases {
		t.Run(c.header, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			if got := bearerFromRequest(req); got != c.want {
				t.Errorf("bearerFromRequest(%q) = %q, want %q", c.header, got, c.want)
			}
		})
	}
}

// Sanity-check: if anyone changes the constant or the /healthz registration,
// the collision check in runHTTP needs to stay in sync.
func TestHealthzPathConstant(t *testing.T) {
	if wellKnownPRMPath == "/healthz" {
		t.Fatal("wellKnownPRMPath collides with /healthz reservation in runHTTP")
	}
}

// Sanity-check that the WWW-Authenticate string we emit matches RFC 6750's grammar.
func TestChallengeFormat(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	mw := challengeMiddleware("https://x/y", next)
	req := httptest.NewRequest("POST", "/mcp", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	got := rec.Header().Get("WWW-Authenticate")
	if !strings.HasPrefix(got, "Bearer ") {
		t.Fatalf("challenge missing scheme: %q", got)
	}
	if !strings.Contains(got, `resource_metadata="`) {
		t.Fatalf("challenge missing resource_metadata param: %q", got)
	}
}
