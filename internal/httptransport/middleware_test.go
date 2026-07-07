package httptransport

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func testKey(t *testing.T) (*rsa.PrivateKey, jwt.Keyfunc) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	return key, func(*jwt.Token) (any, error) { return &key.PublicKey, nil }
}

func signHBO(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return "hbo_" + signed
}

func validClaims() jwt.MapClaims {
	now := time.Now().Unix()
	return jwt.MapClaims{
		"iss":        "https://issuer.example",
		"sub":        "1",
		"iat":        now,
		"exp":        now + 60,
		"scope":      "read write",
		"account_id": float64(42),
		"aud":        "https://host/mcp",
	}
}

func TestPRMHandler(t *testing.T) {
	h := PRMHandler("https://host/mcp", []string{"https://auth.example"}, []string{"read", "write"})

	t.Run("GET returns metadata", func(t *testing.T) {
		req := httptest.NewRequest("GET", WellKnownPRMPath, nil)
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
			ScopesSupported      []string `json:"scopes_supported"`
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
		if len(got.ScopesSupported) != 2 || got.ScopesSupported[0] != "read" || got.ScopesSupported[1] != "write" {
			t.Errorf("scopes_supported = %v", got.ScopesSupported)
		}
	})

}

func TestValidateMiddleware(t *testing.T) {
	prmURL := "https://host" + WellKnownPRMPath
	key, kf := testKey(t)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := ValidateMiddleware(prmURL, kf, "https://issuer.example", "https://host/mcp", next)

	expiredClaims := validClaims()
	expiredClaims["exp"] = time.Now().Add(-time.Minute).Unix()

	wrongAudClaims := validClaims()
	wrongAudClaims["aud"] = "https://other.resource"

	cases := []struct {
		name            string
		header          string
		wantStatus      int
		wantNextCalled  bool
		wantAuthSubstr  string
		wantNoErrorAttr bool
	}{
		{
			name:            "no header → bootstrap challenge",
			header:          "",
			wantStatus:      http.StatusUnauthorized,
			wantAuthSubstr:  `resource_metadata="`,
			wantNoErrorAttr: true,
		},
		{
			name:           "no prefix → invalid_token",
			header:         "Bearer not-an-hbo-token",
			wantStatus:     http.StatusUnauthorized,
			wantAuthSubstr: `error="invalid_token"`,
		},
		{
			name:           "expired → invalid_token + expired description",
			header:         "Bearer " + signHBO(t, key, expiredClaims),
			wantStatus:     http.StatusUnauthorized,
			wantAuthSubstr: `error_description="The access token expired"`,
		},
		{
			name:           "wrong audience → invalid_token",
			header:         "Bearer " + signHBO(t, key, wrongAudClaims),
			wantStatus:     http.StatusUnauthorized,
			wantAuthSubstr: `error="invalid_token"`,
		},
		{
			name:           "valid → passes through",
			header:         "Bearer " + signHBO(t, key, validClaims()),
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
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

			if rec.Code != c.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, c.wantStatus)
			}
			if called != c.wantNextCalled {
				t.Errorf("next called = %v, want %v", called, c.wantNextCalled)
			}
			if c.wantAuthSubstr != "" {
				if got := rec.Header().Get("WWW-Authenticate"); !strings.Contains(got, c.wantAuthSubstr) {
					t.Errorf("WWW-Authenticate %q does not contain %q", got, c.wantAuthSubstr)
				}
			}
			if c.wantNoErrorAttr {
				if got := rec.Header().Get("WWW-Authenticate"); strings.Contains(got, "error=") {
					t.Errorf("bootstrap challenge should not carry an error attribute: %q", got)
				}
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
			if got := BearerFromRequest(req); got != c.want {
				t.Errorf("BearerFromRequest(%q) = %q, want %q", c.header, got, c.want)
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	HealthHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

// Regression guard: /healthz is reserved by main; the PRM path must not collide.
func TestPRMPathNotHealthz(t *testing.T) {
	if WellKnownPRMPath == "/healthz" {
		t.Fatal("WellKnownPRMPath collides with reserved /healthz")
	}
}
