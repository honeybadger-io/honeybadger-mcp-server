package hbmcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterUsageTools registers get_account_usage, the account's usage against
// its plan's quotas. The API's data is returned as is; the description says
// what each field means.
func RegisterUsageTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	r.AddTool(
		generatedTool("get_account_usage"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetAccountUsage(ctx, v3ClientFor(ctx), req)
		},
	)
}

func handleGetAccountUsage(ctx context.Context, client *apiv3.Client, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	usage, err := client.Account.Usage(ctx)
	if errors.Is(err, apiv3.ErrAmbiguousAccount) {
		// The API's message says what's wrong but not what to use instead.
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get account usage: this credential can access more than one account, so the API can't tell whose usage to report. Use an account-bound API Token (hba_), or a token for a user with access to only one account. (%v)", err)), nil
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get account usage: %v", err)), nil
	}
	return jsonResult(usage)
}
