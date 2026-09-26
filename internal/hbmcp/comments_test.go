package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const testComment = `{"id":"cmt1","fault_id":456,"body":"Investigation","created_at":"2026-09-26T00:00:00Z","author":{"name":"Kevin"}}`

func TestListFaultCommentsEmpty(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusOK, `{"data":[],"pagination":{"has_older":false,"limit":25},"links":{"self":"/x"}}`)
	})
	result, err := handleListFaultComments(context.Background(), client, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"project_id": "Xk9mZp",
		"fault_id":   456,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError || getResultText(result) != "[]" {
		t.Fatalf("expected an empty JSON array, got %+v", result)
	}
}

// list_fault_comments returns every comment, not just the first page.
func TestListFaultCommentsWalksEveryPage(t *testing.T) {
	var calls int
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path == "/v3/older" {
			v3JSON(w, http.StatusOK, `{"data":[{"id":"cmt0","fault_id":456,"created_at":"2026-09-25T00:00:00Z"}],
			  "pagination":{"has_older":false,"limit":1},"links":{"self":"/v3/older"}}`)
			return
		}
		v3JSON(w, http.StatusOK, `{"data":[`+testComment+`],
		  "pagination":{"has_older":true,"limit":1},"links":{"self":"/x","older":"/v3/older"}}`)
	})
	result, err := handleListFaultComments(context.Background(), client, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"project_id": "Xk9mZp", "fault_id": 456,
	}}})
	if err != nil || result.IsError {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
	var got []apiv3.Comment
	if err := json.Unmarshal([]byte(getResultText(result)), &got); err != nil {
		t.Fatalf("result is not a JSON array of comments: %v", err)
	}
	if len(got) != 2 || calls != 2 {
		t.Errorf("got %d comments over %d requests, want 2 over 2", len(got), calls)
	}
}

func TestFaultCommentTools(t *testing.T) {
	const body = "  Investigated this fault.\nSee **details**.  "
	cases := []struct {
		name      string
		handler   func(context.Context, *apiv3.Client, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		method    string
		commentID bool
		hasBody   bool
		response  string
		status    int
		want      string
	}{
		{"get", handleGetFaultComment, "GET", true, false, `{"data":` + testComment + `}`, 200, "Investigation"},
		{"create", handleCreateFaultComment, "POST", false, true, `{"data":` + testComment + `}`, 201, "Investigation"},
		// v3 answers an update with the comment as stored, so the tool returns it.
		{"update", handleUpdateFaultComment, "PATCH", true, true, `{"data":` + testComment + `}`, 200, "Investigation"},
		{"delete", handleDeleteFaultComment, "DELETE", true, false, "", 204, "deleted successfully"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{"project_id": "Xk9mZp", "fault_id": 456}
			if tc.commentID {
				args["comment_id"] = "cmt1"
			}
			if tc.hasBody {
				args["body"] = body
			}
			if tc.name == "delete" {
				args["confirm"] = validConfirm("delete_fault_comment", "Xk9mZp", 456, "cmt1")
			}
			path := "/v3/projects/Xk9mZp/faults/456/comments"
			if tc.commentID {
				path += "/cmt1"
			}
			for _, fail := range []bool{false, true} {
				t.Run(map[bool]string{false: "success", true: "api_error"}[fail], func(t *testing.T) {
					calls := 0
					client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
						calls++
						if r.Method != tc.method || r.URL.Path != path {
							t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, tc.method, path)
						}
						if tc.hasBody {
							var payload struct {
								Body string `json:"body"`
							}
							if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
								t.Errorf("decode request: %v", err)
							}
							if payload.Body != body {
								t.Errorf("body = %q, want %q", payload.Body, body)
							}
						}
						if fail {
							v3JSON(w, http.StatusForbidden, `{"error":{"code":"access_denied","message":"Forbidden"}}`)
							return
						}
						if tc.response == "" {
							w.WriteHeader(tc.status)
							return
						}
						v3JSON(w, tc.status, tc.response)
					})
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

			invalid := map[string][]any{
				"project_id": {nil, ""},
				"fault_id":   {nil, 0, -1, 1.5, "123", true, float64(maxSafeInteger * 2)},
			}
			if tc.commentID {
				invalid["comment_id"] = []any{nil, ""}
			}
			if tc.hasBody {
				invalid["body"] = []any{nil, "", " \n\t ", 123, true}
			}
			for field, values := range invalid {
				for _, value := range values {
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
