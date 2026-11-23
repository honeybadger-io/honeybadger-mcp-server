package hbmcp

import (
	"encoding/json"
	"fmt"
	"slices"
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
// first operation it calls, in the spec's tag order.
func ToolReference(bundle []byte) (string, error) {
	tagOf, order, err := specTags(bundle)
	if err != nil {
		return "", err
	}

	sections := map[string][]mcp.Tool{}
	for name, st := range NewServer(&config.Config{APIURL: "http://localhost"}, "").ListTools() {
		section := noOperationSection
		if ops := toolOperations[name]; len(ops) > 0 {
			tag, ok := tagOf[ops[0]]
			if !ok {
				return "", fmt.Errorf("%s: operation %s has no tag in the bundle", name, ops[0])
			}
			section = tag
		}
		sections[section] = append(sections[section], st.Tool)
	}

	for section := range sections {
		if !slices.Contains(order, section) {
			return "", fmt.Errorf("tag %q isn't in the spec's x-tagGroups, so its tools would be left out", section)
		}
	}

	var b strings.Builder
	for _, section := range order {
		tools := sections[section]
		if len(tools) == 0 {
			continue
		}
		sort.Slice(tools, func(i, j int) bool { return toolRank(tools[i].Name) < toolRank(tools[j].Name) })
		fmt.Fprintf(&b, "### %s\n\n", section)
		for _, t := range tools {
			writeToolEntry(&b, t)
		}
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

// specTags reads each operation's first tag, and the tags in the order the
// spec groups them.
func specTags(bundle []byte) (tagOf map[string]string, order []string, err error) {
	var spec struct {
		Paths     map[string]map[string]yaml.Node `yaml:"paths"`
		TagGroups []struct {
			Tags []string `yaml:"tags"`
		} `yaml:"x-tagGroups"`
	}
	if err := yaml.Unmarshal(bundle, &spec); err != nil {
		return nil, nil, fmt.Errorf("reading the bundle: %w", err)
	}
	tagOf = map[string]string{}
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
	order = []string{noOperationSection}
	for _, g := range spec.TagGroups {
		order = append(order, g.Tags...)
	}
	return tagOf, order, nil
}

// toolRank orders a section's tools list, get, create, update, delete, then
// the rest, by name within each.
func toolRank(name string) string {
	for i, verb := range []string{"list_", "get_", "create_", "update_", "delete_"} {
		if strings.HasPrefix(name, verb) {
			return fmt.Sprintf("%d %s", i, name)
		}
	}
	return "9 " + name
}

func writeToolEntry(b *strings.Builder, t mcp.Tool) {
	suffix := ""
	if t.Annotations.ReadOnlyHint == nil || !*t.Annotations.ReadOnlyHint {
		suffix = " _(changes data; hidden in read-only mode)_"
	}
	fmt.Fprintf(b, "- **%s** - %s%s\n", t.Name, t.Description, suffix)

	required := map[string]bool{}
	for _, r := range t.InputSchema.Required {
		required[r] = true
	}
	names := make([]string, 0, len(t.InputSchema.Properties))
	for n := range t.InputSchema.Properties {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if required[names[i]] != required[names[j]] {
			return required[names[i]]
		}
		return names[i] < names[j]
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
