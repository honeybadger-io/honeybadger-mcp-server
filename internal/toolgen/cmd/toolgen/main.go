// Command toolgen writes internal/hbmcp/zz_generated_tools.go from the vendored
// OpenAPI bundle and the overlay. Run it through make generate.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/toolgen"
)

func main() {
	bundle := flag.String("bundle", "openapi/bundled.yaml", "vendored OpenAPI bundle")
	overlay := flag.String("overlay", "openapi/tools.yaml", "tool overlay")
	out := flag.String("out", "internal/hbmcp/zz_generated_tools.go", "generated Go file")
	flag.Parse()

	if err := run(*bundle, *overlay, *out); err != nil {
		fmt.Fprintln(os.Stderr, "toolgen:", err)
		os.Exit(1)
	}
}

func run(bundlePath, overlayPath, outPath string) error {
	bundle, err := os.ReadFile(bundlePath)
	if err != nil {
		return err
	}
	overlay, err := os.ReadFile(overlayPath)
	if err != nil {
		return err
	}
	src, err := toolgen.Generate(bundle, overlay)
	if err != nil {
		return fmt.Errorf("\n%w", err)
	}
	return os.WriteFile(outPath, src, 0o644)
}
