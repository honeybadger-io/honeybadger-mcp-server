package hbmcp

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/analytics"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	eventToolCall = "mcp.tool_call"

	outcomeOK            = "ok"
	outcomeToolError     = "tool_error"
	outcomeInternalError = "internal_error"
	outcomePanic         = "panic"
)

type instrumenter struct {
	sink    analytics.Sink
	cfg     *config.Config
	version string
	enabled bool
}

func newInstrumenter(sink analytics.Sink, cfg *config.Config, version string) *instrumenter {
	return &instrumenter{
		sink:    sink,
		cfg:     cfg,
		version: version,
		enabled: !analytics.IsNop(sink),
	}
}

// wrap decorates a tool handler with usage analytics. It takes the whole
// mcp.Tool so the argument allowlist is built once at registration rather than
// on every call.
//
// With analytics off it returns the handler untouched. Wrapping and discarding
// the result would still add a context value, a timer, and a panic
// intercept/re-panic per call — and that last one rewrites the stack
// server.WithRecovery sees. stdio runs the original handler, unchanged.
func (i *instrumenter) wrap(tool mcp.Tool, h server.ToolHandlerFunc) server.ToolHandlerFunc {
	if !i.enabled {
		return h
	}

	name := tool.Name
	allowed := allowedArgNames(tool)

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, rec := withUpstreamRecord(ctx)
		start := time.Now()

		// Observe panics, report them, then re-panic so server.WithRecovery
		// still handles the panic exactly as it did before.
		defer func() {
			if r := recover(); r != nil {
				i.emit(ctx, name, allowed, req, outcomePanic, start, rec,
					fmt.Errorf("panic in tool %s: %v", name, r))
				panic(r)
			}
		}()

		result, err := h(ctx, req)
		i.emit(ctx, name, allowed, req, i.classify(result, err, rec), start, rec, err)
		return result, err
	}
}

// classify separates a product signal (the caller got it wrong) from an outage
// signal (the API is failing). It cannot rely on the handler's return value
// alone: handlers convert upstream failures into IsError results with a nil
// error, so an outage is only visible in the upstream record.
func (i *instrumenter) classify(result *mcp.CallToolResult, err error, rec *upstreamRecord) string {
	if err != nil {
		return outcomeInternalError
	}
	called, status, transportErr, _ := rec.snapshot()
	if called && (transportErr || status >= 500) {
		return outcomeInternalError
	}
	if result != nil && result.IsError {
		return outcomeToolError
	}
	return outcomeOK
}

func (i *instrumenter) emit(
	ctx context.Context,
	name string,
	allowed map[string]struct{},
	req mcp.CallToolRequest,
	outcome string,
	start time.Time,
	rec *upstreamRecord,
	err error,
) {
	names, unknown := splitArgNames(req.GetArguments(), allowed)

	data := map[string]any{
		"tool":              name,
		"outcome":           outcome,
		"duration_ms":       time.Since(start).Milliseconds(),
		"arg_names":         names,
		"unknown_arg_count": unknown,
		"read_only":         EffectiveReadOnly(ctx, i.cfg),
		"server_version":    i.version,
	}

	claims := ClaimsFromContext(ctx)
	if claims != nil {
		data["user_id"] = claims.Subject
		data["account_id"] = claims.AccountID
		data["client_id"] = claims.ClientID
		data["project_scoped"] = claims.ProjectScoped
	}

	if called, status, transportErr, durationMs := rec.snapshot(); called {
		data["upstream_ms"] = durationMs
		if !transportErr {
			data["upstream_status"] = status
		}
	}

	// Error detail goes to error tracking, never into the event: messages can
	// carry API response bodies, which are customer data.
	if outcome == outcomeInternalError || outcome == outcomePanic {
		if err == nil {
			err = fmt.Errorf("tool %s failed upstream", name)
		}
		notice := analytics.Notice{
			Err: err,
			// Without this, every tool's errors share the decorator's stack
			// frame and group together.
			Fingerprint: "mcp_tool_call:" + name,
			Context: map[string]any{
				"tool":    name,
				"outcome": outcome,
			},
		}
		if claims != nil {
			notice.Context["account_id"] = claims.AccountID
			notice.Context["client_id"] = claims.ClientID
		}
		if token := i.sink.Notify(notice); token != "" {
			data["error_id"] = token
		}
	}

	i.sink.Emit(analytics.Event{Type: eventToolCall, Data: data})
}

// allowedArgNames builds the set of argument names declared by the tool's
// input schema. Arguments arrive as an arbitrary map and this server enables no
// schema validation, so an undeclared key is both an unbounded-cardinality risk
// and a place a caller could hide customer data.
func allowedArgNames(tool mcp.Tool) map[string]struct{} {
	allowed := make(map[string]struct{}, len(tool.InputSchema.Properties))
	for k := range tool.InputSchema.Properties {
		allowed[k] = struct{}{}
	}
	return allowed
}

func splitArgNames(args map[string]any, allowed map[string]struct{}) ([]string, int) {
	names := make([]string, 0, len(args))
	unknown := 0
	for k := range args {
		if _, ok := allowed[k]; ok {
			names = append(names, k)
			continue
		}
		unknown++
	}
	sort.Strings(names)
	return names, unknown
}
