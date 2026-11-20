package hbmcp

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
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

// bodyFrom decodes the arguments that aren't path parameters into an api-go
// params struct. The tool's body arguments are the spec's body fields by name,
// so every one the tool advertises reaches the request without a line per
// field, and null reaches it as null.
func bodyFrom(req mcp.CallToolRequest, params any, pathParams ...string) string {
	body := map[string]any{}
	for k, v := range req.GetArguments() {
		if !slices.Contains(pathParams, k) {
			body[k] = v
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Sprintf("arguments could not be read as JSON: %v", err)
	}
	return decodeSettings(raw, params)
}
