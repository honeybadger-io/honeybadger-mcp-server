package hbmcp

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

const siteUUID = "9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47"

func TestCreateSiteRequiresAURL(t *testing.T) {
	result, err := handleCreateSite(context.Background(), noRequestClient(t), mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "name": "Home",
	}))
	if err != nil || !result.IsError {
		t.Errorf("result = %s, %v; want it refused", getResultText(result), err)
	}
}

func TestSiteToolsRefuseANonUUIDSiteID(t *testing.T) {
	result, err := handleGetSite(context.Background(), noRequestClient(t), mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "site_id": "not-a-uuid",
	}))
	if err != nil || !result.IsError {
		t.Errorf("result = %s, %v; want it refused", getResultText(result), err)
	}
}

// null resets a site setting, so it must reach the API as null.
func TestUpdateSiteResetsWithNull(t *testing.T) {
	client, body := bodyCapture(t, `{"data":{"id":"`+siteUUID+`","project_id":"Xk9mZp","name":"Home","url":"https://example.com"}}`)
	result, err := handleUpdateSite(context.Background(), client, mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "site_id": siteUUID, "match": nil, "frequency": 15,
	}))
	mustSucceed(t, result, err)
	if v, present := (*body)["match"]; !present || v != nil {
		t.Errorf("match = %v (present %v), want an explicit null", v, present)
	}
	if (*body)["frequency"] != float64(15) {
		t.Errorf("frequency = %v, want 15", (*body)["frequency"])
	}
}

// created_before comes from links.older and must go back exactly as given.
func TestListSiteOutagesPagesBackExactly(t *testing.T) {
	var query url.Values
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/projects/Xk9mZp/sites/" + siteUUID + "/outages"; r.URL.Path != want {
			t.Errorf("path = %s, want %s", r.URL.Path, want)
		}
		query = r.URL.Query()
		v3JSON(w, http.StatusOK, `{"data":[],"time_series":{"has_older":false}}`)
	})
	result, err := handleListSiteOutages(context.Background(), client, mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "site_id": siteUUID, "limit": 10, "created_before": 1704153600.123456,
	}))
	mustSucceed(t, result, err)
	if query.Get("created_before") != "1704153600.123456" || query.Get("limit") != "10" {
		t.Errorf("query = %s", query.Encode())
	}
}

func TestListDeploysFilters(t *testing.T) {
	var query url.Values
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		v3JSON(w, http.StatusOK, `{"data":[],"time_series":{"has_older":false}}`)
	})
	result, err := handleListDeploys(context.Background(), client, mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "environment": "production", "local_username": "ci", "before": "cur1",
	}))
	mustSucceed(t, result, err)
	for key, want := range map[string]string{"environment": "production", "local_username": "ci", "before": "cur1"} {
		if query.Get(key) != want {
			t.Errorf("%s = %q, want %q", key, query.Get(key), want)
		}
	}
}

func TestListCheckInEventsPagesBack(t *testing.T) {
	var path string
	var query url.Values
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path, query = r.URL.Path, r.URL.Query()
		v3JSON(w, http.StatusOK, `{"data":[],"time_series":{"has_older":false}}`)
	})
	result, err := handleListCheckInEvents(context.Background(), client, mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "check_in_id": "c1", "created_before": 1704153600.5,
	}))
	mustSucceed(t, result, err)
	if path != "/v3/projects/Xk9mZp/check_ins/c1/events" || query.Get("created_before") != "1704153600.5" {
		t.Errorf("request = %s?%s", path, query.Encode())
	}
}
