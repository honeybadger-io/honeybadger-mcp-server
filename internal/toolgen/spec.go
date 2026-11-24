package toolgen

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// spec is the parts of the bundle the generator reads.
type spec struct {
	root map[string]any
	ops  map[string]*operation
	// problems collects, while an operation is described, the shapes in it the
	// generator can't flatten faithfully.
	problems *[]string
}

// operation is one API operation's inputs, flattened: path, query and JSON body
// parameters by name, each resolved to plain JSON Schema.
type operation struct {
	id     string
	method string
	params map[string]param
	scopes []string
	// rules are the body's top-level conditions (allOf, if/then and the like),
	// which still apply once its properties are flattened into arguments.
	rules []any
	// problems are reported only if a tool uses the operation, as
	// "<tool>: operation <id>: <problem>" or, for one starting with "." (a
	// parameter), "<tool><problem>".
	problems []string
}

// Keys that describe a schema for humans or docs tooling. Tool descriptions come
// from the overlay, so these would only add tokens to tools/list.
var droppedKeys = map[string]bool{
	"description": true, "example": true, "examples": true, "title": true,
	"deprecated": true, "externalDocs": true, "discriminator": true,
	"$comment": true, "writeOnly": true,
}

func loadSpec(bundle []byte) (*spec, error) {
	var root map[string]any
	if err := yaml.Unmarshal(bundle, &root); err != nil {
		return nil, fmt.Errorf("bundle: %w", err)
	}
	d := &spec{root: root, ops: map[string]*operation{}}
	defaultScopes := scopesOf(root["security"])

	paths, _ := root["paths"].(map[string]any)
	for _, path := range sortedKeys(paths) {
		item, _ := paths[path].(map[string]any)
		shared, _ := item["parameters"].([]any)
		for _, method := range []string{"get", "post", "put", "patch", "delete"} {
			raw, ok := item[method].(map[string]any)
			if !ok {
				continue
			}
			id, _ := raw["operationId"].(string)
			if id == "" {
				continue
			}
			op, err := d.describe(strings.ToUpper(method), shared, raw, defaultScopes)
			if err != nil {
				return nil, fmt.Errorf("bundle: %s: %w", id, err)
			}
			op.id = id
			d.ops[id] = op
		}
	}
	return d, nil
}

func (d *spec) describe(method string, shared []any, raw map[string]any, defaultScopes []string) (*operation, error) {
	op := &operation{method: method, params: map[string]param{}, scopes: defaultScopes}
	d.problems = &op.problems
	defer func() { d.problems = nil }()
	if sec, ok := raw["security"]; ok {
		op.scopes = scopesOf(sec)
	}

	own, _ := raw["parameters"].([]any)
	if err := d.addParams(op, d.effectiveParams(shared, own)); err != nil {
		return nil, err
	}

	d.addBodyParams(op, raw["requestBody"])
	if method == "PATCH" || method == "PUT" {
		// A default says what the server uses when a field is left out. On an
		// update, leaving a field out keeps its value instead, and a default a
		// client or model filled in would reset it.
		for _, p := range op.params {
			dropDefaults(p.schema)
		}
		for _, r := range op.rules {
			dropDefaults(r)
		}
	}
	return op, nil
}

// dropDefaults removes every default keyword from a resolved schema. A
// property that happens to be named "default" is a property, not the keyword,
// so properties maps are walked by value.
func dropDefaults(node any) {
	switch n := node.(type) {
	case map[string]any:
		delete(n, "default")
		for k, v := range n {
			if k == "properties" || k == "patternProperties" {
				for _, prop := range asMap(v) {
					dropDefaults(prop)
				}
				continue
			}
			dropDefaults(v)
		}
	case []any:
		for _, v := range n {
			dropDefaults(v)
		}
	}
}

// effectiveParams are the parameters an operation takes: its own, and the
// shared ones it doesn't replace with one of the same name and location.
// Resolving only these keeps a replaced shared parameter's schema, and any
// problem in it, out of the tool.
func (d *spec) effectiveParams(shared, own []any) []any {
	key := func(raw any) string {
		p, _ := d.deref(raw).(map[string]any)
		return fmt.Sprint(p["name"], "\x00", p["in"])
	}
	replaced := map[string]bool{}
	for _, p := range own {
		replaced[key(p)] = true
	}
	var out []any
	for _, p := range shared {
		if !replaced[key(p)] {
			out = append(out, p)
		}
	}
	return append(out, own...)
}

