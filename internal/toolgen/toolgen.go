// Package toolgen generates the MCP tool definitions from the vendored v3
// OpenAPI bundle and the hand-written overlay (openapi/tools.yaml).
//
// The spec owns each tool's shape: parameter names, types, enums, nullability,
// required and nesting. The overlay owns the words (titles and descriptions)
// and the few deliberate differences, each of which must state its reason.
// Anything the overlay says that no longer matches the spec fails generation,
// so a spec update ends in either a reviewable diff or a list of what needs a
// human.
package toolgen

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Description caps keep tools/list small: it is paid for in every agent
// session's context. Detail belongs in get_reference topics.
const (
	toolDescriptionCap  = 300
	paramDescriptionCap = 120
)

// Tool is one generated tool definition.
type Tool struct {
	Name        string
	Title       string
	Description string
	ReadOnly    bool
	// Confirm marks a delete gated on a confirmation token. The confirm
	// parameter and its note are added when rendering, so their text stays
	// with the gate in the hbmcp package.
	Confirm bool
	// Schema is the input schema tools/list advertises.
	Schema map[string]any
	// StrictSchema is the schema calls are checked against when it differs
	// from Schema, which happens only when a parameter is shallow.
	StrictSchema map[string]any
	// Shallow maps a parameter to the get_reference topic describing its shape.
	Shallow map[string]string
	// Wire says where the API takes each parameter, by MCP name.
	Wire map[string]Wire
}

// Wire is where an operation takes a parameter: In is path, query or body,
// and Name is the spec's name, which a rename makes differ from the tool's.
type Wire struct {
	In   string
	Name string
}

type toolSpec struct {
	Operation   string               `yaml:"operation"`
	Operations  []string             `yaml:"operations"`
	Handwritten bool                 `yaml:"handwritten"`
	Title       string               `yaml:"title"`
	Description string               `yaml:"description"`
	Long        string               `yaml:"long"`
	Confirm     bool                 `yaml:"confirm"`
	Params      map[string]paramSpec `yaml:"params"`
}

type paramSpec struct {
	Description string `yaml:"description"`
	Long        string `yaml:"long"`
	Exclude     string `yaml:"exclude"`
	Rename      string `yaml:"rename"`
	As          string `yaml:"as"`
	Reason      string `yaml:"reason"`
	Shallow     string `yaml:"shallow"`
}

// Build turns the bundle and overlay into tool definitions, sorted by name.
// Every problem is reported at once, one per line, sorted.
func Build(bundle, overlay []byte) ([]Tool, error) {
	tools, _, err := build(bundle, overlay)
	return tools, err
}

