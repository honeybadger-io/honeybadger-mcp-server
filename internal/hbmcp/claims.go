package hbmcp

import (
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const tokenPrefix = "hbo_"

type Claims struct {
	Scopes []string
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
	scope, _ := mc["scope"].(string)
	return &Claims{Scopes: strings.Fields(scope)}, nil
}
