package hbmcp

import (
	"strconv"
	"time"
)

// parseUnixTimestamp converts a timestamp string to Unix epoch (int64).
// Accepts either RFC3339 format or Unix timestamp string.
// Returns 0 if empty or invalid.
// Use for API endpoints that expect Unix timestamps (faults).
func parseUnixTimestamp(ts string) int64 {
	if ts == "" {
		return 0
	}
	// Try parsing as Unix timestamp first
	if unix, err := strconv.ParseInt(ts, 10, 64); err == nil {
		return unix
	}
	// Try parsing as RFC3339
	if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
		return parsed.Unix()
	}
	return 0
}

// parseTimestamp converts a timestamp string to *time.Time.
// Returns nil if empty or invalid.
// Use for API endpoints that expect RFC3339 format (project reports).
func parseTimestamp(ts string) *time.Time {
	if ts == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
		return &parsed
	}
	return nil
}
