package hbmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"strings"
	"testing"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

const siteUUIDArg = "5b7c2d1e-8f3a-4c6b-9d2e-1a2b3c4d5e6f"

func registeredTools(t *testing.T) map[string]mcp.Tool {
	t.Helper()
	tools := map[string]mcp.Tool{}
	for name, st := range NewServer(&config.Config{APIURL: "http://x"}, "t").ListTools() {
		tools[name] = st.Tool
	}
	return tools
}

// checked runs a call through the strict check of tool's registered
// definition, with a handler that only records whether it ran.
func checked(t *testing.T, tools map[string]mcp.Tool, tool string, args any) (text string, ran bool, logs string) {
	t.Helper()
	def, ok := tools[tool]
	if !ok {
		t.Fatalf("no tool %s", tool)
	}
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	h := strictArgs(def, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ran = true
		return mcp.NewToolResultText("ok"), nil
	}, logger)
	result, err := h(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool, Arguments: args}})
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	if ran == result.IsError {
		t.Fatalf("%s: ran = %v but IsError = %v", tool, ran, result.IsError)
	}
	return getResultText(result), ran, buf.String()
}

func TestStrictArgsMessages(t *testing.T) {
	tools := registeredTools(t)
	for name, tc := range map[string]struct {
		tool string
		args map[string]any
		want string
	}{
		"misspelled next to a valid key": {
			"update_fault",
			map[string]any{"project_id": "Xk9mZp", "fault_id": "f1", "assignee_id": "u1", "resolve": true},
			`update_fault has no parameter "resolve". Did you mean "resolved"? The call was not made. If you expected "resolve" to exist, your tool list may be stale; reconnect to refresh tool schemas.`,
		},
		"misspelled site setting next to a valid one": {
			"update_site",
			map[string]any{"project_id": "Xk9mZp", "site_id": siteUUIDArg, "name": "Shop", "frequncy": 5},
			`update_site has no parameter "frequncy". Did you mean "frequency"? The call was not made. If you expected "frequncy" to exist, your tool list may be stale; reconnect to refresh tool schemas.`,
		},
		"unknown with nothing close": {
			"update_fault",
			map[string]any{"project_id": "Xk9mZp", "fault_id": "f1", "colour": "red"},
			`update_fault has no parameter "colour". Its parameters are: assignee_id, fault_id, ignored, project_id, resolve_on_deploy, resolved, snooze, snooze_occurrences. The call was not made. If you expected "colour" to exist, your tool list may be stale; reconnect to refresh tool schemas.`,
		},
		"stale v2 field": {
			"list_projects",
			map[string]any{"account_id": "Ab3kL9"},
			`list_projects no longer accepts "account_id": v3 takes the account from the credential. The call was not made. Your tool list is stale; reconnect to refresh tool schemas.`,
		},
		"renamed parameter": {
			"get_project",
			map[string]any{"id": "Xk9mZp"},
			`get_project no longer accepts "id": renamed to project_id. The call was not made. Your tool list is stale; reconnect to refresh tool schemas.`,
		},
		"wrong type with an enum": {
			"update_site",
			map[string]any{"project_id": "Xk9mZp", "site_id": siteUUIDArg, "frequency": "60"},
			`update_site: frequency must be one of 1, 2, 5, 15 or null; got a string ("60"). The call was not made.`,
		},
		"null where not allowed": {
			"update_site",
			map[string]any{"project_id": "Xk9mZp", "site_id": siteUUIDArg, "name": nil},
			`update_site: name can't be null; omit it to leave the setting unchanged. The call was not made.`,
		},
		"null for a required parameter": {
			"get_site",
			map[string]any{"project_id": nil, "site_id": siteUUIDArg},
			`get_site: project_id can't be null; it is required. The call was not made.`,
		},
		"null for an optional parameter outside an update": {
			"list_faults",
			map[string]any{"project_id": "Xk9mZp", "q": nil},
			`list_faults: q can't be null; omit it instead. The call was not made.`,
		},
		"several problems at one nested place": {
			"update_alarm",
			map[string]any{"project_id": "Xk9mZp", "alarm_id": "a1", "trigger_config": map[string]any{}},
			"update_alarm refused these arguments; the call was not made:\n" +
				"- trigger_config: needs \"type\".\n" +
				"- trigger_config: needs \"config\".\n" +
				"See get_reference topic \"alarms\".",
		},
		"fraction for an integer": {
			"list_deploys",
			map[string]any{"project_id": "Xk9mZp", "limit": 2.5},
			`list_deploys: limit must be a whole number; got 2.5. The call was not made.`,
		},
		"JSON text where an object is expected": {
			"update_alarm",
			map[string]any{"project_id": "Xk9mZp", "alarm_id": "a1", "trigger_config": `{"type":"alert_result_count"}`},
			`update_alarm: trigger_config must be an object; got a string ("{\"type\":\"alert_result_count\"}"). The call was not made.`,
		},
		"numeric ID with a list tool": {
			"get_site",
			map[string]any{"project_id": 12345, "site_id": siteUUIDArg},
			`get_site: project_id must be a string; got the number 12345. IDs are strings now: call list_projects for the current one. If your tool definitions show project_id as a number, they are stale; reconnect to refresh tool schemas.`,
		},
		"numeric ID without a list tool": {
			"update_fault_comment",
			map[string]any{"project_id": "Xk9mZp", "fault_id": "f1", "comment_id": 7, "body": "x"},
			`update_fault_comment: comment_id must be a string; got the number 7. IDs are strings now: call the tool that lists them for the current one. If your tool definitions show comment_id as a number, they are stale; reconnect to refresh tool schemas.`,
		},
		"missing required": {
			"get_site",
			map[string]any{"site_id": siteUUIDArg},
			`get_site: project_id is required. The call was not made.`,
		},
		"unknown key inside a shallow parameter": {
			"create_dashboard",
			map[string]any{"project_id": "Xk9mZp", "title": "Ops", "widgets": []any{map[string]any{"type": "errors", "colour": 1}}},
			`create_dashboard: widgets[0] has no field "colour". The call was not made. See get_reference topic "dashboards".`,
		},
		"several problems": {
			"update_site",
			map[string]any{"project_id": "Xk9mZp", "site_id": siteUUIDArg, "frequncy": 5, "timeout": "30"},
			"update_site refused these arguments; the call was not made:\n" +
				"- frequncy: no such parameter. Did you mean \"frequency\"?\n" +
				"- timeout: must be a whole number or null; got a string (\"30\").\n" +
				"If you expected a parameter to exist, your tool list may be stale; reconnect to refresh tool schemas.",
		},
	} {
		t.Run(name, func(t *testing.T) {
			text, ran, _ := checked(t, tools, tc.tool, tc.args)
			if ran || text != tc.want {
				t.Errorf("ran = %v, text =\n%s\nwant\n%s", ran, text, tc.want)
			}
		})
	}
}

