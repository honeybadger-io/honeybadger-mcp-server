package hbapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://api.example.com"
	apiToken := "test-token"

	client := NewClient().
		WithBaseURL(baseURL).
		WithAuthToken(apiToken)

	if client.baseURL != baseURL {
		t.Errorf("expected baseURL %s, got %s", baseURL, client.baseURL)
	}

	if client.apiToken != apiToken {
		t.Errorf("expected apiToken %s, got %s", apiToken, client.apiToken)
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", client.httpClient.Timeout)
	}
}

func TestNewRequest(t *testing.T) {
	client := NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	tests := []struct {
		name     string
		method   string
		path     string
		body     interface{}
		wantURL  string
		wantAuth string
	}{
		{
			name:     "GET request without body",
			method:   "GET",
			path:     "/projects",
			body:     nil,
			wantURL:  "https://api.example.com/v2/projects",
			wantAuth: "test-token",
		},
		{
			name:   "POST request with body",
			method: "POST",
			path:   "/projects",
			body: map[string]string{
				"name": "test-project",
			},
			wantURL:  "https://api.example.com/v2/projects",
			wantAuth: "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := client.newRequest(context.Background(), tt.method, tt.path, tt.body)
			if err != nil {
				t.Fatalf("newRequest() error = %v", err)
			}

			if req.URL.String() != tt.wantURL {
				t.Errorf("expected URL %s, got %s", tt.wantURL, req.URL.String())
			}

			// Check Basic Auth
			username, password, ok := req.BasicAuth()
			if !ok {
				t.Error("expected Basic Auth to be set")
			}
			if username != tt.wantAuth {
				t.Errorf("expected Basic Auth username %s, got %s", tt.wantAuth, username)
			}
			if password != "" {
				t.Errorf("expected Basic Auth password to be empty, got %s", password)
			}

			if req.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
			}

			if req.Header.Get("Accept") != "application/json" {
				t.Errorf("expected Accept application/json, got %s", req.Header.Get("Accept"))
			}
		})
	}
}

func TestDo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check Basic Auth
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic Auth to be set")
		}
		if username != "test-token" {
			t.Errorf("expected Basic Auth username test-token, got %s", username)
		}
		if password != "" {
			t.Errorf("expected Basic Auth password to be empty, got %s", password)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":   "123",
			"name": "test-project",
		})
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	req, err := client.newRequest(context.Background(), "GET", "/projects/123", nil)
	if err != nil {
		t.Fatalf("newRequest() error = %v", err)
	}

	var result map[string]string
	err = client.do(context.Background(), req, &result)
	if err != nil {
		t.Fatalf("do() error = %v", err)
	}

	if result["id"] != "123" {
		t.Errorf("expected id 123, got %s", result["id"])
	}

	if result["name"] != "test-project" {
		t.Errorf("expected name test-project, got %s", result["name"])
	}
}

func TestDo_ErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectedMsg  string
	}{
		{
			name:         "400 Bad Request with JSON error",
			statusCode:   400,
			responseBody: `{"error": "Invalid request"}`,
			expectedMsg:  "Invalid request",
		},
		{
			name:         "401 Unauthorized with message",
			statusCode:   401,
			responseBody: `{"message": "Invalid API key"}`,
			expectedMsg:  "Invalid API key",
		},
		{
			name:         "404 Not Found",
			statusCode:   404,
			responseBody: `{"error": "Project not found"}`,
			expectedMsg:  "Project not found",
		},
		{
			name:         "500 Internal Server Error",
			statusCode:   500,
			responseBody: `{"error": "Internal server error"}`,
			expectedMsg:  "Internal server error",
		},
		{
			name:         "503 Service Unavailable with no body",
			statusCode:   503,
			responseBody: ``,
			expectedMsg:  "503 Service Unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient().
				WithBaseURL(server.URL).
				WithAuthToken("test-token")

			req, err := client.newRequest(context.Background(), "GET", "/test", nil)
			if err != nil {
				t.Fatalf("newRequest() error = %v", err)
			}

			err = client.do(context.Background(), req, nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("expected APIError, got %T", err)
			}

			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, apiErr.StatusCode)
			}

			if apiErr.Message != tt.expectedMsg {
				t.Errorf("expected error message %s, got %s", tt.expectedMsg, apiErr.Message)
			}
		})
	}
}

func TestDo_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	client.httpClient.Timeout = 50 * time.Millisecond

	req, err := client.newRequest(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("newRequest() error = %v", err)
	}

	err = client.do(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("expected timeout error to contain 'request failed', got %s", err.Error())
	}
}

func TestNewRequest_InvalidBody(t *testing.T) {
	client := NewClient().
		WithBaseURL("https://api.example.com").
		WithAuthToken("test-token")

	invalidBody := make(chan int)

	_, err := client.newRequest(context.Background(), "POST", "/test", invalidBody)
	if err == nil {
		t.Fatal("expected error for invalid body, got nil")
	}

	if !strings.Contains(err.Error(), "failed to marshal request body") {
		t.Errorf("expected marshal error, got %s", err.Error())
	}
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		StatusCode: 400,
		Message:    "Bad Request",
		Body:       map[string]interface{}{"field": "name"},
	}

	expected := "HTTP 400: Bad Request"
	if err.Error() != expected {
		t.Errorf("expected error string %s, got %s", expected, err.Error())
	}
}
