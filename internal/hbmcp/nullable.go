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

// setOrClear returns name as a value when the caller sent one, null when the
// caller sent null or an empty string, and unset when the parameter is absent.
// It's for parameters whose description says an empty string or null clears the
// setting.
func setOrClear(req mcp.CallToolRequest, name string) nullable.Nullable[string] {
	switch v := req.GetArguments()[name].(type) {
	case nil:
		if _, present := req.GetArguments()[name]; present {
			return nullable.NewNullNullable[string]()
		}
	case string:
		if v == "" {
			return nullable.NewNullNullable[string]()
		}
		return nullable.NewNullableWithValue(v)
	}
	return nullable.Nullable[string]{}
}

// changesNothing reports whether an update's parameters would send an empty
// body. Comparing against the zero value no longer compiles once a struct holds
// nullable fields, and the encoded body is what the API sees anyway.
func changesNothing(params any) bool {
	body, err := json.Marshal(params)
	return err == nil && string(body) == "{}"
}
