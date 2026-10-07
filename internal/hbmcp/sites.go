package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/google/uuid"
	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterSiteTools registers the uptime site tools: the sites themselves, and
// their outages and checks, which up, down and cert_will_expire events are
// about.
func RegisterSiteTools(r *toolRegistrar, v3ClientFor V3ClientFactory) {
	r.AddTool(
		mcp.NewTool("list_sites",
			mcp.WithTitleAnnotation("List Sites"),
			mcp.WithDescription("List a project's uptime sites: the URLs Honeybadger checks, with their current state (up, down, paused)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("The ID of the project to list sites for")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListSites(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("get_site",
			mcp.WithTitleAnnotation("Get Site"),
			mcp.WithDescription("Get an uptime site: its URL, check settings and current state. Up, down and cert_will_expire events name the site by ID."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("The ID of the project the site belongs to")),
			mcp.WithString("site_id", mcp.Required(), mcp.Description("The ID of the site, a UUID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetSite(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("create_site",
			append([]mcp.ToolOption{
				mcp.WithTitleAnnotation("Create Site"),
				mcp.WithDescription("Start uptime monitoring for a URL. Only url is required; the rest default as described."),
				mcp.WithReadOnlyHintAnnotation(false),
				mcp.WithDestructiveHintAnnotation(true),
				mcp.WithString("project_id", mcp.Required(), mcp.Description("The ID of the project to create the site in")),
				mcp.WithString("url", mcp.Required(), mcp.Description("The URL to check. Must start with http:// or https://.")),
			}, siteSettingOptions()...)...,
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateSite(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("update_site",
			append([]mcp.ToolOption{
				mcp.WithTitleAnnotation("Update Site"),
				mcp.WithDescription("Change an uptime site's settings. Only the parameters given are changed, and null resets a setting to its default."),
				mcp.WithReadOnlyHintAnnotation(false),
				mcp.WithDestructiveHintAnnotation(true),
				mcp.WithString("project_id", mcp.Required(), mcp.Description("The ID of the project the site belongs to")),
				mcp.WithString("site_id", mcp.Required(), mcp.Description("The ID of the site, a UUID")),
				mcp.WithString("url", mcp.Description("A new URL to check. Must start with http:// or https://.")),
			}, siteSettingOptions()...)...,
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateSite(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("delete_site",
			mcp.WithTitleAnnotation("Delete Site"),
			mcp.WithDescription("Stop monitoring an uptime site and delete it."+confirmNote),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("The ID of the project the site belongs to")),
			mcp.WithString("site_id", mcp.Required(), mcp.Description("The ID of the site to delete, a UUID")),
			withConfirmParam(),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteSite(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("list_site_outages",
			mcp.WithTitleAnnotation("List Site Outages"),
			mcp.WithDescription("List an uptime site's outages, newest first: when it went down and came back up, with the status and reason from the failing check. Use it to explain a down or up event. To page back, pass the created_before value from the query string of the previous response's time_series_links.older, unchanged."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("The ID of the project the site belongs to")),
			mcp.WithString("site_id", mcp.Required(), mcp.Description("The ID of the site, a UUID")),
			timeSeriesLimit(),
			createdBeforeParam(),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListSiteOutages(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		mcp.NewTool("list_uptime_checks",
			mcp.WithTitleAnnotation("List Uptime Checks"),
			mcp.WithDescription("List an uptime site's individual checks, newest first: each check's location, response status and duration. A busy site has many; keep limit small. To page back, pass the created_before value from the query string of the previous response's time_series_links.older, unchanged."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("The ID of the project the site belongs to")),
			mcp.WithString("site_id", mcp.Required(), mcp.Description("The ID of the site, a UUID")),
			timeSeriesLimit(),
			createdBeforeParam(),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListUptimeChecks(ctx, v3ClientFor(ctx), req)
		},
	)
}

// siteSettingFields are the site settings the create and update tools accept
// besides url. They're passed through as given, so an explicit null reaches the
// API, where it resets the setting.
var siteSettingFields = []string{
	"name", "frequency", "locations", "match_type", "match", "request_method",
	"request_body", "request_headers", "timeout", "outage_threshold", "validate_ssl", "active",
}

func siteSettingOptions() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithString("name", mcp.Description("A display name for the site")),
		mcp.WithNumber("frequency", acceptsNull,
			mcp.Description("Minutes between checks: 1, 2, 5 or 15. Defaults to 5; a value more frequent than the plan allows is refused.")),
		mcp.WithArray("locations", acceptsNull,
			mcp.Items(map[string]any{"type": "string", "enum": []string{"Virginia", "Oregon", "London", "Frankfurt", "Singapore"}}),
			mcp.Description("Locations to check from. [] or null checks from every location.")),
		mcp.WithString("match_type", mcp.Enum("success", "exact", "include", "exclude", "jmespath"), acceptsNull,
			mcp.Description("How a response passes: success (any 2xx; leave match out), exact (status code equals match), include or exclude (the body does or doesn't contain match), jmespath (match is a JMESPath expression over the JSON body). Defaults to success.")),
		mcp.WithString("match", acceptsNull, mcp.Description("What match_type compares against. Required unless match_type is success; null clears it.")),
		mcp.WithString("request_method", mcp.Enum("GET", "POST", "PUT", "PATCH", "DELETE"), acceptsNull, mcp.Description("Defaults to GET")),
		mcp.WithString("request_body", acceptsNull, mcp.Description("A body to send with the request, up to 32 KB; null clears it")),
		mcp.WithObject("request_headers", acceptsNull, mcp.AdditionalProperties(map[string]any{"type": "string"}),
			mcp.Description("Headers to send, as name to value. Replaces the stored headers; {} or null clears them.")),
		mcp.WithNumber("timeout", acceptsNull, mcp.Description("Request timeout in seconds, 1 to 120, shorter than the frequency. Defaults to 30; only accounts with the uptime-timeout feature can change it.")),
		mcp.WithNumber("outage_threshold", acceptsNull, mcp.Description("Consecutive failures before an outage is declared. null declares one when half the locations fail.")),
		mcp.WithBoolean("validate_ssl", acceptsNull, mcp.Description("Fail the check when the TLS certificate doesn't validate. Defaults to true.")),
		mcp.WithBoolean("active", acceptsNull, mcp.Description("Whether the site is checked. Defaults to true.")),
	}
}

// siteBody gathers the given site settings into a request body.
func siteBody(req mcp.CallToolRequest) []byte {
	args := req.GetArguments()
	body := map[string]any{}
	for _, field := range append([]string{"url"}, siteSettingFields...) {
		if v, ok := args[field]; ok {
			body[field] = v
		}
	}
	encoded, _ := json.Marshal(body)
	return encoded
}

// requireSite reads the project and site IDs every site tool takes.
func requireSite(req mcp.CallToolRequest) (projectID string, siteID apiv3.SiteID, errMsg string) {
	projectID = req.GetString("project_id", "")
	if projectID == "" {
		return "", siteID, "project_id is required"
	}
	raw := req.GetString("site_id", "")
	if raw == "" {
		return "", siteID, "site_id is required"
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", siteID, fmt.Sprintf("site_id must be a site ID, which is a UUID: %v", err)
	}
	return projectID, id, ""
}

func handleListSites(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	sites, err := client.Sites.ListAll(ctx, projectID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list sites: %v", err)), nil
	}
	return jsonResult(sites)
}

func handleGetSite(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, siteID, msg := requireSite(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	site, err := client.Sites.Get(ctx, projectID, siteID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get site: %v", err)), nil
	}
	return jsonResult(site)
}

func handleCreateSite(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID := req.GetString("project_id", "")
	if projectID == "" {
		return mcp.NewToolResultError("project_id is required"), nil
	}
	if req.GetString("url", "") == "" {
		return mcp.NewToolResultError("url is required"), nil
	}
	var params apiv3.SiteCreateParams
	if msg := decodeSettings(siteBody(req), &params); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	site, err := client.Sites.Create(ctx, projectID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create site: %v", err)), nil
	}
	return jsonResult(site)
}

func handleUpdateSite(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, siteID, msg := requireSite(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	var params apiv3.SiteUpdateParams
	if msg := decodeSettings(siteBody(req), &params); msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	if changesNothing(params) {
		return mcp.NewToolResultError("provide at least one setting to change"), nil
	}
	site, err := client.Sites.Update(ctx, projectID, siteID, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update site: %v", err)), nil
	}
	return jsonResult(site)
}

func handleDeleteSite(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, siteID, msg := requireSite(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	id := siteID.String()

	if !deletionConfirmed(ctx, req, "delete_site", projectID, id) {
		var summary string
		site, err := client.Sites.Get(ctx, projectID, siteID)
		switch {
		case err == nil && site.Name == "":
			summary = fmt.Sprintf("delete the unnamed site %s (id %s) from project %s and stop monitoring it", site.Url, id, projectID)
		case err == nil:
			summary = fmt.Sprintf("delete site %q (%s, id %s) from project %s and stop monitoring it", site.Name, site.Url, id, projectID)
		case unreadable(err):
			summary = fmt.Sprintf("delete site %s from project %s and stop monitoring it", id, projectID) + unreadableNote
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Failed to look up site: %v", err)), nil
		}
		return deletionPreview(ctx, req, "delete_site", summary, projectID, id), nil
	}

	if err := client.Sites.Delete(ctx, projectID, siteID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete site: %v", err)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Site %s deleted successfully", id)), nil
}

func handleListSiteOutages(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, siteID, msg := requireSite(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	opts, msg := olderThanOptions(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	outages, err := client.Sites.ListOutages(ctx, projectID, siteID, opts...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list outages: %v", err)), nil
	}
	return jsonResult(outages)
}

func handleListUptimeChecks(ctx context.Context, client *apiv3.Client, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, siteID, msg := requireSite(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	opts, msg := olderThanOptions(req)
	if msg != "" {
		return mcp.NewToolResultError(msg), nil
	}
	checks, err := client.Sites.ListUptimeChecks(ctx, projectID, siteID, opts...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list uptime checks: %v", err)), nil
	}
	return jsonResult(checks)
}

// timeSeriesLimit and createdBeforeParam declare the paging parameters of the
// collections that page back by timestamp.
func timeSeriesLimit() mcp.ToolOption {
	return mcp.WithInteger("limit", mcp.Min(1), mcp.Max(100), mcp.Description("Most items to return, up to 100 (default 25)"))
}

func createdBeforeParam() mcp.ToolOption {
	return mcp.WithNumber("created_before",
		mcp.Description("Page back: the created_before value from the query string of the previous response's time_series_links.older, unchanged"))
}

// pageLimit reads limit. A fractional or non-positive value is refused rather
// than truncated, as the API refuses it.
func pageLimit(req mcp.CallToolRequest) (apiv3.Option, string) {
	raw, present := req.GetArguments()["limit"]
	if !present || raw == nil {
		return nil, ""
	}
	var n float64
	switch v := raw.(type) {
	case float64:
		n = v
	case int:
		n = float64(v)
	case int64:
		n = float64(v)
	default:
		return nil, "limit must be a whole number from 1 to 100"
	}
	if n < 1 || n != math.Trunc(n) {
		return nil, "limit must be a whole number from 1 to 100"
	}
	return apiv3.Limit(int(n)), ""
}

// olderThanOptions reads limit and created_before. A created_before that isn't a
// number is refused: dropping it would serve the newest page again, and a caller
// paging back would loop on it.
func olderThanOptions(req mcp.CallToolRequest) ([]apiv3.Option, string) {
	var opts []apiv3.Option
	limit, msg := pageLimit(req)
	if msg != "" {
		return nil, msg
	}
	if limit != nil {
		opts = append(opts, limit)
	}
	switch v := req.GetArguments()["created_before"].(type) {
	case nil:
	case float64:
		opts = append(opts, apiv3.OlderThan(v))
	case string:
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, "created_before must be the number from time_series_links.older's created_before, not the whole URL"
		}
		opts = append(opts, apiv3.OlderThan(n))
	default:
		return nil, "created_before must be a number"
	}
	return opts, ""
}
