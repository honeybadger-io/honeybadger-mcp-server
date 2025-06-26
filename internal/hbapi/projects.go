package hbapi

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// ProjectsService handles operations for the projects resource
type ProjectsService struct {
	client *Client
}

// ProjectsResponse represents the API response for listing projects
type ProjectsResponse struct {
	Results []Project              `json:"results"`
	Links   map[string]interface{} `json:"links"`
}

// ListAll returns all projects across all accounts accessible by the authenticated user.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/projects/#get-a-project-list-or-project-details
//
// GET /projects
func (p *ProjectsService) ListAll(ctx context.Context) (*ProjectsResponse, error) {
	req, err := p.client.newRequest(ctx, "GET", "/projects", nil)
	if err != nil {
		return nil, err
	}

	var response ProjectsResponse
	if err := p.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// ListByAccountID returns all projects filtered by account_id.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/projects/#get-a-project-list-or-project-details
//
// GET /projects?account_id={accountID}
func (p *ProjectsService) ListByAccountID(ctx context.Context, accountID string) (*ProjectsResponse, error) {
	path := fmt.Sprintf("/projects?account_id=%s", accountID)

	req, err := p.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response ProjectsResponse
	if err := p.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Get returns a single project by ID with full project details.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/projects/#get-a-project-list-or-project-details
//
// GET /projects/{id}
func (p *ProjectsService) Get(ctx context.Context, id int) (*Project, error) {
	path := fmt.Sprintf("/projects/%d", id)
	req, err := p.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result Project
	if err := p.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ProjectRequest represents the request parameters for creating or updating a project
type ProjectRequest struct {
	Name                  string `json:"name,omitempty"`
	ResolveErrorsOnDeploy *bool  `json:"resolve_errors_on_deploy,omitempty"`
	DisablePublicLinks    *bool  `json:"disable_public_links,omitempty"`
	UserURL               string `json:"user_url,omitempty"`
	SourceURL             string `json:"source_url,omitempty"`
	PurgeDays             *int   `json:"purge_days,omitempty"`
	UserSearchField       string `json:"user_search_field,omitempty"`
}

// Create creates a new project with the given parameters.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/projects/#create-a-project
//
// POST /projects?account_id={accountID}
func (p *ProjectsService) Create(ctx context.Context, accountID string, req ProjectRequest) (*Project, error) {
	body := map[string]interface{}{
		"project": req,
	}

	path := fmt.Sprintf("/projects?account_id=%s", accountID)
	httpReq, err := p.client.newRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var result Project
	if err := p.client.do(ctx, httpReq, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateResult represents the result of an update operation
type UpdateResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Update updates an existing project with the given parameters.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/projects/#update-a-project
//
// PUT /projects/{id}
func (p *ProjectsService) Update(ctx context.Context, id int, req ProjectRequest) (*UpdateResult, error) {
	body := map[string]interface{}{
		"project": req,
	}

	path := fmt.Sprintf("/projects/%d", id)
	httpReq, err := p.client.newRequest(ctx, "PUT", path, body)
	if err != nil {
		return nil, err
	}

	// Update API returns empty body on success, so we don't decode a result
	if err := p.client.do(ctx, httpReq, nil); err != nil {
		return nil, err
	}

	return &UpdateResult{
		Success: true,
		Message: fmt.Sprintf("Project %d was successfully updated", id),
	}, nil
}

// DeleteResult represents the result of a delete operation
type DeleteResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Delete deletes a project by ID.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/projects/#delete-a-project
//
// DELETE /projects/{id}
func (p *ProjectsService) Delete(ctx context.Context, id int) (*DeleteResult, error) {
	path := fmt.Sprintf("/projects/%d", id)
	req, err := p.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return nil, err
	}

	if err := p.client.do(ctx, req, nil); err != nil {
		return nil, err
	}

	return &DeleteResult{
		Success: true,
		Message: fmt.Sprintf("Project %d deleted successfully", id),
	}, nil
}

// ProjectGetOccurrenceCountsOptions represents options for getting occurrence counts
type ProjectGetOccurrenceCountsOptions struct {
	Period      string // "hour", "day", "week", or "month"
	Environment string // Filter by environment
}

// ProjectOccurrenceCount represents a single occurrence count data point [timestamp, count]
type ProjectOccurrenceCount [2]int64

// ProjectGetOccurrenceCountsResponse represents the response from single project occurrence counts API
type ProjectGetOccurrenceCountsResponse []ProjectOccurrenceCount

// GetAllOccurrenceCounts gets occurrence counts for all projects.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/projects/#get-a-count-of-occurrences-for-all-projects-or-a-single-project
//
// GET /projects/occurrences
func (p *ProjectsService) GetAllOccurrenceCounts(ctx context.Context, options ProjectGetOccurrenceCountsOptions) (ProjectGetAllOccurrenceCountsResponse, error) {
	path := "/projects/occurrences"

	// Build query parameters using url.Values
	params := url.Values{}
	if options.Period != "" {
		params.Set("period", options.Period)
	}
	if options.Environment != "" {
		params.Set("environment", options.Environment)
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	req, err := p.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result ProjectGetAllOccurrenceCountsResponse
	if err := p.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ProjectGetAllOccurrenceCountsResponse represents the response from all projects occurrence counts API
// The map key is the project ID as a string
type ProjectGetAllOccurrenceCountsResponse map[string][]ProjectOccurrenceCount

// GetOccurrenceCounts gets occurrence counts for a specific project.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/projects/#get-a-count-of-occurrences-for-all-projects-or-a-single-project
//
// GET /projects/{projectID}/occurrences
func (p *ProjectsService) GetOccurrenceCounts(ctx context.Context, projectID int, options ProjectGetOccurrenceCountsOptions) (ProjectGetOccurrenceCountsResponse, error) {
	path := fmt.Sprintf("/projects/%d/occurrences", projectID)

	// Build query parameters using url.Values
	params := url.Values{}
	if options.Period != "" {
		params.Set("period", options.Period)
	}
	if options.Environment != "" {
		params.Set("environment", options.Environment)
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	req, err := p.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result ProjectGetOccurrenceCountsResponse
	if err := p.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ProjectIntegration represents a Honeybadger project integration (channel)
type ProjectIntegration struct {
	ID                   int                    `json:"id"`
	Active               bool                   `json:"active"`
	Events               []string               `json:"events"`
	SiteIDs              []string               `json:"site_ids"`
	Options              map[string]interface{} `json:"options"`
	ExcludedEnvironments []string               `json:"excluded_environments"`
	Filters              []interface{}          `json:"filters"`
	Type                 string                 `json:"type"`
}

// GetIntegrations gets all integrations (channels) for a specific project.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/projects/#get-a-list-of-integrations-for-a-project
//
// GET /projects/{projectID}/integrations
func (p *ProjectsService) GetIntegrations(ctx context.Context, projectID int) ([]ProjectIntegration, error) {
	path := fmt.Sprintf("/projects/%d/integrations", projectID)
	req, err := p.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result []ProjectIntegration
	if err := p.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ProjectReportType represents the type of report to fetch
type ProjectReportType string

const (
	ProjectNoticesByClass    ProjectReportType = "notices_by_class"
	ProjectNoticesByLocation ProjectReportType = "notices_by_location"
	ProjectNoticesByUser     ProjectReportType = "notices_by_user"
	ProjectNoticesPerDay     ProjectReportType = "notices_per_day"
)

// ProjectGetReportOptions represents options for getting report data
type ProjectGetReportOptions struct {
	Start       *time.Time // ISO 8601 format date/time
	Stop        *time.Time // ISO 8601 format date/time
	Environment string     // Filter by environment
}

// GetReport gets report data for a specific project.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/projects/#get-report-data
//
// GET /projects/{projectID}/reports/{reportType}
func (p *ProjectsService) GetReport(ctx context.Context, projectID int, reportType ProjectReportType, options ProjectGetReportOptions) ([][]interface{}, error) {
	path := fmt.Sprintf("/projects/%d/reports/%s", projectID, reportType)

	// Build query parameters using url.Values
	params := url.Values{}
	if options.Start != nil {
		params.Set("start", options.Start.Format(time.RFC3339))
	}
	if options.Stop != nil {
		params.Set("stop", options.Stop.Format(time.RFC3339))
	}
	if options.Environment != "" {
		params.Set("environment", options.Environment)
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	req, err := p.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result [][]interface{}
	if err := p.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return result, nil
}
