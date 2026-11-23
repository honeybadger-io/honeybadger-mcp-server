package hbmcp

import (
	"os"
	"strings"
	"testing"
)

// The README's tool reference is rendered from the registered tools, so it
// can't drift from what tools/list advertises.
func TestToolReferenceListsEveryToolOnce(t *testing.T) {
	bundle, err := os.ReadFile("../../openapi/bundled.yaml")
	if err != nil {
		t.Fatal(err)
	}
	md, err := ToolReference(bundle)
	if err != nil {
		t.Fatal(err)
	}
	for name := range registeredTools(t) {
		if n := strings.Count(md, "- **"+name+"** - "); n != 1 {
			t.Errorf("%s appears %d times", name, n)
		}
	}
	for _, want := range []string{
		"### Reference and search\n\n- **get_reference** - ",
		"### Projects\n",
		"  - `project_id` : The ID of the project to retrieve (string, required)\n",
		"  - `user_url` : User link shown on faults, e.g. 'http://example.com/admin/users/[user_id]'. Null clears it. (string or null, optional)\n",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("reference lacks %q", want)
		}
	}
	// A tool sits under its first operation's spec tag, hand-written or not.
	faults := md[strings.Index(md, "### Faults\n"):]
	faults = faults[:strings.Index(faults[1:], "\n### ")+1]
	for _, name := range []string{"update_fault", "list_faults"} {
		if !strings.Contains(faults, "- **"+name+"** - ") {
			t.Errorf("%s isn't under Faults", name)
		}
	}
	if !strings.Contains(md, "- **create_project** - Create a new Honeybadger project _(changes data; hidden in read-only mode)_") {
		t.Error("a write tool isn't marked as hidden in read-only mode")
	}
}

// A tag the spec doesn't group would leave its tools out of the README, so it
// fails generation instead.
func TestToolReferenceRefusesAnUngroupedTag(t *testing.T) {
	bundle, err := os.ReadFile("../../openapi/bundled.yaml")
	if err != nil {
		t.Fatal(err)
	}
	ungrouped := strings.Replace(string(bundle), "  tags:\n  - Faults\n", "  tags:\n", 1)
	if ungrouped == string(bundle) {
		t.Fatal("couldn't take Faults out of x-tagGroups; the bundle's layout changed")
	}
	_, err = ToolReference([]byte(ungrouped))
	if err == nil || !strings.Contains(err.Error(), `tag "Faults" isn't in the spec's x-tagGroups`) {
		t.Errorf("err = %v, want the ungrouped tag named", err)
	}
}
