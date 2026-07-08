package hbmcp

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

//go:embed docs/badgerql.md
var topicBadgerQL string

//go:embed docs/charts.md
var topicCharts string

//go:embed docs/dashboards.md
var topicDashboards string

//go:embed docs/alarms.md
var topicAlarms string

type referenceTopic struct {
	name    string
	summary string
	content string
}

// Order is the stable composition order: fundamentals, then the query
// language, then the layers that consume it. Topics must not overlap —
// shared content (e.g. chart_config) lives in exactly one topic and the
// others point to it.
var referenceTopics = []referenceTopic{
	{"badgerql", "BadgerQL query language: syntax, functions, type hints, time ranges, response schema", topicBadgerQL},
	{"charts", "Visualizations: view types, chart_config options, shareable Insights URLs", topicCharts},
	{"dashboards", "Dashboards: widget types and structure, grid layout, create/update workflow", topicDashboards},
	{"alarms", "Alarms: trigger_config schema, states, evaluation timing, common patterns", topicAlarms},
}

func topicNames() []string {
	names := make([]string, len(referenceTopics))
	for i, t := range referenceTopics {
		names[i] = t.name
	}
	return names
}

func referenceIndex() string {
	var sb strings.Builder
	sb.WriteString("Honeybadger reference topics. Fetch the ones you need in a single call: get_reference with topics: [\"name\", ...]. Use [\"all\"] for everything. Skip any topic whose content is still visible in your context.\n")
	for _, t := range referenceTopics {
		fmt.Fprintf(&sb, "\n- %s (%d KB) — %s", t.name, (len(t.content)+1023)/1024, t.summary)
	}
	return sb.String()
}

func composeReference(requested []string) (string, error) {
	want := make(map[string]bool, len(requested))
	for _, name := range requested {
		if name == "all" {
			for _, t := range referenceTopics {
				want[t.name] = true
			}
			continue
		}
		valid := false
		for _, t := range referenceTopics {
			if t.name == name {
				valid = true
				break
			}
		}
		if !valid {
			return "", fmt.Errorf("unknown topic %q; valid topics: %s, or \"all\"", name, strings.Join(topicNames(), ", "))
		}
		want[name] = true
	}

	var parts []string
	for _, t := range referenceTopics {
		if want[t.name] {
			parts = append(parts, t.content)
		}
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}

// RegisterReferenceTools registers the get_reference documentation tool
func RegisterReferenceTools(r *toolRegistrar) {
	r.AddTool(
		mcp.NewTool("get_reference",
			mcp.WithDescription("Returns Honeybadger reference documentation by topic: badgerql (query language), charts (view types, chart_config, shareable URLs), dashboards (widgets, layout), alarms (trigger_config, states, patterns). Fetch all topics you need in one call, e.g. topics: [\"badgerql\", \"charts\"]; skip topics still visible in your context. Call with no arguments for a topic index, or [\"all\"] for everything."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithArray("topics",
				mcp.Description("Reference topics to fetch: badgerql, charts, dashboards, alarms, or all. Omit for an index of topics."),
				mcp.WithStringItems(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetReference(ctx, req)
		},
	)
}

func handleGetReference(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requested := req.GetStringSlice("topics", nil)
	if len(requested) == 0 {
		return mcp.NewToolResultText(referenceIndex()), nil
	}

	content, err := composeReference(requested)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(content), nil
}

// ServerInstructions is the always-on context sent to clients at session
// init: the reference topic map, the fetch-once rule, and the cross-tool
// fundamentals. Keep it small — every session pays for it. Interface facts
// belong in reference topics, not here; procedural methodology belongs in
// client-side skills, not the server.
func ServerInstructions() string {
	var sb strings.Builder
	sb.WriteString("Honeybadger provides error tracking, uptime monitoring, and Insights (log/event analytics queried with BadgerQL).\n\n")
	sb.WriteString("Reference documentation is split into non-overlapping topics served by the get_reference tool:\n")
	for _, t := range referenceTopics {
		fmt.Fprintf(&sb, "- %s — %s\n", t.name, t.summary)
	}
	sb.WriteString("\nTool descriptions state which topics they require. Fetch required topics in a single get_reference call before using those tools, and never re-fetch a topic whose content is still visible in your context.\n\n")
	sb.WriteString("Fundamentals: discover before you analyze — resolve projects with list_projects, verify data exists with a minimal count query, and preview event fields before writing analytical queries. Build queries incrementally with narrow time ranges and small limits, then widen. Verify a query returns the expected shape via query_insights before embedding it in a dashboard widget or alarm trigger. After running queries or creating dashboards and alarms, give the user Honeybadger UI links to the results.")
	return sb.String()
}
