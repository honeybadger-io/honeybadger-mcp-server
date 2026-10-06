package hbmcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

func RegisterIntegrationTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	r.AddTool(
		mcp.NewTool("get_integration",
			mcp.WithTitleAnnotation("Get Integration"),
			mcp.WithDescription("Get a single notification integration by ID. To interpret integration fields, fetch reference topic: integrations (via get_reference)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the integration belongs to"),
			),
			mcp.WithString("integration_id",
				mcp.Required(),
				mcp.Description("The ID of the integration to retrieve"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetIntegration(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("create_integration", append([]mcp.ToolOption{
			mcp.WithTitleAnnotation("Create Integration"),
			mcp.WithDescription("Create a notification integration for a project. IMPORTANT: Requires reference topic: integrations — fetch via get_reference first (skip if still visible in your context) for the list of types, each type's config settings, and event names."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the integration in"),
			),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("Integration type: WebHook, Email, PagerDutyV2, Slack, and so on. OAuth types (Slack, GitHub and the like) are created unconnected: send the user to the result's links.web to connect it. They can be active from the start; they send nothing until connected."),
			),
			mcp.WithString("config",
				mcp.Description(`JSON object of the type's own settings, e.g. {"url":"https://example.com/hook","label":"Deploys"} for WebHook or {"integration_key":"..."} for PagerDutyV2. The shared settings (events, site_ids and the rest) are separate parameters, not part of config.`),
			),
		}, integrationSettingOptions()...)...),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateIntegration(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("update_integration", append([]mcp.ToolOption{
			mcp.WithTitleAnnotation("Update Integration"),
			mcp.WithDescription("Update a notification integration. Only the parameters given are changed; the type can't be. IMPORTANT: Requires reference topic: integrations — fetch via get_reference first (skip if still visible in your context) for config settings and event names."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the integration belongs to"),
			),
			mcp.WithString("integration_id",
				mcp.Required(),
				mcp.Description("The ID of the integration to update"),
			),
			mcp.WithString("config",
				mcp.Description("JSON object of the type's own settings to change, the same keys get_integration returns under config. A secret sent back masked, exactly as read, is left unchanged."),
			),
		}, integrationSettingOptions()...)...),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateIntegration(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("delete_integration",
			mcp.WithTitleAnnotation("Delete Integration"),
			mcp.WithDescription("Delete a notification integration. Also deletes all associated tickets and configurations."+confirmNote),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project the integration belongs to"),
			),
			mcp.WithString("integration_id",
				mcp.Required(),
				mcp.Description("The ID of the integration to delete"),
			),
			withConfirmParam(),
		),
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
	"included_environments", "excluded_environments", "filters", "all_sites", "all_check_ins",
}

// integrationSettingOptions declares the shared settings, in the same order, for
// create_integration and update_integration.
func integrationSettingOptions() []mcp.ToolOption {
	list := func(name, description string) mcp.ToolOption {
		return mcp.WithArray(name, mcp.WithStringItems(), mcp.Description(description))
	}
	return []mcp.ToolOption{
		mcp.WithBoolean("active", mcp.Description("Whether the integration sends notifications")),
		list("events", "Event names to notify on, e.g. occurred, resolved, assigned, down, check_in_missing. Adding one the type doesn't support is refused. On create, omit for the type's defaults."),
		mcp.WithString("rate", mcp.Description("Period for the rate_exceeded event: minute, hour, day, and so on")),
		mcp.WithNumber("threshold", mcp.Description("Occurrences within rate before rate_exceeded fires; must be greater than 0")),
		mcp.WithNumber("notification_limit", mcp.Description("Most notifications in a 10-minute window before flood control (the flooded event) steps in")),
		list("site_ids", "Uptime sites whose up and down events notify, when all_sites is off. An empty list means none; an ID from outside the project is refused. Sending a non-empty list turns all_sites off."),
		list("check_in_ids", "Check-ins whose events notify, when all_check_ins is off. An empty list means none; an ID from outside the project is refused. Sending a non-empty list turns all_check_ins off."),
		list("alarm_alert_ids", "Alarms whose alert events notify"),
		list("alarm_ok_ids", "Alarms whose recovery events notify"),
		list("included_environments", "When non-empty, the only environments that notify, including ones that haven't reported yet. Empty means every environment not excluded."),
		list("excluded_environments", "Environments that never notify; wins over included_environments. Names are stored as given, so one can be excluded before it first reports."),
		mcp.WithArray("filters",
			mcp.Description(`Ordered list of {"event": ..., "query": ...} objects: each event with a filter notifies only for errors matching its query. event is an event name, or "all" for every event. Replaces the stored filters; send [] or null to clear them.`),
			mcp.Items(map[string]any{
				"type":     "object",
				"required": []string{"event", "query"},
				"properties": map[string]any{
					"event": map[string]any{"type": "string"},
					"query": map[string]any{"type": "string"},
				},
			}),
		),
		mcp.WithBoolean("all_sites", mcp.Description("Follow every uptime site in the project, including ones added later. When true, leave site_ids out.")),
		mcp.WithBoolean("all_check_ins", mcp.Description("Follow every check-in in the project, including ones added later. When true, leave check_in_ids out.")),
	}
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
	if raw := req.GetString("config", ""); raw != "" {
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
			for name, value := range fields {
				one, _ := json.Marshal(map[string]json.RawMessage{name: value})
				scratch := reflect.New(reflect.TypeOf(params).Elem()).Interface()
				if json.Unmarshal(one, scratch) != nil {
					typeErr.Field = name
					break
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
