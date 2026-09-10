package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"unicode"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestListFaultCommentsEmpty(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer api.Close()
	client := hbapi.NewClient().WithBaseURL(api.URL).WithAuthToken("test-token")
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"project_id": 123,
		"fault_id":   456,
	}}}

	result, err := handleListFaultComments(context.Background(), client, req)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError || getResultText(result) != "[]" {
		t.Fatalf("expected an empty JSON array, got %+v", result)
	}
}

func TestFaultCommentTools(t *testing.T) {
	const body = "  Investigated this fault.\nSee **details**.  "
	cases := []struct {
		name      string
		handler   func(context.Context, *hbapi.Client, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		method    string
		commentID bool
		hasBody   bool
		response  string
		status    int
		want      string
	}{
		{"list", handleListFaultComments, "GET", false, false, `{"results":[{"id":789,"body":"Investigation"}]}`, 200, "Investigation"},
		{"get", handleGetFaultComment, "GET", true, false, `{"id":789,"body":"Investigation"}`, 200, "Investigation"},
		{"create", handleCreateFaultComment, "POST", false, true, `{"id":789,"body":"Investigation"}`, 201, "Investigation"},
		{"update", handleUpdateFaultComment, "PUT", true, true, "", 204, "updated successfully"},
		{"delete", handleDeleteFaultComment, "DELETE", true, false, "", 204, "deleted successfully"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{"project_id": 123, "fault_id": 456}
			if tc.commentID {
				args["comment_id"] = 789
			}
			if tc.hasBody {
				args["body"] = body
			}
			path := "/v2/projects/123/faults/456/comments"
			if tc.commentID {
				path += "/789"
			}
			for _, fail := range []bool{false, true} {
				t.Run(map[bool]string{false: "success", true: "api_error"}[fail], func(t *testing.T) {
					calls := 0
					api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls++
						if r.Method != tc.method || r.URL.Path != path {
							t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, tc.method, path)
						}
						if tc.hasBody {
							var payload struct {
								Comment struct {
									Body string `json:"body"`
								} `json:"comment"`
							}
							if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
								t.Errorf("decode request: %v", err)
							}
							if payload.Comment.Body != body {
								t.Errorf("body = %q, want %q", payload.Comment.Body, body)
							}
						}
						w.Header().Set("Content-Type", "application/json")
						if fail {
							w.WriteHeader(http.StatusForbidden)
							_, _ = w.Write([]byte(`{"error":"Forbidden"}`))
							return
						}
						w.WriteHeader(tc.status)
						_, _ = w.Write([]byte(tc.response))
					}))
					defer api.Close()
					client := hbapi.NewClient().WithBaseURL(api.URL).WithAuthToken("test-token")
					result, err := tc.handler(context.Background(), client, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
					if err != nil {
						t.Fatal(err)
					}
					if calls != 1 || result.IsError != fail {
						t.Fatalf("calls = %d, result = %+v", calls, result)
					}
					want := tc.want
					if fail {
						want = "Failed to " + tc.name + " fault comment"
					}
					if !strings.Contains(getResultText(result), want) {
						t.Errorf("result = %s, want %q", getResultText(result), want)
					}
				})
			}
			fields := []string{"project_id", "fault_id"}
			if tc.commentID {
				fields = append(fields, "comment_id")
			}
			if tc.hasBody {
				fields = append(fields, "body")
			}
			for _, field := range fields {
				invalid := []any{nil, 0, -1, 1.5, "123", true, float64(maxSafeInteger * 2)}
				if field == "body" {
					invalid = []any{nil, "", " \n\t ", 123, true}
				}
				for _, value := range invalid {
					t.Run("invalid_"+field, func(t *testing.T) {
						badArgs := make(map[string]any, len(args))
						for k, v := range args {
							badArgs[k] = v
						}
						if value == nil {
							delete(badArgs, field)
						} else {
							badArgs[field] = value
						}
						// A nil client ensures invalid requests cannot reach the API.
						result, err := tc.handler(context.Background(), nil, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: badArgs}})
						if err != nil || !result.IsError || !strings.Contains(getResultText(result), field) {
							t.Fatalf("result = %+v, err = %v", result, err)
						}
					})
				}
			}
		})
	}
}

func TestFaultCommentBodySchema(t *testing.T) {
	s := server.NewMCPServer("test", "test")
	RegisterCommentTools(newToolRegistrar(s), nil)
	for _, name := range []string{"create_fault_comment", "update_fault_comment"} {
		t.Run(name, func(t *testing.T) {
			tool := s.GetTool(name)
			if tool == nil {
				t.Fatal("tool not registered")
			}
			prop := tool.Tool.InputSchema.Properties["body"].(map[string]any)
			pattern, ok := prop["pattern"].(string)
			if !ok {
				t.Fatal("body schema must advertise a non-blank pattern")
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				t.Fatal(err)
			}
			cases := []struct {
				body  string
				valid bool
			}{
				{"", false},
				{" \t\n\r ", false},
				{"\u00a0\u2003\u202f\u3000", false},
				{"Comment", true},
				{"  First line\nSecond line  ", true},
				{"\u00a0Hello\u3000", true},
				{"\u200b", true},
				{"\ufeff", true},
			}
			for _, tc := range cases {
				if got := re.MatchString(tc.body); got != tc.valid {
					t.Errorf("body %q: pattern accepts = %v, want %v", tc.body, got, tc.valid)
				}
			}
			// Cover every whitespace rune recognized by the handler.
			for r := rune(0); r <= unicode.MaxRune; r++ {
				if unicode.IsSpace(r) && re.MatchString(string(r)) {
					t.Errorf("pattern accepts whitespace U+%04X", r)
				}
			}
		})
	}
}
