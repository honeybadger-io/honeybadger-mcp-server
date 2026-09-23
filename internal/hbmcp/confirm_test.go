package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/confirmtoken"
	"github.com/mark3labs/mcp-go/server"
)

// validConfirm mints the token a stdio caller would receive from a preview.
func validConfirm(tool string, ids ...any) string {
	return processSigner.Mint("", tool, ids, time.Now())
}

var confirmTokenPattern = regexp.MustCompile(`confirm set to "([^"]+)"`)

// fakeAPI answers every GET with a resource named "Thing" and counts requests.
type fakeAPI struct {
	*httptest.Server
	mu      sync.Mutex
	gets    int
	deletes int
}

func newFakeAPI(t *testing.T) *fakeAPI {
	f := &fakeAPI{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Method == http.MethodDelete {
			f.deletes++
			w.WriteHeader(http.StatusNoContent)
			return
		}
		f.gets++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"Thing","title":"Thing","author":"Thing","body":"Looked into it"}`))
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeAPI) counts() (gets, deletes int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.gets, f.deletes
}

func rpc(t *testing.T, s *server.MCPServer, ctx context.Context, method string, params, out any) {
	t.Helper()
	msg, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	resp, _ := json.Marshal(s.HandleMessage(ctx, msg))
	if err := json.Unmarshal(resp, out); err != nil {
		t.Fatalf("%s: %v", method, err)
	}
}

func callTool(t *testing.T, s *server.MCPServer, ctx context.Context, name string, args map[string]any) (text string, isError bool) {
	t.Helper()
	var out struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	rpc(t, s, ctx, "tools/call", map[string]any{"name": name, "arguments": args}, &out)
	if len(out.Result.Content) == 0 {
		t.Fatalf("%s returned no content", name)
	}
	return out.Result.Content[0].Text, out.Result.IsError
}

func withArg(args map[string]any, key string, value any) map[string]any {
	out := make(map[string]any, len(args)+1)
	for k, v := range args {
		out[k] = v
	}
	out[key] = value
	return out
}

// Every delete_* tool must go through deletionConfirmed/deletionPreview. A new
// delete tool fails here until it is added to deleteToolArgs.
func TestDeleteToolsRequireConfirmation(t *testing.T) {
	deleteToolArgs := map[string]map[string]any{
		"delete_project":       {"id": 123},
		"delete_dashboard":     {"project_id": 123, "dashboard_id": "dash1"},
		"delete_alarm":         {"project_id": 123, "alarm_id": "alarm1"},
		"delete_check_in":      {"project_id": 123, "check_in_id": "chk1"},
		"delete_fault_comment": {"project_id": 123, "fault_id": 456, "comment_id": 789},
	}

	api := newFakeAPI(t)
	s := NewServer(&config.Config{
		AuthToken:     "test-token",
		APIURL:        api.URL,
		LogLevel:      "error",
		TransportMode: config.TransportStdio,
	}, "test")
	ctx := context.Background()

	var list struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				InputSchema struct {
					Properties map[string]any `json:"properties"`
				} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	rpc(t, s, ctx, "tools/list", map[string]any{}, &list)

	seen := 0
	for _, tool := range list.Result.Tools {
		if !strings.HasPrefix(tool.Name, "delete_") {
			continue
		}
		seen++
		t.Run(tool.Name, func(t *testing.T) {
			if _, ok := tool.InputSchema.Properties["confirm"]; !ok {
				t.Fatal("missing confirm parameter; declare withConfirmParam()")
			}
			args, ok := deleteToolArgs[tool.Name]
			if !ok {
				t.Fatal("add this tool to deleteToolArgs")
			}
			gets0, deletes0 := api.counts()

			// A truncated 123.9 would preview and delete a different resource.
			for key, value := range args {
				if _, numeric := value.(int); !numeric {
					continue
				}
				text, isErr := callTool(t, s, ctx, tool.Name, withArg(args, key, 123.9))
				if gets, deletes := api.counts(); !isErr || gets != gets0 || deletes != deletes0 {
					t.Fatalf("fractional %s must be rejected before any API call; isError=%v text=%q", key, isErr, text)
				}
			}

			text, isErr := callTool(t, s, ctx, tool.Name, args)
			gets, deletes := api.counts()
			if isErr || !strings.HasPrefix(text, "Not deleted.") || deletes != deletes0 {
				t.Fatalf("unconfirmed call must preview without deleting; isError=%v deletes=%d text=%q", isErr, deletes-deletes0, text)
			}
			if gets != gets0+1 || !strings.Contains(text, "Thing") {
				t.Fatalf("preview must describe the looked-up resource; gets=%d text=%q", gets-gets0, text)
			}
			m := confirmTokenPattern.FindStringSubmatch(text)
			if m == nil {
				t.Fatalf("preview has no token: %q", text)
			}

			text, _ = callTool(t, s, ctx, tool.Name, withArg(args, "confirm", "bogus"))
			if _, deletes = api.counts(); !strings.Contains(text, "invalid or expired") || deletes != deletes0 {
				t.Fatalf("bad token must not delete; deletes=%d text=%q", deletes-deletes0, text)
			}

			gets0, _ = api.counts()
			text, isErr = callTool(t, s, ctx, tool.Name, withArg(args, "confirm", m[1]))
			gets, deletes = api.counts()
			if isErr || deletes != deletes0+1 || gets != gets0 {
				t.Fatalf("confirmed call must delete once without a lookup; isError=%v deletes=%d gets=%d text=%q", isErr, deletes-deletes0, gets-gets0, text)
			}
		})
	}
	if seen != len(deleteToolArgs) {
		t.Errorf("found %d delete_* tools, deleteToolArgs lists %d", seen, len(deleteToolArgs))
	}
}

func TestHTTPDeleteConfirmation(t *testing.T) {
	api := newFakeAPI(t)
	s := NewServer(&config.Config{
		APIURL:        api.URL,
		LogLevel:      "error",
		TransportMode: config.TransportHTTP,
		ConfirmSecret: "0123456789abcdef0123456789abcdef",
	}, "test")
	caller := func(bearer, subject string) context.Context {
		ctx := WithAuthToken(context.Background(), bearer)
		return WithClaims(ctx, &Claims{Subject: subject, Scopes: []string{"write"}})
	}
	args := map[string]any{"project_id": 123, "check_in_id": "chk1"}
	ids := []any{123, "chk1"}
	deletesSince := func(before int) int {
		_, d := api.counts()
		return d - before
	}

	t.Run("client-minted token", func(t *testing.T) {
		_, before := api.counts()
		forged := confirmtoken.New([]byte("alice-bearer")).Mint("alice", "delete_check_in", ids, time.Now())
		if text, _ := callTool(t, s, caller("alice-bearer", "alice"), "delete_check_in", withArg(args, "confirm", forged)); deletesSince(before) != 0 {
			t.Fatalf("token signed with the caller's bearer deleted: %q", text)
		}
	})

	text, _ := callTool(t, s, caller("alice-bearer", "alice"), "delete_check_in", args)
	m := confirmTokenPattern.FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("preview has no token: %q", text)
	}

	t.Run("other caller", func(t *testing.T) {
		_, before := api.counts()
		if text, _ := callTool(t, s, caller("bob-bearer", "bob"), "delete_check_in", withArg(args, "confirm", m[1])); deletesSince(before) != 0 {
			t.Fatalf("alice's token deleted for bob: %q", text)
		}
	})

	t.Run("refreshed bearer", func(t *testing.T) {
		_, before := api.counts()
		text, isErr := callTool(t, s, caller("alice-refreshed-bearer", "alice"), "delete_check_in", withArg(args, "confirm", m[1]))
		if isErr || deletesSince(before) != 1 {
			t.Fatalf("token rejected after alice's bearer refreshed: %q", text)
		}
	})
}
