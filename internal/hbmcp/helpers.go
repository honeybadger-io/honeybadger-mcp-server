package hbmcp

import "time"

// parseTimestamp converts a timestamp string to *time.Time, returns nil if empty or invalid
func parseTimestamp(ts string) *time.Time {
	if ts == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
		return &parsed
	}
	return nil
}

// parseTimestampValue converts a timestamp string to time.Time, returns zero value if empty or invalid
func parseTimestampValue(ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
		return parsed
	}
	return time.Time{}
}
