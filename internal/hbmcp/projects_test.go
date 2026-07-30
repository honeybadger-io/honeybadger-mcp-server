package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/mark3labs/mcp-go/mcp"
)

// getResultText pulls the text out of a tool result. Shared by every tool test.
func getResultText(result *mcp.CallToolResult) string {
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return textContent.Text
		}
		return fmt.Sprintf("%v", result.Content[0])
	}
	return ""
}

func projectArgs(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

func TestHandleListProjects(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		// No account id given, so the credential's own account is used.
		if want := "/v3/accounts/me/projects"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{
			"data": [
				{"id": "Xk9mZp", "account_id": "Ab3kL9", "name": "Production", "active": true,
				 "token": "tok_1", "fault_count": 12, "unresolved_fault_count": 3},
				{"id": "Nm8pQx", "account_id": "Ab3kL9", "name": "Staging", "active": false}
			],
			"pagination": {"page": 1, "per_page": 25, "total_count": 2, "total_pages": 1}
		}`)
	})

	result, err := handleListProjects(context.Background(), client, projectArgs(nil))
	if err != nil {
		t.Fatalf("handleListProjects() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}

	var got projectSummaryResponse
	if err := json.Unmarshal([]byte(getResultText(result)), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(got.Results) != 2 {
		t.Fatalf("got %d projects, want 2", len(got.Results))
	}
	if got.Results[0].ID != "Xk9mZp" {
		t.Errorf("first id = %q, want the opaque string id", got.Results[0].ID)
	}
	if got.Results[0].FaultCount != 12 {
		t.Errorf("fault_count = %d, want 12", got.Results[0].FaultCount)
	}
	// The second project omitted the counts; absent must read as zero, not panic.
	if got.Results[1].FaultCount != 0 {
		t.Errorf("absent fault_count = %d, want 0", got.Results[1].FaultCount)
	}
}

// The summary deliberately drops the large nested arrays that would blow the
// token budget. get_project is where the full record lives.
func TestHandleListProjectsOmitsHeavyFields(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusOK, `{"data":[{"id":"Xk9mZp","account_id":"Ab3kL9","name":"P","active":true,
			"environments":["production","staging"],"sites":[{"id":"s1"}],"teams":[{"id":"t1"}]}]}`)
	})

	result, err := handleListProjects(context.Background(), client, projectArgs(nil))
	if err != nil {
		t.Fatalf("handleListProjects() error = %v", err)
	}
	text := getResultText(result)
	for _, heavy := range []string{"environments", "sites", "teams"} {
		if strings.Contains(text, heavy) {
			t.Errorf("summary leaked the %q array", heavy)
		}
	}
}

func TestHandleListProjects_WithAccountID(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/Ab3kL9/projects"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{"data":[]}`)
	})

	result, err := handleListProjects(context.Background(), client,
		projectArgs(map[string]interface{}{"account_id": "Ab3kL9"}))
	if err != nil {
		t.Fatalf("handleListProjects() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
}

// A credential covering several accounts cannot use the `me` sentinel. The
// handler must recover by asking which account it belongs to, then retrying.
func TestHandleListProjectsRecoversFromAmbiguousAccount(t *testing.T) {
	var paths []string
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v3/accounts/me/projects":
			v3JSON(w, http.StatusUnprocessableEntity,
				`{"error":{"code":"ambiguous_account","message":"\"me\" is ambiguous"}}`)
		case "/v3/token":
			v3JSON(w, http.StatusOK, `{"data":{"kind":"oauth","account_id":"Ab3kL9","scopes":["projects:read"]}}`)
		case "/v3/accounts/Ab3kL9/projects":
			v3JSON(w, http.StatusOK, `{"data":[{"id":"Xk9mZp","account_id":"Ab3kL9","name":"P","active":true}]}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	})

	result, err := handleListProjects(context.Background(), client, projectArgs(nil))
	if err != nil {
		t.Fatalf("handleListProjects() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected recovery, got %s", getResultText(result))
	}
	if len(paths) != 3 || paths[1] != "/v3/token" {
		t.Errorf("request sequence = %v, want the ambiguous call, introspection, then a retry", paths)
	}
}

// If introspection also fails, the caller must see the original problem rather
// than a confusing error about the recovery attempt.
func TestHandleListProjectsReportsOriginalErrorWhenIntrospectionFails(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v3/token" {
			v3JSON(w, http.StatusInternalServerError, `{"error":{"code":"service_unavailable","message":"down"}}`)
			return
		}
		v3JSON(w, http.StatusUnprocessableEntity,
			`{"error":{"code":"ambiguous_account","message":"\"me\" is ambiguous"}}`)
	})

	result, err := handleListProjects(context.Background(), client, projectArgs(nil))
	if err != nil {
		t.Fatalf("handleListProjects() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected an error result")
	}
	if !strings.Contains(getResultText(result), "ambiguous_account") {
		t.Errorf("error = %q, want the original ambiguous_account", getResultText(result))
	}
}

func TestHandleListProjects_Error(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusUnauthorized, `{"error":{"code":"unauthorized","message":"Invalid token"}}`)
	})

	result, err := handleListProjects(context.Background(), client, projectArgs(nil))
	if err != nil {
		t.Fatalf("handleListProjects() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	if !strings.Contains(getResultText(result), "Failed to list projects") {
		t.Errorf("error = %q", getResultText(result))
	}
}

func TestHandleGetProject(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"Production","active":true}}`)
	})

	result, err := handleGetProject(context.Background(), client,
		projectArgs(map[string]interface{}{"id": "Xk9mZp"}))
	if err != nil {
		t.Fatalf("handleGetProject() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if !strings.Contains(getResultText(result), "Production") {
		t.Errorf("result = %q", getResultText(result))
	}
}

