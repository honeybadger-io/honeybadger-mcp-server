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

	// readStatus and readBody, when set, answer every read with that error; an
	// insufficient_scope 403 is what a write-only API token gets.
	readStatus int
	readBody   string
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
		if f.readStatus != 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(f.readStatus)
			_, _ = w.Write([]byte(f.readBody))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// One shape that decodes as any of the resources a delete previews.
		thing := `{"id":"9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47","project_id":"Xk9mZp","fault_id":"456","name":"Thing","title":"Thing","label":"Thing","type":"Thing","key":"k","active":true,"author":{"name":"Thing"},"body":"Looked into it","created_at":"2026-01-01T00:00:00Z"}`
		_, _ = w.Write([]byte(`{"data":` + thing + `}`))
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

// deleteToolArgs holds valid arguments for every delete tool. A new delete tool
// fails TestDeleteToolsRequireConfirmation until it is added here.
var deleteToolArgs = map[string]map[string]any{
	"delete_project":       {"id": "Xk9mZp"},
	"delete_dashboard":     {"project_id": "Xk9mZp", "dashboard_id": "dash1"},
	"delete_alarm":         {"project_id": "Xk9mZp", "alarm_id": "alarm1"},
	"delete_check_in":      {"project_id": "Xk9mZp", "check_in_id": "chk1"},
	"delete_site":          {"project_id": "Xk9mZp", "site_id": "9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47"},
	"delete_fault_comment": {"project_id": "Xk9mZp", "fault_id": 456, "comment_id": "cmt1"},
	"delete_integration":   {"project_id": "Xk9mZp", "integration_id": "int1"},
	"delete_project_key":   {"project_id": "Xk9mZp", "key_id": "key1"},
}

