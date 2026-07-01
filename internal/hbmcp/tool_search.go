package hbmcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type toolInfo struct {
	Name        string
	Description string
	ReadOnly    bool
}

type toolRegistrar struct {
	server  *server.MCPServer
	catalog []toolInfo
}

func newToolRegistrar(s *server.MCPServer) *toolRegistrar {
	return &toolRegistrar{
		server: s,
	}
}

func (r *toolRegistrar) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	r.server.AddTool(tool, handler)
	r.catalog = append(r.catalog, toolInfo{
		Name:        tool.Name,
		Description: tool.Description,
		ReadOnly:    tool.Annotations.ReadOnlyHint != nil && *tool.Annotations.ReadOnlyHint,
	})
}

func searchCatalog(catalog []toolInfo, query string) []toolInfo {
	q := strings.ToLower(query)
	var results []toolInfo
	for _, t := range catalog {
		if strings.Contains(strings.ToLower(t.Name), q) ||
			strings.Contains(strings.ToLower(t.Description), q) {
			results = append(results, t)
		}
	}
	return results
}

func registerSearchTool(s *server.MCPServer, catalog []toolInfo, readOnly bool) {
	s.AddTool(
		mcp.NewTool("search_tools",
			mcp.WithDescription("Search available Honeybadger tools by name or description. Use this to discover tools before calling them."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("Search query to match against tool names and descriptions"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query := strings.TrimSpace(req.GetString("query", ""))
			if query == "" {
				return mcp.NewToolResultError("query is required"), nil
			}

			searchable := catalog
			if readOnly {
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
				sb.WriteString(fmt.Sprintf("Name: %s\nDescription: %s\nRead-only: %s", m.Name, m.Description, readOnlyStr))
			}
			return mcp.NewToolResultText(sb.String()), nil
		},
	)
}

func filterReadOnlyCatalog(catalog []toolInfo) []toolInfo {
	var filtered []toolInfo
	for _, t := range catalog {
		if t.ReadOnly {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
