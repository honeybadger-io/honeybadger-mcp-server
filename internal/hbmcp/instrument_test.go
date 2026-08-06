package hbmcp

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/analytics"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type recordingSink struct {
	events  []analytics.Event
	notices []analytics.Notice
	token   string
}

func (s *recordingSink) Emit(e analytics.Event) { s.events = append(s.events, e) }
func (s *recordingSink) Notify(n analytics.Notice) string {
	s.notices = append(s.notices, n)
	return s.token
}
func (s *recordingSink) Flush() {}

func testTool() mcp.Tool {
	return mcp.NewTool("list_faults",
		mcp.WithDescription("test"),
		mcp.WithNumber("project_id", mcp.Description("project")),
		mcp.WithNumber("limit", mcp.Description("limit")),
	)
}

func testInstrumenter(sink analytics.Sink) *instrumenter {
	return newInstrumenter(sink, &config.Config{TransportMode: config.TransportHTTP}, "1.2.3")
}

func callWrapped(t *testing.T, sink *recordingSink, args map[string]any, h server.ToolHandlerFunc) analytics.Event {
	t.Helper()
	wrapped := testInstrumenter(sink).wrap(testTool(), h)
	req := mcp.CallToolRequest{}
	req.Params.Name = "list_faults"
	req.Params.Arguments = args
	ctx := WithClaims(context.Background(), &Claims{
		Scopes:        []string{"read", "write"},
		Subject:       "user_abc",
		AccountID:     "acct_xyz",
		ClientID:      "app-42",
		ProjectScoped: true,
	})
	_, _ = wrapped(ctx, req)
	if len(sink.events) != 1 {
		t.Fatalf("got %d events, want 1", len(sink.events))
	}
	return sink.events[0]
}

func TestWrap_OKOutcomeAndIdentity(t *testing.T) {
	sink := &recordingSink{}
	ev := callWrapped(t, sink, map[string]any{"project_id": float64(1)},
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("ok"), nil
		})

	if ev.Type != "mcp.tool_call" {
		t.Errorf("Type = %q, want mcp.tool_call", ev.Type)
	}
	if ev.Data["outcome"] != "ok" {
		t.Errorf("outcome = %v, want ok", ev.Data["outcome"])
	}
	if ev.Data["tool"] != "list_faults" {
		t.Errorf("tool = %v, want list_faults", ev.Data["tool"])
	}
	if ev.Data["user_id"] != "user_abc" {
		t.Errorf("user_id = %v, want user_abc", ev.Data["user_id"])
	}
	if ev.Data["account_id"] != "acct_xyz" {
		t.Errorf("account_id = %v, want acct_xyz", ev.Data["account_id"])
	}
	if ev.Data["client_id"] != "app-42" {
		t.Errorf("client_id = %v, want app-42", ev.Data["client_id"])
	}
	if ev.Data["project_scoped"] != true {
		t.Errorf("project_scoped = %v, want true", ev.Data["project_scoped"])
	}
	if ev.Data["server_version"] != "1.2.3" {
		t.Errorf("server_version = %v, want 1.2.3", ev.Data["server_version"])
	}
	if len(sink.notices) != 0 {
		t.Errorf("got %d notices on the ok path, want 0", len(sink.notices))
	}
}

func TestWrap_ToolErrorOutcome(t *testing.T) {
	sink := &recordingSink{}
	ev := callWrapped(t, sink, nil,
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultError("project_id is required"), nil
		})

	if ev.Data["outcome"] != "tool_error" {
		t.Errorf("outcome = %v, want tool_error", ev.Data["outcome"])
	}
	if len(sink.notices) != 0 {
		t.Errorf("tool_error must not Notify; got %d notices", len(sink.notices))
	}
}

func TestWrap_InternalErrorOnReturnedError(t *testing.T) {
	sink := &recordingSink{token: "tok-123"}
	ev := callWrapped(t, sink, nil,
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return nil, errors.New("boom")
		})

	if ev.Data["outcome"] != "internal_error" {
		t.Errorf("outcome = %v, want internal_error", ev.Data["outcome"])
	}
	if ev.Data["error_id"] != "tok-123" {
		t.Errorf("error_id = %v, want tok-123", ev.Data["error_id"])
	}
	if len(sink.notices) != 1 {
		t.Fatalf("got %d notices, want 1", len(sink.notices))
	}
	if sink.notices[0].Fingerprint != "mcp_tool_call:list_faults" {
		t.Errorf("Fingerprint = %q, want mcp_tool_call:list_faults", sink.notices[0].Fingerprint)
	}
}

