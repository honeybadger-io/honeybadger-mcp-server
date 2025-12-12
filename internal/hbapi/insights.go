package hbapi

import (
	"context"
	"fmt"
)

// InsightsService handles operations for the insights resource
type InsightsService struct {
	client *Client
}

// InsightsQueryRequest represents a request to query insights data
type InsightsQueryRequest struct {
	Query    string `json:"query"`
	Ts       string `json:"ts,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

// InsightsQueryMeta represents metadata about an insights query response
type InsightsQueryMeta struct {
	Query     string                   `json:"query"`
	Fields    []string                 `json:"fields"`
	Schema    []map[string]interface{} `json:"schema"`
	Rows      int                      `json:"rows"`
	TotalRows int                      `json:"total_rows"`
	StartAt   string                   `json:"start_at"`
	EndAt     string                   `json:"end_at"`
}

// InsightsQueryResponse represents the response from an insights query
type InsightsQueryResponse struct {
	Results []map[string]interface{} `json:"results"`
	Meta    InsightsQueryMeta        `json:"meta"`
}

// Query executes a BadgerQL query against the project's insights data.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/insights/
//
// POST /v2/projects/{projectID}/insights/queries
func (i *InsightsService) Query(ctx context.Context, projectID int, request InsightsQueryRequest) (*InsightsQueryResponse, error) {
	path := fmt.Sprintf("/projects/%d/insights/queries", projectID)

	req, err := i.client.newRequest(ctx, "POST", path, request)
	if err != nil {
		return nil, err
	}

	var response InsightsQueryResponse
	if err := i.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