func TestStrictArgsRefusesArgumentsThatArentAnObject(t *testing.T) {
	text, ran, _ := checked(t, registeredTools(t), "get_site", "oops")
	if want := "get_site: arguments must be a JSON object. The call was not made."; ran || text != want {
		t.Errorf("ran = %v, text =\n%s\nwant\n%s", ran, text, want)
	}
}

func TestStrictArgsLetsAValidCallThrough(t *testing.T) {
	tools := registeredTools(t)
	for tool, args := range map[string]any{
		"update_site":        map[string]any{"project_id": "Xk9mZp", "site_id": siteUUIDArg, "name": "Shop", "frequency": 1, "match": nil},
		"get_project":        map[string]any{"project_id": "Xk9mZp"},
		"get_reference":      nil,
		"create_dashboard":   map[string]any{"project_id": "Xk9mZp", "title": "Ops", "widgets": []any{map[string]any{"type": "errors"}}},
		"update_fault":       map[string]any{"project_id": "Xk9mZp", "fault_id": "f1", "resolved": true, "assignee_id": nil},
		"query_insights":     map[string]any{"project_id": "Xk9mZp", "query": "fields @ts"},
		"list_fault_notices": map[string]any{"project_id": "Xk9mZp", "fault_id": "f1", "limit": 5},
	} {
		if text, ran, _ := checked(t, tools, tool, args); !ran {
			t.Errorf("%s refused a valid call: %s", tool, text)
		}
	}
}

