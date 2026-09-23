package hbmcp

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

const confirmTTL = 10 * time.Minute

const confirmNote = " Deleting takes two calls: the first deletes nothing and returns a preview with a confirmation token. Show the preview to the user, and only if they approve, call again with the same arguments and confirm set to that token."

// Keys stdio sessions, which carry no per-request token.
var processConfirmKey = func() []byte {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b
}()

func withConfirmParam() mcp.ToolOption {
	return mcp.WithString("confirm",
		mcp.Description("Confirmation token from this tool's previous preview. Pass it only after the user has approved the deletion shown in that preview."),
	)
}

// deletionConfirmed reports whether req carries a live token minted for this
// tool and these ids by the same caller.
func deletionConfirmed(ctx context.Context, req mcp.CallToolRequest, tool string, ids ...any) bool {
	token := req.GetString("confirm", "")
	return token != "" && validConfirmToken(confirmKey(ctx), token, tool, ids, time.Now())
}

// deletionPreview is what a delete handler returns instead of deleting;
// summary completes "This will permanently ...".
func deletionPreview(ctx context.Context, req mcp.CallToolRequest, tool, summary string, ids ...any) *mcp.CallToolResult {
	notice := ""
	if req.GetString("confirm", "") != "" {
		notice = "The confirm token was invalid or expired. "
	}
	token := mintConfirmToken(confirmKey(ctx), tool, ids, time.Now().Add(confirmTTL))
	return mcp.NewToolResultText(fmt.Sprintf(
		"%sNot deleted. This will permanently %s. Show this to the user and ask them to approve. Only if they approve, call %s again with the same arguments and confirm set to %q (valid for %d minutes).",
		notice, summary, tool, token, int(confirmTTL.Minutes())))
}

// In http mode the caller's bearer token keys the MAC, so any replica can
// verify it and one caller's token is useless to another.
func confirmKey(ctx context.Context) []byte {
	if token := AuthTokenFromContext(ctx); token != "" {
		return []byte(token)
	}
	return processConfirmKey
}

func mintConfirmToken(key []byte, tool string, ids []any, expires time.Time) string {
	exp := strconv.FormatInt(expires.Unix(), 36)
	return exp + "." + confirmMAC(key, tool, ids, exp)
}

func validConfirmToken(key []byte, token, tool string, ids []any, now time.Time) bool {
	exp, mac, ok := strings.Cut(token, ".")
	if !ok {
		return false
	}
	unix, err := strconv.ParseInt(exp, 36, 64)
	if err != nil || now.Unix() >= unix {
		return false
	}
	return hmac.Equal([]byte(mac), []byte(confirmMAC(key, tool, ids, exp)))
}

func confirmMAC(key []byte, tool string, ids []any, exp string) string {
	msg, _ := json.Marshal(append([]any{tool, exp}, ids...))
	m := hmac.New(sha256.New, key)
	m.Write(msg)
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil)[:16])
}

func excerpt(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
