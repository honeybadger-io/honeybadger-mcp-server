package hbmcp

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// apiClientTimeout mirrors api-go's default http.Client timeout
// (api-go@v0.8.0/client.go:39-41). WithHTTPClient replaces that client
// wholesale, so failing to set this would remove the upstream timeout and let
// a hung API call pin a request indefinitely.
const apiClientTimeout = 30 * time.Second

// upstreamRecord captures the outcome of the API call a tool handler makes.
//
// This exists because handlers turn API failures into IsError results with a
// nil error (see alarms.go, projects.go), so the handler's return value cannot
// distinguish an outage from a rejected call. The transport can.
//
// Every handler makes exactly one upstream call, so no aggregation is needed.
// The sticky-failure rule is a guard for a future fan-out handler: once a
// failure is recorded, a later success cannot overwrite it.
type upstreamRecord struct {
	mu           sync.Mutex
	called       bool
	status       int
	transportErr bool
	durationMs   int64
}

func (r *upstreamRecord) record(status int, transportErr bool, d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.called && (r.transportErr || r.status >= http.StatusInternalServerError) {
		return
	}
	r.called = true
	r.status = status
	r.transportErr = transportErr
	r.durationMs = d.Milliseconds()
}

func (r *upstreamRecord) snapshot() (called bool, status int, transportErr bool, durationMs int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.called, r.status, r.transportErr, r.durationMs
}

type upstreamKey struct{}

func withUpstreamRecord(ctx context.Context) (context.Context, *upstreamRecord) {
	rec := &upstreamRecord{}
	return context.WithValue(ctx, upstreamKey{}, rec), rec
}

func upstreamRecordFromContext(ctx context.Context) *upstreamRecord {
	rec, _ := ctx.Value(upstreamKey{}).(*upstreamRecord)
	return rec
}

type recordingTransport struct {
	base http.RoundTripper
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := upstreamRecordFromContext(req.Context())
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	if rec == nil {
		return resp, err
	}
	if err != nil {
		rec.record(0, true, time.Since(start))
	} else {
		rec.record(resp.StatusCode, false, time.Since(start))
	}
	return resp, err
}

// newRecordingHTTPClient builds the client handed to api-go via
// WithHTTPClient. api-go builds its requests with http.NewRequestWithContext
// using the handler's context, so the record placed by the instrumenter is
// reachable from inside RoundTrip.
func newRecordingHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   apiClientTimeout,
		Transport: &recordingTransport{base: http.DefaultTransport},
	}
}
