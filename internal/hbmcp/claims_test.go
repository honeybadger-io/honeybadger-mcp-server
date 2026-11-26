package hbmcp

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func signedToken(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tokenPrefix + signed
}

func testKey(t *testing.T) (*rsa.PrivateKey, jwt.Keyfunc) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	return key, func(*jwt.Token) (any, error) { return &key.PublicKey, nil }
}

func baseClaims() jwt.MapClaims {
	now := time.Now().Unix()
	return jwt.MapClaims{
		"iss":   "http://localhost:3001",
		"sub":   "42",
		"exp":   now + 60,
		"scope": "read write",
	}
}

func TestParseAccessToken_Success(t *testing.T) {
	key, kf := testKey(t)
	raw := signedToken(t, key, baseClaims())

	got, err := ParseAccessToken(raw, kf, "http://localhost:3001", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Subject != "42" {
		t.Errorf("Subject = %q, want 42", got.Subject)
	}
	if !got.HasScope("read") || !got.HasScope("write") {
		t.Errorf("Scopes = %v", got.Scopes)
	}
	if got.HasScope("admin") {
		t.Errorf("unexpected admin scope")
	}
}

func TestParseAccessToken_RequiresSubject(t *testing.T) {
	key, kf := testKey(t)
	for name, sub := range map[string]any{"missing": nil, "empty": "", "non-string": 42} {
		t.Run(name, func(t *testing.T) {
			c := baseClaims()
			if sub == nil {
				delete(c, "sub")
			} else {
				c["sub"] = sub
			}
			if _, err := ParseAccessToken(signedToken(t, key, c), kf, "", ""); err == nil {
				t.Fatal("expected rejection of token without a usable sub")
			}
		})
	}
}

func TestParseAccessToken_MissingPrefix(t *testing.T) {
	key, kf := testKey(t)
	signed, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, baseClaims()).SignedString(key)
	if _, err := ParseAccessToken(signed, kf, "", ""); err == nil {
		t.Fatal("expected error for missing prefix")
	}
}

func TestParseAccessToken_Expired(t *testing.T) {
	key, kf := testKey(t)
	c := baseClaims()
	c["exp"] = time.Now().Add(-time.Minute).Unix()
	raw := signedToken(t, key, c)

	_, err := ParseAccessToken(raw, kf, "", "")
	if err == nil {
		t.Fatal("expected expired error")
	}
	if !errors.Is(err, jwt.ErrTokenExpired) {
		t.Errorf("error %v does not wrap jwt.ErrTokenExpired", err)
	}
}

func TestParseAccessToken_IssuerMismatch(t *testing.T) {
	key, kf := testKey(t)
	raw := signedToken(t, key, baseClaims())
	if _, err := ParseAccessToken(raw, kf, "https://other.issuer", ""); err == nil {
		t.Fatal("expected issuer mismatch")
	}
}

func TestParseAccessToken_BadSignature(t *testing.T) {
	signer, _ := testKey(t)
	_, verifierKF := testKey(t)
	raw := signedToken(t, signer, baseClaims())
	if _, err := ParseAccessToken(raw, verifierKF, "", ""); err == nil {
		t.Fatal("expected signature error")
	}
}

func TestParseAccessToken_NoExp(t *testing.T) {
	key, kf := testKey(t)
	c := baseClaims()
	delete(c, "exp")
	raw := signedToken(t, key, c)
	if _, err := ParseAccessToken(raw, kf, "", ""); err == nil {
		t.Fatal("expected error for missing exp")
	}
}

func TestParseAccessToken_WrongAlgorithm(t *testing.T) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, baseClaims())
	signed, _ := tok.SignedString([]byte("hmac-secret"))
	raw := tokenPrefix + signed

	_, kf := testKey(t)
	if _, err := ParseAccessToken(raw, kf, "", ""); err == nil {
		t.Fatal("expected rejection of HS256 token")
	}
}

