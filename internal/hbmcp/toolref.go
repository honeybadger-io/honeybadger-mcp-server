package hbmcp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v3"
)

// noOperationSection holds the tools that reach no API: they have no spec tag.
const noOperationSection = "Reference and search"

// ToolReference renders the README's tool reference from the registered tools,
// so it says what tools/list says. Each tool sits under the spec tag of the
// first operation it calls. The overlay's order is the layout: a section
// appears where its first tool is listed, and tools and parameters appear as
// the overlay lists them.
func ToolReference(bundle, overlay []byte) (string, error) {
	tagOf, err := specTags(bundle)
	if err != nil {
		return "", err
	}
	toolOrder, paramOrder, err := overlayOrder(overlay)
	if err != nil {
		return "", err
	}

	registered := NewServer(&config.Config{APIURL: "http://localhost"}, "").ListTools()
	var order []string
	sections := map[string][]mcp.Tool{}
	for _, name := range toolOrder {
		st, ok := registered[name]
		if !ok {
			continue
		}
		section := noOperationSection
		if ops := toolOperations[name]; len(ops) > 0 {
			tag, ok := tagOf[ops[0]]
			if !ok {
				return "", fmt.Errorf("%s: operation %s has no tag in the bundle", name, ops[0])
			}
			section = tag
		}
		if _, seen := sections[section]; !seen {
			order = append(order, section)
		}
		sections[section] = append(sections[section], st.Tool)
	}

	var b strings.Builder
	for _, section := range order {
		fmt.Fprintf(&b, "### %s\n\n", section)
		for _, t := range sections[section] {
			writeToolEntry(&b, t, paramOrder[t.Name])
		}
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

// specTags reads each operation's first tag.
func specTags(bundle []byte) (map[string]string, error) {
	var spec struct {
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal(bundle, &spec); err != nil {
		return nil, fmt.Errorf("reading the bundle: %w", err)
	}
	tagOf := map[string]string{}
	for _, item := range spec.Paths {
		for _, node := range item {
			var op struct {
				OperationID string   `yaml:"operationId"`
				Tags        []string `yaml:"tags"`
			}
			if node.Decode(&op) == nil && op.OperationID != "" && len(op.Tags) > 0 {
				tagOf[op.OperationID] = op.Tags[0]
			}
		}
	}
	return tagOf, nil
}

// overlayOrder reads the tools in the order the overlay lists them, and each
// tool's parameters likewise. A YAML mapping keeps its order only as nodes.
func overlayOrder(overlay []byte) (tools []string, params map[string][]string, err error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(overlay, &doc); err != nil {
		return nil, nil, fmt.Errorf("reading the overlay: %w", err)
	}
	if len(doc.Content) == 0 {
		return nil, nil, nil
	}
	params = map[string][]string{}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		name, spec := root.Content[i].Value, root.Content[i+1]
		tools = append(tools, name)
		for j := 0; j+1 < len(spec.Content); j += 2 {
			if spec.Content[j].Value != "params" {
				continue
			}
			params[name] = mappingKeys(spec.Content[j+1])
		}
	}
	return tools, params, nil
}

// mappingKeys lists a mapping's keys in order, with a YAML merge (<<: *anchor,
// or a list of anchors) expanded in place into the keys it brings in.
func mappingKeys(m *yaml.Node) []string {
	if m.Kind == yaml.AliasNode {
		m = m.Alias
	}
	var keys []string
	for k := 0; k+1 < len(m.Content); k += 2 {
		key, value := m.Content[k], m.Content[k+1]
		if key.Value != "<<" {
			keys = append(keys, key.Value)
			continue
		}
		merged := []*yaml.Node{value}
		if value.Kind == yaml.SequenceNode {
			merged = value.Content
		}
		for _, mm := range merged {
			keys = append(keys, mappingKeys(mm)...)
		}
	}
	return keys
}

// writeToolEntry lists a tool's parameters in the overlay's order, then any
// it doesn't list (confirm, a hand-written tool's), required first and by
// name.
func writeToolEntry(b *strings.Builder, t mcp.Tool, order []string) {
	suffix := ""
	if t.Annotations.ReadOnlyHint == nil || !*t.Annotations.ReadOnlyHint {
		suffix = " _(changes data; hidden in read-only mode)_"
	}
	fmt.Fprintf(b, "- **%s** - %s%s\n", t.Name, t.Description, suffix)

	required := map[string]bool{}
	for _, r := range t.InputSchema.Required {
		required[r] = true
	}
	rank := map[string]int{}
	for i, n := range order {
		if rank[n] == 0 { // a key listed again through a merge keeps its first place
			rank[n] = i + 1
		}
	}
	names := make([]string, 0, len(t.InputSchema.Properties))
	for n := range t.InputSchema.Properties {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		x, y := names[i], names[j]
		if (rank[x] > 0) != (rank[y] > 0) {
			return rank[x] > 0
		}
		if rank[x] != rank[y] {
			return rank[x] < rank[y]
		}
		if required[x] != required[y] {
			return required[x]
		}
		return x < y
	})
	for _, n := range names {
		raw, _ := json.Marshal(t.InputSchema.Properties[n])
		var p map[string]any
		_ = json.Unmarshal(raw, &p)
		need := "optional"
		if required[n] {
			need = "required"
		}
		fmt.Fprintf(b, "  - `%s` : %v (%s, %s)\n", n, p["description"], strings.Join(schemaTypes(p), " or "), need)
	}
	b.WriteString("\n")
}
