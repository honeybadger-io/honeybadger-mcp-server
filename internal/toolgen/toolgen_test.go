package toolgen

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/bundle.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// createWidget is an overlay entry that describes every parameter, so tests can
// add one override at a time.
const createWidget = `
create_widget:
  operation: createWidget
  title: Create Widget
  description: Create a widget.
  params:
    project_id: { description: Project ID. }
    name: { description: Widget name. }
    color: { description: Color. }
    size: { description: Size. }
    parts: { description: Parts. }
    labels: { description: Labels. }
    kind: { description: Kind. }
`

func buildOne(t *testing.T, overlay string) Tool {
	t.Helper()
	tools, err := Build(fixture(t), []byte(overlay))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(tools))
	}
	return tools[0]
}

// asJSON normalises a schema for comparison, so map and slice types don't matter.
func asJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	_ = json.Unmarshal(b, &out)
	b, _ = json.Marshal(out)
	return string(b)
}

func TestFlattensPathAndBodyIntoOneStrictObject(t *testing.T) {
	tool := buildOne(t, createWidget)

	want := `{
		"type": "object",
		"additionalProperties": false,
		"required": ["name", "project_id"],
		"properties": {
			"project_id": {"type": "string", "description": "Project ID."},
			"name": {"type": "string", "description": "Widget name."},
			"color": {"type": ["string", "null"], "enum": ["red", "blue", null], "description": "Color."},
			"size": {"type": ["integer", "null"], "default": 3, "description": "Size."},
			"parts": {"type": "array", "description": "Parts.", "items": {
				"type": "object", "additionalProperties": false, "required": ["label"],
				"properties": {"label": {"type": "string"}, "weight": {"type": "number"}},
				"if": {"properties": {"label": {"const": "heavy"}}},
				"then": {"properties": {"weight": {"type": "number", "minimum": 10}}, "required": ["label"]}}},
			"labels": {"type": "object", "additionalProperties": {"type": "string"}, "description": "Labels."},
			"kind": {"type": "string", "enum": ["a", "b"], "description": "Kind."}
		}
	}`
	if got := asJSON(t, tool.Schema); got != asJSON(t, jsonValue(t, want)) {
		t.Errorf("schema =\n%s\nwant\n%s", got, asJSON(t, jsonValue(t, want)))
	}
	if tool.Name != "create_widget" || tool.Title != "Create Widget" || tool.Description != "Create a widget." {
		t.Errorf("tool = %q %q %q", tool.Name, tool.Title, tool.Description)
	}
}

func jsonValue(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("bad want JSON: %v", err)
	}
	return v
}

func TestQueryParametersAndOverrides(t *testing.T) {
	tool := buildOne(t, `
list_widgets:
  operation: listWidgets
  title: List Widgets
  description: List widgets.
  params:
    project_id: { description: Project ID. }
    since: { description: Since when., as: date, reason: Agents write dates. }
    label: { exclude: Not useful to agents. }
    per_page: { description: Page size. }
`)
	want := `{
		"type": "object",
		"additionalProperties": false,
		"required": ["project_id"],
		"properties": {
			"project_id": {"type": "string", "description": "Project ID."},
			"since": {"type": "string", "description": "Since when."},
			"per_page": {"type": "integer", "minimum": 1, "maximum": 100, "description": "Page size."}
		}
	}`
	if got := asJSON(t, tool.Schema); got != asJSON(t, jsonValue(t, want)) {
		t.Errorf("schema =\n%s\nwant\n%s", got, asJSON(t, jsonValue(t, want)))
	}
}

func TestShallowAdvertisesTheTopLevelTypeAndKeepsTheFullSchemaForChecking(t *testing.T) {
	tool := buildOne(t, strings.Replace(createWidget, "parts: { description: Parts. }", "parts: { description: Parts., shallow: dashboards }", 1))

	parts := tool.Schema["properties"].(map[string]any)["parts"]
	if got, want := asJSON(t, parts), `{"description":"Parts. Shape: get_reference topic dashboards.","items":{"type":"object"},"type":"array"}`; got != want {
		t.Errorf("advertised parts = %s, want %s", got, want)
	}
	full := tool.StrictSchema["properties"].(map[string]any)["parts"].(map[string]any)
	if got := asJSON(t, full["items"]); !strings.Contains(got, `"label"`) {
		t.Errorf("strict parts items = %s, want the full Part schema", got)
	}
	if !reflect.DeepEqual(tool.Shallow, map[string]string{"parts": "dashboards"}) {
		t.Errorf("Shallow = %v", tool.Shallow)
	}
}

