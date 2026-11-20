package hbmcp

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
)

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

// jsonArg re-encodes an object or array argument as JSON text, for the
// handlers that decode it into api-go types with their own field checks. The
// strict check has made sure it is one. Absent or null comes back as "".
func jsonArg(req mcp.CallToolRequest, name string) string {
	v := req.GetArguments()[name]
	if v == nil {
		return ""
	}
	encoded, _ := json.Marshal(v) // decoded JSON always encodes again
	return string(encoded)
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
