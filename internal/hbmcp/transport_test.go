package hbmcp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// recordingTransport captures the headers of the request it's asked to round-trip.
type recordingTransport struct {
	gotAuthHeader  string
	gotOtherHeader string
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.gotAuthHeader = req.Header.Get("Authorization")
	r.gotOtherHeader = req.Header.Get("X-Other")
	return &http.Response{StatusCode: 200, Body: http.NoBody, Header: http.Header{}, Request: req}, nil
}

func TestBearerTransport_SetsBearerHeader(t *testing.T) {
	rec := &recordingTransport{}
	bt := &bearerTransport{token: "jwt-xyz", base: rec}

	req := httptest.NewRequest("GET", "http://example.test/v2/projects", nil)
	if _, err := bt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	if rec.gotAuthHeader != "Bearer jwt-xyz" {
		t.Errorf("forwarded Authorization = %q, want %q", rec.gotAuthHeader, "Bearer jwt-xyz")
	}
}

func TestBearerTransport_ReplacesBasicAuth(t *testing.T) {
	rec := &recordingTransport{}
	bt := &bearerTransport{token: "jwt-xyz", base: rec}

	req := httptest.NewRequest("GET", "http://example.test/v2/projects", nil)
	req.SetBasicAuth("pat-from-api-go", "")
	if _, err := bt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	if rec.gotAuthHeader != "Bearer jwt-xyz" {
		t.Errorf("forwarded Authorization = %q, want Bearer to replace Basic", rec.gotAuthHeader)
	}
}

func TestBearerTransport_PreservesOtherHeaders(t *testing.T) {
	rec := &recordingTransport{}
	bt := &bearerTransport{token: "jwt-xyz", base: rec}

	req := httptest.NewRequest("GET", "http://example.test/v2/projects", nil)
	req.Header.Set("X-Other", "keepme")
	if _, err := bt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	if rec.gotOtherHeader != "keepme" {
		t.Errorf("X-Other header lost: got %q, want %q", rec.gotOtherHeader, "keepme")
	}
}

func TestBearerTransport_DoesNotMutateCaller(t *testing.T) {
	rec := &recordingTransport{}
	bt := &bearerTransport{token: "jwt-xyz", base: rec}

	req := httptest.NewRequest("GET", "http://example.test/v2/projects", nil)
	req.SetBasicAuth("pat", "")
	originalAuth := req.Header.Get("Authorization")

	if _, err := bt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	// RoundTripper contract: must not modify the incoming request.
	if got := req.Header.Get("Authorization"); got != originalAuth {
		t.Errorf("caller's request was mutated: Authorization = %q, want %q", got, originalAuth)
	}
}
