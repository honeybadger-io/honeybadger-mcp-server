package hbmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// The strict check refuses a call whose arguments don't match the tool's
// definition before the handler runs: an unknown, misspelled or stale key, a
// value of the wrong type, null where it isn't allowed. Handlers read only the
// keys they know, so without it such a call would drop the argument and report
// success. Each refusal names the field and the fix, since the caller is
// usually an agent that can correct the call.
//
// mcp-go's WithInputSchemaValidation would check the same schemas, but it runs
// before tool middleware and its generic message can't be replaced.

const (
	notMade     = "The call was not made."
	maybeStale  = "your tool list may be stale; reconnect to refresh tool schemas."
	staleTools  = "Your tool list is stale; reconnect to refresh tool schemas."
	maxGotQuote = 40
)

// strictArgs wraps a tool's handler with the check. The schema is compiled
// once, here: the generated strict schema when the tool has one (it keeps the
// shape of shallow parameters and the body's rules), else the advertised one.
func strictArgs(tool mcp.Tool, h server.ToolHandlerFunc, logger *slog.Logger) server.ToolHandlerFunc {
	c := newChecker(tool)
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		problems := c.check(req.Params.Arguments)
		if len(problems) == 0 {
			return h(ctx, req)
		}
		client, version := clientOf(ctx)
		for _, p := range problems {
			logger.Warn("tool arguments refused", "tool", c.tool, "field", p.field, "kind", p.kind,
				"client", client, "client_version", version)
		}
		return mcp.NewToolResultError(c.message(problems)), nil
	}
}

func clientOf(ctx context.Context) (name, version string) {
	if s, ok := server.ClientSessionFromContext(ctx).(server.SessionWithClientInfo); ok {
		info := s.GetClientInfo()
		return info.Name, info.Version
	}
	return "", ""
}

type checker struct {
	tool     string
	schema   *jsonschema.Schema
	props    map[string]map[string]any // top-level parameter schemas
	required []string
}

func newChecker(tool mcp.Tool) *checker {
	var doc any
	if strict, ok := generatedStrictSchemas[tool.Name]; ok {
		doc = strict
	} else {
		s := tool.InputSchema
		if s.AdditionalProperties == nil {
			s.AdditionalProperties = false
		}
		doc = s
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		panic(fmt.Sprintf("hbmcp: %s: encoding its input schema: %v", tool.Name, err))
	}
	parsed, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		panic(fmt.Sprintf("hbmcp: %s: reading its input schema: %v", tool.Name, err))
	}
	compiler := jsonschema.NewCompiler()
	url := "tool:///" + tool.Name + ".json"
	if err := compiler.AddResource(url, parsed); err != nil {
		panic(fmt.Sprintf("hbmcp: %s: %v", tool.Name, err))
	}
	schema, err := compiler.Compile(url)
	if err != nil {
		panic(fmt.Sprintf("hbmcp: %s: compiling its input schema: %v", tool.Name, err))
	}
	var shape struct {
		Properties map[string]map[string]any `json:"properties"`
		Required   []string                  `json:"required"`
	}
	_ = json.Unmarshal(raw, &shape)
	return &checker{tool: tool.Name, schema: schema, props: shape.Properties, required: shape.Required}
}

// problem is one thing wrong with a call's arguments.
type problem struct {
	field string // the top-level parameter, for the log
	path  string // where in the arguments, e.g. widgets[0]
	kind  string // unknown, stale, type, null, id_number, required, nested, other
	short string // as a line in a list of problems
	full  string // as the whole message, when it's the only problem
	// target is the parameter an unknown key points to: its rename or the
	// closest declared name.
	target string
	// topic is the reference topic for a shallow parameter's shape.
	topic string
}

func (c *checker) check(arguments any) []problem {
	var args any = map[string]any{}
	switch a := arguments.(type) {
	case nil:
	case map[string]any:
		args = a
	default:
		return []problem{{kind: "other", short: "arguments must be a JSON object.",
			full: c.tool + ": arguments must be a JSON object. " + notMade}}
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return []problem{{kind: "other", short: "arguments could not be read.",
			full: fmt.Sprintf("%s: arguments could not be read: %v. %s", c.tool, err, notMade)}}
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return []problem{{kind: "other", short: "arguments could not be read.",
			full: fmt.Sprintf("%s: arguments could not be read: %v. %s", c.tool, err, notMade)}}
	}
	verr, ok := c.schema.Validate(instance).(*jsonschema.ValidationError)
	if !ok {
		return nil
	}
	// Values as JSON gives them (numbers are float64), to describe what was sent.
	var plain map[string]any
	_ = json.Unmarshal(raw, &plain)

	var leaves []*jsonschema.ValidationError
	collectLeaves(verr, &leaves)
	var problems []problem
	seen := map[string]bool{} // one problem per location, unknown keys apart
	for _, e := range leaves {
		for _, p := range c.describe(e, plain) {
			key := p.path + "\x00" + p.kind + "\x00" + p.short
			if p.kind == "type" || p.kind == "null" || p.kind == "id_number" {
				key = p.path
			}
			if !seen[key] {
				seen[key] = true
				problems = append(problems, p)
			}
		}
	}
	problems = dropPointedAt(problems)
	sort.SliceStable(problems, func(i, j int) bool { return problems[i].path < problems[j].path })
	return problems
}