func TestAnnotationsComeFromScopes(t *testing.T) {
	for _, tc := range []struct {
		overlay  string
		readOnly bool
	}{
		{"list_widgets:\n  operation: listWidgets\n  params: {project_id: {description: x}, since: {exclude: x}, label: {exclude: x}, per_page: {exclude: x}}", true},
		{"search_widgets:\n  operation: searchWidgets\n  params: {q: {description: x}}", true}, // a POST that only reads
		{"delete_widget:\n  operation: deleteWidget\n  confirm: true\n  params: {project_id: {description: x}, widget_id: {description: x}}", false},
	} {
		tool := buildOne(t, tc.overlay+"\n  title: T\n  description: D.\n")
		if tool.ReadOnly != tc.readOnly {
			t.Errorf("%s: ReadOnly = %v, want %v", tool.Name, tool.ReadOnly, tc.readOnly)
		}
	}
}

func TestOperationsMergeRequiresOnlyWhatEveryOneRequires(t *testing.T) {
	tool := buildOne(t, `
get_counts:
  operations: [getProjectCounts, listAccountCounts]
  title: Get Counts
  description: Counts.
  params:
    project_id: { description: Project ID; omit for the whole account. }
    period: { description: Period. }
`)
	if got := asJSON(t, tool.Schema["required"]); got != "null" {
		t.Errorf("required = %s, want none: listAccountCounts has no project_id", got)
	}
}

func TestConfirmIsMarkedOnDeletes(t *testing.T) {
	tool := buildOne(t, `
delete_widget:
  operation: deleteWidget
  confirm: true
  title: Delete Widget
  description: Delete a widget.
  params: {project_id: {description: x}, widget_id: {description: x}}
`)
	if !tool.Confirm {
		t.Error("Confirm = false")
	}
	if _, ok := tool.Schema["properties"].(map[string]any)["confirm"]; ok {
		t.Error("confirm is in the schema; render adds it, so its text lives with the delete gate")
	}
}

func TestHandwrittenToolsAreSkipped(t *testing.T) {
	tools, err := Build(fixture(t), []byte("get_reference:\n  handwritten: true\n"))
	if err != nil || len(tools) != 0 {
		t.Fatalf("Build = %v, %v; want no tools", tools, err)
	}
}

func TestOverlayProblemsFailGeneration(t *testing.T) {
	long := strings.Repeat("x", 301)
	for name, tc := range map[string]struct {
		overlay string
		want    string
	}{
		"body field without a description": {
			strings.Replace(createWidget, "    color: { description: Color. }\n", "", 1),
			"create_widget.color: no description",
		},
		"exclude of a field not in the spec": {
			createWidget + "    colour: { exclude: Gone. }\n",
			"create_widget.colour: excluded but not in the spec",
		},
		"date of a field not in the spec": {
			createWidget + "    colour: { as: date, reason: x, description: x }\n",
			"create_widget.colour: converted but not in the spec",
		},
		"described field not in the spec": {
			createWidget + "    colour: { description: x }\n",
			"create_widget.colour: not in the spec",
		},
		"date on a string": {
			strings.Replace(createWidget, "name: { description: Widget name. }", "name: { description: Widget name., as: date, reason: x }", 1),
			"create_widget.name: as: date needs a Unix-seconds number; spec says string",
		},
		"override without a reason": {
			strings.Replace(createWidget, "name: { description: Widget name. }", "name: { description: Widget name., as: date }", 1),
			"create_widget.name: as needs a reason",
		},
		"confirm on a POST": {
			strings.Replace(createWidget, "  title: Create Widget", "  confirm: true\n  title: Create Widget", 1),
			"create_widget: confirm needs a DELETE operation",
		},
		"operations disagree": {
			"get_counts:\n  operations: [getProjectCounts, listCountsBySize]\n  title: T\n  description: D.\n  params: {project_id: {description: x}, period: {description: x}}\n",
			"get_counts.period: operations disagree on its schema",
		},
		"unknown operation": {
			"get_counts:\n  operation: getCountz\n  title: T\n  description: D.\n",
			"get_counts: operation getCountz isn't in the spec",
		},
		"tool description over the cap": {
			strings.Replace(createWidget, "description: Create a widget.", "description: "+long, 1),
			"create_widget: description is 301 characters; the cap is 300 (add long: <reason>)",
		},
		"parameter description over the cap": {
			strings.Replace(createWidget, "description: Widget name.", "description: "+long[:121], 1),
			"create_widget.name: description is 121 characters; the cap is 120 (add long: <reason>)",
		},
		"missing tool description": {
			strings.Replace(createWidget, "  description: Create a widget.\n", "", 1),
			"create_widget: no description",
		},
		"missing title": {
			strings.Replace(createWidget, "  title: Create Widget\n", "", 1),
			"create_widget: no title",
		},
		"misspelled overlay key": {
			strings.Replace(createWidget, "  title: Create Widget", "  titel: Create Widget", 1),
			"field titel not found",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Build(fixture(t), []byte(tc.overlay))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v\nwant it to contain %q", err, tc.want)
			}
		})
	}
}

