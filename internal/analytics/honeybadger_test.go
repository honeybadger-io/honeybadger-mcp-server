package analytics

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	honeybadger "github.com/honeybadger-io/honeybadger-go"
)

// Configuration.Backend cannot be faked from this package — Backend.Event
// takes []*eventPayload, an unexported type — so these tests point Endpoint at
// an httptest server and use Sync for determinism.
func TestHoneybadgerSink_EmitPostsEvent(t *testing.T) {
	type captured struct {
		path string
		body []byte
	}
	got := make(chan captured, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- captured{path: r.URL.Path, body: b}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	sink := newHoneybadgerSink(honeybadger.Configuration{
		APIKey:   "test-key",
		Env:      "test",
		Endpoint: srv.URL,
		Sync:     true,
	})

	sink.Emit(Event{Type: "mcp.tool_call", Data: map[string]any{
		"tool":    "list_faults",
		"outcome": "ok",
	}})

	// Bounded: an unbuffered receive would hang until the whole test binary
	// times out if the request is never sent, hiding the real failure.
	var c captured
	select {
	case c = <-got:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the event request; none was sent")
	}
	if c.path != "/v1/events" {
		t.Errorf("path = %q, want /v1/events", c.path)
	}
	var payload map[string]any
	if err := json.Unmarshal(c.body, &payload); err != nil {
		t.Fatalf("body is not JSON: %v (body=%s)", err, c.body)
	}
	if payload["event_type"] != "mcp.tool_call" {
		t.Errorf("event_type = %v, want mcp.tool_call (payload=%v)", payload["event_type"], payload)
	}
	if payload["tool"] != "list_faults" {
		t.Errorf("tool = %v, want list_faults", payload["tool"])
	}
}

func TestHoneybadgerSink_NotifyReturnsToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	sink := newHoneybadgerSink(honeybadger.Configuration{
		APIKey:   "test-key",
		Env:      "test",
		Endpoint: srv.URL,
		Sync:     true,
	})

	token := sink.Notify(Notice{
		Err:         errors.New("boom"),
		Fingerprint: "mcp_tool_call:list_faults",
		Context:     map[string]any{"tool": "list_faults"},
	})
	if token == "" {
		t.Error("Notify returned an empty token, want a correlation token")
	}
}

func TestNopSink_NotifyReturnsEmpty(t *testing.T) {
	s := NewNopSink()
	s.Emit(Event{Type: "x"})
	s.Flush()
	if token := s.Notify(Notice{Err: errors.New("boom")}); token != "" {
		t.Errorf("token = %q, want empty", token)
	}
}

// The activation gate compares against NewNopSink(), so equality must hold.
func TestNopSink_ValuesCompareEqual(t *testing.T) {
	if NewNopSink() != NewNopSink() {
		t.Error("nop sinks do not compare equal; the activation gate check would break")
	}
}
