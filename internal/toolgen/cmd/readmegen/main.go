// Command readmegen rewrites README.md's tool reference, between its marker
// comments, from the registered tools. Run it through make generate, after
// toolgen.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbmcp"
)

const (
	begin = "<!-- BEGIN GENERATED TOOLS: run make generate; edit openapi/tools.yaml, not this list -->"
	end   = "<!-- END GENERATED TOOLS -->"
)

func main() {
	if err := run("openapi/bundled.yaml", "openapi/tools.yaml", "README.md"); err != nil {
		fmt.Fprintln(os.Stderr, "readmegen:", err)
		os.Exit(1)
	}
}

func run(bundlePath, overlayPath, readmePath string) error {
	bundle, err := os.ReadFile(bundlePath)
	if err != nil {
		return err
	}
	overlay, err := os.ReadFile(overlayPath)
	if err != nil {
		return err
	}
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}
	before, rest, ok := strings.Cut(string(readme), begin)
	if !ok {
		return fmt.Errorf("%s has no %q marker", readmePath, begin)
	}
	_, after, ok := strings.Cut(rest, end)
	if !ok {
		return fmt.Errorf("%s has no %q marker", readmePath, end)
	}
	reference, err := hbmcp.ToolReference(bundle, overlay)
	if err != nil {
		return err
	}
	out := before + begin + "\n\n" + reference + "\n\n" + end + after
	return os.WriteFile(readmePath, []byte(out), 0o644)
}
