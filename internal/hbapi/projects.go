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
func (p *ProjectsService) Create(ctx context.Context, req ProjectRequest) (*Project, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	body := map[string]interface{}{
		"project": req,
	}

	httpReq, err := p.client.newRequest(ctx, "POST", "/projects", body)
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

// Delete deletes a project by ID
func (p *ProjectsService) Delete(ctx context.Context, id int) error {

	path := fmt.Sprintf("/projects/%d", id)
	req, err := p.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return p.client.do(ctx, req, nil)
}
