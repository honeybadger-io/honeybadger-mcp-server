package hbmcp

import (
	"context"
	"strings"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/logging"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ClientFactory builds a v2 client. Tools still on v2 take this.
//
// Temporary: it disappears when every tool has moved to V3ClientFactory. Keeping
// both lets the migration land tool by tool with the build and tests green,
// rather than breaking every handler at once.
type ClientFactory func(ctx context.Context) *hbapi.Client

// V3ClientFactory builds a v3 client. Migrated tools take this.
type V3ClientFactory func(ctx context.Context) *apiv3.Client

// EffectiveReadOnly decides whether to hide the writing tools from this request.
//
// Three sources, in descending order of authority:
//
//  1. Introspection. The API's own account of what the credential holds, which
//     covers all three credential kinds and is granular — an OAuth token's JWT
//     carries only legacy read/write, because the expansion to granular
//     permissions happens server-side.
//  2. OAuth claims. Coarse but free, and correct as far as it goes. Used when
//     introspection was unavailable.
//  3. For an opaque credential with neither, the whole catalog. Hiding the
//     writing tools there would deny an account token writes it legitimately
//     holds: nothing here knows its scopes, and absent knowledge is not absent
//     permission. The API refuses what the credential cannot do, naming the
//     missing scope as it does.
//
// In stdio mode there is no per-request credential, so the startup --read-only
// flag decides.
func EffectiveReadOnly(ctx context.Context, cfg *config.Config) bool {
	if cfg.TransportMode != config.TransportHTTP {
		return cfg.ReadOnly
	}

	if info := TokenInfoFromContext(ctx); info != nil {
		return !grantsAnyWrite(info.Scopes)
	}

	if kind := CredentialKindFromContext(ctx); kind != KindUnknown && !kind.Verifiable() {
		return false
	}

	claims := ClaimsFromContext(ctx)
	return claims == nil || !claims.HasScope("write")
}

// grantsAnyWrite reports whether any scope permits writing.
//
// Granular scopes read "faults:write", "checkins:write" and so on, and the legacy
// aliases are the bare words. Matching the suffix keeps this working as the
// catalog grows rather than requiring a list of every writing scope.
func grantsAnyWrite(scopes []string) bool {
	for _, s := range scopes {
		if s == "write" || strings.HasSuffix(s, ":write") || strings.HasSuffix(s, ":create") {
			return true
		}
	}
	return false
}

func filterReadOnlyTools(tools []mcp.Tool) []mcp.Tool {
	var readOnlyTools []mcp.Tool
	for _, tool := range tools {
		if tool.Annotations.ReadOnlyHint != nil && *tool.Annotations.ReadOnlyHint {
			readOnlyTools = append(readOnlyTools, tool)
		}
	}
	return readOnlyTools
}

func NewServer(cfg *config.Config, version string) *server.MCPServer {
	s, _ := NewServerWithCatalog(cfg, version)
	return s
}

// NewServerWithCatalog also returns the full tool catalog (including
// search_tools) so callers like the HTTP landing page can list the
// server's tools without an MCP session.
func NewServerWithCatalog(cfg *config.Config, version string) (*server.MCPServer, []ToolInfo) {
	logger := logging.SetupLogger(cfg.LogLevel)

	hooks := &server.Hooks{}
	hooks.AddOnRegisterSession(func(ctx context.Context, session server.ClientSession) {
		logger.Info("Client session registered", "session_id", session.SessionID())
	})
	hooks.AddOnUnregisterSession(func(ctx context.Context, session server.ClientSession) {
		logger.Info("Client session unregistered", "session_id", session.SessionID())
	})
	hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
		logger.Error("Error in request", "method", method, "request_id", id, "error", err)
	})
	hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
		logger.Debug("Processing request", "method", method, "request_id", id)
	})

	serverOptions := []server.ServerOption{
		server.WithToolCapabilities(true),
		server.WithInstructions(ServerInstructions()),
		server.WithLogging(),
		server.WithRecovery(),
		server.WithHooks(hooks),
	}
	serverOptions = append(serverOptions, server.WithToolFilter(func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
		if EffectiveReadOnly(ctx, cfg) {
			tools = filterReadOnlyTools(tools)
		}

		// Scope filtering needs the credential's actual permissions, which only
		// introspection supplies. Without it every tool stays advertised and the
		// API refuses what the credential cannot do — showing a tool that then
		// fails is better than hiding one the caller was entitled to.
		if info := TokenInfoFromContext(ctx); info != nil {
			tools = filterByScopes(tools, info.Scopes)
		}
		return tools
	}))

	s := server.NewMCPServer("honeybadger-mcp-server", version, serverOptions...)

	clientFor := newClientFactory(cfg)
	v3ClientFor := newV3ClientFactory(cfg)
	r := newToolRegistrar(s)
	RegisterReferenceTools(r, newReferenceFetcher(cfg.InstructionsURL, logger))
	RegisterProjectTools(r, clientFor, v3ClientFor)
	RegisterFaultTools(r, clientFor, v3ClientFor)
	RegisterInsightsTools(r, v3ClientFor)
	RegisterStreamTools(r, v3ClientFor)
	RegisterDashboardTools(r, clientFor)
	RegisterAlarmTools(r, clientFor)
	RegisterCheckInTools(r, clientFor)
	registerSearchTool(s, r.catalog, cfg)

	return s, append(r.catalog, searchToolInfo)
}

func newClientFactory(cfg *config.Config) ClientFactory {
	if cfg.TransportMode == config.TransportHTTP {
		// No fallback to cfg.AuthToken — the 401 middleware must catch
		// bearer-less requests; a fallback would mask that regression.
		return func(ctx context.Context) *hbapi.Client {
			return hbapi.NewClient().
				WithBaseURL(cfg.APIURL).
				WithBearerToken(AuthTokenFromContext(ctx))
		}
	}
	return func(ctx context.Context) *hbapi.Client {
		return hbapi.NewClient().
			WithBaseURL(cfg.APIURL).
			WithAuthToken(cfg.AuthToken)
	}
}

// newV3ClientFactory builds the per-request v3 client.
//
// The base client is built once and each request derives from it. apiv3 clients
// are immutable, so WithBearerToken returns a fresh client rather than mutating a
// shared one — which is what keeps one request's credential from reaching
// another's in http mode.
func newV3ClientFactory(cfg *config.Config) V3ClientFactory {
	base := apiv3.NewClient().WithBaseURL(cfg.APIURL)

	if cfg.TransportMode == config.TransportHTTP {
		return func(ctx context.Context) *apiv3.Client {
			return base.WithBearerToken(AuthTokenFromContext(ctx))
		}
	}

	// v3 accepts only Bearer credentials: a scoped API token (hbt_ or hba_) or an
	// OAuth access token. The personal auth tokens this flag used to carry are
	// rejected outright, so the configured value must now be a scoped token.
	return func(ctx context.Context) *apiv3.Client {
		return base.WithBearerToken(cfg.AuthToken)
	}
}