// addParams adds an operation's path and query parameters.
func (d *spec) addParams(op *operation, list []any) error {
	for _, rawParam := range list {
		p, ok := d.deref(rawParam).(map[string]any)
		if !ok {
			continue
		}
		name, _ := p["name"].(string)
		in, _ := p["in"].(string)
		if in != "path" && in != "query" {
			continue
		}
		schema, ok := d.resolve(p["schema"]).(map[string]any)
		if !ok {
			return fmt.Errorf("parameter %s has no schema", name)
		}
		required := 0
		if in == "path" || p["required"] == true {
			required = 1
		}
		if have, ok := op.params[name]; ok && have.in != in {
			d.fail(".%s: in both %s and %s", name, have.in, in)
		}
		op.params[name] = param{schema: schema, required: required, in: in}
	}
	return nil
}

// addBodyParams adds a JSON request body's properties as parameters.
func (d *spec) addBodyParams(op *operation, requestBody any) {
	body, ok := d.deref(requestBody).(map[string]any)
	if !ok {
		return
	}
	content, _ := body["content"].(map[string]any)
	media, _ := content["application/json"].(map[string]any)
	schema, ok := d.resolve(media["schema"]).(map[string]any)
	if !ok {
		return
	}
	var b flatBody
	b.add(schema)
	op.rules = b.rules
	for _, name := range b.conditionalOnly() {
		d.fail("body property %s is defined only inside a condition", name)
	}
	for _, name := range b.twice {
		d.fail("body property %s is defined twice, differently", name)
	}
	for name, s := range b.props {
		if have, ok := op.params[name]; ok {
			d.fail(".%s: in both %s and body", name, have.in)
		}
		p := param{schema: s.(map[string]any), in: "body"}
		if b.required[name] {
			p.required = 1
		}
		op.params[name] = p
	}
}

// scopesOf lists the scopes in a security requirement list.
func scopesOf(sec any) []string {
	var scopes []string
	for _, req := range asSlice(sec) {
		m, _ := req.(map[string]any)
		for _, list := range m {
			for _, s := range asSlice(list) {
				scopes = append(scopes, fmt.Sprint(s))
			}
		}
	}
	return scopes
}

// deref follows a $ref to its target, without resolving the target's insides.
func (d *spec) deref(node any) any {
	for {
		m, ok := node.(map[string]any)
		if !ok {
			return node
		}
		ref, ok := m["$ref"].(string)
		if !ok {
			return node
		}
		// Every hop, so a constraint beside a $ref deeper in a chain is
		// caught too.
		d.checkRefSiblings(m)
		node = d.lookup(ref)
	}
}

// readOnly reports whether a schema, or any schema its $ref chain passes
// through, is marked readOnly: set by the server, never sent.
func (d *spec) readOnly(node any) bool {
	for {
		m, ok := node.(map[string]any)
		if !ok {
			return false
		}
		if m["readOnly"] == true {
			return true
		}
		ref, ok := m["$ref"].(string)
		if !ok {
			return false
		}
		node = d.lookup(ref)
	}
}

func (d *spec) lookup(ref string) any {
	var node any = d.root
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		m, _ := node.(map[string]any)
		node = m[part]
	}
	return node
}

// resolve copies a schema into plain JSON Schema for a tool: references
// inlined, a single-entry allOf unwrapped, "X or null" folded into X's type,
// docs-only keys dropped, read-only properties removed, and objects closed to
// unknown properties unless the spec gives a schema for them.
func (d *spec) resolve(node any) any {
	switch n := d.deref(node).(type) {
	case map[string]any:
		return d.resolveMap(n)
	case []any:
		out := make([]any, len(n))
		for i, v := range n {
			out[i] = d.resolve(v)
		}
		return out
	default:
		return n
	}
}