func TestHandleGetProject_MissingID(t *testing.T) {
	result, err := handleGetProject(context.Background(), offlineV3Client(), projectArgs(nil))
	if err != nil {
		t.Fatalf("handleGetProject() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing id")
	}
	if !strings.Contains(getResultText(result), "id is required") {
		t.Errorf("error = %q", getResultText(result))
	}
}

func TestHandleGetProject_NotFound(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusNotFound, `{"error":{"code":"not_found","message":"Resource not found"}}`)
	})

	result, err := handleGetProject(context.Background(), client,
		projectArgs(map[string]interface{}{"id": "nope"}))
	if err != nil {
		t.Fatalf("handleGetProject() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
}

func TestHandleCreateProject(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusCreated, `{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"New","active":true}}`)
	})

	result, err := handleCreateProject(context.Background(), client,
		projectArgs(map[string]interface{}{"name": "New"}))
	if err != nil {
		t.Fatalf("handleCreateProject() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if body["name"] != "New" {
		t.Errorf("sent body = %v", body)
	}
}

func TestHandleCreateProject_MissingName(t *testing.T) {
	result, err := handleCreateProject(context.Background(), offlineV3Client(), projectArgs(nil))
	if err != nil {
		t.Fatalf("handleCreateProject() error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "name is required") {
		t.Errorf("expected 'name is required', got %q", getResultText(result))
	}
}

func TestHandleUpdateProject(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusOK, `{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"Renamed","active":true}}`)
	})

	result, err := handleUpdateProject(context.Background(), client,
		projectArgs(map[string]interface{}{"id": "Xk9mZp", "name": "Renamed"}))
	if err != nil {
		t.Fatalf("handleUpdateProject() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if body["name"] != "Renamed" {
		t.Errorf("sent body = %v", body)
	}
}

func TestHandleUpdateProject_MissingID(t *testing.T) {
	result, err := handleUpdateProject(context.Background(), offlineV3Client(),
		projectArgs(map[string]interface{}{"name": "Renamed"}))
	if err != nil {
		t.Fatalf("handleUpdateProject() error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "id is required") {
		t.Errorf("expected 'id is required', got %q", getResultText(result))
	}
}

func TestHandleDeleteProject(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if want := "/v3/accounts/me/projects/Xk9mZp"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		// A delete answers 204 with no body.
		w.WriteHeader(http.StatusNoContent)
	})

	result, err := handleDeleteProject(context.Background(), client,
		projectArgs(map[string]interface{}{"id": "Xk9mZp"}))
	if err != nil {
		t.Fatalf("handleDeleteProject() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if !strings.Contains(getResultText(result), "Xk9mZp") {
		t.Errorf("result should name what was deleted, got %q", getResultText(result))
	}
}

func TestHandleDeleteProject_Error(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusForbidden,
			`{"error":{"code":"insufficient_scope","message":"Insufficient scope",
			  "details":{"required_scope":"projects:write","token_scopes":["projects:read"]}}}`)
	})

	result, err := handleDeleteProject(context.Background(), client,
		projectArgs(map[string]interface{}{"id": "Xk9mZp"}))
	if err != nil {
		t.Fatalf("handleDeleteProject() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	// The scope the credential lacks is the actionable part.
	if !strings.Contains(getResultText(result), "projects:write") {
		t.Errorf("error should name the missing scope, got %q", getResultText(result))
	}
}

func TestHandleGetProjectReport(t *testing.T) {
	mockResponse := `[["RuntimeError", 8347], ["SocketError", 4651]]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/reports/notices_by_class" {
			t.Errorf("expected path /v2/projects/123/reports/notices_by_class, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": float64(123),
				"report":     "notices_by_class",
			},
		},
	}

	result, err := handleGetProjectReport(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetProjectReport() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}
	resultText := getResultText(result)
	if !strings.Contains(resultText, "RuntimeError") {
		t.Error("Result should contain RuntimeError")
	}
}

func TestHandleGetProjectReport_InvalidReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors": "Invalid report type"}`))
	}))
	defer server.Close()

	client := hbapi.NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id": float64(123),
				"report":     "invalid_report_type",
			},
		},
	}

	result, err := handleGetProjectReport(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetProjectReport() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid report type")
	}
	resultText := getResultText(result)
	if !strings.Contains(resultText, "Failed to get project report") {
		t.Error("Error message should contain 'Failed to get project report'")
	}
}

