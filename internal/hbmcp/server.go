package hbmcp

import (
	"context"

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

// In http mode the token's scope is authoritative; in stdio there's no token,
// so the startup --read-only flag decides. Missing claims fails closed.
func EffectiveReadOnly(ctx context.Context, cfg *config.Config) bool {
	if cfg.TransportMode == config.TransportHTTP {
		claims := ClaimsFromContext(ctx)
		return claims == nil || !claims.HasScope("write")
	}
	return cfg.ReadOnly
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
			return filterReadOnlyTools(tools)
		}
		return tools
	}))

	s := server.NewMCPServer("honeybadger-mcp-server", version, serverOptions...)

	clientFor := newClientFactory(cfg)
	v3ClientFor := newV3ClientFactory(cfg)
	r := newToolRegistrar(s)
	RegisterReferenceTools(r, newReferenceFetcher(cfg.InstructionsURL, logger))
	RegisterProjectTools(r, clientFor)
	RegisterFaultTools(r, clientFor)
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
