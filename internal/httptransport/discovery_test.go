package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
			got, err := NormalizePublicURL(c.in)
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
	cases := []struct{ in, want string }{
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
			if got := NormalizeEndpointPath(c.in); got != c.want {
				t.Errorf("NormalizeEndpointPath(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestASMetadataURL(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"origin only", "https://app.honeybadger.io", "https://app.honeybadger.io/.well-known/oauth-authorization-server", false},
		{"trailing slash", "https://app.honeybadger.io/", "https://app.honeybadger.io/.well-known/oauth-authorization-server", false},
		{"path inserted per RFC 8414", "https://app.honeybadger.io/oauth", "https://app.honeybadger.io/.well-known/oauth-authorization-server/oauth", false},
		{"path with trailing slash", "https://app.honeybadger.io/oauth/", "https://app.honeybadger.io/.well-known/oauth-authorization-server/oauth", false},
		{"missing scheme", "app.honeybadger.io", "", true},
		{"missing host", "https://", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := asMetadataURL(c.in)
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

func TestDiscoverAS_PathIssuer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/oauth-authorization-server/oauth" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":   "https://issuer.example/oauth",
			"jwks_uri": "https://issuer.example/.well-known/jwks.json",
		})
	}))
	defer srv.Close()

	md, err := DiscoverAS(context.Background(), srv.URL+"/oauth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md.Issuer != "https://issuer.example/oauth" {
		t.Errorf("unexpected metadata: %+v", md)
	}
}

func TestDiscoverAS_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/oauth-authorization-server" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":   "https://issuer.example",
			"jwks_uri": "https://issuer.example/.well-known/jwks.json",
		})
	}))
	defer srv.Close()

	md, err := DiscoverAS(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md.Issuer != "https://issuer.example" || md.JWKSURI != "https://issuer.example/.well-known/jwks.json" {
		t.Errorf("unexpected metadata: %+v", md)
	}
}

func TestDiscoverAS_MissingFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"issuer": "https://x"}) // no jwks_uri
	}))
	defer srv.Close()

	if _, err := DiscoverAS(context.Background(), srv.URL); err == nil {
		t.Fatal("expected error for missing jwks_uri")
	}
}

func TestDiscoverAS_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if _, err := DiscoverAS(context.Background(), srv.URL); err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestVerifyJWKSReachable(t *testing.T) {
	t.Run("good JWKS", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"keys":[{"kty":"RSA","kid":"k1"}]}`))
		}))
		defer srv.Close()
		if err := VerifyJWKSReachable(context.Background(), srv.URL); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("no keys", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"keys":[]}`))
		}))
		defer srv.Close()
		if err := VerifyJWKSReachable(context.Background(), srv.URL); err == nil {
			t.Error("expected error for empty keys")
		}
	})
	t.Run("key missing kty", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"keys":[{}]}`))
		}))
		defer srv.Close()
		if err := VerifyJWKSReachable(context.Background(), srv.URL); err == nil {
			t.Error("expected error for key missing kty")
		}
	})
	t.Run("500 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		if err := VerifyJWKSReachable(context.Background(), srv.URL); err == nil {
			t.Error("expected error for 500")
		}
	})
	t.Run("malformed body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`not json`))
		}))
		defer srv.Close()
		if err := VerifyJWKSReachable(context.Background(), srv.URL); err == nil {
			t.Error("expected decode error")
		}
	})
}
