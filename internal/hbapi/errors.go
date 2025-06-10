package hbapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error [%s]: %s", e.Code, e.Message)
}

func WrapError(resp *http.Response, err error) error {
	if err != nil {
		return &APIError{
			Code:    "internal_error",
			Message: fmt.Sprintf("HTTP request failed: %v", err),
		}
	}
	
	apiErr := &APIError{
		Code:    mapStatusToErrorCode(resp.StatusCode),
		Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
	}
	
	if resp.Body != nil {
		body, readErr := io.ReadAll(resp.Body)
		if readErr == nil && len(body) > 0 {
			var errorResponse map[string]interface{}
			if json.Unmarshal(body, &errorResponse) == nil {
				if msg, ok := errorResponse["message"].(string); ok {
					apiErr.Message = msg
				} else if msg, ok := errorResponse["error"].(string); ok {
					apiErr.Message = msg
				}
				apiErr.Details = errorResponse
			}
		}
	}
	
	return apiErr
}

func mapStatusToErrorCode(statusCode int) string {
	switch statusCode {
	case 400:
		return "bad_request"
	case 401:
		return "unauthorized"
	case 404:
		return "not_found"
	case 429:
		return "rate_limited"
	case 500, 501, 502, 503, 504, 505:
		return "internal_error"
	default:
		if statusCode >= 400 && statusCode < 500 {
			return "bad_request"
		}
		return "internal_error"
	}
}