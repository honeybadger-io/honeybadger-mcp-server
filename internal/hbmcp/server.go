package hbmcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates a new MCP server instance with configured tools
func NewServer(cfg *config.Config) *server.MCPServer {
	// Create new MCP server
	s := server.NewMCPServer(
		"honeybadger-mcp-server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

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

	return s
}