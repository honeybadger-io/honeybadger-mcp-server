package hbmcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

func RegisterIntegrationTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	r.AddTool(
		generatedTool("get_integration"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetIntegration(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("create_integration"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateIntegration(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("update_integration"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateIntegration(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("delete_integration"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteIntegration(ctx, v3ClientFor(ctx), req)
		},
	)
}

func handleGetIntegration(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	integrationID := req.GetString("integration_id", "")
	if integrationID == "" {
		return mcp.NewToolResultError("integration_id is required"), nil
	}

	integration, err := client.Integrations.Get(ctx, projectID, integrationID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get integration: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(integration)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// integrationSettingFields are the settings every integration type shares. They
// sit beside config in the API's body, not inside it.
var integrationSettingFields = []string{
	"active", "events", "rate", "threshold", "notification_limit",
	"site_ids", "check_in_ids", "alarm_alert_ids", "alarm_ok_ids",
	"included_environments", "excluded_environments", "filters",
}

// integrationBody collects the shared settings and config into the API's body
// shape, as JSON for decoding into the create or update params. Decoding does the
// type checking: a site id that isn't a UUID or a fractional threshold is refused
// there rather than sent.
func integrationBody(req mcp.CallToolRequest, extra map[string]any) ([]byte, string) {
	args := req.GetArguments()
	body := map[string]any{}
	for k, v := range extra {
		body[k] = v
	}
	for _, field := range integrationSettingFields {
		if v, ok := args[field]; ok {
			body[field] = v
		}
	}
	raw, problem := jsonTextArg(req, "config")
	if problem != "" {
		return nil, problem
	}
	if raw != "" {
		var config map[string]any
		if err := json.Unmarshal([]byte(raw), &config); err != nil {
			return nil, fmt.Sprintf("Failed to parse config JSON: %v", err)
		}
		// The flat shape this tool used to take put these inside config; the API
		// now refuses them there, so say where they go instead.
		for _, field := range integrationSettingFields {
			if _, ok := config[field]; ok {
				return nil, fmt.Sprintf("%s is its own parameter, not a config setting: config holds only the type's settings", field)
			}
		}
		body["config"] = config
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, "Failed to encode the integration"
	}
	return encoded, ""
}

// decodeSettings decodes an integration body into params, and on failure says
// which setting was wrong in the tool's terms.
//
// A nullable field decodes through its own unmarshaler, so the type error it
// returns doesn't name the field. When that happens, the settings are decoded
// one at a time to find the one that fails.
func decodeSettings(raw []byte, params any) string {
	err := json.Unmarshal(raw, params)
	if err == nil {
		return ""
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) && typeErr.Field == "" {
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) == nil {
			// Sorted, so the same bad request always names the same setting.
			for _, name := range slices.Sorted(maps.Keys(fields)) {
				one, _ := json.Marshal(map[string]json.RawMessage{name: fields[name]})
				scratch := reflect.New(reflect.TypeOf(params).Elem()).Interface()
				var fieldErr *json.UnmarshalTypeError
				if errors.As(json.Unmarshal(one, scratch), &fieldErr) {
					// Describe this field's own error, so the name and the
					// expected type can't come from two different settings.
					fieldErr.Field = name
					return describeSettingError(fieldErr)
				}
			}
		}
	}
	return describeSettingError(err)
}

// describeSettingError turns a decoding failure into the tool's terms. The raw
// errors name Go types and struct fields, which mean nothing to the caller.
func describeSettingError(err error) string {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		want := "a string"
		switch typeErr.Type.Kind() {
		case reflect.Int:
			want = "a whole number"
		case reflect.Bool:
			want = "true or false"
		case reflect.Slice:
			want = "a list of strings"
			if typeErr.Field == "filters" {
				want = `a list of {"event": ..., "query": ...} objects`
			}
		}
		return fmt.Sprintf("%s must be %s", typeErr.Field, want)
	}
	// Only site_ids decodes into UUIDs, and the UUID parser doesn't say which
	// field it was reading.
	if strings.Contains(err.Error(), "UUID") {
		return fmt.Sprintf("site_ids must be site IDs, which are UUIDs: %v", err)
	}
	return fmt.Sprintf("Invalid integration settings: %v", err)
}

func handleCreateIntegration(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	integrationType := req.GetString("type", "")
	if integrationType == "" {
		return mcp.NewToolResultError("type is required"), nil
	}
	if msg := rejectStaleSchemaFields("create_integration", req); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	raw, msg := integrationBody(req, map[string]any{"type": integrationType})
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	var params apiv3.IntegrationCreateParams
	if msg := decodeSettings(raw, &params); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}

	integration, err := client.Integrations.Create(ctx, projectID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create integration: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(integration)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateIntegration(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	integrationID := req.GetString("integration_id", "")
	if integrationID == "" {
		return mcp.NewToolResultError("integration_id is required"), nil
	}

	if msg := rejectStaleSchemaFields("update_integration", req); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	raw, msg := integrationBody(req, nil)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	var params apiv3.IntegrationUpdateParams
	if msg := decodeSettings(raw, &params); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	if changesNothing(params) {
		return mcp.NewToolResultError("provide at least one setting to change"), nil
	}

	integration, err := client.Integrations.Update(ctx, projectID, integrationID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update integration: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(integration)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteIntegration(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	integrationID := req.GetString("integration_id", "")
	if integrationID == "" {
		return mcp.NewToolResultError("integration_id is required"), nil
	}

	if !deletionConfirmed(ctx, req, "delete_integration", projectID, integrationID) {
		integration, err := client.Integrations.Get(ctx, projectID, integrationID)
		var summary string
		switch {
		case err == nil:
			name := integration.Type
			if label := integrationLabel(integration); label != "" {
				name = fmt.Sprintf("%s integration %q", integration.Type, label)
			} else {
				name += " integration"
			}
			summary = fmt.Sprintf("delete the %s (id %s) from project %s, with its tickets and configuration", name, integrationID, projectID)
		case unreadable(err):
			summary = fmt.Sprintf("delete integration %s from project %s, with its tickets and configuration", integrationID, projectID) + unreadableNote
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Failed to look up integration: %v", err)), nil
		}
		return deletionPreview(ctx, req, "delete_integration", summary, projectID, integrationID), nil
	}

	if err := client.Integrations.Delete(ctx, projectID, integrationID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete integration: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Integration %s deleted successfully", integrationID)), nil
}

// integrationLabel returns an integration's label, or "" when it has none. A
// project can hold several integrations of one type, and the label is what tells
// them apart in a deletion preview.
func integrationLabel(integration *apiv3.Integration) string {
	label, _ := integration.Config["label"].(string)
	return label
}