// build also returns the hand-written tools' names, sorted.
func build(bundle, overlay []byte) ([]Tool, []string, error) {
	var specs map[string]toolSpec
	dec := yaml.NewDecoder(bytes.NewReader(overlay))
	dec.KnownFields(true)
	if err := dec.Decode(&specs); err != nil && err != io.EOF { // an empty overlay is EOF
		return nil, nil, fmt.Errorf("overlay: %w", err)
	}
	doc, err := loadSpec(bundle)
	if err != nil {
		return nil, nil, err
	}

	var tools []Tool
	var handwritten, problems []string
	for _, name := range sortedKeys(specs) {
		spec := specs[name]
		if spec.Handwritten {
			handwritten = append(handwritten, name)
			continue
		}
		tool, errs := doc.buildTool(name, spec)
		problems = append(problems, errs...)
		if len(errs) == 0 {
			tools = append(tools, tool)
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return nil, nil, errors.New(strings.Join(problems, "\n"))
	}
	return tools, handwritten, nil
}

// param is one tool input gathered from an operation.
type param struct {
	schema   map[string]any
	required int    // how many of the tool's operations require it
	in       string // path, query or body
}

// toolBuilder builds one tool, collecting every problem rather than stopping
// at the first.
type toolBuilder struct {
	name     string
	spec     toolSpec
	problems []string
}

func (b *toolBuilder) fail(format string, args ...any) {
	b.problems = append(b.problems, fmt.Sprintf(format, args...))
}

func (d *spec) buildTool(name string, spec toolSpec) (Tool, []string) {
	b := &toolBuilder{name: name, spec: spec}
	ops := b.operations(d)
	if len(b.problems) > 0 {
		return Tool{}, b.problems
	}
	params := b.mergeParams(ops)
	b.checkTool(ops)
	b.checkOverridesMatchSpec(params)

	advertised := map[string]any{}
	strict := map[string]any{}
	shallow := map[string]string{}
	wire := map[string]Wire{}
	var required []any
	for _, pname := range sortedKeys(params) {
		ps := spec.Params[pname]
		if ps.Exclude != "" {
			continue
		}
		shown, full, ok := b.paramSchemas(pname, params[pname].schema, ps)
		if !ok {
			continue
		}
		mcpName := pname
		if ps.Rename != "" {
			mcpName = ps.Rename
		}
		advertised[mcpName] = shown
		wire[mcpName] = Wire{In: params[pname].in, Name: pname}
		strict[mcpName] = full
		if ps.Shallow != "" {
			shallow[pname] = ps.Shallow
		}
		if params[pname].required == len(ops) {
			required = append(required, mcpName)
		}
	}
	sort.Slice(required, func(i, j int) bool { return required[i].(string) < required[j].(string) })

	tool := Tool{
		Name:        name,
		Title:       spec.Title,
		Description: spec.Description,
		ReadOnly:    readOnly(ops),
		Confirm:     spec.Confirm,
		Schema:      objectSchema(advertised, required),
		Wire:        wire,
	}
	var rules []any
	for _, op := range ops {
		rules = append(rules, op.rules...)
	}
	// Shallow parameters and body rules are checked but not advertised: rules
	// at the top of a tool's schema make some clients refuse the tool.
	if len(shallow) > 0 || len(rules) > 0 {
		tool.StrictSchema = objectSchema(strict, required)
		if len(rules) > 0 {
			tool.StrictSchema["allOf"] = rules
		}
	}
	if len(shallow) > 0 {
		tool.Shallow = shallow
	}
	return tool, b.problems
}

// operations looks up the operations the tool names.
func (b *toolBuilder) operations(d *spec) []*operation {
	ids := b.spec.Operations
	if b.spec.Operation != "" {
		ids = append([]string{b.spec.Operation}, ids...)
	}
	if len(ids) == 0 {
		b.fail("%s: no operation", b.name)
	}
	var ops []*operation
	for _, id := range ids {
		op, ok := d.ops[id]
		if !ok {
			b.fail("%s: operation %s isn't in the spec", b.name, id)
			continue
		}
		ops = append(ops, op)
	}
	return ops
}

// mergeParams combines the operations' parameters. A parameter is required
// only when every operation requires it.
func (b *toolBuilder) mergeParams(ops []*operation) map[string]*param {
	params := map[string]*param{}
	for _, op := range ops {
		for pname, p := range op.params {
			have, ok := params[pname]
			if !ok {
				params[pname] = &param{schema: p.schema, required: p.required, in: p.in}
				continue
			}
			if !reflect.DeepEqual(have.schema, p.schema) {
				b.fail("%s.%s: operations disagree on its schema", b.name, pname)
			}
			have.required += p.required
		}
	}
	return params
}

func (b *toolBuilder) checkTool(ops []*operation) {
	if b.spec.Title == "" {
		b.fail("%s: no title", b.name)
	}
	b.checkDescription(b.name, b.spec.Description, b.spec.Long, toolDescriptionCap)
	if !b.spec.Confirm {
		return
	}
	for _, op := range ops {
		if op.method != "DELETE" {
			b.fail("%s: confirm needs a DELETE operation", b.name)
			return
		}
	}
}

func (b *toolBuilder) checkDescription(where, description, long string, limit int) {
	if description == "" {
		b.fail("%s: no description", where)
	} else if n := len(description); n > limit && long == "" {
		b.fail("%s: description is %d characters; the cap is %d (add long: <reason>)", where, n, limit)
	}
}

// checkOverridesMatchSpec fails on overlay entries for parameters the spec
// doesn't have, so a removed or renamed field can't leave a stale override.
func (b *toolBuilder) checkOverridesMatchSpec(params map[string]*param) {
	for _, pname := range sortedKeys(b.spec.Params) {
		if _, ok := params[pname]; ok {
			continue
		}
		ps := b.spec.Params[pname]
		switch {
		case ps.Exclude != "":
			b.fail("%s.%s: excluded but not in the spec", b.name, pname)
		case ps.Rename != "":
			b.fail("%s.%s: renamed but not in the spec", b.name, pname)
		case ps.As != "":
			b.fail("%s.%s: converted but not in the spec", b.name, pname)
		default:
			b.fail("%s.%s: not in the spec", b.name, pname)
		}
	}
}

// paramSchemas returns the schema tools/list shows for a parameter and the one
// calls are checked against; they differ only for a shallow parameter.
func (b *toolBuilder) paramSchemas(pname string, schema map[string]any, ps paramSpec) (shown, full map[string]any, ok bool) {
	where := b.name + "." + pname
	b.checkDescription(where, ps.Description, ps.Long, paramDescriptionCap)
	if ps.Description == "" {
		return nil, nil, false
	}
	if ps.Rename != "" && ps.Reason == "" {
		b.fail("%s: rename needs a reason", where)
	}
	if ps.As != "" && ps.Reason == "" {
		b.fail("%s: as needs a reason", where)
	}

	full = withDescription(schema, ps.Description)
	switch ps.As {
	case "":
	case "date":
		if t := nonNullType(schema); t != "number" {
			b.fail("%s: as: date needs a Unix-seconds number; spec says %s", where, t)
		}
		full = map[string]any{"type": "string", "description": ps.Description}
	default:
		b.fail("%s: as: %s isn't a known conversion (date)", where, ps.As)
	}
	shown = full
	if ps.Shallow != "" {
		shown = shallowSchema(schema, ps.Description+" Shape: get_reference topic "+ps.Shallow+".")
	}
	return shown, full, true
}

func objectSchema(props map[string]any, required []any) map[string]any {
	s := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func withDescription(schema map[string]any, description string) map[string]any {
	out := make(map[string]any, len(schema)+1)
	for k, v := range schema {
		out[k] = v
	}
	out["description"] = description
	return out
}

// shallowSchema keeps only the top-level type; for an array of objects, the
// items say only that they are objects.
func shallowSchema(schema map[string]any, description string) map[string]any {
	out := map[string]any{"type": schema["type"], "description": description}
	if items, ok := schema["items"].(map[string]any); ok {
		out["items"] = map[string]any{"type": items["type"]}
	}
	return out
}

// nonNullType names a schema's type, leaving out null.
func nonNullType(schema map[string]any) string {
	switch t := schema["type"].(type) {
	case string:
		return t
	case []any:
		for _, v := range t {
			if v != "null" {
				return fmt.Sprint(v)
			}
		}
	}
	return "untyped"
}

// readOnly holds when every operation needs only :read scopes. The HTTP method
// isn't enough: some reads are POSTs (an Insights query).
func readOnly(ops []*operation) bool {
	for _, op := range ops {
		if len(op.scopes) == 0 {
			return false
		}
		for _, s := range op.scopes {
			if !strings.HasSuffix(s, ":read") {
				return false
			}
		}
	}
	return true
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
