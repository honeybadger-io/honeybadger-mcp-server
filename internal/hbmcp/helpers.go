package hbmcp

import (
	"math"
	"strconv"
	"strings"
)

// maxSafeInteger is the largest integer a float64 can represent exactly
// (2^53). JSON numbers decode to float64, so IDs above this can't round-trip
// without silent rounding and must be rejected rather than truncated.
const maxSafeInteger = 1 << 53

// acceptsNull is an mcp.PropertyOption that makes a property schema accept JSON
// null in addition to its declared type, e.g. {"type": ["number", "null"]}.
// Handlers must inspect the raw argument via req.GetArguments() to distinguish
// an explicit null from an omitted key.
func acceptsNull(schema map[string]any) {
	if t, ok := schema["type"].(string); ok {
		schema["type"] = []any{t, "null"}
	}
}

// requireFaultID reads a fault id. v3 fault ids are opaque strings, but a whole
// number is still accepted, since callers may hold ids from before the change
// and the API accepts either. A fractional number is refused rather than
// truncated, so 456.9 can't act on fault 456.
func requireFaultID(args map[string]any, name string) (string, bool) {
	switch v := args[name].(type) {
	case string:
		if id := strings.TrimSpace(v); id != "" {
			return id, true
		}
	case float64:
		if v >= 1 && v <= maxSafeInteger && v == math.Trunc(v) {
			return strconv.FormatInt(int64(v), 10), true
		}
	case int: // arguments constructed in Go rather than decoded from JSON
		if v >= 1 {
			return strconv.Itoa(v), true
		}
	}
	return "", false
}
