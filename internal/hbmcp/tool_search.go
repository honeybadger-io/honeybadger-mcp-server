package hbmcp

import (
	"context"
	"fmt"
	"log/slog"
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
	logger  *slog.Logger
	catalog []ToolInfo
}

func newToolRegistrar(s *server.MCPServer, logger *slog.Logger) *toolRegistrar {
	return &toolRegistrar{server: s, logger: logger}
}

// AddTool registers a tool with its arguments checked against its definition
// before the handler runs.
func (r *toolRegistrar) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	r.server.AddTool(tool, strictArgs(tool, handler, r.logger))
	r.catalog = append(r.catalog, ToolInfo{
		Name:        tool.Name,
		Description: tool.Description,
		ReadOnly:    tool.Annotations.ReadOnlyHint != nil && *tool.Annotations.ReadOnlyHint,
	})
}

func searchCatalog(catalog []ToolInfo, query string) []ToolInfo {
	q := strings.ToLower(query)
	var results []ToolInfo
	for _, t := range catalog {
		if strings.Contains(strings.ToLower(t.Name), q) ||
			strings.Contains(strings.ToLower(t.Description), q) {
			results = append(results, t)
		}
	}
	return results
}

var searchToolInfo = ToolInfo{
	Name:        "search_tools",
	Description: "Search available Honeybadger tools by name or description. Use this to discover tools before calling them.",
	ReadOnly:    true,
}

func registerSearchTool(s *server.MCPServer, catalog []ToolInfo, cfg *config.Config, logger *slog.Logger) {
	tool := mcp.NewTool(searchToolInfo.Name,
		mcp.WithTitleAnnotation("Search Tools"),
		mcp.WithDescription(searchToolInfo.Description),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query to match against tool names and descriptions"),
		),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := strings.TrimSpace(req.GetString("query", ""))
		if query == "" {
			return mcp.NewToolResultError("query is required"), nil
		}

		searchable := catalog
		if EffectiveReadOnly(ctx, cfg) {
			searchable = filterReadOnlyCatalog(catalog)
		}
		// Search has to apply the same scope filter as tools/list, or it
		// becomes a way to discover tools the catalog deliberately hid.
		if scopes, ok := knownScopes(ctx); ok {
			searchable = filterCatalogByScopes(searchable, scopes)
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
	}
	s.AddTool(tool, strictArgs(tool, handler, logger))
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

// generatedTool returns a tool definition generated from the spec. A name the
// overlay doesn't define panics at startup, so a missing entry can't register
// an empty tool.
func generatedTool(name string) mcp.Tool {
	t, ok := generatedTools[name]
	if !ok {
		panic("hbmcp: no generated definition for " + name + "; add it to openapi/tools.yaml and run make generate")
	}
	return t
}
