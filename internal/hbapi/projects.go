package hbapi

import (
	"fmt"
)

// ListProjects returns all projects, optionally filtered by account_id
func (c *Client) ListProjects(accountID string) ([]Project, error) {
	path := "/projects"
	if accountID != "" {
		path = fmt.Sprintf("/projects?account_id=%s", accountID)
	}

	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response ProjectsResponse
	if err := c.do(req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// GetProject returns a single project by ID
func (c *Client) GetProject(id string) (*Project, error) {
	if id == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	path := fmt.Sprintf("/projects/%s", id)
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result Project
	if err := c.do(req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateProject creates a new project with the given name
func (c *Client) CreateProject(name string) (*Project, error) {
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	body := map[string]interface{}{
		"project": map[string]interface{}{
			"name": name,
		},
	}

	req, err := c.newRequest("POST", "/projects", body)
	if err != nil {
		return nil, err
	}

	var result Project
	if err := c.do(req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateProject updates an existing project with the given updates
func (c *Client) UpdateProject(id string, updates map[string]interface{}) (*Project, error) {
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
	req, err := c.newRequest("PUT", path, body)
	if err != nil {
		return nil, err
	}

	var result Project
	if err := c.do(req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteProject deletes a project by ID
func (c *Client) DeleteProject(id string) error {
	if id == "" {
		return fmt.Errorf("project ID cannot be empty")
	}

	path := fmt.Sprintf("/projects/%s", id)
	req, err := c.newRequest("DELETE", path, nil)
	if err != nil {
		return err
	}

	return c.do(req, nil)
}