package hbmcp

import (
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const tokenPrefix = "hbo_"

type Claims struct {
	Scopes []string
	// Subject and AccountID are hashids from the AS — opaque, not raw
	// database IDs. ProjectScoped records only whether the token carried a
	// project_id, because that claim (unlike the other two) is emitted raw.
	Subject       string
	AccountID     string
	ClientID      string
	ProjectScoped bool
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
	sub, _ := mc["sub"].(string)
	accountID, _ := mc["account_id"].(string)
	clientID, _ := mc["client_id"].(string)
	// The AS .compacts its payload, so these are absent on some tokens.
	// Absence must leave the field empty, never fail validation.
	_, projectScoped := mc["project_id"]
	return &Claims{
		Scopes:        strings.Fields(scope),
		Subject:       sub,
		AccountID:     accountID,
		ClientID:      clientID,
		ProjectScoped: projectScoped,
	}, nil
}
