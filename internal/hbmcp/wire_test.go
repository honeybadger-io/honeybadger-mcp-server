package hbmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/toolgen"
	"github.com/mark3labs/mcp-go/server"
)

// wireSamples are values for parameters whose schema alone doesn't make a
// value the handler can use: a pattern a sample string won't match ("*"
// applies to every tool), or an integration's config, which must suit its type.
var wireSamples = map[string]map[string]any{
	"*": {
		"url": "https://example.com/", "report_period": "1 day", "grace_period": "5 minutes",
		"evaluation_period": "5m", "lookback_lag": "1m",
	},
	"create_integration": {"type": "WebHook", "config": map[string]any{"url": "https://example.com/hook"}},
}

// Every parameter a generated tool advertises reaches the API request: a path
// parameter in the URL, a query parameter in the query string, a body field
// in the JSON body. A tool that accepts a parameter and never sends it would
// report success for something that didn't happen.
func TestEveryAdvertisedParameterReachesTheRequest(t *testing.T) {
	bundle, err := os.ReadFile("../../openapi/bundled.yaml")
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := os.ReadFile("../../openapi/tools.yaml")
	if err != nil {
		t.Fatal(err)
	}
	tools, err := toolgen.Build(bundle, overlay)
	if err != nil {
		t.Fatal(err)
	}

	api := &recordingAPI{}
	srv := httptest.NewServer(api)
	defer srv.Close()
	s := NewServer(&config.Config{APIURL: srv.URL, AuthToken: "hbu_test", TransportMode: config.TransportStdio}, "test")

	for _, tool := range tools {
		for _, param := range sortedNames(tool.Wire) {
			wire := tool.Wire[param]
			args := wireArgs(tool, param)
			api.reset()
			text := callToolText(t, s, tool.Name, args)
			method := ""
			if tool.Confirm {
				// The first call only previews; the delete happens on the second,
				// with the token the preview gave, and that's the request to check.
				m := confirmToken.FindStringSubmatch(text)
				if m == nil {
					t.Errorf("%s.%s: no preview token: %s", tool.Name, param, text)
					continue
				}
				args["confirm"] = m[1]
				api.reset()
				text = callToolText(t, s, tool.Name, args)
				method = http.MethodDelete
			}
			if !api.saw(method, wire, param, args[param]) {
				t.Errorf("%s.%s: not sent (%s); tool said: %s", tool.Name, param, wire, text)
			}
		}
	}
}

// wireArgs is a call to tool with param set and every required parameter
// filled in.
func wireArgs(tool toolgen.Tool, param string) map[string]any {
	schema := tool.StrictSchema
	if schema == nil {
		schema = tool.Schema
	}
	props := schema["properties"].(map[string]any)
	args := map[string]any{}
	for _, r := range asAnySlice(schema["required"]) {
		args[r.(string)] = sampleFor(tool.Name, r.(string), props[r.(string)])
	}
	args[param] = sampleFor(tool.Name, param, props[param])
	// An update refuses a call that changes nothing, so a path parameter is
	// checked alongside one body field.
	if tool.Wire[param] == "path" {
		if body := firstBodyParam(tool.Wire); body != "" {
			if _, set := args[body]; !set {
				args[body] = sampleFor(tool.Name, body, props[body])
			}
		}
	}
	for k, v := range wireSamples[tool.Name] {
		args[k] = v
	}
	return args
}

var confirmToken = regexp.MustCompile(`confirm set to "([^"]+)"`)

// sampleFor makes a value its schema accepts.
func sampleFor(tool, param string, raw any) any {
	if v, ok := wireSamples[tool][param]; ok {
		return v
	}
	if v, ok := wireSamples["*"][param]; ok {
		return v
	}
	return sample(param, raw)
}

