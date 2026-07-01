// Package httptransport holds Streamable-HTTP concerns: PRM, AS/JWKS
// discovery, the 401 middleware, and small HTTP helpers.
package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const WellKnownPRMPath = "/.well-known/oauth-protected-resource"

// Path rejected so the PRM-served and 401-advertised URLs stay in sync.
func NormalizePublicURL(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse public-url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("public-url must include scheme and host, got %q", raw)
	}
	if p := strings.TrimSuffix(u.Path, "/"); p != "" {
		return "", fmt.Errorf("public-url must be origin-only (no path), got %q", raw)
	}
	return u.Scheme + "://" + u.Host, nil
}

// Leading slash: ServeMux panics without one. Trailing slash: PRM drift.
func NormalizeEndpointPath(p string) string {
	if p == "" {
		return "/mcp"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if len(p) > 1 {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

type ASMetadata struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

func DiscoverAS(ctx context.Context, authServer string) (*ASMetadata, error) {
	u := strings.TrimSuffix(authServer, "/") + "/.well-known/oauth-authorization-server"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch AS metadata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("AS metadata returned %d: %s", resp.StatusCode, string(body))
	}
	var md ASMetadata
	if err := json.NewDecoder(resp.Body).Decode(&md); err != nil {
		return nil, fmt.Errorf("decode AS metadata: %w", err)
	}
	if md.Issuer == "" || md.JWKSURI == "" {
		return nil, errors.New("AS metadata missing issuer or jwks_uri")
	}
	return &md, nil
}

// Force a JWKS fetch at startup — keyfunc.NewDefault otherwise lazy-loads
// and swallows the initial fetch/parse failure until the first request.
func VerifyJWKSReachable(ctx context.Context, jwksURI string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURI, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS returned %d", resp.StatusCode)
	}
	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}
	if len(jwks.Keys) == 0 {
		return errors.New("JWKS has no keys")
	}
	// kty is required per RFC 7517 §4.1; catches malformed keys keyfunc silently drops.
	for i, k := range jwks.Keys {
		if k.Kty == "" {
			return fmt.Errorf("JWKS key %d missing required kty", i)
		}
	}
	return nil
}
