package hbmcp

import "context"

type authTokenKey struct{}
type claimsKey struct{}

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
