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
)

// validConfirm mints the token a stdio caller would receive from a preview.
func validConfirm(tool string, ids ...any) string {
	return mintConfirmToken(processConfirmKey, tool, ids, time.Now().Add(confirmTTL))
}

func TestConfirmToken(t *testing.T) {
	key := []byte("caller-token")
	now := time.Now()
	good := mintConfirmToken(key, "delete_project", []any{123}, now.Add(confirmTTL))

	if !validConfirmToken(key, good, "delete_project", []any{123}, now) {
		t.Fatal("freshly minted token rejected")
	}

	exp, mac, _ := strings.Cut(good, ".")
	rejects := map[string]struct {
		key   []byte
		token string
		tool  string
		ids   []any
		now   time.Time
	}{
		"expired":       {key, good, "delete_project", []any{123}, now.Add(confirmTTL + time.Second)},
		"other tool":    {key, good, "delete_dashboard", []any{123}, now},
		"other id":      {key, good, "delete_project", []any{124}, now},
		"other caller":  {[]byte("someone-else"), good, "delete_project", []any{123}, now},
		"extended exp":  {key, "zzzzzzz." + mac, "delete_project", []any{123}, now},
		"tampered mac":  {key, exp + "." + strings.Repeat("A", len(mac)), "delete_project", []any{123}, now},
		"no separator":  {key, exp + mac, "delete_project", []any{123}, now},
		"garbage":       {key, "not-a-token", "delete_project", []any{123}, now},
		"id split move": {key, mintConfirmToken(key, "delete_fault_comment", []any{1, 23, 4}, now.Add(time.Minute)), "delete_fault_comment", []any{12, 3, 4}, now},
	}
	for name, tc := range rejects {
		t.Run(name, func(t *testing.T) {
			if validConfirmToken(tc.key, tc.token, tc.tool, tc.ids, tc.now) {
				t.Error("token accepted")
			}
		})
	}
}

func TestConfirmKeyBindsHTTPCaller(t *testing.T) {
	alice := WithAuthToken(context.Background(), "alice-token")
	bob := WithAuthToken(context.Background(), "bob-token")
	token := mintConfirmToken(confirmKey(alice), "delete_project", []any{1}, time.Now().Add(time.Minute))

	if !validConfirmToken(confirmKey(alice), token, "delete_project", []any{1}, time.Now()) {
		t.Fatal("token rejected for the caller it was minted for")
	}
	if validConfirmToken(confirmKey(bob), token, "delete_project", []any{1}, time.Now()) {
		t.Error("token accepted for a different caller")
	}
	if validConfirmToken(confirmKey(context.Background()), token, "delete_project", []any{1}, time.Now()) {
		t.Error("http token accepted by the stdio key")
	}
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

	var mu sync.Mutex
	deletes := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			mu.Lock()
			deletes++
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"Thing","title":"Thing","author":"Ann","body":"Looked into it"}`))
	}))
	defer api.Close()

	s := NewServer(&config.Config{
		AuthToken:     "test-token",
		APIURL:        api.URL,
		LogLevel:      "error",
		TransportMode: config.TransportStdio,
	}, "test")

	rpc := func(t *testing.T, method string, params any, out any) {
		t.Helper()
		msg, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
		resp, _ := json.Marshal(s.HandleMessage(context.Background(), msg))
		if err := json.Unmarshal(resp, out); err != nil {
			t.Fatalf("%s: %v", method, err)
		}
	}
	type callResult struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	call := func(t *testing.T, name string, args map[string]any) (string, bool) {
		t.Helper()
		var out callResult
		rpc(t, "tools/call", map[string]any{"name": name, "arguments": args}, &out)
		if len(out.Result.Content) == 0 {
			t.Fatalf("%s returned no content", name)
		}
		return out.Result.Content[0].Text, out.Result.IsError
	}
	deleteCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return deletes
	}
	tokenPattern := regexp.MustCompile(`confirm set to "([^"]+)"`)

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
	rpc(t, "tools/list", map[string]any{}, &list)

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
			before := deleteCount()

			text, isErr := call(t, tool.Name, args)
			if isErr || !strings.HasPrefix(text, "Not deleted.") || deleteCount() != before {
				t.Fatalf("unconfirmed call must preview without deleting; isError=%v deletes=%d text=%q", isErr, deleteCount()-before, text)
			}
			m := tokenPattern.FindStringSubmatch(text)
			if m == nil {
				t.Fatalf("preview has no token: %q", text)
			}

			bad := map[string]any{"confirm": "bogus"}
			for k, v := range args {
				bad[k] = v
			}
			text, _ = call(t, tool.Name, bad)
			if !strings.Contains(text, "invalid or expired") || deleteCount() != before {
				t.Fatalf("bad token must not delete; deletes=%d text=%q", deleteCount()-before, text)
			}

			good := map[string]any{"confirm": m[1]}
			for k, v := range args {
				good[k] = v
			}
			text, isErr = call(t, tool.Name, good)
			if isErr || deleteCount() != before+1 {
				t.Fatalf("confirmed call must delete once; isError=%v deletes=%d text=%q", isErr, deleteCount()-before, text)
			}
		})
	}
	if seen != len(deleteToolArgs) {
		t.Errorf("found %d delete_* tools, deleteToolArgs lists %d", seen, len(deleteToolArgs))
	}
}
