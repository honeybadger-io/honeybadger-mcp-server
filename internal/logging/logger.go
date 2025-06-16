package logging

import (
	"log/slog"
	"os"
	"strings"
)

// SetupLogger configures slog with the specified log level
func SetupLogger(level string) *slog.Logger {
	var slogLevel slog.Level

	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn", "warning":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo // default to info
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)

	// Set as default logger
	slog.SetDefault(logger)

	return logger
}

// ParseLogLevel converts a string log level to slog.Level
func ParseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