func TestLongLetsADescriptionPassTheCap(t *testing.T) {
	long := strings.Repeat("x", 301)
	overlay := strings.Replace(createWidget, "description: Create a widget.", "description: "+long+"\n  long: Carried over from v2; trimmed later.", 1)
	if _, err := Build(fixture(t), []byte(overlay)); err != nil {
		t.Errorf("Build: %v", err)
	}
}

func TestProblemsAreAllReportedSorted(t *testing.T) {
	overlay := strings.Replace(createWidget, "    color: { description: Color. }\n", "", 1)
	overlay = strings.Replace(overlay, "    size: { description: Size. }\n", "", 1)
	_, err := Build(fixture(t), []byte(overlay))
	want := "create_widget.color: no description\ncreate_widget.size: no description"
	if err == nil || err.Error() != want {
		t.Errorf("err = %v\nwant\n%s", err, want)
	}
}

func TestGenerateWritesParseableGo(t *testing.T) {
	src, err := Generate(fixture(t), []byte(createWidget+"\ndelete_widget:\n  operation: deleteWidget\n  confirm: true\n  title: Delete Widget\n  description: Delete a widget.\n  params: {project_id: {description: x}, widget_id: {description: x}}\n"))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "zz_generated_tools.go", src, 0); err != nil {
		t.Fatalf("generated source doesn't parse: %v\n%s", err, src)
	}
	for _, want := range []string{
		"// Code generated by toolgen. DO NOT EDIT.",
		`"create_widget": {`,
		`"delete_widget": {`,
		"confirmParam,",
		`"Delete a widget." + confirmNote`,
	} {
		if !strings.Contains(string(src), want) {
			t.Errorf("generated source lacks %q", want)
		}
	}
}

// A body's top-level rules (which fields a kind requires) still hold once its
// properties are flattened into the tool's arguments. They are checked, not
// advertised: some clients refuse a tool whose top-level schema has allOf.
func TestBodyRulesAreKeptForChecking(t *testing.T) {
	tool := buildOne(t, createWidget)
	if _, ok := tool.Schema["allOf"]; ok {
		t.Errorf("advertised schema has allOf: %s", asJSON(t, tool.Schema["allOf"]))
	}
	want := `[{"if":{"properties":{"kind":{"const":"b"}}},"then":{"required":["size"]}}]`
	if tool.StrictSchema == nil {
		t.Fatal("StrictSchema = nil, want the body's rules")
	}
	if got := asJSON(t, tool.StrictSchema["allOf"]); got != want {
		t.Errorf("strict allOf = %s, want %s", got, want)
	}
}

// A body composed with allOf (a shared input plus what create adds) flattens
// into one set of arguments; its conditions stay rules.
func TestComposedBodyFlattens(t *testing.T) {
	tool := buildOne(t, `
update_widget:
  operation: updateWidget
  title: Update Widget
  description: Update a widget.
  params:
    project_id: { description: x }
    widget_id: { description: x }
    label: { description: x }
    weight: { description: x }
    note: { description: x }
    default: { description: x }
`)
	if got, want := asJSON(t, tool.Schema["required"]), `["label","project_id","weight","widget_id"]`; got != want {
		t.Errorf("required = %s, want %s", got, want)
	}
	if got, want := asJSON(t, tool.StrictSchema["allOf"]), `[{"if":{"properties":{"label":{"const":"heavy"}}},"then":{"properties":{"weight":{"minimum":10,"type":"number"}},"required":["label"]}}]`; got != want {
		t.Errorf("rules = %s, want %s", got, want)
	}
}

