package hbapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type APIError struct {
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message"`
	Body       interface{} `json:"body,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

func WrapError(resp *http.Response, err error) error {
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}

	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Message:    resp.Status,
	}

	if resp.Body != nil {
		body, readErr := io.ReadAll(resp.Body)
		if readErr == nil && len(body) > 0 {
			var jsonBody interface{}
			if json.Unmarshal(body, &jsonBody) == nil {
				apiErr.Body = jsonBody

				// Try to extract a more specific message
				if bodyMap, ok := jsonBody.(map[string]interface{}); ok {
					if msg, ok := bodyMap["message"].(string); ok {
						apiErr.Message = msg
					} else if msg, ok := bodyMap["errors"].(string); ok {
						apiErr.Message = msg
					}
				}
			} else {
				// If not JSON, store as string
				apiErr.Body = string(body)
				apiErr.Message = string(body)
			}
		}
	}

	return apiErr
}
