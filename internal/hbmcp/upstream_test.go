package hbmcp

import (
	"context"
	"io"
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

// RoundTrip returns when headers arrive; api-go decodes the body afterwards.
// A connection dropped mid-body is an outage, not a clean 200.
func TestRecordingTransport_RecordsBodyReadFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("truncated"))
		// Returning without writing the promised bytes makes the client's
		// body read fail with ErrUnexpectedEOF.
	}))
	defer srv.Close()

	ctx, rec := withUpstreamRecord(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := newRecordingHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Fatal("expected a body read error")
	}

	called, _, transportErr, _ := rec.snapshot()
	if !called {
		t.Fatal("called = false, want true")
	}
	if !transportErr {
		t.Error("transportErr = false; a truncated body was recorded as a clean 200")
	}
}

// A fully-read body keeps the status and extends the recorded duration to
// cover transfer, rather than stopping at the header.
func TestRecordingTransport_SuccessfulBodyKeepsStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	ctx, rec := withUpstreamRecord(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := newRecordingHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}

	_, status, transportErr, _ := rec.snapshot()
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if transportErr {
		t.Error("transportErr = true on a clean read")
	}
}

// A 5xx recorded at header time must survive the body being read to EOF.
func TestRecordingTransport_ServerErrorSurvivesBodyRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	ctx, rec := withUpstreamRecord(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := newRecordingHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	_, status, _, _ := rec.snapshot()
	if status != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 to survive the body read", status)
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