func TestStrictArgsLogsEachRefusal(t *testing.T) {
	_, _, logs := checked(t, registeredTools(t), "update_fault",
		map[string]any{"project_id": "Xk9mZp", "fault_id": "f1", "resolve": true})
	if n := strings.Count(logs, "level=WARN"); n != 1 {
		t.Fatalf("got %d WARN records, want 1:\n%s", n, logs)
	}
	for _, want := range []string{`msg="tool arguments refused"`, "tool=update_fault", "field=resolve", "kind=unknown"} {
		if !strings.Contains(logs, want) {
			t.Errorf("log lacks %s:\n%s", want, logs)
		}
	}
}

// Every tool refuses a key it doesn't declare, naming it.
func TestEveryToolRefusesUnknownKeys(t *testing.T) {
	tools := registeredTools(t)
	for name := range tools {
		text, ran, _ := checked(t, tools, name, map[string]any{"zz_unknown": 1})
		if ran || !strings.Contains(text, "zz_unknown") || !strings.Contains(text, "no parameter") && !strings.Contains(text, "no such parameter") {
			t.Errorf("%s: ran = %v, text = %s", name, ran, text)
		}
	}
}

// Every declared parameter refuses a value of a type it doesn't take, and
// null unless its schema allows it.
func TestEveryParameterIsTypeChecked(t *testing.T) {
	tools := registeredTools(t)
	for name, tool := range tools {
		props := schemaProperties(t, tool)
		for param, schema := range props {
			types := allowedTypes(schema)
			wrong, ok := valueOutside(types)
			if ok {
				text, ran, _ := checked(t, tools, name, map[string]any{param: wrong})
				if ran || !mentionsProblem(text, param, "must be") {
					t.Errorf("%s.%s given %#v: ran = %v, text = %s", name, param, wrong, ran, text)
				}
			}
			text, ran, _ := checked(t, tools, name, map[string]any{param: nil})
			refusedNull := mentionsProblem(text, param, "can't be null")
			if allowsNullType(types) == refusedNull {
				t.Errorf("%s.%s given null: allows null = %v but refused = %v (ran = %v): %s", name, param, allowsNullType(types), refusedNull, ran, text)
			}
		}
	}
}

func TestEveryStaleFieldGivesItsReason(t *testing.T) {
	tools := registeredTools(t)
	for tool, fields := range staleSchemaFields {
		for field, reason := range fields {
			text, ran, _ := checked(t, tools, tool, map[string]any{field: "x"})
			if ran || !strings.Contains(text, reason) || !strings.Contains(text, "reconnect") {
				t.Errorf("%s.%s: ran = %v, text = %s", tool, field, ran, text)
			}
		}
	}
}

func schemaProperties(t *testing.T, tool mcp.Tool) map[string]map[string]any {
	t.Helper()
	raw, _ := json.Marshal(tool.InputSchema)
	var s struct {
		Properties map[string]map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	return s.Properties
}

func allowedTypes(schema map[string]any) []string {
	switch t := schema["type"].(type) {
	case string:
		return []string{t}
	case []any:
		var out []string
		for _, v := range t {
			out = append(out, v.(string))
		}
		sort.Strings(out)
		return out
	}
	return nil
}

func allowsNullType(types []string) bool {
	for _, t := range types {
		if t == "null" {
			return true
		}
	}
	return false
}

// valueOutside returns a value of a JSON type the schema doesn't take.
func valueOutside(types []string) (any, bool) {
	has := map[string]bool{}
	for _, t := range types {
		has[t] = true
	}
	for _, c := range []struct {
		typ   string
		value any
	}{{"boolean", true}, {"string", "x"}, {"object", map[string]any{}}, {"array", []any{}}} {
		if !has[c.typ] {
			return c.value, true
		}
	}
	return nil, false
}

// mentionsProblem matches both the single-problem and the list form.
func mentionsProblem(text, param, problem string) bool {
	return strings.Contains(text, ": "+param+" "+problem) || strings.Contains(text, "- "+param+": "+problem)
}
