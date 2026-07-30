package httptransport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbmcp"
)

func PRMHandler(resource string, authServers, scopes []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"resource":              resource,
			"authorization_servers": authServers,
			"scopes_supported":      scopes,
		})
	})
}

// ValidateMiddleware authenticates the request's Bearer credential.
//
// v3 accepts three kinds and they can be verified to different depths:
//
//   - An OAuth access token (hbo_) is a signed JWT, so its signature, issuer,
//     expiry, and audience are all checked here. The audience check is what stops
//     a token minted for another resource being replayed at this server
//     (RFC 8707), so it stays mandatory.
//   - A scoped API token (hbt_ or hba_) is opaque. Nothing about it can be
//     verified locally, so it is accepted on its prefix and forwarded for the API
//     to judge. This server deliberately makes no authorization decision about
//     it; the API is the authority either way.
//
// The trade is explicit: opaque credentials get no audience binding and no local
// expiry check, so an expired one surfaces as a failure on the first API call
// rather than as a challenge here.
//
// Expired tokens get an error_description so MCP clients trigger their refresh-on-401 path.
func ValidateMiddleware(prmURL string, keyfn jwt.Keyfunc, expectedIssuer, expectedAudience string, next http.Handler) http.Handler {
	bootstrap := fmt.Sprintf(`Bearer resource_metadata="%s"`, prmURL)
	invalidToken := fmt.Sprintf(`Bearer error="invalid_token", resource_metadata="%s"`, prmURL)
	expiredToken := fmt.Sprintf(`Bearer error="invalid_token", error_description="The access token expired", resource_metadata="%s"`, prmURL)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := BearerFromRequest(r)
		if raw == "" {
			w.Header().Set("WWW-Authenticate", bootstrap)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		kind := hbmcp.ClassifyCredential(raw)
		if kind == hbmcp.KindUnknown {
			w.Header().Set("WWW-Authenticate", invalidToken)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := hbmcp.WithAuthToken(r.Context(), raw)
		ctx = hbmcp.WithCredentialKind(ctx, kind)

		if kind.Verifiable() {
			claims, err := hbmcp.ParseAccessToken(raw, keyfn, expectedIssuer, expectedAudience)
			if err != nil {
				challenge := invalidToken
				if errors.Is(err, jwt.ErrTokenExpired) {
					challenge = expiredToken
				}
				w.Header().Set("WWW-Authenticate", challenge)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx = hbmcp.WithClaims(ctx, claims)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func BearerFromRequest(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
}

// For LB target-group health checks.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
