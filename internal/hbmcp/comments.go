package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// nonBlankCommentPattern requires a character outside the Unicode whitespace set
// used by strings.TrimSpace. An explicit set avoids differences in \s across
// JSON Schema clients and Go regular expressions.
const nonBlankCommentPattern = "[^\t-\r \u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]"

// How each comment write behaves with an account token (hba_), which has no
// user behind it.
const (
	commentCreateNote = " With an account-scoped API Token, the comment is attributed to the token's name."
	commentUpdateNote = " Only the comment's author can edit it, so an account-scoped API Token, which has no author, is refused with access_denied."
	commentDeleteNote = " An account-scoped API Token can delete a comment only when it can manage the project."
)

// RegisterCommentTools registers the fault comment tools.
func RegisterCommentTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	r.AddTool(
		mcp.NewTool("list_fault_comments",
			mcp.WithTitleAnnotation("List Fault Comments"),
			mcp.WithDescription("List every comment on a fault, newest first."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			withProjectParam(),
			withFaultParam(),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFaultComments(ctx, v3ClientFor(ctx), req)
		},
	)
	r.AddTool(
		mcp.NewTool("get_fault_comment",
			mcp.WithTitleAnnotation("Get Fault Comment"),
			mcp.WithDescription("Get a single comment on a fault by ID."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			withProjectParam(),
			withFaultParam(),
			withCommentParam(),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetFaultComment(ctx, v3ClientFor(ctx), req)
		},
	)
	r.AddTool(
		mcp.NewTool("create_fault_comment",
			mcp.WithTitleAnnotation("Create Fault Comment"),
			mcp.WithDescription("Add a comment to a fault. @mentions notify the named project members."+commentCreateNote),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			withProjectParam(),
			withFaultParam(),
			withCommentBodyParam(),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateFaultComment(ctx, v3ClientFor(ctx), req)
		},
	)
	r.AddTool(
		mcp.NewTool("update_fault_comment",
			mcp.WithTitleAnnotation("Update Fault Comment"),
			mcp.WithDescription("Replace the body of an existing fault comment. Returns the comment as stored."+commentUpdateNote),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			withProjectParam(),
			withFaultParam(),
			withCommentParam(),
			withCommentBodyParam(),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateFaultComment(ctx, v3ClientFor(ctx), req)
		},
	)
	r.AddTool(
		mcp.NewTool("delete_fault_comment",
			mcp.WithTitleAnnotation("Delete Fault Comment"),
			mcp.WithDescription("Delete an existing comment from a fault."+commentDeleteNote+confirmNote),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			withProjectParam(),
			withFaultParam(),
			withCommentParam(),
			withConfirmParam(),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteFaultComment(ctx, v3ClientFor(ctx), req)
		},
	)
}

func withProjectParam() mcp.ToolOption {
	return mcp.WithString("project_id", mcp.Required(), mcp.Description("The ID of the project containing the fault"))
}

func withFaultParam() mcp.ToolOption {
	return mcp.WithString("fault_id", mcp.Required(), mcp.Description("The ID of the fault the comments belong to"))
}

func withCommentParam() mcp.ToolOption {
	return mcp.WithString("comment_id", mcp.Required(), mcp.Description("The ID of the comment"))
}

func withCommentBodyParam() mcp.ToolOption {
	return mcp.WithString("body", mcp.Required(), mcp.Description("The comment text (must not be blank)"),
		mcp.MinLength(1), mcp.Pattern(nonBlankCommentPattern))
}

// requireComment reads the ids every single-comment tool needs.
func requireComment(req mcp.CallToolRequest) (projectID, faultID, commentID, errMsg string) {
	projectID, faultID, errMsg = requireProjectAndFault(req)
	if errMsg != "" {
		return "", "", "", errMsg
	}
	commentID = req.GetString("comment_id", "")
	if commentID == "" {
		return "", "", "", "comment_id is required"
	}
	return projectID, faultID, commentID, ""
}

// requireCommentBody refuses a missing or blank body, which the schema pattern
// catches only for clients that validate it.
func requireCommentBody(req mcp.CallToolRequest) (string, bool) {
	body, ok := req.GetArguments()["body"].(string)
	return body, ok && strings.TrimSpace(body) != ""
}

func handleListFaultComments(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, faultID, msg := requireProjectAndFault(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	comments, err := client.Faults.ListAllComments(ctx, projectID, faultID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list fault comments: %v", err)), nil
	}
	return jsonResult(comments)
}

func handleGetFaultComment(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, faultID, commentID, msg := requireComment(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	comment, err := client.Faults.GetComment(ctx, projectID, faultID, commentID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get fault comment: %v", err)), nil
	}
	return jsonResult(comment)
}

func handleCreateFaultComment(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, faultID, msg := requireProjectAndFault(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	body, ok := requireCommentBody(req)
	if !ok {
		return mcp.NewToolResultError("body must be a non-empty string"), nil
	}
	comment, err := client.Faults.AddComment(ctx, projectID, faultID, body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create fault comment: %v", err)), nil
	}
	return jsonResult(comment)
}

func handleUpdateFaultComment(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, faultID, commentID, msg := requireComment(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	body, ok := requireCommentBody(req)
	if !ok {
		return mcp.NewToolResultError("body must be a non-empty string"), nil
	}
	comment, err := client.Faults.UpdateComment(ctx, projectID, faultID, commentID, body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update fault comment: %v", err)), nil
	}
	return jsonResult(comment)
}

func handleDeleteFaultComment(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, faultID, commentID, msg := requireComment(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	if !deletionConfirmed(ctx, req, "delete_fault_comment", projectID, faultID, commentID) {
		comment, err := client.Faults.GetComment(ctx, projectID, faultID, commentID)
		var summary string
		switch {
		case err == nil:
			summary = fmt.Sprintf("delete comment %s by %s on fault %s: %q",
				commentID, commentAuthor(comment), faultID, excerpt(nullableString(comment.Body), 80))
		case unreadable(err):
			summary = fmt.Sprintf("delete comment %s on fault %s", commentID, faultID) + unreadableNote
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Failed to look up fault comment: %v", err)), nil
		}
		return deletionPreview(ctx, req, "delete_fault_comment", summary, projectID, faultID, commentID), nil
	}
	if err := client.Faults.DeleteComment(ctx, projectID, faultID, commentID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete fault comment: %v", err)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Fault comment %s deleted successfully", commentID)), nil
}

// commentAuthor names who wrote a comment, for a deletion preview.
func commentAuthor(c *apiv3.Comment) string {
	if a, err := c.Author.Get(); err == nil {
		if name, err := a.Name.Get(); err == nil && name != "" {
			return name
		}
		if email, err := a.Email.Get(); err == nil && email != "" {
			return email
		}
	}
	return "an unknown author"
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// nullableString returns the value, or "" when it is null or absent.
func nullableString(n interface{ Get() (string, error) }) string {
	v, _ := n.Get()
	return v
}
