package hbmcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbapi"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/logging"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

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
	s := server.NewMCPServer(
		"honeybadger-mcp-server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithRecovery(),
		server.WithHooks(hooks),
	)

	// Create API client
	apiClient := hbapi.NewClient().
		WithBaseURL(cfg.APIURL).
		WithAuthToken(cfg.AuthToken)

	// Register the ping tool
	s.AddTool(
		mcp.NewTool("ping",
			mcp.WithDescription("Test connectivity to the MCP server"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result := map[string]interface{}{
				"status":    "pong",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			jsonBytes, err := json.Marshal(result)
			if err != nil {
				return mcp.NewToolResultError("Failed to marshal response"), nil
			}
			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	)

	// Register project tools
	RegisterProjectTools(s, apiClient)

	// Register fault tools
	RegisterFaultTools(s, apiClient)

	return s
}
