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
	if !got.HasScope("read") || !got.HasScope("write") {
		t.Errorf("Scopes = %v", got.Scopes)
	}
	if got.HasScope("admin") {
		t.Errorf("unexpected admin scope")
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