// A default says what the server uses when a field is left out. That holds
// for a create (POST) but not an update (PATCH or PUT), where leaving a field
// out keeps its value, so a filled-in default would reset it.
func TestDefaultsAreKeptOnlyWhereOmittingUsesThem(t *testing.T) {
	create := buildOne(t, createWidget)
	if got := create.Schema["properties"].(map[string]any)["size"].(map[string]any)["default"]; got != 3 {
		t.Errorf("create_widget.size default = %v, want 3", got)
	}
	update := buildOne(t, `
update_widget:
  operation: updateWidget
  title: Update Widget
  description: Update a widget.
  params:
    project_id: { description: x }
    widget_id: { description: x }
    label: { description: x }
    weight: { description: x }
    note: { description: x }
    default: { description: x }
`)
	if _, ok := update.Schema["properties"].(map[string]any)["default"]; !ok {
		t.Error("update_widget lost its property named default")
	}
	if got, ok := update.Schema["properties"].(map[string]any)["note"].(map[string]any)["default"]; ok {
		t.Errorf("update_widget.note default = %v, want none on a PATCH", got)
	}
}

// Each parameter records where the API takes it, so a test can check every
// advertised parameter reaches the request.
func TestToolsRecordWhereEachParameterGoes(t *testing.T) {
	tool := buildOne(t, `
list_widgets:
  operation: listWidgets
  title: List Widgets
  description: List widgets.
  params:
    project_id: { description: x }
    since: { description: x, as: date, reason: x }
    label: { exclude: x }
    per_page: { description: x }
`)
	want := map[string]string{"project_id": "path", "since": "query", "per_page": "query"}
	if !reflect.DeepEqual(tool.Wire, want) {
		t.Errorf("Wire = %v, want %v", tool.Wire, want)
	}
	if got := buildOne(t, createWidget).Wire["name"]; got != "body" {
		t.Errorf("create_widget name = %v, want body", got)
	}
}

// Spec shapes the generator can't flatten faithfully fail generation, naming
// the shape, rather than quietly producing a schema that drops a constraint or
// a field.
func TestUnsupportedSpecShapesFailGeneration(t *testing.T) {
	bundle, err := os.ReadFile("testdata/unsupported.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for op, want := range map[string]string{
		"getThing":          "get_thing.thing_id: in both path and query",
		"refSiblings":       "ref_siblings: operation refSiblings: minimum next to a $ref to Count would be lost",
		"allOfOverlap":      "all_of_overlap: operation allOfOverlap: minimum both beside and inside a single-entry allOf",
		"conditionalOnly":   "conditional_only: operation conditionalOnly: body property extra is defined only inside a condition",
		"refInAllOf":        "ref_in_all_of: operation refInAllOf: maximum next to a $ref to Count would be lost",
		"nestedConditional": "nested_conditional: operation nestedConditional: body property extra is defined only inside a condition",
		"duplicateProp":     "duplicate_prop: operation duplicateProp: body property count is defined twice, differently",
	} {
		name := toSnake(op)
		overlay := name + ":\n  operation: " + op + "\n  title: T\n  description: D.\n"
		_, err := Build(bundle, []byte(overlay))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v\nwant it to contain %q", op, err, want)
		}
	}
}

// A delete gated on confirm takes the confirm token in its strict schema too,
// or a tool with body rules would refuse the confirming call.
func TestConfirmIsInTheStrictSchema(t *testing.T) {
	bundle, err := os.ReadFile("testdata/unsupported.yaml")
	if err != nil {
		t.Fatal(err)
	}
	tools, err := Build(bundle, []byte("delete_things:\n  operation: deleteThings\n  confirm: true\n  title: T\n  description: D.\n  params:\n    thing_id: { description: x }\n    kind: { description: x }\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tools[0].StrictSchema["properties"].(map[string]any)["confirm"]; !ok {
		t.Errorf("strict schema lacks confirm: %s", asJSON(t, tools[0].StrictSchema))
	}
}

func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

// A problem in a shared parameter an operation replaces with its own is
// moot: the operation's parameter is the one the tool takes.
func TestAnOverriddenSharedParameterDoesntFailGeneration(t *testing.T) {
	bundle, err := os.ReadFile("testdata/unsupported.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Build(bundle, []byte("overrides_shared:\n  operation: overridesShared\n  title: T\n  description: D.\n  params:\n    shared_id: { description: x }\n")); err != nil {
		t.Errorf("Build: %v", err)
	}
}
