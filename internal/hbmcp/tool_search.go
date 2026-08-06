package hbmcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ToolInfo struct {
	Name        string
	Description string
	ReadOnly    bool
}

type toolRegistrar struct {
	server  *server.MCPServer
	catalog []ToolInfo
}

func newToolRegistrar(s *server.MCPServer) *toolRegistrar {
	return &toolRegistrar{
		server: s,
	}
}

func (r *toolRegistrar) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	r.server.AddTool(tool, handler)
	r.catalog = append(r.catalog, ToolInfo{
		Name:        tool.Name,
		Description: tool.Description,
		ReadOnly:    tool.Annotations.ReadOnlyHint != nil && *tool.Annotations.ReadOnlyHint,
	})
}

// searchNormalizer folds the separators used in tool names ("list_faults") and
// in prose ("check-in") down to spaces so a query typed either way matches.
var searchNormalizer = strings.NewReplacer("_", " ", "-", " ")

func normalizeSearchText(s string) string {
	return searchNormalizer.Replace(strings.ToLower(s))
}

// searchCatalog matches every whitespace-separated term in query against the
// tool's name and description combined. All terms must match (AND), in any
// order; each term is still a substring match, so "project" finds "projects".
func searchCatalog(catalog []ToolInfo, query string) []ToolInfo {
	terms := strings.Fields(normalizeSearchText(query))
	if len(terms) == 0 {
		return nil
	}

	var results []ToolInfo
	for _, t := range catalog {
		haystack := normalizeSearchText(t.Name + " " + t.Description)
		matched := true
		for _, term := range terms {
			if !strings.Contains(haystack, term) {
				matched = false
				break
			}
		}
		if matched {
			results = append(results, t)
		}
	}
	return results
}

var searchToolInfo = ToolInfo{
	Name:        "search_tools",
	Description: "Search available Honeybadger tools by name or description. Multi-word queries match tools containing all of the words, in any order. Use this to discover tools before calling them.",
	ReadOnly:    true,
}

func registerSearchTool(s *server.MCPServer, catalog []ToolInfo, cfg *config.Config) {
	s.AddTool(
		mcp.NewTool(searchToolInfo.Name,
			mcp.WithTitleAnnotation("Search Tools"),
			mcp.WithDescription(searchToolInfo.Description),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("Search query to match against tool names and descriptions. Multiple words are all required, in any order (e.g. \"list faults\")"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query := req.GetString("query", "")
			// Separators normalize to spaces, so a query of "_" carries no
			// search terms even though it survives TrimSpace.
			if len(strings.Fields(normalizeSearchText(query))) == 0 {
				return mcp.NewToolResultError("query is required"), nil
			}

			searchable := catalog
			if EffectiveReadOnly(ctx, cfg) {
				searchable = filterReadOnlyCatalog(catalog)
			}

			matches := searchCatalog(searchable, query)
			if len(matches) == 0 {
				return mcp.NewToolResultText("No tools found matching the query."), nil
			}

			var sb strings.Builder
			for i, m := range matches {
				if i > 0 {
					sb.WriteString("\n\n")
				}
				readOnlyStr := "no"
				if m.ReadOnly {
					readOnlyStr = "yes"
				}
				fmt.Fprintf(&sb, "Name: %s\nDescription: %s\nRead-only: %s", m.Name, m.Description, readOnlyStr)
			}
			return mcp.NewToolResultText(sb.String()), nil
		},
	)
}

func filterReadOnlyCatalog(catalog []ToolInfo) []ToolInfo {
	var filtered []ToolInfo
	for _, t := range catalog {
		if t.ReadOnly {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
