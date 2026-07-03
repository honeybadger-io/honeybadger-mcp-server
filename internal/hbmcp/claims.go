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

// No audience validation: the authorization server issues audience-less
// tokens (no aud/resource claim, no RFC 8707 support), and their only
// consumers are the Honeybadger API and this server, which relays each
// token to that same API — so there is no second resource for a token to
// be confused with. Revisit if the AS ever mints tokens for other
// resources or signs other token types with the same key.
func ParseAccessToken(raw string, keyfunc jwt.Keyfunc, expectedIssuer string) (*Claims, error) {
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
	tok, err := jwt.NewParser(opts...).Parse(strings.TrimPrefix(raw, tokenPrefix), keyfunc)
	if err != nil {
		return nil, err
	}
	mc, _ := tok.Claims.(jwt.MapClaims)
	scope, _ := mc["scope"].(string)
	return &Claims{Scopes: strings.Fields(scope)}, nil
}
