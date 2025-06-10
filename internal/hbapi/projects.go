package hbapi

import (
	"encoding/json"
	"fmt"
)

// ListProjects returns all projects as raw JSON
func (c *Client) ListProjects() (json.RawMessage, error) {
	req, err := c.newRequest("GET", "/v2/projects", nil)
	if err != nil {
		return nil, err
	}

	var result json.RawMessage
	if err := c.do(req, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetProject returns a single project by ID as raw JSON
func (c *Client) GetProject(id string) (json.RawMessage, error) {
	if id == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	path := fmt.Sprintf("/v2/projects/%s", id)
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result json.RawMessage
	if err := c.do(req, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// CreateProject creates a new project with the given name
func (c *Client) CreateProject(name string) (json.RawMessage, error) {
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	body := map[string]interface{}{
		"project": map[string]interface{}{
			"name": name,
		},
	}

	req, err := c.newRequest("POST", "/v2/projects", body)
	if err != nil {
		return nil, err
	}

	var result json.RawMessage
	if err := c.do(req, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateProject updates an existing project with the given updates
func (c *Client) UpdateProject(id string, updates map[string]interface{}) (json.RawMessage, error) {
	if id == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}
	if updates == nil || len(updates) == 0 {
		return nil, fmt.Errorf("updates cannot be empty")
	}

	body := map[string]interface{}{
		"project": updates,
	}

	path := fmt.Sprintf("/v2/projects/%s", id)
	req, err := c.newRequest("PUT", path, body)
	if err != nil {
		return nil, err
	}

	var result json.RawMessage
	if err := c.do(req, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DeleteProject deletes a project by ID
func (c *Client) DeleteProject(id string) error {
	if id == "" {
		return fmt.Errorf("project ID cannot be empty")
	}

	path := fmt.Sprintf("/v2/projects/%s", id)
	req, err := c.newRequest("DELETE", path, nil)
	if err != nil {
		return err
	}

	return c.do(req, nil)
}