// Every delete_* tool must go through deletionConfirmed/deletionPreview.
func TestDeleteToolsRequireConfirmation(t *testing.T) {
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
	const secret = "0123456789abcdef0123456789abcdef"
	s := NewServer(&config.Config{
		APIURL:        api.URL,
		LogLevel:      "error",
		TransportMode: config.TransportHTTP,
		ConfirmSecret: secret,
	}, "test")
	caller := func(bearer, subject string) context.Context {
		ctx := WithAuthToken(context.Background(), bearer)
		return WithClaims(ctx, &Claims{Subject: subject, Scopes: []string{"write"}})
	}
	args := map[string]any{"project_id": "Xk9mZp", "check_in_id": "chk1"}
	ids := []any{"Xk9mZp", "chk1"}
	deletesSince := func(before int) int {
		_, d := api.counts()
		return d - before
	}

	t.Run("client-minted token", func(t *testing.T) {
		_, before := api.counts()
		forged := confirmtoken.New([]byte("alice-bearer")).Mint("sub:alice", "delete_check_in", ids, time.Now())
		if text, _ := callTool(t, s, caller("alice-bearer", "alice"), "delete_check_in", withArg(args, "confirm", forged)); deletesSince(before) != 0 {
			t.Fatalf("token signed with the caller's bearer deleted: %q", text)
		}
	})

	t.Run("per-process key", func(t *testing.T) {
		_, before := api.counts()
		stdio := processSigner.Mint("sub:alice", "delete_check_in", ids, time.Now())
		if text, _ := callTool(t, s, caller("alice-bearer", "alice"), "delete_check_in", withArg(args, "confirm", stdio)); deletesSince(before) != 0 {
			t.Fatalf("http mode accepted a token not signed with MCP_CONFIRM_SECRET: %q", text)
		}
	})

	t.Run("minted by another replica", func(t *testing.T) {
		_, before := api.counts()
		replica := confirmtoken.New([]byte(secret)).Mint("sub:alice", "delete_check_in", ids, time.Now())
		text, isErr := callTool(t, s, caller("alice-bearer", "alice"), "delete_check_in", withArg(args, "confirm", replica))
		if isErr || deletesSince(before) != 1 {
			t.Fatalf("token signed with the shared secret rejected: %q", text)
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

// An opaque token (hbt_, hba_) has no subject, so a confirmation is bound to the
// token itself: another caller's token cannot use it, the same token can.
func TestHTTPDeleteConfirmationBindsOpaqueTokens(t *testing.T) {
	api := newFakeAPI(t)
	s := NewServer(&config.Config{
		APIURL:        api.URL,
		LogLevel:      "error",
		TransportMode: config.TransportHTTP,
		ConfirmSecret: "0123456789abcdef0123456789abcdef",
	}, "test")
	opaque := func(bearer string) context.Context {
		ctx := WithAuthToken(context.Background(), bearer)
		return WithCredentialKind(ctx, ClassifyCredential(bearer))
	}
	args := map[string]any{"project_id": "Xk9mZp", "check_in_id": "chk1"}

	text, _ := callTool(t, s, opaque("hbt_alice"), "delete_check_in", args)
	m := confirmTokenPattern.FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("preview has no token: %q", text)
	}

	_, before := api.counts()
	if text, _ := callTool(t, s, opaque("hbt_bob"), "delete_check_in", withArg(args, "confirm", m[1])); func() int { _, d := api.counts(); return d - before }() != 0 {
		t.Fatalf("alice's confirmation deleted for bob: %q", text)
	}

	text, isErr := callTool(t, s, opaque("hbt_alice"), "delete_check_in", withArg(args, "confirm", m[1]))
	if _, after := api.counts(); isErr || after-before != 1 {
		t.Fatalf("alice's own confirmation was refused: %q", text)
	}
}

// A write-only API token may delete but not read, so the preview's lookup is
// refused. The preview must still be issued, naming the resource by id, and the
// confirmed call must still delete. Before, such a token was offered every delete
// tool and could never use one.
func TestDeleteToolsWorkForWriteOnlyTokens(t *testing.T) {
	api := newFakeAPI(t)
	api.readStatus = http.StatusForbidden
	api.readBody = `{"error":{"code":"insufficient_scope","message":"Insufficient scope","details":{"required_scope":"x:read"}}}`
	s := NewServer(&config.Config{
		AuthToken:     "test-token",
		APIURL:        api.URL,
		LogLevel:      "error",
		TransportMode: config.TransportStdio,
	}, "test")
	ctx := context.Background()

	for tool, args := range deleteToolArgs {
		t.Run(tool, func(t *testing.T) {
			_, deletes0 := api.counts()
			text, isErr := callTool(t, s, ctx, tool, args)
			if isErr || !strings.HasPrefix(text, "Not deleted.") || !strings.Contains(text, "not allowed to read") {
				t.Fatalf("a refused read must still preview by id; isError=%v text=%q", isErr, text)
			}
			m := confirmTokenPattern.FindStringSubmatch(text)
			if m == nil {
				t.Fatalf("preview has no token: %q", text)
			}
			text, isErr = callTool(t, s, ctx, tool, withArg(args, "confirm", m[1]))
			if _, deletes := api.counts(); isErr || deletes != deletes0+1 {
				t.Fatalf("confirmed call must delete once; isError=%v text=%q", isErr, text)
			}
		})
	}
}

// Only insufficient_scope degrades a preview. A lookup that finds nothing, or is
// refused for any other reason, must still fail with no confirmation token, or a
// preview would mint one for a resource that does not exist or is off-limits.
func TestDeletePreviewStillFailsOnOtherReadErrors(t *testing.T) {
	for name, fail := range map[string]struct {
		status int
		body   string
	}{
		"not found":     {http.StatusNotFound, `{"error":{"code":"not_found","message":"Resource not found"}}`},
		"access denied": {http.StatusForbidden, `{"error":{"code":"access_denied","message":"Access denied"}}`},
	} {
		t.Run(name, func(t *testing.T) {
			api := newFakeAPI(t)
			api.readStatus, api.readBody = fail.status, fail.body
			s := NewServer(&config.Config{
				AuthToken:     "test-token",
				APIURL:        api.URL,
				LogLevel:      "error",
				TransportMode: config.TransportStdio,
			}, "test")

			for tool, args := range deleteToolArgs {
				text, isErr := callTool(t, s, context.Background(), tool, args)
				if !isErr || !strings.Contains(text, "Failed to look up") {
					t.Errorf("%s: a %s lookup must fail the preview; isError=%v text=%q", tool, name, isErr, text)
				}
				if confirmTokenPattern.MatchString(text) {
					t.Errorf("%s: issued a confirmation token after a %s lookup: %q", tool, name, text)
				}
			}
			if _, deletes := api.counts(); deletes != 0 {
				t.Errorf("deleted %d resources", deletes)
			}
		})
	}
}
