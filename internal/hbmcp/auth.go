package hbmcp

import "context"

type authTokenKey struct{}

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
