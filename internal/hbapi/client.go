package hbapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client

	// Resource services
	Projects *ProjectsService
	Faults   *FaultsService
}

func NewClient() *Client {
	c := &Client{
		baseURL: "https://api.honeybadger.io", // Default base URL
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Initialize resource services
	c.Projects = &ProjectsService{client: c}
	c.Faults = &FaultsService{client: c}
	return c
}

// WithAuthToken sets the API token for the client
func (c *Client) WithAuthToken(apiToken string) *Client {
	c.apiToken = apiToken
	return c
}

// WithBaseURL sets the base URL for the client
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = baseURL
	return c
}

// WithHTTPClient sets a custom HTTP client
func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	c.httpClient = httpClient
	return c
}

// ProjectsAPI returns the projects service
func (c *Client) ProjectsAPI() *ProjectsService {
	return c.Projects
}

func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	// Add /v2 prefix to all paths
	url := fmt.Sprintf("%s/v2%s", c.baseURL, path)

	var buf io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		buf = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set HTTP Basic Auth with token as username, no password
	req.SetBasicAuth(c.apiToken, "")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func (c *Client) do(ctx context.Context, req *http.Request, v interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check if the error is due to context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return WrapError(resp, nil)
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