// dropPointedAt drops "X is required" when another problem already points the
// caller at X: a renamed parameter, or a misspelling of it.
func dropPointedAt(problems []problem) []problem {
	pointed := map[string]bool{}
	for _, p := range problems {
		if p.target != "" {
			pointed[p.target] = true
		}
	}
	kept := problems[:0]
	for _, p := range problems {
		if p.kind == "required" && pointed[p.field] {
			continue
		}
		kept = append(kept, p)
	}
	return kept
}

func collectLeaves(e *jsonschema.ValidationError, out *[]*jsonschema.ValidationError) {
	if len(e.Causes) == 0 {
		*out = append(*out, e)
		return
	}
	for _, c := range e.Causes {
		collectLeaves(c, out)
	}
}

func (c *checker) describe(e *jsonschema.ValidationError, args map[string]any) []problem {
	loc := e.InstanceLocation
	path := pathString(loc)
	top := len(loc) == 0
	field := ""
	if len(loc) > 0 {
		field = loc[0]
	}
	topic := ""
	if field != "" {
		topic = generatedShallowTopics[c.tool][field]
	}

	switch k := e.ErrorKind.(type) {
	case *kind.AdditionalProperties:
		var out []problem
		for _, name := range k.Properties {
			if top {
				out = append(out, c.unknownTopLevel(name))
				continue
			}
			out = append(out, c.nested(field, path, fmt.Sprintf("has no field %q.", name), topic))
		}
		return out
	case *kind.Required:
		var out []problem
		for _, name := range k.Missing {
			if top {
				out = append(out, problem{field: name, path: name, kind: "required", short: "required.",
					full: fmt.Sprintf("%s: %s is required. %s", c.tool, name, notMade)})
				continue
			}
			out = append(out, c.nested(field, path, fmt.Sprintf("needs %q.", name), topic))
		}
		return out
	case *kind.Type, *kind.Enum:
		if len(loc) == 1 {
			return []problem{c.wrongValue(field, args[field])}
		}
	}
	return []problem{c.nested(field, path, e.ErrorKind.LocalizedString(printer)+".", topic)}
}

var printer = message.NewPrinter(language.English)

func (c *checker) unknownTopLevel(name string) problem {
	if reason, ok := staleSchemaFields[c.tool][name]; ok {
		return problem{field: name, path: name, kind: "stale", target: renameTarget(reason),
			short: "no longer accepted: " + reason + ".",
			full:  fmt.Sprintf("%s no longer accepts %q: %s. %s %s", c.tool, name, reason, notMade, staleTools)}
	}
	if near := c.closest(name); near != "" {
		return problem{field: name, path: name, kind: "unknown", target: near,
			short: fmt.Sprintf("no such parameter. Did you mean %q?", near),
			full: fmt.Sprintf("%s has no parameter %q. Did you mean %q? %s If you expected %q to exist, %s",
				c.tool, name, near, notMade, name, maybeStale)}
	}
	names := make([]string, 0, len(c.props))
	for n := range c.props {
		names = append(names, n)
	}
	sort.Strings(names)
	return problem{field: name, path: name, kind: "unknown",
		short: "no such parameter.",
		full: fmt.Sprintf("%s has no parameter %q. Its parameters are: %s. %s If you expected %q to exist, %s",
			c.tool, name, strings.Join(names, ", "), notMade, name, maybeStale)}
}

// closest is the declared parameter within two edits of name, if any.
func (c *checker) closest(name string) string {
	best, bestDist := "", 3
	for n := range c.props {
		if d := editDistance(name, n); d < bestDist || (d == bestDist && n < best) {
			best, bestDist = n, d
		}
	}
	return best
}

func (c *checker) nested(field, path, what, topic string) problem {
	full := fmt.Sprintf("%s: %s %s %s", c.tool, path, what, notMade)
	if topic != "" {
		full += fmt.Sprintf(" See get_reference topic %q.", topic)
	}
	return problem{field: field, path: path, kind: "nested", short: what, full: full, topic: topic}
}

