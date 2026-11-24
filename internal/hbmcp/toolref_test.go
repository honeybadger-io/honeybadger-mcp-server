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
	md, err := ToolReference(bundle, overlayBytes(t))
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

// Projects come first after the API-free tools: every other tool takes a
// project_id, so it's where a reader starts.
func TestToolReferenceListsProjectsFirst(t *testing.T) {
	bundle, err := os.ReadFile("../../openapi/bundled.yaml")
	if err != nil {
		t.Fatal(err)
	}
	md, err := ToolReference(bundle, overlayBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	ref, projects, faults := strings.Index(md, "### Reference and search\n"), strings.Index(md, "### Projects\n"), strings.Index(md, "### Faults\n")
	if ref >= projects || projects >= faults {
		t.Errorf("section offsets: reference %d, projects %d, faults %d; want reference, then projects, then faults", ref, projects, faults)
	}
}

// The overlay's order is the README's: a section where its first tool is,
// tools and parameters as the overlay lists them.
func TestToolReferenceFollowsTheOverlaysOrder(t *testing.T) {
	bundle, err := os.ReadFile("../../openapi/bundled.yaml")
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := os.ReadFile("../../openapi/tools.yaml")
	if err != nil {
		t.Fatal(err)
	}
	md, err := ToolReference(bundle, overlay)
	if err != nil {
		t.Fatal(err)
	}
	inOrder := func(what, text string, items ...string) {
		t.Helper()
		last := -1
		for _, s := range items {
			i := strings.Index(text, s)
			if i < 0 || i < last {
				t.Errorf("%s: %q is out of order (at %d, after %d)", what, s, i, last)
			}
			last = i
		}
	}
	inOrder("tools", md, "- **list_faults** - ", "- **get_fault** - ", "- **update_fault** - ", "- **get_fault_counts** - ")
	get := md[strings.Index(md, "- **get_fault** - "):]
	get = get[:strings.Index(get, "\n\n")]
	inOrder("get_fault's parameters", get, "  - `project_id` :", "  - `fault_id` :")
	// update_site takes the create settings through a YAML merge; they keep the
	// overlay's order after the tool's own.
	upd := md[strings.Index(md, "- **update_site** - "):]
	upd = upd[:strings.Index(upd, "\n\n")]
	inOrder("update_site's parameters", upd, "  - `project_id` :", "  - `site_id` :", "  - `url` :", "  - `name` :", "  - `frequency` :", "  - `active` :")
}

func overlayBytes(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../../openapi/tools.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
