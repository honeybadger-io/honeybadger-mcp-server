package hbapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInsightsQuery(t *testing.T) {
	mockResponse := `{
		"results": [
			{"ts": "2024-01-01T00:00:00Z", "count": 10, "name": "web"},
			{"ts": "2024-01-01T01:00:00Z", "count": 15, "name": "api"}
		],
		"meta": {
			"query": "stats count() by event_type::str",
			"fields": ["ts", "count", "name"],
			"schema": [
				{"name": "ts", "type": "DateTime"},
				{"name": "count", "type": "UInt64"},
				{"name": "name", "type": "String"}
			],
			"rows": 2,
			"total_rows": 2,
			"start_at": "2024-01-01T00:00:00Z",
			"end_at": "2024-01-01T03:00:00Z"
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/insights/queries" {
			t.Errorf("expected path /v2/projects/123/insights/queries, got %s", r.URL.Path)
		}
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
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	response, err := client.Insights.Query(context.Background(), 123, InsightsQueryRequest{
		Query: "stats count() by event_type::str",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(response.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(response.Results))
	}

	if response.Meta.Query != "stats count() by event_type::str" {
		t.Errorf("expected query in meta, got %s", response.Meta.Query)
	}

	if len(response.Meta.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(response.Meta.Fields))
	}

	if response.Meta.Rows != 2 {
		t.Errorf("expected 2 rows, got %d", response.Meta.Rows)
	}

	if response.Meta.TotalRows != 2 {
		t.Errorf("expected 2 total rows, got %d", response.Meta.TotalRows)
	}
}

func TestInsightsQuery_WithAllOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [],
			"meta": {
				"query": "fields @ts, message::str",
				"fields": [],
				"schema": [],
				"rows": 0,
				"total_rows": 0,
				"start_at": "2024-01-01T00:00:00Z",
				"end_at": "2024-01-07T00:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	response, err := client.Insights.Query(context.Background(), 123, InsightsQueryRequest{
		Query:    "fields @ts, message::str",
		Ts:       "week",
		Timezone: "America/New_York",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if response.Meta.Query != "fields @ts, message::str" {
		t.Errorf("expected query in meta, got %s", response.Meta.Query)
	}
}

func TestInsightsQuery_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors": "Invalid API token"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("invalid-token")

	_, err := client.Insights.Query(context.Background(), 123, InsightsQueryRequest{
		Query: "stats count()",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 401 {
		t.Errorf("expected status code 401, got %d", apiErr.StatusCode)
	}
}

func TestInsightsQuery_ProjectNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors": "Project not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Insights.Query(context.Background(), 999, InsightsQueryRequest{
		Query: "stats count()",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 404 {
		t.Errorf("expected status code 404, got %d", apiErr.StatusCode)
	}
}

func TestInsightsQuery_InvalidQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors": "Invalid query syntax"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Insights.Query(context.Background(), 123, InsightsQueryRequest{
		Query: "INVALID QUERY",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 422 {
		t.Errorf("expected status code 422, got %d", apiErr.StatusCode)
	}
}
