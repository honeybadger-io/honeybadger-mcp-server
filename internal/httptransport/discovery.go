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

// RFC 8414 §3: the well-known segment is inserted between the host and any
// issuer path component (https://as.example/foo resolves to
// https://as.example/.well-known/oauth-authorization-server/foo), matching
// the URL compliant clients derive from the PRM's authorization_servers.
func asMetadataURL(authServer string) (string, error) {
	u, err := url.Parse(authServer)
	if err != nil {
		return "", fmt.Errorf("parse authorization-server: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("authorization-server must include scheme and host, got %q", authServer)
	}
	return u.Scheme + "://" + u.Host + "/.well-known/oauth-authorization-server" + strings.TrimSuffix(u.Path, "/"), nil
}

func DiscoverAS(ctx context.Context, authServer string) (*ASMetadata, error) {
	u, err := asMetadataURL(authServer)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch AS metadata: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
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
