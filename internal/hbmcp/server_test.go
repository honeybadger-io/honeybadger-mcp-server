package hbmcp

import (
	"testing"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
)

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		AuthToken: "test-token",
		APIURL:    "https://api.honeybadger.io/v2",
		LogLevel:  "info",
	}

	server := NewServer(cfg)
	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	// Basic test to ensure server is created
	// Note: mcp-go doesn't expose tool listing in the public API
	// We'll test functionality through manual testing
}
