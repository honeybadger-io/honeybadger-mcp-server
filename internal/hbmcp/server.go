package hbmcp

import (
	"context"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/analytics"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/logging"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ClientFactory func(ctx context.Context) *hbapi.Client

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
	s, _, _ := NewServerWithCatalog(cfg, version)
	return s
}

// newSink is the activation gate. Analytics runs only on the hosted HTTP
// transport with a key we configured. stdio always gets the no-op sink,
// whatever HONEYBADGER_API_KEY contains — in stdio that key belongs to the
// customer running the binary, and sending our telemetry to their project
// would pollute their data and consume their quota.
func newSink(cfg *config.Config) analytics.Sink {
	if cfg.TransportMode != config.TransportHTTP || cfg.HoneybadgerAPIKey == "" {
		return analytics.NewNopSink()
	}
	env := cfg.HoneybadgerEnv
	if env == "" {
		env = "production"
	}
	return analytics.NewHoneybadgerSink(cfg.HoneybadgerAPIKey, env)
}

// NewServerWithCatalog also returns the full tool catalog (including
// search_tools) so callers like the HTTP landing page can list the
// server's tools without an MCP session.
func NewServerWithCatalog(cfg *config.Config, version string) (*server.MCPServer, []ToolInfo, analytics.Sink) {
	logger := logging.SetupLogger(cfg.LogLevel)

	sink := newSink(cfg)
	analyticsOn := sink != analytics.NewNopSink()
	if !analyticsOn {
		logger.Info("Usage analytics disabled", "reason", "no HONEYBADGER_API_KEY or non-http transport")
	}

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

	inst := newInstrumenter(sink, cfg, version)
	clientFor := newClientFactory(cfg, analyticsOn)
	r := newToolRegistrar(s, inst)
	RegisterReferenceTools(r, newReferenceFetcher(cfg.InstructionsURL, logger))
	RegisterProjectTools(r, clientFor)
	RegisterFaultTools(r, clientFor)
	RegisterInsightsTools(r, clientFor)
	RegisterStreamTools(r, clientFor)
	RegisterDashboardTools(r, clientFor)
	RegisterAlarmTools(r, clientFor)
	RegisterCheckInTools(r, clientFor)
	registerSearchTool(s, r.catalog, cfg, inst)

	return s, append(r.catalog, searchToolInfo), sink
}

func newClientFactory(cfg *config.Config, analyticsOn bool) ClientFactory {
	if cfg.TransportMode == config.TransportHTTP {
		// No fallback to cfg.AuthToken — the 401 middleware must catch
		// bearer-less requests; a fallback would mask that regression.
		return func(ctx context.Context) *hbapi.Client {
			c := hbapi.NewClient().
				WithBaseURL(cfg.APIURL).
				WithBearerToken(AuthTokenFromContext(ctx))
			if analyticsOn {
				c = c.WithHTTPClient(newRecordingHTTPClient())
			}
			return c
		}
	}
	return func(ctx context.Context) *hbapi.Client {
		return hbapi.NewClient().
			WithBaseURL(cfg.APIURL).
			WithAuthToken(cfg.AuthToken)
	}
}
