package hbmcp

import (
	"context"

	"github.com/honeybadger-io/api-go/apiv3"
)

type authTokenKey struct{}
type claimsKey struct{}
type credentialKindKey struct{}
type tokenInfoKey struct{}

func WithAuthToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, authTokenKey{}, token)
}

func AuthTokenFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(authTokenKey{}).(string); ok {
		return v
	}
	return ""
}

func WithClaims(ctx context.Context, c *Claims) context.Context {
	if c == nil {
		return ctx
	}
	return context.WithValue(ctx, claimsKey{}, c)
}

func ClaimsFromContext(ctx context.Context) *Claims {
	v, _ := ctx.Value(claimsKey{}).(*Claims)
	return v
}

// WithCredentialKind records which sort of credential the request carried, so
// downstream code can tell a verifiable OAuth token from an opaque scoped one.
func WithCredentialKind(ctx context.Context, kind CredentialKind) context.Context {
	if kind == KindUnknown {
		return ctx
	}
	return context.WithValue(ctx, credentialKindKey{}, kind)
}

// CredentialKindFromContext returns the request's credential kind, or
// KindUnknown when none was recorded.
func CredentialKindFromContext(ctx context.Context) CredentialKind {
	if v, ok := ctx.Value(credentialKindKey{}).(CredentialKind); ok {
		return v
	}
	return KindUnknown
}

// WithTokenInfo records what the request's credential permits, as reported by the
// API. Absent when introspection was unavailable, which callers must treat as
// "unknown", never as "nothing permitted".
func WithTokenInfo(ctx context.Context, info *apiv3.TokenInfo) context.Context {
	if info == nil {
		return ctx
	}
	return context.WithValue(ctx, tokenInfoKey{}, info)
}

// TokenInfoFromContext returns the request credential's description, or nil when
// none was recorded.
func TokenInfoFromContext(ctx context.Context) *apiv3.TokenInfo {
	info, _ := ctx.Value(tokenInfoKey{}).(*apiv3.TokenInfo)
	return info
}
