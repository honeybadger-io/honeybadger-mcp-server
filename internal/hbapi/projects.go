package hbapi

import (
	"fmt"
)

// ListProjects returns all projects as JSON array, optionally filtered by account_id
func (c *Client) ListProjects(accountID string) ([]map[string]interface{}, error) {
	path := "/v2/projects"
	if accountID != "" {
		path = fmt.Sprintf("/v2/projects?account_id=%s", accountID)
	}

	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response map[string]interface{}
	if err := c.do(req, &response); err != nil {
		return nil, err
	}

	// Return the projects array from the "results" field
	if projects, ok := response["results"].([]interface{}); ok {
		// Convert []interface{} to []map[string]interface{} for type safety
		result := make([]map[string]interface{}, len(projects))
		for i, project := range projects {
			result[i] = project.(map[string]interface{})
		}
		return result, nil
	}

	// Return empty array if "results" field not found or wrong type
	return []map[string]interface{}{}, nil
}

// GetProject returns a single project by ID as JSON object
func (c *Client) GetProject(id string) (map[string]interface{}, error) {
	if id == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	path := fmt.Sprintf("/v2/projects/%s", id)
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := c.do(req, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// CreateProject creates a new project with the given name
func (c *Client) CreateProject(name string) (map[string]interface{}, error) {
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

	var result map[string]interface{}
	if err := c.do(req, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateProject updates an existing project with the given updates
func (c *Client) UpdateProject(id string, updates map[string]interface{}) (map[string]interface{}, error) {
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

	var result map[string]interface{}
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