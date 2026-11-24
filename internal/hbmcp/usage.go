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
		mcp.NewTool("get_account_usage",
			mcp.WithTitleAnnotation("Get Account Usage"),
			mcp.WithDescription("Get the account's usage against its plan's quotas. "+
				"errors_limit and errors_consumed: errors allowed and recorded in the current billing period. "+
				"insights_daily_limit and insights_daily_consumed: Insights bytes allowed and ingested in the current UTC day; insights_limited is true while Insights data is being dropped for exceeding the limit. "+
				"Uptime monitors and check-ins: uptime_consumed and check_ins_consumed are always present. Plans without a monitor pool add uptime_limit and check_ins_limit; plans where both share one pool add monitors_limit and monitors_consumed instead. On plans with a project limit, uptime_limit applies to each project while uptime_consumed counts the whole account. "+
				"A null limit means no cap; a limit of 0 is a real cap that allows nothing. "+
				"Needs a credential for one account; a project-bound credential is refused."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
		),
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
