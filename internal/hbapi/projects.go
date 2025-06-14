package hbapi

import (
	"context"
	"fmt"
)

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

// ProjectGetOccurrenceCountsOptions represents options for getting occurrence counts
type ProjectGetOccurrenceCountsOptions struct {
	Period      string `json:"period,omitempty"`      // "hour", "day", "week", or "month"
	Environment string `json:"environment,omitempty"` // Filter by environment
}

// ProjectOccurrenceCount represents a single occurrence count data point [timestamp, count]
type ProjectOccurrenceCount [2]int64

// ProjectGetOccurrenceCountsResponse represents the response from single project occurrence counts API
type ProjectGetOccurrenceCountsResponse []ProjectOccurrenceCount

// ProjectGetAllOccurrenceCountsResponse represents the response from all projects occurrence counts API
// The map key is the project ID as a string
type ProjectGetAllOccurrenceCountsResponse map[string][]ProjectOccurrenceCount

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
	Start       string `json:"start,omitempty"`       // ISO 8601 format date/time
	Stop        string `json:"stop,omitempty"`        // ISO 8601 format date/time
	Environment string `json:"environment,omitempty"` // Filter by environment
}

// ProjectsService handles operations for the projects resource
type ProjectsService struct {
	client *Client
}

// ListAll returns all projects
func (p *ProjectsService) ListAll(ctx context.Context) ([]Project, error) {
	req, err := p.client.newRequest(ctx, "GET", "/projects", nil)
	if err != nil {
		return nil, err
	}

	var response ProjectsResponse
	if err := p.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// ListByAccountID returns all projects filtered by account_id
func (p *ProjectsService) ListByAccountID(ctx context.Context, accountID string) ([]Project, error) {
	path := fmt.Sprintf("/projects?account_id=%s", accountID)

	req, err := p.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response ProjectsResponse
	if err := p.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// Get returns a single project by ID
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

// Create creates a new project with the given parameters
func (p *ProjectsService) Create(ctx context.Context, accountID string, req ProjectRequest) (*Project, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account ID cannot be empty")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

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

// Update updates an existing project with the given parameters
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

// Delete deletes a project by ID
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

// GetAllOccurrenceCounts gets occurrence counts for all projects
func (p *ProjectsService) GetAllOccurrenceCounts(ctx context.Context, options ProjectGetOccurrenceCountsOptions) (ProjectGetAllOccurrenceCountsResponse, error) {
	path := "/projects/occurrences"

	// Add query parameters if provided
	queryParams := make([]string, 0)
	if options.Period != "" {
		queryParams = append(queryParams, fmt.Sprintf("period=%s", options.Period))
	}
	if options.Environment != "" {
		queryParams = append(queryParams, fmt.Sprintf("environment=%s", options.Environment))
	}

	if len(queryParams) > 0 {
		path += "?" + fmt.Sprintf("%s", queryParams[0])
		for _, param := range queryParams[1:] {
			path += "&" + param
		}
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

// GetOccurrenceCounts gets occurrence counts for a specific project
func (p *ProjectsService) GetOccurrenceCounts(ctx context.Context, projectID int, options ProjectGetOccurrenceCountsOptions) (ProjectGetOccurrenceCountsResponse, error) {
	path := fmt.Sprintf("/projects/%d/occurrences", projectID)

	// Add query parameters if provided
	queryParams := make([]string, 0)
	if options.Period != "" {
		queryParams = append(queryParams, fmt.Sprintf("period=%s", options.Period))
	}
	if options.Environment != "" {
		queryParams = append(queryParams, fmt.Sprintf("environment=%s", options.Environment))
	}

	if len(queryParams) > 0 {
		path += "?" + fmt.Sprintf("%s", queryParams[0])
		for _, param := range queryParams[1:] {
			path += "&" + param
		}
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

// GetIntegrations gets all integrations for a specific project
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

// GetReport gets report data for a specific project
func (p *ProjectsService) GetReport(ctx context.Context, projectID int, reportType ProjectReportType, options ProjectGetReportOptions) ([][]interface{}, error) {
	path := fmt.Sprintf("/projects/%d/reports/%s", projectID, reportType)

	// Add query parameters if provided
	queryParams := make([]string, 0)
	if options.Start != "" {
		queryParams = append(queryParams, fmt.Sprintf("start=%s", options.Start))
	}
	if options.Stop != "" {
		queryParams = append(queryParams, fmt.Sprintf("stop=%s", options.Stop))
	}
	if options.Environment != "" {
		queryParams = append(queryParams, fmt.Sprintf("environment=%s", options.Environment))
	}

	if len(queryParams) > 0 {
		path += "?" + queryParams[0]
		for _, param := range queryParams[1:] {
			path += "&" + param
		}
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
