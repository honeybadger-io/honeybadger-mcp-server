package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oapi-codegen/nullable"
)

// RegisterAlarmTools registers all alarm-related MCP tools
// RegisterAlarmTools registers the alarm tools, all on v3.
func RegisterAlarmTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	// list_alarms tool
	r.AddTool(
		generatedTool("list_alarms"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListAlarms(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_alarm tool
	r.AddTool(
		generatedTool("get_alarm"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetAlarm(ctx, v3ClientFor(ctx), req)
		},
	)

	// create_alarm tool
	r.AddTool(
		generatedTool("create_alarm"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateAlarm(ctx, v3ClientFor(ctx), req)
		},
	)

	// update_alarm tool
	r.AddTool(
		generatedTool("update_alarm"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateAlarm(ctx, v3ClientFor(ctx), req)
		},
	)

	// delete_alarm tool
	r.AddTool(
		generatedTool("delete_alarm"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteAlarm(ctx, v3ClientFor(ctx), req)
		},
	)

	// get_alarm_history tool
	r.AddTool(
		generatedTool("get_alarm_history"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetAlarmHistory(ctx, v3ClientFor(ctx), req)
		},
	)
}

func handleListAlarms(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	response, err := client.Alarms.List(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list alarms: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetAlarm(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	alarmID := req.GetString("alarm_id", "")
	if alarmID == "" {
		return mcp.NewToolResultError("alarm_id is required"), nil
	}

	alarm, err := client.Alarms.Get(ctx, projectID, alarmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get alarm: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(alarm)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// parseTrigger reads a trigger_config argument. Create and update take the same
// shape, as the API does, and it's sent whole.
//
// Unknown keys are refused, and config must name its value: a misspelt key such
// as "threshold" would otherwise decode as a value of 0 and store an alarm that
// fires on any result.
func parseTrigger(raw string) (*apiv3.AlarmTriggerConfig, error) {
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	var trigger apiv3.AlarmTriggerConfig
	if err := dec.Decode(&trigger); err != nil {
		return nil, err
	}
	var present struct {
		Config map[string]json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal([]byte(raw), &present); err != nil {
		return nil, err
	}
	if v, ok := present.Config["value"]; !ok || string(v) == "null" {
		return nil, fmt.Errorf("config.value is required")
	}
	return &trigger, nil
}

func handleCreateAlarm(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}
	period := req.GetString("evaluation_period", "")
	if period == "" {
		return mcp.NewToolResultError("evaluation_period is required"), nil
	}
	lag := req.GetString("lookback_lag", "")
	if lag == "" {
		return mcp.NewToolResultError("lookback_lag is required ('0s' for none)"), nil
	}
	rawTrigger, problem := jsonTextArg(req, "trigger_config")
	if problem != "" {
		return mcp.NewToolResultError(problem), nil
	}
	if rawTrigger == "" {
		return mcp.NewToolResultError("trigger_config is required"), nil
	}
	trigger, err := parseTrigger(rawTrigger)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse trigger_config JSON: %v", err)), nil
	}

	params := apiv3.AlarmCreateParams{
		Name:             name,
		Query:            query,
		EvaluationPeriod: period,
		LookbackLag:      lag,
		TriggerConfig:    *trigger,
		Description:      setIfGiven(req, "description"),
	}

	// On create, null means the same as leaving it out: every current stream.
	ids, isNull, given, problem := streamIDsArg(req)
	if problem != "" {
		return mcp.NewToolResultError(problem), nil
	}
	if given && !isNull {
		params.StreamIds = nullable.NewNullableWithValue(ids)
	}

	alarm, err := client.Alarms.Create(ctx, projectID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create alarm: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(alarm)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleUpdateAlarm changes whichever of an alarm's fields the caller supplies.
func handleUpdateAlarm(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	alarmID := req.GetString("alarm_id", "")
	if alarmID == "" {
		return mcp.NewToolResultError("alarm_id is required"), nil
	}
	// Presence, not emptiness: an empty description clears it, while omitting a
	// field leaves it alone.
	args := req.GetArguments()
	params := apiv3.AlarmUpdateParams{Description: setOrClear(req, "description")}
	for field, target := range map[string]**string{
		"name":              &params.Name,
		"query":             &params.Query,
		"evaluation_period": &params.EvaluationPeriod,
		"lookback_lag":      &params.LookbackLag,
	} {
		if v, ok := args[field].(string); ok {
			value := v
			*target = &value
		}
	}
	ids, isNull, given, problem := streamIDsArg(req)
	switch {
	case problem != "":
		return mcp.NewToolResultError(problem), nil
	case isNull:
		params.StreamIds = nullable.NewNullNullable[[]string]()
	case given:
		params.StreamIds = nullable.NewNullableWithValue(ids)
	}
	rawTrigger, problem := jsonTextArg(req, "trigger_config")
	if problem != "" {
		return mcp.NewToolResultError(problem), nil
	}
	if raw := rawTrigger; raw != "" {
		trigger, err := parseTrigger(raw)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse trigger_config JSON: %v", err)), nil
		}
		params.TriggerConfig = trigger
	}
	if changesNothing(params) {
		return mcp.NewToolResultError("provide at least one field to change"), nil
	}

	alarm, err := client.Alarms.Update(ctx, projectID, alarmID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update alarm: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(alarm)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleDeleteAlarm(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	alarmID := req.GetString("alarm_id", "")
	if alarmID == "" {
		return mcp.NewToolResultError("alarm_id is required"), nil
	}

	if !deletionConfirmed(ctx, req, "delete_alarm", projectID, alarmID) {
		alarm, err := client.Alarms.Get(ctx, projectID, alarmID)
		var summary string
		switch {
		case err == nil:
			summary = fmt.Sprintf("delete alarm %q (id %s) from project %s", alarm.Name, alarmID, projectID)
		case unreadable(err):
			summary = fmt.Sprintf("delete alarm %s from project %s", alarmID, projectID) + unreadableNote
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Failed to look up alarm: %v", err)), nil
		}
		return deletionPreview(ctx, req, "delete_alarm", summary, projectID, alarmID), nil
	}

	err := client.Alarms.Delete(ctx, projectID, alarmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete alarm: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(map[string]any{"deleted": true, "alarm_id": alarmID})
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetAlarmHistory(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}

	alarmID := req.GetString("alarm_id", "")
	if alarmID == "" {
		return mcp.NewToolResultError("alarm_id is required"), nil
	}

	// Page size is fixed at 25, so this endpoint takes page but no per_page.
	var opts []apiv3.Option
	if page := req.GetInt("page", 0); page > 0 {
		opts = append(opts, apiv3.Page(page, 0))
	}

	history, err := client.Alarms.ListHistory(ctx, projectID, alarmID, opts...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get alarm history: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(history)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// streamIDsArg reads stream_ids. Its schema says a string holding a JSON array,
// as v2's tool took it, but clients also send the array itself, and JSON null
// (or the string "null") means every current stream. Anything else is refused
// rather than dropped, so an alarm isn't left on streams the caller meant to
// change.
func streamIDsArg(req mcp.CallToolRequest) (ids []string, isNull, given bool, problem string) {
	raw, present := req.GetArguments()["stream_ids"]
	if !present {
		return nil, false, false, ""
	}
	switch v := raw.(type) {
	case nil:
		return nil, true, true, ""
	case string:
		v = strings.TrimSpace(v)
		switch v {
		case "":
			return nil, false, false, ""
		case "null":
			return nil, true, true, ""
		}
		if err := json.Unmarshal([]byte(v), &ids); err != nil {
			return nil, false, false, fmt.Sprintf("Failed to parse stream_ids JSON: %v", err)
		}
		return ids, false, true, ""
	case []any:
		ids = make([]string, 0, len(v))
		for _, item := range v {
			id, ok := item.(string)
			if !ok {
				return nil, false, false, fmt.Sprintf("stream_ids must be a list of stream ID strings; got %v", item)
			}
			ids = append(ids, id)
		}
		return ids, false, true, ""
	default:
		return nil, false, false, "stream_ids must be a JSON array of stream IDs, or null"
	}
}
