package hbmcp

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
)

// notYetGenerated are the tools still defined in Go while the overlay grows a
// domain at a time. It shrinks to nothing; then every tool is generated or
// listed as hand-written in openapi/tools.yaml.
var notYetGenerated = []string{
	"get_project_integrations",
	"list_dashboards", "get_dashboard", "create_dashboard", "update_dashboard", "delete_dashboard",
	"list_alarms", "get_alarm", "get_alarm_history", "create_alarm", "update_alarm", "delete_alarm",
	"get_integration", "create_integration", "update_integration", "delete_integration",
	"list_ingestion_keys", "create_ingestion_key", "update_ingestion_key", "delete_ingestion_key",
	"query_insights", "list_streams",
}

// Every registered tool comes from the overlay, generated or hand-written, and
// every generated tool is registered with exactly its generated definition.
func TestToolsComeFromTheOverlay(t *testing.T) {
	registered := NewServer(&config.Config{APIURL: "http://x"}, "t").ListTools()
	for name, st := range registered {
		gen, generated := generatedTools[name]
		switch {
		case generated:
			if !reflect.DeepEqual(st.Tool, gen) {
				t.Errorf("%s is registered with a definition other than the generated one", name)
			}
		case slices.Contains(generatedHandwritten, name), slices.Contains(notYetGenerated, name):
		default:
			t.Errorf("%s isn't in openapi/tools.yaml", name)
		}
	}
	for name := range generatedTools {
		if _, ok := registered[name]; !ok {
			t.Errorf("%s is generated but has no handler", name)
		}
	}
}

// A generated tool is read-only exactly when every operation it reaches needs
// only read scopes, as the scope filter sees them. Read-only mode hides the
// others, so the two must agree.
func TestGeneratedAnnotationsMatchScopes(t *testing.T) {
	for name, tool := range generatedTools {
		scopes, resolved := toolRequiredScopes(name)
		if !resolved {
			t.Errorf("%s: its operations aren't all in the scope map", name)
			continue
		}
		readOnly := len(scopes) > 0
		for _, s := range scopes {
			if !strings.HasSuffix(s, ":read") {
				readOnly = false
			}
		}
		if got := *tool.Annotations.ReadOnlyHint; got != readOnly {
			t.Errorf("%s: readOnlyHint = %v; its scopes %v say %v", name, got, scopes, readOnly)
		}
		if got := *tool.Annotations.DestructiveHint; got == readOnly {
			t.Errorf("%s: destructiveHint = %v with readOnlyHint %v", name, got, readOnly)
		}
	}
}

func TestShallowTopicsAreReferenceTopics(t *testing.T) {
	for tool, params := range generatedShallowTopics {
		for param, topic := range params {
			if !slices.Contains(referenceTopicNames, topic) {
				t.Errorf("%s.%s: shallow topic %q isn't a get_reference topic", tool, param, topic)
			}
		}
	}
}
