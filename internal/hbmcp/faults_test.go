package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func faultArgs(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

func TestHandleListFaults(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/faults"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{
			"data": [{"id":"f1","project_id":"Xk9mZp","klass":"RuntimeError","message":"boom","notices_count":42}],
			"pagination": {"page":1,"per_page":25,"total_count":1,"total_pages":1}
		}`)
	})

	result, err := handleListFaults(context.Background(), client,
		faultArgs(map[string]interface{}{"project_id": "Xk9mZp"}))
	if err != nil {
		t.Fatalf("handleListFaults() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if !strings.Contains(getResultText(result), "RuntimeError") {
		t.Errorf("result = %q", getResultText(result))
	}
}

// q passes straight through; v3 expresses environment and status filters inside it.
func TestHandleListFaults_WithSearch(t *testing.T) {
	var gotQuery string
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		v3JSON(w, http.StatusOK, `{"data":[]}`)
	})

	_, err := handleListFaults(context.Background(), client, faultArgs(map[string]interface{}{
		"project_id": "Xk9mZp",
		"q":          "environment:production is:resolved",
	}))
	if err != nil {
		t.Fatalf("handleListFaults() error = %v", err)
	}
	if want := "environment:production is:resolved"; gotQuery != want {
		t.Errorf("q = %q, want %q", gotQuery, want)
	}
}

// limit maps to per_page: both cap how many faults one call returns.
func TestHandleListFaults_LimitMapsToPerPage(t *testing.T) {
	var query url.Values
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		v3JSON(w, http.StatusOK, `{"data":[]}`)
	})

	_, err := handleListFaults(context.Background(), client, faultArgs(map[string]interface{}{
		"project_id": "Xk9mZp",
		"limit":      10,
		"page":       3,
	}))
	if err != nil {
		t.Fatalf("handleListFaults() error = %v", err)
	}
	if got := query.Get("per_page"); got != "10" {
		t.Errorf("per_page = %q, want 10", got)
	}
	if got := query.Get("page"); got != "3" {
		t.Errorf("page = %q, want 3", got)
	}
}

// v3's listFaults takes only page, per_page and q. A caller passing a filter it
// cannot honour must be told, not silently given unfiltered results.
func TestHandleListFaultsRejectsFiltersV3Lacks(t *testing.T) {
	for _, field := range []string{"created_after", "occurred_after", "occurred_before", "order"} {
		result, err := handleListFaults(context.Background(), offlineV3Client(),
			faultArgs(map[string]interface{}{"project_id": "Xk9mZp", field: "whatever"}))
		if err != nil {
			t.Fatalf("%s: handleListFaults() error = %v", field, err)
		}
		if !result.IsError {
			t.Errorf("%s: accepted; results would have been unfiltered", field)
			continue
		}
		if !strings.Contains(getResultText(result), field) {
			t.Errorf("%s: error does not name the field: %q", field, getResultText(result))
		}
	}
}

func TestHandleListFaults_MissingProjectID(t *testing.T) {
	result, err := handleListFaults(context.Background(), offlineV3Client(), faultArgs(nil))
	if err != nil {
		t.Fatalf("handleListFaults() error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "project_id is required") {
		t.Errorf("expected 'project_id is required', got %q", getResultText(result))
	}
}

func TestHandleListFaults_Error(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusNotFound, `{"error":{"code":"not_found","message":"Resource not found"}}`)
	})

	result, err := handleListFaults(context.Background(), client,
		faultArgs(map[string]interface{}{"project_id": "nope"}))
	if err != nil {
		t.Fatalf("handleListFaults() error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "Failed to list faults") {
		t.Errorf("got %q", getResultText(result))
	}
}

func TestHandleGetFault(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/faults/f1"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK,
			`{"data":{"id":"f1","project_id":"Xk9mZp","klass":"RuntimeError","action":null}}`)
	})

	result, err := handleGetFault(context.Background(), client,
		faultArgs(map[string]interface{}{"project_id": "Xk9mZp", "fault_id": "f1"}))
	if err != nil {
		t.Fatalf("handleGetFault() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if !strings.Contains(getResultText(result), "RuntimeError") {
		t.Errorf("result = %q", getResultText(result))
	}
}

func TestHandleGetFault_MissingIDs(t *testing.T) {
	for _, tc := range []struct {
		args map[string]interface{}
		want string
	}{
		{nil, "project_id is required"},
		{map[string]interface{}{"project_id": "Xk9mZp"}, "fault_id is required"},
	} {
		result, err := handleGetFault(context.Background(), offlineV3Client(), faultArgs(tc.args))
		if err != nil {
			t.Fatalf("handleGetFault() error = %v", err)
		}
		if !result.IsError || !strings.Contains(getResultText(result), tc.want) {
			t.Errorf("args %v: got %q, want %q", tc.args, getResultText(result), tc.want)
		}
	}
}

func TestHandleGetFault_Error(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusNotFound, `{"error":{"code":"not_found","message":"Resource not found"}}`)
	})

	result, err := handleGetFault(context.Background(), client,
		faultArgs(map[string]interface{}{"project_id": "Xk9mZp", "fault_id": "nope"}))
	if err != nil {
		t.Fatalf("handleGetFault() error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "Failed to get fault") {
		t.Errorf("got %q", getResultText(result))
	}
}

// v3 replaced v2's mutable PUT with an endpoint per action, so resolving hits
// /faults/resolve with the fault in a list.
func TestHandleUpdateFaultResolve(t *testing.T) {
	var paths []string
	var body map[string]any
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusNoContent)
	})

	result, err := handleUpdateFault(context.Background(), client, faultArgs(map[string]interface{}{
		"project_id": "Xk9mZp", "fault_id": "f1", "resolved": true,
	}))
	if err != nil {
		t.Fatalf("handleUpdateFault() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if len(paths) != 1 || paths[0] != "/v3/accounts/me/projects/Xk9mZp/faults/resolve" {
		t.Errorf("paths = %v", paths)
	}
	ids, _ := body["fault_ids"].([]any)
	if len(ids) != 1 || ids[0] != "f1" {
		t.Errorf("fault_ids = %v", body["fault_ids"])
	}
}

// resolved:false is a different endpoint, not the same one with a flag.
func TestHandleUpdateFaultUnresolveAndUnignore(t *testing.T) {
	var paths []string
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := handleUpdateFault(context.Background(), client, faultArgs(map[string]interface{}{
		"project_id": "Xk9mZp", "fault_id": "f1", "resolved": false, "ignored": false,
	}))
	if err != nil {
		t.Fatalf("handleUpdateFault() error = %v", err)
	}
	want := []string{
		"/v3/accounts/me/projects/Xk9mZp/faults/unresolve",
		"/v3/accounts/me/projects/Xk9mZp/faults/unignore",
	}
	if len(paths) != 2 || paths[0] != want[0] || paths[1] != want[1] {
		t.Errorf("paths = %v, want %v", paths, want)
	}
}

// Both flags in one request means two calls, and the summary reports both.
func TestHandleUpdateFaultBothFlags(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	result, err := handleUpdateFault(context.Background(), client, faultArgs(map[string]interface{}{
		"project_id": "Xk9mZp", "fault_id": "f1", "resolved": true, "ignored": true,
	}))
	if err != nil {
		t.Fatalf("handleUpdateFault() error = %v", err)
	}

	var applied map[string]any
	if err := json.Unmarshal([]byte(getResultText(result)), &applied); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if applied["resolved"] != true || applied["ignored"] != true {
		t.Errorf("summary = %v, want both changes reported", applied)
	}
}

// assignee_id and resolve_on_deploy cannot be expressed in v3 — the assign
// endpoint does not specify its body, and resolve-on-deploy does not exist.
func TestHandleUpdateFaultRejectsUnsupportedFields(t *testing.T) {
	for field, value := range map[string]interface{}{
		"assignee_id":       12,
		"resolve_on_deploy": true,
	} {
		result, err := handleUpdateFault(context.Background(), offlineV3Client(),
			faultArgs(map[string]interface{}{
				"project_id": "Xk9mZp", "fault_id": "f1", "resolved": true, field: value,
			}))
		if err != nil {
			t.Fatalf("%s: handleUpdateFault() error = %v", field, err)
		}
		if !result.IsError {
			t.Errorf("%s: accepted; it would have been silently ignored", field)
			continue
		}
		if !strings.Contains(getResultText(result), field) {
			t.Errorf("%s: error does not name it: %q", field, getResultText(result))
		}
	}
}

// A non-boolean must be refused rather than coerced: the typed getter would turn
// null into false and quietly unresolve the fault.
func TestHandleUpdateFaultRejectsNonBoolean(t *testing.T) {
	result, err := handleUpdateFault(context.Background(), offlineV3Client(),
		faultArgs(map[string]interface{}{
			"project_id": "Xk9mZp", "fault_id": "f1", "resolved": "yes",
		}))
	if err != nil {
		t.Fatalf("handleUpdateFault() error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "must be a boolean") {
		t.Errorf("got %q, want a boolean type error", getResultText(result))
	}
}

func TestHandleUpdateFault_NoFields(t *testing.T) {
	result, err := handleUpdateFault(context.Background(), offlineV3Client(),
		faultArgs(map[string]interface{}{"project_id": "Xk9mZp", "fault_id": "f1"}))
	if err != nil {
		t.Fatalf("handleUpdateFault() error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "at least one of") {
		t.Errorf("got %q", getResultText(result))
	}
}

func TestHandleUpdateFault_Error(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusForbidden,
			`{"error":{"code":"insufficient_scope","message":"Insufficient scope",
			  "details":{"required_scope":"faults:write","token_scopes":["faults:read"]}}}`)
	})

	result, err := handleUpdateFault(context.Background(), client, faultArgs(map[string]interface{}{
		"project_id": "Xk9mZp", "fault_id": "f1", "resolved": true,
	}))
	if err != nil {
		t.Fatalf("handleUpdateFault() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected an error result")
	}
	if !strings.Contains(getResultText(result), "faults:write") {
		t.Errorf("error should name the missing scope, got %q", getResultText(result))
	}
}

func TestHandleListFaultNotices(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/faults/f1/notices"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit = %q, want 5", got)
		}
		// Notice ids are UUIDs, unlike every other v3 resource.
		v3JSON(w, http.StatusOK, `{
			"data":[{"id":"11111111-1111-4111-8111-111111111111","fault_id":"f1","project_id":"Xk9mZp"}],
			"pagination":{"has_older":false,"has_newer":false,"limit":5}
		}`)
	})

	result, err := handleListFaultNotices(context.Background(), client, faultArgs(map[string]interface{}{
		"project_id": "Xk9mZp", "fault_id": "f1", "limit": 5,
	}))
	if err != nil {
		t.Fatalf("handleListFaultNotices() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
}

// Notices page by cursor in v3, so the timestamp filters have no equivalent.
func TestHandleListFaultNoticesRejectsTimestampFilters(t *testing.T) {
	for _, field := range []string{"created_after", "created_before"} {
		result, err := handleListFaultNotices(context.Background(), offlineV3Client(),
			faultArgs(map[string]interface{}{
				"project_id": "Xk9mZp", "fault_id": "f1", field: "2026-01-01T00:00:00Z",
			}))
		if err != nil {
			t.Fatalf("%s: error = %v", field, err)
		}
		if !result.IsError || !strings.Contains(getResultText(result), field) {
			t.Errorf("%s: got %q", field, getResultText(result))
		}
	}
}

func TestHandleListFaultNotices_MissingIDs(t *testing.T) {
	result, err := handleListFaultNotices(context.Background(), offlineV3Client(), faultArgs(nil))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "project_id is required") {
		t.Errorf("got %q", getResultText(result))
	}
}

func TestHandleListFaultAffectedUsers(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/faults/f1/affected_users"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{"data":{"users":[{"email":"a@example.com","count":3}]}}`)
	})

	result, err := handleListFaultAffectedUsers(context.Background(), client,
		faultArgs(map[string]interface{}{"project_id": "Xk9mZp", "fault_id": "f1"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if !strings.Contains(getResultText(result), "a@example.com") {
		t.Errorf("result = %q", getResultText(result))
	}
}

// v3's affected-users endpoint takes no search parameter.
func TestHandleListFaultAffectedUsersRejectsSearch(t *testing.T) {
	result, err := handleListFaultAffectedUsers(context.Background(), offlineV3Client(),
		faultArgs(map[string]interface{}{"project_id": "Xk9mZp", "fault_id": "f1", "q": "alice"}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "q") {
		t.Errorf("got %q", getResultText(result))
	}
}

func TestHandleGetFaultCounts(t *testing.T) {
	var gotPath, gotQuery string
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.Query().Get("q")
		v3JSON(w, http.StatusOK, `{"data":{"total":42,"unresolved":7}}`)
	})

	result, err := handleGetFaultCounts(context.Background(), client, faultArgs(map[string]interface{}{
		"project_id": "Xk9mZp", "q": "is:unresolved",
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	if want := "/v3/accounts/me/projects/Xk9mZp/faults/summary"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if gotQuery != "is:unresolved" {
		t.Errorf("q = %q", gotQuery)
	}
	if !strings.Contains(getResultText(result), "42") {
		t.Errorf("result = %q", getResultText(result))
	}
}

func TestHandleGetFaultCounts_MissingProjectID(t *testing.T) {
	result, err := handleGetFaultCounts(context.Background(), offlineV3Client(), faultArgs(nil))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !result.IsError || !strings.Contains(getResultText(result), "project_id is required") {
		t.Errorf("got %q", getResultText(result))
	}
}
