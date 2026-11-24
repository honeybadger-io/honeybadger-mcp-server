package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// assertUsagePassedThrough runs get_account_usage against a stub returning
// data and checks the tool's output is that data, key for key: what the API
// sends, including null and absent keys, is what the agent sees.
func assertUsagePassedThrough(t *testing.T, data string) {
	t.Helper()
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if want := "/v3/account/usage"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		v3JSON(w, http.StatusOK, `{"data": `+data+`}`)
	})

	result, err := handleGetAccountUsage(context.Background(), client, mcpRequest(nil))
	if err != nil {
		t.Fatalf("handleGetAccountUsage() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %s", getResultText(result))
	}
	var got, want map[string]any
	if err := json.Unmarshal([]byte(getResultText(result)), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(data), &want); err != nil {
		t.Fatalf("bad stub JSON: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%s\nwant\n%s", getResultText(result), data)
	}
}

// Plans without a monitor pool cap uptime monitors and check-ins separately.
func TestHandleGetAccountUsage_SeparateCaps(t *testing.T) {
	assertUsagePassedThrough(t, `{"id": "Ab3kL9", "quota": {
		"errors_limit": 150000, "errors_consumed": 41250,
		"uptime_limit": 10, "uptime_consumed": 4,
		"check_ins_limit": 20, "check_ins_consumed": 12,
		"insights_daily_limit": 1073741824, "insights_daily_consumed": 812000000, "insights_limited": false
	}}`)
}

// On pooled plans monitors_* replaces the separate caps, and a metered pool
// has no cap: monitors_limit stays null rather than disappearing. A null
// Insights limit is uncapped and stays null too.
func TestHandleGetAccountUsage_MeteredPool(t *testing.T) {
	assertUsagePassedThrough(t, `{"id": "Ab3kL9", "quota": {
		"errors_limit": 300000, "errors_consumed": 0,
		"uptime_consumed": 5, "check_ins_consumed": 3,
		"monitors_limit": null, "monitors_consumed": 8,
		"insights_daily_limit": null, "insights_daily_consumed": 35541, "insights_limited": false
	}}`)
}

// An Insights limit of 0 is a real cap that blocks ingestion, so it must
// reach the agent as 0, not be dropped or read as uncapped.
func TestHandleGetAccountUsage_ZeroInsightsLimit(t *testing.T) {
	assertUsagePassedThrough(t, `{"id": "Ab3kL9", "quota": {
		"errors_limit": 300000, "errors_consumed": 0,
		"uptime_consumed": 0, "check_ins_consumed": 0,
		"monitors_limit": 150, "monitors_consumed": 0,
		"insights_daily_limit": 0, "insights_daily_consumed": 0, "insights_limited": true
	}}`)
}

// A project-bound credential is refused with a 403; the API's reason reaches
// the caller as is.
func TestHandleGetAccountUsage_ErrorPassesThrough(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusForbidden, `{"error":{"code":"access_denied","message":"This credential is limited to one project"}}`)
	})

	result, err := handleGetAccountUsage(context.Background(), client, mcpRequest(nil))
	if err != nil {
		t.Fatalf("handleGetAccountUsage() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	text := getResultText(result)
	for _, want := range []string{"Failed to get account usage", "403", "access_denied", "limited to one project"} {
		if !strings.Contains(text, want) {
			t.Errorf("error %q does not contain %q", text, want)
		}
	}
}

// A personal token whose user belongs to several accounts can't say whose
// usage to read; the caller is told which credential works instead.
func TestHandleGetAccountUsage_AmbiguousAccount(t *testing.T) {
	client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
		v3JSON(w, http.StatusUnprocessableEntity, `{"error":{"code":"ambiguous_account","message":"This credential can access more than one account"}}`)
	})

	result, err := handleGetAccountUsage(context.Background(), client, mcpRequest(nil))
	if err != nil {
		t.Fatalf("handleGetAccountUsage() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	text := getResultText(result)
	for _, want := range []string{"more than one account", "account-bound API Token (hba_)", "ambiguous_account"} {
		if !strings.Contains(text, want) {
			t.Errorf("error %q does not contain %q", text, want)
		}
	}
}
