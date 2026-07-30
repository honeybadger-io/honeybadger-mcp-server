package hbmcp

import "context"

type authTokenKey struct{}
type claimsKey struct{}
type credentialKindKey struct{}

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
