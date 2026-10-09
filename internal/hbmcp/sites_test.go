package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
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
		v3JSON(w, http.StatusOK, `{"data":[],"pagination":{"has_older":false}}`)
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
		v3JSON(w, http.StatusOK, `{"data":[],"pagination":{"has_older":false}}`)
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
		v3JSON(w, http.StatusOK, `{"data":[],"pagination":{"has_older":false}}`)
	})
	result, err := handleListCheckInEvents(context.Background(), client, mcpRequest(map[string]any{
		"project_id": "Xk9mZp", "check_in_id": "c1", "created_before": 1704153600.5,
	}))
	mustSucceed(t, result, err)
	if path != "/v3/projects/Xk9mZp/check_ins/c1/events" || query.Get("created_before") != "1704153600.5" {
		t.Errorf("request = %s?%s", path, query.Encode())
	}
}

// The position comes out as next_created_before, digits intact and with no URL
// in the output, and goes back in to fetch exactly the next older page.
func TestOutagesHandBackTheirPagingPosition(t *testing.T) {
	var sent []string
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		sent = append(sent, r.URL.Query().Get("created_before"))
		v3JSON(w, http.StatusOK, `{"data":[],"pagination":{"has_older":true,"has_newer":false,"limit":2},
			"links":{"self":"/v3/x","older":"/v3/projects/Xk9mZp/sites/`+siteUUID+`/outages?created_before=1704153600.123456&limit=2"}}`)
	})
	args := map[string]any{"project_id": "Xk9mZp", "site_id": siteUUID, "limit": 2}

	result, err := handleListSiteOutages(context.Background(), client, mcpRequest(args))
	mustSucceed(t, result, err)
	text := getResultText(result)
	if !strings.Contains(text, `"next_created_before":1704153600.123456`) {
		t.Fatalf("output = %s", text)
	}
	var page map[string]any
	if err := json.Unmarshal([]byte(text), &page); err != nil {
		t.Fatal(err)
	}
	if _, present := page["time_series_links"]; present {
		t.Errorf("output carries the links: %s", text)
	}

	// An agent reads the number as JSON and passes it straight back.
	args["created_before"] = page["next_created_before"]
	result, err = handleListSiteOutages(context.Background(), client, mcpRequest(args))
	mustSucceed(t, result, err)
	if len(sent) != 2 || sent[1] != "1704153600.123456" {
		t.Errorf("created_before sent = %q, want the second request to carry 1704153600.123456", sent)
	}
}
