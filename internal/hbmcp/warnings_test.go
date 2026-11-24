package hbmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/mark3labs/mcp-go/mcp"
)

// droppedDataWarnings is a meta.warnings array as the API sends it. The tools
// pass it through untouched, so the output must carry exactly this array.
const droppedDataWarnings = `[{
	"code": "error_rate_limited",
	"message": "Honeybadger accepts at most limit_per_minute errors a minute from each of this project's API keys between start_at and end_at and drops the rest.",
	"start_at": "2026-11-23T10:00:00Z",
	"end_at": "2026-11-23T11:00:00Z",
	"limit_per_minute": 1
}]`

// warningsCase is one tool whose operation reports meta.warnings: the handler,
// its arguments, and the response body with a meta object spliced in.
type warningsCase struct {
	name    string
	handler func(context.Context, *apiv3.Client, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	args    map[string]any
	body    func(meta string) string
}

// testWarningsPassThrough checks that a tool's output carries meta.warnings
// unchanged as a top-level warnings key, and leaves the key out when the API
// reports none.
func testWarningsPassThrough(t *testing.T, tc warningsCase) {
	t.Helper()
	run := func(t *testing.T, meta string) map[string]any {
		t.Helper()
		client := newV3TestClient(t, func(w http.ResponseWriter, r *http.Request) {
			v3JSON(w, http.StatusOK, tc.body(meta))
		})
		result, err := tc.handler(context.Background(), client, mcpRequest(tc.args))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if result.IsError {
			t.Fatalf("expected success, got %s", getResultText(result))
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(getResultText(result)), &out); err != nil {
			t.Fatalf("output is not a JSON object: %v\n%s", err, getResultText(result))
		}
		return out
	}

	t.Run(tc.name+" with warnings", func(t *testing.T) {
		out := run(t, `{"request_id":"r1","warnings":`+droppedDataWarnings+`}`)
		var want any
		if err := json.Unmarshal([]byte(droppedDataWarnings), &want); err != nil {
			t.Fatal(err)
		}
		if got := out["warnings"]; !reflect.DeepEqual(got, want) {
			t.Errorf("warnings = %#v, want %#v", got, want)
		}
	})

	t.Run(tc.name+" without warnings", func(t *testing.T) {
		out := run(t, `{"request_id":"r1"}`)
		if got, ok := out["warnings"]; ok {
			t.Errorf("warnings = %#v, want the key left out", got)
		}
	})
}

func TestListToolsPassWarningsThrough(t *testing.T) {
	for _, tc := range []warningsCase{
		{
			name:    "list_faults",
			handler: handleListFaults,
			args:    map[string]any{"project_id": "Xk9mZp"},
			body: func(meta string) string {
				return `{"data":[{"id":"1","project_id":"Xk9mZp"}],"pagination":{"page":1,"per_page":25},"meta":` + meta + `}`
			},
		},
		{
			name:    "list_fault_notices",
			handler: handleListFaultNotices,
			args:    map[string]any{"project_id": "Xk9mZp", "fault_id": "1"},
			body: func(meta string) string {
				return `{"data":[{"id":"11111111-1111-4111-8111-111111111111","fault_id":"1","project_id":"Xk9mZp"}],` +
					`"pagination":{"has_older":false,"has_newer":false,"limit":25},"meta":` + meta + `}`
			},
		},
	} {
		testWarningsPassThrough(t, tc)
	}
}

// The summary and a single project's occurrences return one object rather than
// a list; api-go flattens their warnings next to the data.
func TestCountToolsPassWarningsThrough(t *testing.T) {
	for _, tc := range []warningsCase{
		{
			name:    "get_fault_counts",
			handler: handleGetFaultCounts,
			args:    map[string]any{"project_id": "Xk9mZp"},
			body: func(meta string) string {
				return `{"data":{"total":3,"environments":[{"environment":"production","resolved":false,"ignored":false,"count":3}]},"meta":` + meta + `}`
			},
		},
		{
			name:    "get_project_occurrence_counts",
			handler: handleGetProjectOccurrenceCounts,
			args:    map[string]any{"project_id": "Xk9mZp"},
			body: func(meta string) string {
				return `{"data":{"project_id":"Xk9mZp","buckets":[{"timestamp":1795428000,"count":2}]},"meta":` + meta + `}`
			},
		},
	} {
		testWarningsPassThrough(t, tc)
	}
}
