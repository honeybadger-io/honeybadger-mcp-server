package hbmcp

import (
	"errors"
	"math"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const tokenPrefix = "hbo_"

type Claims struct {
	Subject string
	Scopes  []string
	// ScopeVersion is the token's scope_version claim: 1 for legacy read/write,
	// 2 for granular permissions. 0 when the claim is absent or malformed, as on
	// tokens minted before the AS added it.
	ScopeVersion int
}

// Granular reports whether the scopes are granular permissions ("faults:write")
// that the API honors as written, rather than the legacy read/write aliases.
//
// A version 1 token is never granular, even if it carries a granular name: the
// API drops every name but read and write on those, so honoring one here would
// offer a tool the API refuses. Without the claim, a token is granular only if it
// has scopes and none is a legacy alias; the AS refuses to mix the two.
//
// An unknown version is not granular, so it is judged by the legacy rule, which
// grants nothing without "write".
func (c *Claims) Granular() bool {
	switch c.ScopeVersion {
	case 2:
		return true
	case 0:
		return len(c.Scopes) > 0 && !c.HasScope("read") && !c.HasScope("write")
	default:
		return false
	}
}

func (c *Claims) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// The AS audience-binds tokens per RFC 8707: hosts discover the resource
// identifier from PRM and send it as resource= on the authorize and token
// requests, and the AS mints it into aud. A missing or mismatched aud is
// rejected (jwt.WithAudience treats absent aud as invalid) — without that
// the binding does nothing.
func ParseAccessToken(raw string, keyfunc jwt.Keyfunc, expectedIssuer, expectedAudience string) (*Claims, error) {
	if !strings.HasPrefix(raw, tokenPrefix) {
		return nil, errors.New("token missing hbo_ prefix")
	}
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithExpirationRequired(),
	}
	if expectedIssuer != "" {
		opts = append(opts, jwt.WithIssuer(expectedIssuer))
	}
	if expectedAudience != "" {
		opts = append(opts, jwt.WithAudience(expectedAudience))
	}
	tok, err := jwt.NewParser(opts...).Parse(strings.TrimPrefix(raw, tokenPrefix), keyfunc)
	if err != nil {
		return nil, err
	}
	mc, _ := tok.Claims.(jwt.MapClaims)
	// Delete confirmations are bound to the subject, so callers without one
	// would share an identity. RFC 9068 requires sub on JWT access tokens.
	sub, err := mc.GetSubject()
	if err != nil || sub == "" {
		return nil, errors.New("token missing sub")
	}
	scope, _ := mc["scope"].(string)
	return &Claims{Subject: sub, Scopes: strings.Fields(scope), ScopeVersion: scopeVersion(mc)}, nil
}

// scopeVersion reads the scope_version claim, or 0 when it is absent or not a
// whole number.
func scopeVersion(mc jwt.MapClaims) int {
	v, ok := mc["scope_version"].(float64)
	if !ok || v != math.Trunc(v) || v < 0 || v > math.MaxInt32 {
		return 0
	}
	return int(v)
}