func TestParseAccessToken_AudienceMatch(t *testing.T) {
	key, kf := testKey(t)
	c := baseClaims()
	c["aud"] = "https://mcp.honeybadger.io"
	raw := signedToken(t, key, c)

	if _, err := ParseAccessToken(raw, kf, "", "https://mcp.honeybadger.io"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAccessToken_AudienceMismatch(t *testing.T) {
	key, kf := testKey(t)
	c := baseClaims()
	c["aud"] = "https://other.resource"
	raw := signedToken(t, key, c)

	_, err := ParseAccessToken(raw, kf, "", "https://mcp.honeybadger.io")
	if err == nil {
		t.Fatal("expected audience mismatch")
	}
	if !errors.Is(err, jwt.ErrTokenInvalidAudience) {
		t.Errorf("error %v does not wrap jwt.ErrTokenInvalidAudience", err)
	}
}

func TestParseAccessToken_MissingAudienceRejected(t *testing.T) {
	key, kf := testKey(t)
	raw := signedToken(t, key, baseClaims()) // no aud claim

	if _, err := ParseAccessToken(raw, kf, "", "https://mcp.honeybadger.io"); err == nil {
		t.Fatal("expected error for missing aud when an audience is expected")
	}
}

func TestParseAccessToken_EmptyScope(t *testing.T) {
	key, kf := testKey(t)
	c := baseClaims()
	c["scope"] = ""
	raw := signedToken(t, key, c)

	got, err := ParseAccessToken(raw, kf, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Scopes) != 0 {
		t.Errorf("Scopes = %v, want empty", got.Scopes)
	}
}

// The AS stamps scope_version on every new token: 1 for legacy read/write, 2 for
// granular permissions. JSON numbers arrive as float64.
func TestParseAccessToken_ScopeVersion(t *testing.T) {
	key, kf := testKey(t)
	for _, tc := range []struct {
		name  string
		claim any
		want  int
	}{
		{"version 1", float64(1), 1},
		{"version 2", float64(2), 2},
		{"absent", nil, 0},
		{"not a number", "2", 0},
		{"not an integer", float64(1.5), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := baseClaims()
			if tc.claim != nil {
				c["scope_version"] = tc.claim
			}
			got, err := ParseAccessToken(signedToken(t, key, c), kf, "", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ScopeVersion != tc.want {
				t.Errorf("ScopeVersion = %d, want %d", got.ScopeVersion, tc.want)
			}
		})
	}
}

func TestClaimsGranular(t *testing.T) {
	for _, tc := range []struct {
		name   string
		claims Claims
		want   bool
	}{
		{"version 2", Claims{ScopeVersion: 2, Scopes: []string{"faults:read"}}, true},
		{"version 2 with no scopes", Claims{ScopeVersion: 2}, true},
		// A version 1 token only ever honors read/write. A stray granular name on
		// it was dropped by the AS and must stay dropped here.
		{"version 1 with a stray granular name", Claims{ScopeVersion: 1, Scopes: []string{"read", "faults:write"}}, false},
		{"version 1", Claims{ScopeVersion: 1, Scopes: []string{"read", "write"}}, false},
		// Tokens minted before the claim existed: the scope names decide.
		{"no version, legacy names", Claims{Scopes: []string{"read", "write"}}, false},
		{"no version, read only", Claims{Scopes: []string{"read"}}, false},
		{"no version, mixed names", Claims{Scopes: []string{"write", "faults:read"}}, false},
		{"no version, granular names", Claims{Scopes: []string{"faults:read", "faults:write"}}, true},
		{"no version, no scopes", Claims{}, false},
		// An unknown future version is neither; callers treat it as legacy, which
		// grants nothing without a write scope.
		{"version 3", Claims{ScopeVersion: 3, Scopes: []string{"faults:write"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.claims.Granular(); got != tc.want {
				t.Errorf("Granular() = %v, want %v", got, tc.want)
			}
		})
	}
}