func (d *spec) resolveMap(n map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range n {
		if droppedKeys[k] || strings.HasPrefix(k, "x-") || k == "$ref" {
			continue
		}
		out[k] = v
	}
	d.simplify(out)
	serverSet := d.resolveChildren(out)
	delete(out, "readOnly")
	dropFromRequired(out, serverSet)

	// Close only a schema that is itself an object type: a subschema under if,
	// then or allOf only adds rules, and closing it would refuse the
	// properties its parent declares.
	if _, ok := out["properties"]; ok {
		if _, set := out["additionalProperties"]; !set && isObjectType(out["type"]) {
			out["additionalProperties"] = false
		}
	}
	return out
}

// simplify unwraps a single-entry allOf and folds "X or null" into X, so
// tools/list shows a plain type wherever the spec allows it.
func (d *spec) simplify(out map[string]any) {
	if all, ok := out["allOf"].([]any); ok && len(all) == 1 {
		delete(out, "allOf")
		inner := d.resolveMap(d.deref(all[0]).(map[string]any))
		if both := sharedKeywords(out, inner); len(both) > 0 {
			d.fail("%s both beside and inside a single-entry allOf", strings.Join(both, ", "))
		}
		mergeMissing(out, inner)
	}
	for _, key := range []string{"anyOf", "oneOf"} {
		alts, ok := out[key].([]any)
		if !ok {
			continue
		}
		if folded, ok := d.foldNullable(alts); ok {
			delete(out, key)
			mergeMissing(out, folded)
		}
	}
}

func mergeMissing(dst, src map[string]any) {
	for k, v := range src {
		if _, own := dst[k]; !own {
			dst[k] = v
		}
	}
}

// resolveChildren resolves the subschemas in out, leaving out read-only
// properties, and returns their names.
func (d *spec) resolveChildren(out map[string]any) map[string]bool {
	serverSet := map[string]bool{}
	for k, v := range out {
		switch k {
		case "properties":
			props := map[string]any{}
			for name, raw := range v.(map[string]any) {
				if d.readOnly(raw) {
					serverSet[name] = true // set by the server, never sent
					continue
				}
				props[name] = d.resolve(raw)
			}
			out[k] = props
		case "items", "additionalProperties", "allOf", "anyOf", "oneOf", "not", "if", "then", "else":
			out[k] = d.resolve(v)
		case "patternProperties":
			schemas := map[string]any{}
			for pattern, raw := range v.(map[string]any) {
				schemas[pattern] = d.resolve(raw)
			}
			out[k] = schemas
		}
	}
	return serverSet
}

// dropFromRequired removes the read-only names from required, and only those:
// a subschema under then may require a property its parent declares.
func dropFromRequired(out map[string]any, names map[string]bool) {
	if len(names) == 0 {
		return
	}
	var required []any
	for _, r := range asSlice(out["required"]) {
		if !names[fmt.Sprint(r)] {
			required = append(required, r)
		}
	}
	if len(required) > 0 {
		out["required"] = required
	} else {
		delete(out, "required")
	}
}

