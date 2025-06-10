package hbapi

import (
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
	
	client := NewClient(baseURL, apiToken)
	
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
	client := NewClient("https://api.example.com", "test-token")
	
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
			wantURL:  "https://api.example.com/projects",
			wantAuth: "test-token",
		},
		{
			name:   "POST request with body",
			method: "POST",
			path:   "/projects",
			body: map[string]string{
				"name": "test-project",
			},
			wantURL:  "https://api.example.com/projects",
			wantAuth: "test-token",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := client.newRequest(tt.method, tt.path, tt.body)
			if err != nil {
				t.Fatalf("newRequest() error = %v", err)
			}
			
			if req.URL.String() != tt.wantURL {
				t.Errorf("expected URL %s, got %s", tt.wantURL, req.URL.String())
			}
			
			if req.Header.Get("X-API-Key") != tt.wantAuth {
				t.Errorf("expected X-API-Key %s, got %s", tt.wantAuth, req.Header.Get("X-API-Key"))
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
		if r.Header.Get("X-API-Key") != "test-token" {
			t.Errorf("expected X-API-Key test-token, got %s", r.Header.Get("X-API-Key"))
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"id":   "123",
			"name": "test-project",
		})
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "test-token")
	req, err := client.newRequest("GET", "/projects/123", nil)
	if err != nil {
		t.Fatalf("newRequest() error = %v", err)
	}
	
	var result map[string]string
	err = client.do(req, &result)
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

func TestDo_ErrorMapping(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedCode   string
		expectedMsg    string
	}{
		{
			name:         "400 Bad Request",
			statusCode:   400,
			responseBody: `{"error": "Invalid request"}`,
			expectedCode: "bad_request",
			expectedMsg:  "Invalid request",
		},
		{
			name:         "401 Unauthorized",
			statusCode:   401,
			responseBody: `{"message": "Invalid API key"}`,
			expectedCode: "unauthorized",
			expectedMsg:  "Invalid API key",
		},
		{
			name:         "404 Not Found",
			statusCode:   404,
			responseBody: `{"error": "Project not found"}`,
			expectedCode: "not_found",
			expectedMsg:  "Project not found",
		},
		{
			name:         "429 Rate Limited",
			statusCode:   429,
			responseBody: `{"message": "Rate limit exceeded"}`,
			expectedCode: "rate_limited",
			expectedMsg:  "Rate limit exceeded",
		},
		{
			name:         "500 Internal Server Error",
			statusCode:   500,
			responseBody: `{"error": "Internal server error"}`,
			expectedCode: "internal_error",
			expectedMsg:  "Internal server error",
		},
		{
			name:         "503 Service Unavailable",
			statusCode:   503,
			responseBody: `{}`,
			expectedCode: "internal_error",
			expectedMsg:  "HTTP 503: 503 Service Unavailable",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()
			
			client := NewClient(server.URL, "test-token")
			req, err := client.newRequest("GET", "/test", nil)
			if err != nil {
				t.Fatalf("newRequest() error = %v", err)
			}
			
			err = client.do(req, nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			
			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("expected APIError, got %T", err)
			}
			
			if apiErr.Code != tt.expectedCode {
				t.Errorf("expected error code %s, got %s", tt.expectedCode, apiErr.Code)
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
	
	client := NewClient(server.URL, "test-token")
	client.httpClient.Timeout = 50 * time.Millisecond
	
	req, err := client.newRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("newRequest() error = %v", err)
	}
	
	err = client.do(req, nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("expected timeout error to contain 'request failed', got %s", err.Error())
	}
}

func TestNewRequest_InvalidBody(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	
	invalidBody := make(chan int)
	
	_, err := client.newRequest("POST", "/test", invalidBody)
	if err == nil {
		t.Fatal("expected error for invalid body, got nil")
	}
	
	if !strings.Contains(err.Error(), "failed to marshal request body") {
		t.Errorf("expected marshal error, got %s", err.Error())
	}
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		Code:    "bad_request",
		Message: "Invalid input",
		Details: map[string]interface{}{"field": "name"},
	}
	
	expected := "API error [bad_request]: Invalid input"
	if err.Error() != expected {
		t.Errorf("expected error string %s, got %s", expected, err.Error())
	}
}

func TestMapStatusToErrorCode(t *testing.T) {
	tests := []struct {
		statusCode   int
		expectedCode string
	}{
		{400, "bad_request"},
		{401, "unauthorized"},
		{404, "not_found"},
		{429, "rate_limited"},
		{500, "internal_error"},
		{501, "internal_error"},
		{502, "internal_error"},
		{503, "internal_error"},
		{504, "internal_error"},
		{505, "internal_error"},
		{403, "bad_request"}, // Other 4xx
		{418, "bad_request"}, // Other 4xx
		{599, "internal_error"}, // Other 5xx
	}
	
	for _, tt := range tests {
		t.Run(string(rune(tt.statusCode)), func(t *testing.T) {
			code := mapStatusToErrorCode(tt.statusCode)
			if code != tt.expectedCode {
				t.Errorf("status %d: expected %s, got %s", tt.statusCode, tt.expectedCode, code)
			}
		})
	}
}