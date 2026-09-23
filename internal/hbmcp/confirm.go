package hbmcp

import (
	"context"
	"fmt"
	"time"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/confirmtoken"
	"github.com/mark3labs/mcp-go/mcp"
)

const confirmNote = " Deleting takes two calls: the first deletes nothing and returns a preview with a confirmation token. Show the preview to the user, and only if they approve, call again with the same arguments and confirm set to that token."

// Signs for stdio, where one process mints and verifies every token; http
// mode injects a signer keyed by the shared MCP_CONFIRM_SECRET instead.
var processSigner = confirmtoken.NewRandom()

type confirmSignerKey struct{}

func withConfirmSigner(ctx context.Context, s *confirmtoken.Signer) context.Context {
	return context.WithValue(ctx, confirmSignerKey{}, s)
}

func confirmSigner(ctx context.Context) *confirmtoken.Signer {
	if s, ok := ctx.Value(confirmSignerKey{}).(*confirmtoken.Signer); ok {
		return s
	}
	return processSigner
}

// The subject survives access-token refreshes, unlike the bearer itself.
func confirmCaller(ctx context.Context) string {
	if claims := ClaimsFromContext(ctx); claims != nil {
		return claims.Subject
	}
	return ""
}

func withConfirmParam() mcp.ToolOption {
	return mcp.WithString("confirm",
		mcp.Description("Confirmation token from this tool's previous preview. Pass it only after the user has approved the deletion shown in that preview."),
	)
}

// deletionConfirmed reports whether req carries a live token minted for this
// tool and these ids by the same caller.
func deletionConfirmed(ctx context.Context, req mcp.CallToolRequest, tool string, ids ...any) bool {
	token := req.GetString("confirm", "")
	return token != "" && confirmSigner(ctx).Valid(token, confirmCaller(ctx), tool, ids, time.Now())
}

// deletionPreview is what a delete handler returns instead of deleting;
// summary completes "This will permanently ...".
func deletionPreview(ctx context.Context, req mcp.CallToolRequest, tool, summary string, ids ...any) *mcp.CallToolResult {
	notice := ""
	if req.GetString("confirm", "") != "" {
		notice = "The confirm token was invalid or expired. "
	}
	token := confirmSigner(ctx).Mint(confirmCaller(ctx), tool, ids, time.Now())
	return mcp.NewToolResultText(fmt.Sprintf(
		"%sNot deleted. This will permanently %s. Show this to the user and ask them to approve. Only if they approve, call %s again with the same arguments and confirm set to %q (valid for %d minutes).",
		notice, summary, tool, token, int(confirmtoken.TTL.Minutes())))
}

func excerpt(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