// A 5xx from the API is an outage, even though the handler reports it as a
// tool result error with a nil Go error.
func TestWrap_UpstreamServerErrorIsInternalError(t *testing.T) {
	sink := &recordingSink{token: "tok-500"}
	ev := callWrapped(t, sink, nil,
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if rec := upstreamRecordFromContext(ctx); rec != nil {
				rec.record(http.StatusInternalServerError, false, 3*time.Millisecond)
			}
			return mcp.NewToolResultError("Failed to list faults: HTTP 500"), nil
		})

	if ev.Data["outcome"] != "internal_error" {
		t.Errorf("outcome = %v, want internal_error", ev.Data["outcome"])
	}
	if ev.Data["upstream_status"] != 500 {
		t.Errorf("upstream_status = %v, want 500", ev.Data["upstream_status"])
	}
	if len(sink.notices) != 1 {
		t.Errorf("got %d notices, want 1", len(sink.notices))
	}
}

// A 4xx is attributable to the call, not to an outage.
func TestWrap_UpstreamClientErrorIsToolError(t *testing.T) {
	sink := &recordingSink{}
	ev := callWrapped(t, sink, nil,
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if rec := upstreamRecordFromContext(ctx); rec != nil {
				rec.record(http.StatusNotFound, false, time.Millisecond)
			}
			return mcp.NewToolResultError("Failed: HTTP 404"), nil
		})

	if ev.Data["outcome"] != "tool_error" {
		t.Errorf("outcome = %v, want tool_error", ev.Data["outcome"])
	}
	if len(sink.notices) != 0 {
		t.Errorf("4xx must not Notify; got %d notices", len(sink.notices))
	}
}

func TestWrap_ArgNamesAllowlistedAndSorted(t *testing.T) {
	sink := &recordingSink{}
	ev := callWrapped(t, sink, map[string]any{
		"project_id":           float64(1),
		"limit":                float64(10),
		"customer@example.com": "injected",
		"another_undeclared":   true,
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	})

	names, ok := ev.Data["arg_names"].([]string)
	if !ok {
		t.Fatalf("arg_names is %T, want []string", ev.Data["arg_names"])
	}
	if len(names) != 2 || names[0] != "limit" || names[1] != "project_id" {
		t.Errorf("arg_names = %v, want [limit project_id]", names)
	}
	if ev.Data["unknown_arg_count"] != 2 {
		t.Errorf("unknown_arg_count = %v, want 2", ev.Data["unknown_arg_count"])
	}
	for _, n := range names {
		if n == "customer@example.com" {
			t.Fatal("undeclared argument name leaked into the event")
		}
	}
}

func TestWrap_PanicIsRecordedAndRepanics(t *testing.T) {
	sink := &recordingSink{token: "tok-panic"}
	wrapped := testInstrumenter(sink).wrap(testTool(),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			panic("kaboom")
		})

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("panic did not propagate; WithRecovery would never see it")
			}
		}()
		req := mcp.CallToolRequest{}
		req.Params.Name = "list_faults"
		_, _ = wrapped(context.Background(), req)
	}()

	if len(sink.events) != 1 {
		t.Fatalf("got %d events, want 1", len(sink.events))
	}
	if sink.events[0].Data["outcome"] != "panic" {
		t.Errorf("outcome = %v, want panic", sink.events[0].Data["outcome"])
	}
	if len(sink.notices) != 1 {
		t.Errorf("panic must Notify; got %d notices", len(sink.notices))
	}
}

func TestWrap_ResultAndErrorPassThroughUnchanged(t *testing.T) {
	sink := &recordingSink{}
	want := mcp.NewToolResultText("payload")
	wantErr := errors.New("boom")
	wrapped := testInstrumenter(sink).wrap(testTool(),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return want, wantErr
		})

	req := mcp.CallToolRequest{}
	req.Params.Name = "list_faults"
	got, err := wrapped(context.Background(), req)
	if got != want {
		t.Error("result was not passed through unchanged")
	}
	if !errors.Is(err, wantErr) {
		t.Error("error was not passed through unchanged")
	}
}
