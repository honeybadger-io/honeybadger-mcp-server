package hbmcp

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oapi-codegen/nullable"
)

// The v3 inputs mark the fields an update can clear as nullable.Nullable: unset
// leaves the field alone, a value sets it, and null clears it. Tool parameters
// are plain strings, so these helpers map them onto that.

// setIfGiven returns name as a value when the caller sent a non-empty string,
// and unset otherwise.
func setIfGiven(req mcp.CallToolRequest, name string) nullable.Nullable[string] {
	if v := req.GetString(name, ""); v != "" {
		return nullable.NewNullableWithValue(v)
	}
	return nullable.Nullable[string]{}
}

// setOrClear returns name as a value when the caller sent a string, empty or
// not, null when the caller sent null, and unset when the parameter is absent.
// Null is what clears a setting, as the spec has it.
func setOrClear(req mcp.CallToolRequest, name string) nullable.Nullable[string] {
	v, present := req.GetArguments()[name]
	switch s, ok := v.(string); {
	case !present:
		return nullable.Nullable[string]{}
	case ok:
		return nullable.NewNullableWithValue(s)
	default:
		return nullable.NewNullNullable[string]()
	}
}

// changesNothing reports whether an update's parameters would send an empty
// body. Comparing against the zero value no longer compiles once a struct holds
// nullable fields, and the encoded body is what the API sees anyway.
func changesNothing(params any) bool {
	body, err := json.Marshal(params)
	return err == nil && string(body) == "{}"
}