func TestHandleGetProjectReport_WithOptions(t *testing.T) {
	mockResponse := `[["inquiries#create", 2904]]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/projects/456/reports/notices_by_location" {
			t.Errorf("expected path /v2/projects/456/reports/notices_by_location, got %s", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("start") != "2023-01-01T00:00:00Z" {
			t.Errorf("expected start=2023-01-01T00:00:00Z, got %s", query.Get("start"))
		}
		if query.Get("stop") != "2023-01-31T23:59:59Z" {
			t.Errorf("expected stop=2023-01-31T23:59:59Z, got %s", query.Get("stop"))
		}
		if query.Get("environment") != "production" {
			t.Errorf("expected environment=production, got %s", query.Get("environment"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := hbapi.NewClient().WithBaseURL(server.URL).WithAuthToken("test-token")
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"project_id":  float64(456),
				"report":      "notices_by_location",
				"start":       "2023-01-01T00:00:00Z",
				"stop":        "2023-01-31T23:59:59Z",
				"environment": "production",
			},
		},
	}

	result, err := handleGetProjectReport(context.Background(), client, req)
	if err != nil {
		t.Fatalf("handleGetProjectReport() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error: %s", getResultText(result))
	}
}

// The project write schema carries every setting v2 accepted, so they must reach
// the wire rather than being refused.
func TestHandleUpdateProjectSendsSettings(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusOK, `{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"App","active":true}}`)
	})

	result, err := handleUpdateProject(context.Background(), client, projectArgs(map[string]interface{}{
		"id":                       "Xk9mZp",
		"name":                     "App",
		"purge_days":               float64(30),
		"user_url":                 "http://example.com/users/[user_id]",
		"resolve_errors_on_deploy": true,
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}

	for key, want := range map[string]any{
		"name":                     "App",
		"purge_days":               float64(30),
		"user_url":                 "http://example.com/users/[user_id]",
		"resolve_errors_on_deploy": true,
	} {
		if body[key] != want {
			t.Errorf("%s = %v, want %v", key, body[key], want)
		}
	}
}

// Turning a setting off is a real request, and false must survive to the wire —
// the typed getter cannot tell false from absent, which is why the handler reads
// the raw arguments.
func TestHandleUpdateProjectSendsFalseSettings(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusOK, `{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"App","active":true}}`)
	})

	if _, err := handleUpdateProject(context.Background(), client, projectArgs(map[string]interface{}{
		"id": "Xk9mZp", "name": "App", "disable_public_links": false,
	})); err != nil {
		t.Fatalf("error = %v", err)
	}

	v, present := body["disable_public_links"]
	if !present {
		t.Fatal("disable_public_links was dropped; false is a value, not an absence")
	}
	if v != false {
		t.Errorf("disable_public_links = %v, want false", v)
	}
}

// Settings the caller did not mention must stay absent, so an update does not
// blank them.
func TestHandleUpdateProjectOmitsUnmentionedSettings(t *testing.T) {
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		v3JSON(w, http.StatusOK, `{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"App","active":true}}`)
	})

	if _, err := handleUpdateProject(context.Background(), client, projectArgs(map[string]interface{}{
		"id": "Xk9mZp", "name": "App",
	})); err != nil {
		t.Fatalf("error = %v", err)
	}

	for _, absent := range []string{"purge_days", "user_url", "source_url", "disable_public_links"} {
		if _, present := body[absent]; present {
			t.Errorf("%q was sent unmentioned; it would overwrite the current value", absent)
		}
	}
}

// The update body is the same schema as create, with name required, so the tool
// says so rather than letting the API reject it.
func TestHandleUpdateProjectRequiresName(t *testing.T) {
	result, err := handleUpdateProject(context.Background(), offlineV3Client(),
		projectArgs(map[string]interface{}{"id": "Xk9mZp", "purge_days": float64(30)}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "name is required") {
		t.Errorf("got %q", getResultText(result))
	}
}
