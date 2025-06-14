package hbapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// FaultsService handles operations for the faults resource
type FaultsService struct {
	client *Client
}

// FaultListOptions represents options for listing faults
type FaultListOptions struct {
	Q              string `json:"q,omitempty"`               // Search string
	CreatedAfter   string `json:"created_after,omitempty"`   // Timestamp string
	OccurredAfter  string `json:"occurred_after,omitempty"`  // Timestamp string
	OccurredBefore string `json:"occurred_before,omitempty"` // Timestamp string
	Limit          int    `json:"limit,omitempty"`           // Max 25
	Order          string `json:"order,omitempty"`           // "recent" or "frequent"
}

// FaultListResponse represents the API response for listing faults
type FaultListResponse struct {
	Results []Fault                `json:"results"`
	Links   map[string]interface{} `json:"links"`
}

// List returns a list of faults for a project with optional filtering and ordering.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/faults/#get-a-fault-list-or-fault-details
//
// GET /v2/projects/{projectID}/faults
func (f *FaultsService) List(ctx context.Context, projectID int, options FaultListOptions) (*FaultListResponse, error) {
	path := fmt.Sprintf("/projects/%d/faults", projectID)

	// Build query parameters using url.Values
	params := url.Values{}
	if options.Q != "" {
		params.Set("q", options.Q)
	}
	if options.CreatedAfter != "" {
		params.Set("created_after", options.CreatedAfter)
	}
	if options.OccurredAfter != "" {
		params.Set("occurred_after", options.OccurredAfter)
	}
	if options.OccurredBefore != "" {
		params.Set("occurred_before", options.OccurredBefore)
	}
	if options.Limit > 0 {
		params.Set("limit", strconv.Itoa(options.Limit))
	}
	if options.Order != "" {
		params.Set("order", options.Order)
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	req, err := f.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response FaultListResponse
	if err := f.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Get returns a single fault by ID with full fault details.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/faults/#get-a-fault-list-or-fault-details
//
// GET /v2/projects/{projectID}/faults/{faultID}
func (f *FaultsService) Get(ctx context.Context, projectID, faultID int) (*Fault, error) {
	path := fmt.Sprintf("/projects/%d/faults/%d", projectID, faultID)
	req, err := f.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result Fault
	if err := f.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// FaultListNoticesOptions represents options for listing notices for a fault
type FaultListNoticesOptions struct {
	CreatedAfter  string `json:"created_after,omitempty"`  // Timestamp string
	CreatedBefore string `json:"created_before,omitempty"` // Timestamp string
	Limit         int    `json:"limit,omitempty"`          // Max 25
}

// FaultNoticesResponse represents the API response for listing fault notices
type FaultNoticesResponse struct {
	Results []Notice               `json:"results"`
	Links   map[string]interface{} `json:"links"`
}

// ListNotices returns a list of notices for a specific fault with optional filtering.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/faults/#get-a-list-of-notices
//
// GET /v2/projects/{projectID}/faults/{faultID}/notices
func (f *FaultsService) ListNotices(ctx context.Context, projectID, faultID int, options FaultListNoticesOptions) (*FaultNoticesResponse, error) {
	path := fmt.Sprintf("/projects/%d/faults/%d/notices", projectID, faultID)

	// Build query parameters using url.Values
	params := url.Values{}
	if options.CreatedAfter != "" {
		params.Set("created_after", options.CreatedAfter)
	}
	if options.CreatedBefore != "" {
		params.Set("created_before", options.CreatedBefore)
	}
	if options.Limit > 0 {
		params.Set("limit", strconv.Itoa(options.Limit))
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	req, err := f.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response FaultNoticesResponse
	if err := f.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// FaultAffectedUser represents a user affected by a fault
type FaultAffectedUser struct {
	User  string `json:"user"`  // Email or user identifier
	Count int    `json:"count"` // Number of occurrences for this user
}

// FaultListAffectedUsersOptions represents options for listing affected users for a fault
type FaultListAffectedUsersOptions struct {
	Q string `json:"q,omitempty"` // Search string
}

// ListAffectedUsers returns a list of users affected by a specific fault.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/faults/#get-a-list-of-affected-users
//
// GET /v2/projects/{projectID}/faults/{faultID}/affected_users
func (f *FaultsService) ListAffectedUsers(ctx context.Context, projectID, faultID int, options FaultListAffectedUsersOptions) ([]FaultAffectedUser, error) {
	path := fmt.Sprintf("/projects/%d/faults/%d/affected_users", projectID, faultID)

	// Build query parameters if search provided
	if options.Q != "" {
		path += fmt.Sprintf("?q=%s", url.QueryEscape(options.Q))
	}

	req, err := f.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var users []FaultAffectedUser
	if err := f.client.do(ctx, req, &users); err != nil {
		return nil, err
	}

	return users, nil
}