func sample(name string, raw any) any {
	schema, _ := raw.(map[string]any)
	if enum := asAnySlice(schema["enum"]); len(enum) > 0 {
		for _, v := range enum {
			if v != nil {
				return v
			}
		}
	}
	if c, ok := schema["const"]; ok {
		return c
	}
	switch first(schemaTypes(schema)) {
	case "string":
		return sampleString(name, schema)
	case "integer":
		if m, ok := schema["minimum"].(int); ok {
			return float64(m)
		}
		return 1.0
	case "number":
		return 1.0
	case "boolean":
		return true
	case "array":
		return []any{sample(name, schema["items"])}
	case "object":
		out := map[string]any{}
		props, _ := schema["properties"].(map[string]any)
		for _, r := range asAnySlice(schema["required"]) {
			out[r.(string)] = sample(r.(string), props[r.(string)])
		}
		if len(props) == 0 {
			if extra, ok := schema["additionalProperties"].(map[string]any); ok {
				out["k"] = sample("k", extra)
			}
		}
		return out
	}
	return "s-" + name
}

func firstBodyParam(wire map[string]string) string {
	for _, name := range sortedNames(wire) {
		if wire[name] == "body" {
			return name
		}
	}
	return ""
}

func sampleString(name string, schema map[string]any) string {
	switch schema["format"] {
	case "uuid":
		return "5b7c2d1e-8f3a-4c6b-9d2e-1a2b3c4d5e6f"
	case "date-time":
		return "2026-01-02T03:04:05Z"
	}
	if strings.Contains(fmt.Sprint(schema["description"]), "RFC 3339") {
		return "2026-01-02"
	}
	return "s-" + name
}

func first(types []string) string {
	for _, t := range types {
		if t != "null" {
			return t
		}
	}
	return ""
}

func asAnySlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func sortedNames(m map[string]string) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func callToolText(t *testing.T, s *server.MCPServer, tool string, args map[string]any) string {
	t.Helper()
	msg, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": tool, "arguments": args}})
	raw, _ := json.Marshal(s.HandleMessage(context.Background(), msg))
	var resp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &resp)
	if len(resp.Result.Content) == 0 {
		return string(raw)
	}
	return resp.Result.Content[0].Text
}

// recordingAPI answers every request plausibly and remembers what it was sent.
type recordingAPI struct {
	mu       sync.Mutex
	requests []recorded
}

type recorded struct {
	method string
	path   string
	query  map[string][]string
	body   map[string]any
}

func (a *recordingAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	a.mu.Lock()
	a.requests = append(a.requests, recorded{method: r.Method, path: r.URL.Path, query: r.URL.Query(), body: body})
	a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	last := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	if r.Method == http.MethodGet && !strings.HasPrefix(last, "s-") && !strings.Contains(last, "-4c6b-") {
		_, _ = io.WriteString(w, `{"data":[],"links":{},"pagination":{}}`)
		return
	}
	_, _ = io.WriteString(w, `{"data":{"id":"5b7c2d1e-8f3a-4c6b-9d2e-1a2b3c4d5e6f","name":"x"}}`)
}

func (a *recordingAPI) reset() {
	a.mu.Lock()
	a.requests = nil
	a.mu.Unlock()
}

// saw reports whether a request (with method, when given) carried value where
// wire says, as the value itself rather than only its key: a field sent empty
// or zero hasn't reached the API.
func (a *recordingAPI) saw(method, in, name string, value any) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, r := range a.requests {
		if method != "" && r.method != method {
			continue
		}
		switch in {
		case "path":
			if strings.Contains(r.path, fmt.Sprint(value)) {
				return true
			}
		case "query":
			if slices.Contains(r.query[name], queryForm(value)) {
				return true
			}
		case "body":
			if got, ok := r.body[name]; ok && sameJSON(got, value) {
				return true
			}
		}
	}
	return false
}

// queryForm is how a sample appears in a query string. A date parameter's
// sample (2026-01-02) is sent as the Unix seconds its handler converts it to.
func queryForm(v any) string {
	switch x := v.(type) {
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case string:
		if at, err := time.Parse(time.DateOnly, x); err == nil {
			return strconv.FormatInt(at.Unix(), 10)
		}
	}
	return fmt.Sprint(v)
}

func sameJSON(a, b any) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	var na, nb any
	_ = json.Unmarshal(ja, &na)
	_ = json.Unmarshal(jb, &nb)
	return reflect.DeepEqual(na, nb)
}
