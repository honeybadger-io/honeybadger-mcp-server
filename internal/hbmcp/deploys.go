package hbmcp

import (
	"context"
	"fmt"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterDeployTools registers the deploy tools: deployed events and a fault's
// last_notice_deploy name a deploy by ID.
func RegisterDeployTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	r.AddTool(
		generatedTool("list_deploys"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListDeploys(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("get_deploy"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetDeploy(ctx, v3ClientFor(ctx), req)
		},
	)
}

func handleListDeploys(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	var opts []apiv3.Option
	if env := req.GetString("environment", ""); env != "" {
		opts = append(opts, apiv3.InEnvironment(env))
	}
	if who := req.GetString("local_username", ""); who != "" {
		opts = append(opts, apiv3.DeployedBy(who))
	}
	limit, msg := pageLimit(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	if limit != nil {
		opts = append(opts, limit)
	}
	if cursor := req.GetString("before", ""); cursor != "" {
		opts = append(opts, apiv3.Before(cursor))
	}
	if cursor := req.GetString("after", ""); cursor != "" {
		opts = append(opts, apiv3.After(cursor))
	}

	deploys, err := client.Deploys.List(ctx, projectID, opts...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list deploys: %v", err)), nil
	}
	return jsonResult(deploys)
}

func handleGetDeploy(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	deployID := req.GetString("deploy_id", "")
	if deployID == "" {
		return mcp.NewToolResultError("deploy_id is required"), nil
	}
	deploy, err := client.Deploys.Get(ctx, projectID, deployID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get deploy: %v", err)), nil
	}
	return jsonResult(deploy)
}
