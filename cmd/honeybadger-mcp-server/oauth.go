package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const wellKnownPRMPath = "/.well-known/oauth-protected-resource"

// Operator-supplied origin; rejecting a path keeps the PRM-served URL and the
// 401-advertised URL from drifting (both build absolute URLs by appending here).
func normalizePublicURL(raw string) (string, error) {
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

// ServeMux panics on patterns missing a leading slash; trailing slash would
// make the advertised PRM resource differ from where clients POST.
func normalizeEndpointPath(p string) string {
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

func prmHandler(resource string, authServers []string) http.Handler {
	body, _ := json.Marshal(struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
	}{Resource: resource, AuthorizationServers: authServers})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(body)
	})
}

// Only handles the no-bearer bootstrap case so unauthenticated clients can
// discover OAuth; deliberately does NOT validate present tokens (thin relay).
// Mid-session expiry surfaces as a tool error, not a 401 — see CLAUDE.md.
func challengeMiddleware(prmURL string, next http.Handler) http.Handler {
	challenge := fmt.Sprintf(`Bearer resource_metadata="%s"`, prmURL)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")) == "" {
			w.Header().Set("WWW-Authenticate", challenge)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerFromRequest(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
}
