package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
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
		generatedTool("list_sites"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListSites(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("get_site"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetSite(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("create_site"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateSite(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("update_site"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleUpdateSite(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("delete_site"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleDeleteSite(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("list_site_outages"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListSiteOutages(ctx, v3ClientFor(ctx), req)
		},
	)

	r.AddTool(
		generatedTool("list_uptime_checks"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListUptimeChecks(ctx, v3ClientFor(ctx), req)
		},
	)
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
	if msg := bodyFrom(req, &params, "project_id"); msg != "" {
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
	// name and url have no default to reset to, so a null for either would
	// otherwise be dropped while the rest of the update reported success.
	for _, field := range []string{"name", "url"} {
		if v, present := req.GetArguments()[field]; present && v == nil {
			return mcp.NewToolResultError(field + " can't be reset; omit it to keep the current value"), nil
		}
	}
	var params apiv3.SiteUpdateParams
	if msg := bodyFrom(req, &params, "project_id", "site_id"); msg != "" {
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
	return jsonResult(timestampPageOf(outages))
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
	return jsonResult(timestampPageOf(checks))
}

// timestampPage is what the outage, uptime check and check-in event tools
// return. Those collections page back by a Unix timestamp the API puts only
// inside links.older's URL, so the position is lifted out once here and handed
// back as next_created_before; callers never handle a URL.
type timestampPage[T any] struct {
	Data     []T  `json:"data"`
	HasOlder bool `json:"has_older"`
	// NextCreatedBefore is the link's created_before verbatim. It's a number
	// with significant microseconds, so it's passed through as written rather
	// than reformatted.
	NextCreatedBefore json.Number `json:"next_created_before,omitempty"`
}

func timestampPageOf[T any](resp *apiv3.ListResponse[T]) timestampPage[T] {
	page := timestampPage[T]{Data: resp.Data}
	if resp.TimeSeries != nil {
		page.HasOlder = resp.TimeSeries.HasOlder
	}
	if resp.TimeSeriesLinks == nil {
		return page
	}
	older, err := resp.TimeSeriesLinks.Older.Get()
	if err != nil {
		return page
	}
	link, err := url.Parse(older)
	if err != nil {
		return page
	}
	if v := link.Query().Get("created_before"); v != "" {
		if _, err := strconv.ParseFloat(v, 64); err == nil {
			page.NextCreatedBefore = json.Number(v)
		}
	}
	return page
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
	if v, ok := req.GetArguments()["created_before"].(float64); ok {
		opts = append(opts, apiv3.OlderThan(v))
	}
	return opts, ""
}
