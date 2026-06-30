package hbmcp

import (
	"context"
	"net/http"
	"time"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/logging"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Matches api-go's NewClient default; preserved explicitly because WithHTTPClient overrides it.
const apiClientTimeout = 30 * time.Second

type ClientFactory func(ctx context.Context) *hbapi.Client

func filterReadOnlyTools(tools []mcp.Tool) []mcp.Tool {
	var readOnlyTools []mcp.Tool
	for _, tool := range tools {
		if tool.Annotations.ReadOnlyHint != nil && *tool.Annotations.ReadOnlyHint {
			readOnlyTools = append(readOnlyTools, tool)
		}
	}
	return readOnlyTools
}

func NewServer(cfg *config.Config) *server.MCPServer {
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
		server.WithLogging(),
		server.WithRecovery(),
		server.WithHooks(hooks),
	}
	if cfg.ReadOnly {
		serverOptions = append(serverOptions, server.WithToolFilter(func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
			return filterReadOnlyTools(tools)
		}))
	}

	s := server.NewMCPServer("honeybadger-mcp-server", "1.0.0", serverOptions...)

	clientFor := newClientFactory(cfg)
	r := newToolRegistrar(s)
	RegisterProjectTools(r, clientFor)
	RegisterFaultTools(r, clientFor)
	RegisterInsightsTools(r, clientFor)
	RegisterDashboardTools(r, clientFor)
	RegisterAlarmTools(r, clientFor)
	registerSearchTool(s, r.catalog, cfg.ReadOnly)

	return s
}

func newClientFactory(cfg *config.Config) ClientFactory {
	if cfg.TransportMode == config.TransportHTTP {
		sharedTransport := http.DefaultTransport
		// No fallback to cfg.AuthToken on missing ctx token — the 401 challenge
		// middleware must catch bearer-less requests, and silently using a
		// startup PAT would mask a middleware regression.
		return func(ctx context.Context) *hbapi.Client {
			return hbapi.NewClient().
				WithBaseURL(cfg.APIURL).
				WithHTTPClient(&http.Client{
					Timeout:   apiClientTimeout,
					Transport: &bearerTransport{token: AuthTokenFromContext(ctx), base: sharedTransport},
				})
		}
	}
	return func(ctx context.Context) *hbapi.Client {
		return hbapi.NewClient().
			WithBaseURL(cfg.APIURL).
			WithAuthToken(cfg.AuthToken)
	}
}
