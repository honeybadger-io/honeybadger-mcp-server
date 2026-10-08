package hbmcp

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// maxSafeInteger is the largest integer a float64 can represent exactly
// (2^53). JSON numbers decode to float64, so IDs above this can't round-trip
// without silent rounding and must be rejected rather than truncated.
const maxSafeInteger = 1 << 53

// acceptsNull is an mcp.PropertyOption that makes a property schema accept JSON
// null in addition to its declared type, e.g. {"type": ["number", "null"]}.
// Handlers must inspect the raw argument via req.GetArguments() to distinguish
// an explicit null from an omitted key.
//
// An enum must list null too, or a schema-checking client still refuses it, so
// acceptsNull adds it; pass acceptsNull after mcp.Enum.
func acceptsNull(schema map[string]any) {
	if t, ok := schema["type"].(string); ok {
		schema["type"] = []any{t, "null"}
	}
	if values, ok := schema["enum"].([]string); ok {
		withNull := make([]any, 0, len(values)+1)
		for _, v := range values {
			withNull = append(withNull, v)
		}
		schema["enum"] = append(withNull, nil)
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

// refuseNonStrings refuses any of the named parameters sent as something other
// than a string or null. mcp-go's GetString reads a number, boolean, array or
// object as absent, so without this a filter or setting sent in the wrong type
// would be skipped while the call reported success.
func refuseNonStrings(req mcp.CallToolRequest, names ...string) string {
	args := req.GetArguments()
	for _, name := range names {
		switch args[name].(type) {
		case nil, string:
		default:
			return fmt.Sprintf("%s must be a string; got %T", name, args[name])
		}
	}
	return ""
}

// jsonTextArg reads a parameter whose schema says a string holding JSON. Clients
// also send the JSON value itself, an object or an array, and reading only
// strings would drop it while the call reported success; so a value is encoded
// back to JSON text. Absent, null and an empty string all come back as "". Any
// other type is refused.
func jsonTextArg(req mcp.CallToolRequest, name string) (text, problem string) {
	switch v := req.GetArguments()[name].(type) {
	case nil:
		return "", ""
	case string:
		return strings.TrimSpace(v), ""
	case map[string]any, []any:
		encoded, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Sprintf("%s could not be read as JSON: %v", name, err)
		}
		return string(encoded), ""
	default:
		return "", name + " must be JSON: an object or array, or a string holding one"
	}
}

// positiveIntArg reads a whole number of at least 1. JSON numbers arrive as
// float64, but clients also send ints and numeric strings ("30"), so those are
// taken too. Anything else, a fraction, or a value under 1 is refused with a
// message rather than dropped: dropping it would report success for a setting
// that never changed.
func positiveIntArg(args map[string]any, name string) (n int, given bool, problem string) {
	raw, present := args[name]
	if !present || raw == nil {
		return 0, false, ""
	}
	var f float64
	switch v := raw.(type) {
	case float64:
		f = v
	case int:
		f = float64(v)
	case int64:
		f = float64(v)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, false, name + " must be a whole number of at least 1"
		}
		f = parsed
	default:
		return 0, false, name + " must be a whole number of at least 1"
	}
	if f < 1 || f != math.Trunc(f) || f > maxSafeInteger {
		return 0, false, name + " must be a whole number of at least 1"
	}
	return int(f), true, ""
}
