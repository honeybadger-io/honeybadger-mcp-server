package hbmcp

import (
	"context"
	"testing"
)

func TestAuthTokenRoundTrip(t *testing.T) {
	ctx := WithAuthToken(context.Background(), "abc123")
	if got := AuthTokenFromContext(ctx); got != "abc123" {
		t.Fatalf("AuthTokenFromContext = %q, want %q", got, "abc123")
	}
}

func TestAuthTokenFromContext_Empty(t *testing.T) {
	if got := AuthTokenFromContext(context.Background()); got != "" {
		t.Fatalf("AuthTokenFromContext on bare ctx = %q, want empty", got)
	}
}

func TestWithAuthToken_EmptyNotStored(t *testing.T) {
	ctx := WithAuthToken(context.Background(), "")
	if got := AuthTokenFromContext(ctx); got != "" {
		t.Fatalf("AuthTokenFromContext after empty WithAuthToken = %q, want empty", got)
	}
}

func TestWithAuthToken_Overwrite(t *testing.T) {
	ctx := WithAuthToken(context.Background(), "first")
	ctx = WithAuthToken(ctx, "second")
	if got := AuthTokenFromContext(ctx); got != "second" {
		t.Fatalf("AuthTokenFromContext after overwrite = %q, want %q", got, "second")
	}
}