// wrongValue describes a top-level parameter given a value its schema doesn't
// take, in terms of what the schema does take.
func (c *checker) wrongValue(field string, got any) problem {
	schema := c.props[field]
	types := schemaTypes(schema)
	if got == nil {
		short := "can't be null; omit it instead."
		switch {
		case slices.Contains(c.required, field):
			short = "can't be null; it is required."
		case strings.HasPrefix(c.tool, "update_"):
			short = "can't be null; omit it to leave the setting unchanged."
		}
		return problem{field: field, path: field, kind: "null", short: short,
			full: fmt.Sprintf("%s: %s %s %s", c.tool, field, short, notMade)}
	}
	if n, ok := got.(float64); ok && isIDParam(field) && slices.Contains(types, "string") {
		list := "call the tool that lists them"
		if t := listToolFor(field); t != "" {
			list = "call " + t
		}
		short := fmt.Sprintf("must be a string; got the number %s. IDs are strings now: %s for the current one.", formatNumber(n), list)
		return problem{field: field, path: field, kind: "id_number", short: short,
			full: fmt.Sprintf("%s: %s %s If your tool definitions show %s as a number, they are stale; reconnect to refresh tool schemas.",
				c.tool, field, short, field)}
	}
	short := fmt.Sprintf("must be %s; got %s.", expected(schema, types), describeValue(got))
	return problem{field: field, path: field, kind: "type", short: short,
		full: fmt.Sprintf("%s: %s %s %s", c.tool, field, short, notMade)}
}

func schemaTypes(schema map[string]any) []string {
	switch t := schema["type"].(type) {
	case string:
		return []string{t}
	case []any:
		out := make([]string, 0, len(t))
		for _, v := range t {
			out = append(out, fmt.Sprint(v))
		}
		return out
	}
	return nil
}

// expected names what a schema takes: its enum when it has one, else its types.
func expected(schema map[string]any, types []string) string {
	if enum, ok := schema["enum"].([]any); ok {
		var values []string
		hasNull := false
		for _, v := range enum {
			if v == nil {
				hasNull = true
				continue
			}
			b, _ := json.Marshal(v)
			values = append(values, string(b))
		}
		list := strings.Join(values, ", ")
		if hasNull {
			return "one of " + list + " or null"
		}
		if len(values) > 1 {
			return "one of " + strings.Join(values[:len(values)-1], ", ") + " or " + values[len(values)-1]
		}
		return list
	}
	names := map[string]string{
		"string": "a string", "integer": "a whole number", "number": "a number",
		"boolean": "true or false", "array": "an array", "object": "an object",
	}
	var parts []string
	nullable := false
	for _, t := range types {
		if t == "null" {
			nullable = true
			continue
		}
		parts = append(parts, names[t])
	}
	if nullable {
		parts = append(parts, "null")
	}
	return strings.Join(parts, " or ")
}

func describeValue(v any) string {
	switch x := v.(type) {
	case string:
		if len(x) > maxGotQuote {
			x = x[:maxGotQuote] + "…"
		}
		return "a string (" + strconv.Quote(x) + ")"
	case float64:
		return formatNumber(x)
	case bool:
		return strconv.FormatBool(x)
	case []any:
		return "an array"
	case map[string]any:
		return "an object"
	}
	return fmt.Sprintf("%T", v)
}

func formatNumber(n float64) string {
	return strconv.FormatFloat(n, 'f', -1, 64)
}

func isIDParam(name string) bool {
	return name == "id" || strings.HasSuffix(name, "_id")
}

// listToolFor names the tool that lists what an ID parameter refers to, when
// one follows the list_<resource>s pattern.
func listToolFor(param string) string {
	name := "list_" + strings.TrimSuffix(param, "_id") + "s"
	if _, ok := generatedTools[name]; ok {
		return name
	}
	return ""
}

func pathString(loc []string) string {
	var b strings.Builder
	for i, part := range loc {
		if _, err := strconv.Atoi(part); err == nil && i > 0 {
			b.WriteString("[" + part + "]")
			continue
		}
		if i > 0 {
			b.WriteString(".")
		}
		b.WriteString(part)
	}
	return b.String()
}

// message is the refusal: a problem's own sentence when there's one, else a
// list.
func (c *checker) message(problems []problem) string {
	if len(problems) == 1 {
		return problems[0].full
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s refused these arguments; the call was not made:\n", c.tool)
	unknown := false
	var topics []string
	for _, p := range problems {
		fmt.Fprintf(&b, "- %s: %s\n", p.path, p.short)
		unknown = unknown || p.kind == "unknown" || p.kind == "stale"
		if p.topic != "" && !slices.Contains(topics, p.topic) {
			topics = append(topics, p.topic)
		}
	}
	if unknown {
		b.WriteString("If you expected a parameter to exist, " + maybeStale + "\n")
	}
	for _, t := range topics {
		fmt.Fprintf(&b, "See get_reference topic %q.\n", t)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// editDistance is the Levenshtein distance between a and b.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// renameTarget is the new name in a stale field's "renamed to X" reason.
func renameTarget(reason string) string {
	if target, ok := strings.CutPrefix(reason, "renamed to "); ok {
		return target
	}
	return ""
}
