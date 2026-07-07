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
		ctx := hbmcp.WithAuthToken(r.Context(), raw)
		ctx = hbmcp.WithClaims(ctx, claims)
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
