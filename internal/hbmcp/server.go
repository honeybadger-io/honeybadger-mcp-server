package hbmcp

import (
	"context"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/logging"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// filterReadOnlyTools filters tools to only include those marked as read-only
func filterReadOnlyTools(tools []mcp.Tool) []mcp.Tool {
	var readOnlyTools []mcp.Tool
	for _, tool := range tools {
		if tool.Annotations.ReadOnlyHint != nil && *tool.Annotations.ReadOnlyHint {
			readOnlyTools = append(readOnlyTools, tool)
		}
	}
	return readOnlyTools
}

// NewServer creates a new MCP server instance with configured tools
func NewServer(cfg *config.Config) *server.MCPServer {
	// Setup logger with configured level
	logger := logging.SetupLogger(cfg.LogLevel)

	// Create lifecycle hooks for logging and error handling
	hooks := &server.Hooks{}

	// Log client session lifecycle
	hooks.AddOnRegisterSession(func(ctx context.Context, session server.ClientSession) {
		logger.Info("Client session registered",
			"session_id", session.SessionID())
	})

	hooks.AddOnUnregisterSession(func(ctx context.Context, session server.ClientSession) {
		logger.Info("Client session unregistered",
			"session_id", session.SessionID())
	})

	// Log and handle errors
	hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
		logger.Error("Error in request",
			"method", method,
			"request_id", id,
			"error", err)
	})

	// Optional: Log all incoming requests (can be verbose)
	hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
		logger.Debug("Processing request",
			"method", method,
			"request_id", id)
	})

	// Create new MCP server with enhanced configuration
	serverOptions := []server.ServerOption{
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithRecovery(),
		server.WithHooks(hooks),
	}

	// Add read-only filter if needed
	if cfg.ReadOnly {
		serverOptions = append(serverOptions, server.WithToolFilter(func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
			return filterReadOnlyTools(tools)
		}))
	}

	s := server.NewMCPServer("honeybadger-mcp-server", "1.0.0", serverOptions...)

	// Create API client
	apiClient := hbapi.NewClient().
		WithBaseURL(cfg.APIURL).
		WithAuthToken(cfg.AuthToken)

	// Register project tools
	RegisterProjectTools(s, apiClient)

	// Register fault tools
	RegisterFaultTools(s, apiClient)

	// Register insights tools
	RegisterInsightsTools(s, apiClient)

	// Register dashboard tools
	RegisterDashboardTools(s, apiClient)

	return s
}
