package toolgen

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// spec is the parts of the bundle the generator reads.
type spec struct {
	root map[string]any
	ops  map[string]*operation
}

// operation is one API operation's inputs, flattened: path, query and JSON body
// parameters by name, each resolved to plain JSON Schema.
type operation struct {
	method string
	params map[string]param
	scopes []string
	// rules are the body's top-level conditions (allOf, if/then and the like),
	// which still apply once its properties are flattened into arguments.
	rules []any
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
			d.ops[id] = op
		}
	}
	return d, nil
}

func (d *spec) describe(method string, shared []any, raw map[string]any, defaultScopes []string) (*operation, error) {
	op := &operation{method: method, params: map[string]param{}, scopes: defaultScopes}
	if sec, ok := raw["security"]; ok {
		op.scopes = scopesOf(sec)
	}

	own, _ := raw["parameters"].([]any)
	for _, list := range [][]any{shared, own} { // an operation's own parameter replaces a shared one
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
				return nil, fmt.Errorf("parameter %s has no schema", name)
			}
			required := 0
			if in == "path" || p["required"] == true {
				required = 1
			}
			op.params[name] = param{schema: schema, required: required}
		}
	}

	d.addBodyParams(op, raw["requestBody"])
	return op, nil
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
	for name, s := range b.props {
		p := param{schema: s.(map[string]any)}
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
		mergeMissing(out, d.resolveMap(d.deref(all[0]).(map[string]any)))
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
				if m, _ := d.deref(raw).(map[string]any); m["readOnly"] == true {
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
}

func (b *flatBody) add(schema map[string]any) {
	if b.props == nil {
		b.props, b.required = map[string]any{}, map[string]bool{}
	}
	for name, p := range asMap(schema["properties"]) {
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
