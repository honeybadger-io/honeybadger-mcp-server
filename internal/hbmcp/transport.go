package hbmcp

import "net/http"

// bearerTransport overrides api-go's Basic auth header (req.SetBasicAuth in
// its newRequest path) with an OAuth Bearer token for hosted multi-tenant use.
type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(r)
}
