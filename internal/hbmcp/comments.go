package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/mark3labs/mcp-go/mcp"
)

// nonBlankCommentPattern requires a character outside the Unicode whitespace set
// used by strings.TrimSpace. An explicit set avoids differences in \s across
// JSON Schema clients and Go regular expressions.
const nonBlankCommentPattern = "[^\t-\r \u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]"

// RegisterCommentTools registers all fault-comment-related MCP tools.
func RegisterCommentTools(r *toolRegistrar, clientFor ClientFactory) {
	r.AddTool(
		mcp.NewTool("list_fault_comments",
			mcp.WithTitleAnnotation("List Fault Comments"),
			mcp.WithDescription("List comments on a fault. Returns the first page of comments; pagination is not currently supported."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The ID of the project containing the fault"), mcp.Min(1)),
			mcp.WithInteger("fault_id", mcp.Required(), mcp.Description("The ID of the fault containing the comments"), mcp.Min(1)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFaultComments(ctx, clientFor(ctx), req)
		},
	)
	r.AddTool(
		mcp.NewTool("get_fault_comment",
			mcp.WithTitleAnnotation("Get Fault Comment"),
			mcp.WithDescription("Get a single comment on a fault by ID."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The ID of the project containing the fault"), mcp.Min(1)),
			mcp.WithInteger("fault_id", mcp.Required(), mcp.Description("The ID of the fault containing the comments"), mcp.Min(1)),
			mcp.WithInteger("comment_id", mcp.Required(), mcp.Description("The ID of the comment"), mcp.Min(1)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetFaultComment(ctx, clientFor(ctx), req)
		},
	)
	r.AddTool(
		mcp.NewTool("create_fault_comment",
			mcp.WithTitleAnnotation("Create Fault Comment"),
			mcp.WithDescription("Add a comment to a fault."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The ID of the project containing the fault"), mcp.Min(1)),
			mcp.WithInteger("fault_id", mcp.Required(), mcp.Description("The ID of the fault containing the comments"), mcp.Min(1)),
			mcp.WithString("body", mcp.Required(), mcp.Description("The comment text (must not be blank)"), mcp.MinLength(1), mcp.Pattern(nonBlankCommentPattern)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateFaultComment(ctx, clientFor(ctx), req)
		},
	)
	r.AddTool(
		mcp.NewTool("update_fault_comment",
			mcp.WithTitleAnnotation("Update Fault Comment"),
			mcp.WithDescription("Replace the body of an existing fault comment."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The ID of the project containing the fault"), mcp.Min(1)),
			mcp.WithInteger("fault_id", mcp.Required(), mcp.Description("The ID of the fault containing the comments"), mcp.Min(1)),
			mcp.WithInteger("comment_id", mcp.Required(), mcp.Description("The ID of the comment"), mcp.Min(1)),
			mcp.WithString("body", mcp.Required(), mcp.Description("The comment text (must not be blank)"), mcp.MinLength(1), mcp.Pattern(nonBlankCommentPattern)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateFaultComment(ctx, clientFor(ctx), req)
		},
	)
	r.AddTool(
		mcp.NewTool("delete_fault_comment",
			mcp.WithTitleAnnotation("Delete Fault Comment"),
			mcp.WithDescription("Delete an existing comment from a fault."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithInteger("project_id", mcp.Required(), mcp.Description("The ID of the project containing the fault"), mcp.Min(1)),
			mcp.WithInteger("fault_id", mcp.Required(), mcp.Description("The ID of the fault containing the comments"), mcp.Min(1)),
			mcp.WithInteger("comment_id", mcp.Required(), mcp.Description("The ID of the comment"), mcp.Min(1)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteFaultComment(ctx, clientFor(ctx), req)
		},
	)
}

func handleListFaultComments(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	projectID, ok := requireID(args, "project_id")
	if !ok {
		return mcp.NewToolResultError("project_id must be a positive integer"), nil
	}
	faultID, ok := requireID(args, "fault_id")
	if !ok {
		return mcp.NewToolResultError("fault_id must be a positive integer"), nil
	}
	value, err := client.Comments.List(ctx, projectID, faultID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list fault comments: %v", err)), nil
	}
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleGetFaultComment(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	projectID, ok := requireID(args, "project_id")
	if !ok {
		return mcp.NewToolResultError("project_id must be a positive integer"), nil
	}
	faultID, ok := requireID(args, "fault_id")
	if !ok {
		return mcp.NewToolResultError("fault_id must be a positive integer"), nil
	}
	commentID, ok := requireID(args, "comment_id")
	if !ok {
		return mcp.NewToolResultError("comment_id must be a positive integer"), nil
	}
	value, err := client.Comments.Get(ctx, projectID, faultID, commentID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get fault comment: %v", err)), nil
	}
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleCreateFaultComment(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	projectID, ok := requireID(args, "project_id")
	if !ok {
		return mcp.NewToolResultError("project_id must be a positive integer"), nil
	}
	faultID, ok := requireID(args, "fault_id")
	if !ok {
		return mcp.NewToolResultError("fault_id must be a positive integer"), nil
	}
	body, ok := args["body"].(string)
	if !ok || strings.TrimSpace(body) == "" {
		return mcp.NewToolResultError("body must be a non-empty string"), nil
	}
	value, err := client.Comments.Create(ctx, projectID, faultID, body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create fault comment: %v", err)), nil
	}
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func handleUpdateFaultComment(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	projectID, ok := requireID(args, "project_id")
	if !ok {
		return mcp.NewToolResultError("project_id must be a positive integer"), nil
	}
	faultID, ok := requireID(args, "fault_id")
	if !ok {
		return mcp.NewToolResultError("fault_id must be a positive integer"), nil
	}
	commentID, ok := requireID(args, "comment_id")
	if !ok {
		return mcp.NewToolResultError("comment_id must be a positive integer"), nil
	}
	body, ok := args["body"].(string)
	if !ok || strings.TrimSpace(body) == "" {
		return mcp.NewToolResultError("body must be a non-empty string"), nil
	}
	err := client.Comments.Update(ctx, projectID, faultID, commentID, body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update fault comment: %v", err)), nil
	}
	return mcp.NewToolResultText("Fault comment updated successfully"), nil
}

func handleDeleteFaultComment(ctx context.Context, client *hbapi.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	projectID, ok := requireID(args, "project_id")
	if !ok {
		return mcp.NewToolResultError("project_id must be a positive integer"), nil
	}
	faultID, ok := requireID(args, "fault_id")
	if !ok {
		return mcp.NewToolResultError("fault_id must be a positive integer"), nil
	}
	commentID, ok := requireID(args, "comment_id")
	if !ok {
		return mcp.NewToolResultError("comment_id must be a positive integer"), nil
	}
	err := client.Comments.Delete(ctx, projectID, faultID, commentID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete fault comment: %v", err)), nil
	}
	return mcp.NewToolResultText("Fault comment deleted successfully"), nil
}
