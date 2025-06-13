package hbapi

import (
	"context"
	"fmt"
)

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
func (p *ProjectsService) ListByAccountID(ctx context.Context, accountID int) ([]Project, error) {
	path := fmt.Sprintf("/projects?account_id=%d", accountID)

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
func (p *ProjectsService) Get(ctx context.Context, id string) (*Project, error) {
	if id == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	path := fmt.Sprintf("/projects/%s", id)
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

// Create creates a new project with the given name
func (p *ProjectsService) Create(ctx context.Context, name string) (*Project, error) {
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	body := map[string]interface{}{
		"project": map[string]interface{}{
			"name": name,
		},
	}

	req, err := p.client.newRequest(ctx, "POST", "/projects", body)
	if err != nil {
		return nil, err
	}

	var result Project
	if err := p.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Update updates an existing project with the given updates
func (p *ProjectsService) Update(ctx context.Context, id string, updates map[string]interface{}) (*Project, error) {
	if id == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}
	if updates == nil || len(updates) == 0 {
		return nil, fmt.Errorf("updates cannot be empty")
	}

	body := map[string]interface{}{
		"project": updates,
	}

	path := fmt.Sprintf("/projects/%s", id)
	req, err := p.client.newRequest(ctx, "PUT", path, body)
	if err != nil {
		return nil, err
	}

	var result Project
	if err := p.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete deletes a project by ID
func (p *ProjectsService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("project ID cannot be empty")
	}

	path := fmt.Sprintf("/projects/%s", id)
	req, err := p.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return p.client.do(ctx, req, nil)
}
