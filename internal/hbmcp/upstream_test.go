package hbmcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRecordingTransport_RecordsStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, rec := withUpstreamRecord(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := newRecordingHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	called, status, transportErr, _ := rec.snapshot()
	if !called {
		t.Fatal("called = false, want true")
	}
	if status != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", status)
	}
	if transportErr {
		t.Error("transportErr = true, want false")
	}
}

func TestRecordingTransport_RecordsTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing is listening now

	ctx, rec := withUpstreamRecord(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if _, err := newRecordingHTTPClient().Do(req); err == nil {
		t.Fatal("expected a transport error")
	}

	called, _, transportErr, _ := rec.snapshot()
	if !called {
		t.Fatal("called = false, want true")
	}
	if !transportErr {
		t.Error("transportErr = false, want true")
	}
}

// A later success must not mask an earlier failure.
func TestUpstreamRecord_FailureIsSticky(t *testing.T) {
	rec := &upstreamRecord{}
	rec.record(500, false, 5*time.Millisecond)
	rec.record(200, false, 1*time.Millisecond)

	_, status, _, _ := rec.snapshot()
	if status != 500 {
		t.Errorf("status = %d, want 500 (failure should be sticky)", status)
	}
}

func TestUpstreamRecord_NilSafeWhenAnalyticsOff(t *testing.T) {
	// No record in the context: the transport must not panic.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := newRecordingHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if rec := upstreamRecordFromContext(context.Background()); rec != nil {
		t.Error("expected no record in a bare context")
	}
}

// api-go's default client carries a 30s timeout; replacing the client must
// not silently drop it (api-go@v0.8.0/client.go:39-41).
func TestRecordingHTTPClient_PreservesTimeout(t *testing.T) {
	if got := newRecordingHTTPClient().Timeout; got != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", got)
	}
}
