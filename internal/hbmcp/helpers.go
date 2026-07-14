package hbmcp

import (
	"math"
	"time"
)

// maxSafeInteger is the largest integer a float64 can represent exactly
// (2^53). JSON numbers decode to float64, so IDs above this can't round-trip
// without silent rounding and must be rejected rather than truncated.
const maxSafeInteger = 1 << 53

// nullable is an mcp.PropertyOption that makes a property schema accept JSON
// null in addition to its declared type, e.g. {"type": ["number", "null"]}.
// Handlers must inspect the raw argument via req.GetArguments() to distinguish
// an explicit null from an omitted key.
func nullable(schema map[string]any) {
	if t, ok := schema["type"].(string); ok {
		schema["type"] = []any{t, "null"}
	}
}

// requireID extracts a positive integer argument, rejecting the fractional,
// string, and negative values that req.GetInt would silently coerce. Use for
// resource IDs in destructive handlers, where a truncated 456.9 would target
// the wrong resource.
func requireID(args map[string]any, name string) (int, bool) {
	switch v := args[name].(type) {
	case float64:
		if v >= 1 && v <= maxSafeInteger && v == math.Trunc(v) {
			return int(v), true
		}
	case int: // arguments constructed in Go rather than decoded from JSON
		if v >= 1 {
			return v, true
		}
	}
	return 0, false
}

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