// foldNullable turns [X, {type: null}] into X accepting null.
func (d *spec) foldNullable(alts []any) (map[string]any, bool) {
	if len(alts) != 2 {
		return nil, false
	}
	var other map[string]any
	nulls := 0
	for _, a := range alts {
		m, _ := d.deref(a).(map[string]any)
		if m["type"] == "null" && len(m) == 1 {
			nulls++
		} else {
			other = m
		}
	}
	if nulls != 1 || other == nil {
		return nil, false
	}
	x := d.resolveMap(other)
	switch t := x["type"].(type) {
	case string:
		x["type"] = []any{t, "null"}
	case []any:
		x["type"] = append(append([]any{}, t...), "null")
	default:
		return nil, false
	}
	if enum, ok := x["enum"].([]any); ok {
		x["enum"] = append(append([]any{}, enum...), nil)
	}
	return x, true
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func isObjectType(t any) bool {
	switch x := t.(type) {
	case string:
		return x == "object"
	case []any:
		for _, v := range x {
			if v == "object" {
				return true
			}
		}
	}
	return false
}

// ruleKeys are the keywords that constrain an object as a whole rather than
// one property.
var ruleKeys = []string{"if", "then", "else", "oneOf", "anyOf", "not", "dependentRequired"}

// flatBody is a request body flattened into arguments: allOf parts (a shared
// input plus what create adds) merge their properties and required names, and
// conditions (if/then and the like) become rules that still apply.
type flatBody struct {
	props    map[string]any
	required map[string]bool
	rules    []any
	// twice are properties two allOf parts define differently; one would
	// replace the other.
	twice []string
}

func (b *flatBody) add(schema map[string]any) {
	if b.props == nil {
		b.props, b.required = map[string]any{}, map[string]bool{}
	}
	for name, p := range asMap(schema["properties"]) {
		if have, ok := b.props[name]; ok && !reflect.DeepEqual(have, p) {
			b.twice = append(b.twice, name)
		}
		b.props[name] = p
	}
	for _, r := range asSlice(schema["required"]) {
		b.required[fmt.Sprint(r)] = true
	}
	rule := map[string]any{}
	for _, k := range ruleKeys {
		if v, ok := schema[k]; ok {
			rule[k] = v
		}
	}
	if len(rule) > 0 {
		b.rules = append(b.rules, rule)
	}
	for _, part := range asSlice(schema["allOf"]) {
		b.add(part.(map[string]any))
	}
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// fail records a shape the generator can't flatten faithfully in the
// operation being described.
func (d *spec) fail(format string, args ...any) {
	if d.problems == nil {
		return
	}
	// The same node can be dereferenced from several places; one report is
	// enough.
	if p := fmt.Sprintf(format, args...); !slices.Contains(*d.problems, p) {
		*d.problems = append(*d.problems, p)
	}
}

// checkRefSiblings fails on constraints written next to a $ref: resolving the
// reference replaces the node, and they would be lost. readOnly is honoured
// (see readOnly), and docs keys are dropped anyway.
func (d *spec) checkRefSiblings(node any) {
	m, ok := node.(map[string]any)
	if !ok {
		return
	}
	ref, ok := m["$ref"].(string)
	if !ok {
		return
	}
	var lost []string
	for k := range m {
		if k != "$ref" && k != "readOnly" && !droppedKeys[k] && !strings.HasPrefix(k, "x-") {
			lost = append(lost, k)
		}
	}
	if len(lost) > 0 {
		sort.Strings(lost)
		d.fail("%s next to a $ref to %s would be lost", strings.Join(lost, ", "), ref[strings.LastIndex(ref, "/")+1:])
	}
}

// sharedKeywords are the keywords a schema and its single allOf entry both
// set, which merging would collapse to one.
func sharedKeywords(outer, inner map[string]any) []string {
	var both []string
	for k := range inner {
		if _, ok := outer[k]; ok && k != "readOnly" {
			both = append(both, k)
		}
	}
	sort.Strings(both)
	return both
}

// conditionalOnly lists properties a body defines only inside a condition
// (then, else, oneOf, anyOf): flattening exposes only its plain properties.
func (b *flatBody) conditionalOnly() []string {
	seen := map[string]bool{}
	var only []string
	note := func(props map[string]any) {
		for name := range props {
			if _, plain := b.props[name]; !plain && !seen[name] {
				seen[name] = true
				only = append(only, name)
			}
		}
	}
	// branch notes a conditional schema's properties, and those of the
	// schemas it composes, however deep.
	var branch func(schema map[string]any)
	branch = func(schema map[string]any) {
		note(asMap(schema["properties"]))
		for _, k := range []string{"then", "else"} {
			if sub := asMap(schema[k]); sub != nil {
				branch(sub)
			}
		}
		for _, k := range []string{"allOf", "oneOf", "anyOf"} {
			for _, sub := range asSlice(schema[k]) {
				branch(asMap(sub))
			}
		}
	}
	for _, rule := range b.rules {
		r := asMap(rule)
		for _, k := range []string{"then", "else"} {
			if sub := asMap(r[k]); sub != nil {
				branch(sub)
			}
		}
		for _, k := range []string{"oneOf", "anyOf"} {
			for _, alt := range asSlice(r[k]) {
				branch(asMap(alt))
			}
		}
	}
	sort.Strings(only)
	return only
}
