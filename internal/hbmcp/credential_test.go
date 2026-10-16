package hbmcp

import (
	"context"
	"testing"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
)

func TestClassifyCredential(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want CredentialKind
	}{
		{"hbo_eyJhbGci...", KindOAuth},
		{"hbt_abc123", KindUserToken},
		{"hba_abc123", KindAccountToken},
		{"", KindUnknown},
		{"abc123", KindUnknown},
		{"Bearer hbt_abc", KindUnknown}, // the scheme must already be stripped
		{"HBT_abc123", KindUnknown},     // prefixes are case-sensitive
		{"hb_abc123", KindUnknown},
	} {
		if got := ClassifyCredential(tc.raw); got != tc.want {
			t.Errorf("ClassifyCredential(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

// Only OAuth tokens can be checked without calling the API. If this ever reports
// true for a scoped token, something is claiming to verify an opaque string.
func TestOnlyOAuthIsVerifiable(t *testing.T) {
	if !KindOAuth.Verifiable() {
		t.Error("KindOAuth.Verifiable() = false")
	}
	for _, k := range []CredentialKind{KindUserToken, KindAccountToken, KindUnknown} {
		if k.Verifiable() {
			t.Errorf("%q.Verifiable() = true; nothing about an opaque token is checkable locally", k)
		}
	}
}

// An OAuth request without the write scope must not be offered writing tools.
func TestEffectiveReadOnlyOAuthWithoutWriteScope(t *testing.T) {
	cfg := &config.Config{TransportMode: config.TransportHTTP}
	ctx := WithCredentialKind(context.Background(), KindOAuth)
	ctx = WithClaims(ctx, &Claims{Scopes: []string{"read"}})

	if !EffectiveReadOnly(ctx, cfg) {
		t.Error("a read-only OAuth token was offered the writing tools")
	}
}

func TestEffectiveReadOnlyOAuthWithWriteScope(t *testing.T) {
	cfg := &config.Config{TransportMode: config.TransportHTTP}
	ctx := WithCredentialKind(context.Background(), KindOAuth)
	ctx = WithClaims(ctx, &Claims{Scopes: []string{"read", "write"}})

	if EffectiveReadOnly(ctx, cfg) {
		t.Error("a write-scoped OAuth token was denied the writing tools")
	}
}

// Missing claims on a verifiable credential still fails closed.
func TestEffectiveReadOnlyOAuthWithoutClaimsFailsClosed(t *testing.T) {
	cfg := &config.Config{TransportMode: config.TransportHTTP}
	ctx := WithCredentialKind(context.Background(), KindOAuth)

	if !EffectiveReadOnly(ctx, cfg) {
		t.Error("an OAuth request with no claims was offered the writing tools")
	}
}

// A scoped token is opaque, so hiding the writing tools would deny an account
// token writes it legitimately holds. The API refuses what it cannot do.
func TestEffectiveReadOnlyOpaqueTokensSeeEverything(t *testing.T) {
	cfg := &config.Config{TransportMode: config.TransportHTTP}
	for _, kind := range []CredentialKind{KindUserToken, KindAccountToken} {
		ctx := WithCredentialKind(context.Background(), kind)
		if EffectiveReadOnly(ctx, cfg) {
			t.Errorf("%q was treated as read-only; its permissions are unknown here, not absent", kind)
		}
	}
}

// stdio has no per-request credential, so the flag decides.
func TestEffectiveReadOnlyStdioUsesFlag(t *testing.T) {
	for _, readOnly := range []bool{true, false} {
		cfg := &config.Config{TransportMode: config.TransportStdio, ReadOnly: readOnly}
		if got := EffectiveReadOnly(context.Background(), cfg); got != readOnly {
			t.Errorf("stdio with --read-only=%v gave %v", readOnly, got)
		}
	}
}

func TestCredentialKindRoundTrip(t *testing.T) {
	ctx := WithCredentialKind(context.Background(), KindAccountToken)
	if got := CredentialKindFromContext(ctx); got != KindAccountToken {
		t.Errorf("got %q, want %q", got, KindAccountToken)
	}
	// An unknown kind is never recorded, so it cannot be mistaken for a real one.
	if got := CredentialKindFromContext(WithCredentialKind(context.Background(), KindUnknown)); got != KindUnknown {
		t.Errorf("got %q, want %q", got, KindUnknown)
	}
}

// Introspected scopes outrank OAuth claims, because the claims carry only legacy
// read/write while the API reports the granular set.
func TestEffectiveReadOnlyPrefersIntrospectedScopes(t *testing.T) {
	cfg := &config.Config{TransportMode: config.TransportHTTP}

	// Claims say read-only; introspection says the credential can write faults.
	ctx := WithCredentialKind(context.Background(), KindOAuth)
	ctx = WithClaims(ctx, &Claims{Scopes: []string{"read"}})
	ctx = WithTokenInfo(ctx, &apiv3.TokenInfo{Scopes: []string{"faults:read", "faults:write"}})

	if EffectiveReadOnly(ctx, cfg) {
		t.Error("introspection reported a writing scope and the tools were still hidden")
	}
}

func TestEffectiveReadOnlyIntrospectedReadOnlyHidesWrites(t *testing.T) {
	cfg := &config.Config{TransportMode: config.TransportHTTP}
	ctx := WithCredentialKind(context.Background(), KindAccountToken)
	ctx = WithTokenInfo(ctx, &apiv3.TokenInfo{Scopes: []string{"faults:read", "insights:read"}})

	if !EffectiveReadOnly(ctx, cfg) {
		t.Error("a credential holding only read scopes was offered the writing tools")
	}
}

func TestGrantsAnyWrite(t *testing.T) {
	for _, tc := range []struct {
		scopes []string
		want   bool
	}{
		{[]string{"faults:write"}, true},
		{[]string{"faults:read", "checkins:write"}, true},
		{[]string{"projects:create"}, true}, // creating is writing
		{[]string{"write"}, true},           // legacy alias
		{[]string{"faults:read"}, false},
		{[]string{"read"}, false},
		{nil, false},
		{[]string{}, false},
		{[]string{"writer"}, false},          // must not match on substring
		{[]string{"read:write_logs"}, false}, // nor mid-scope
	} {
		if got := grantsAnyWrite(tc.scopes); got != tc.want {
			t.Errorf("grantsAnyWrite(%v) = %v, want %v", tc.scopes, got, tc.want)
		}
	}
}
