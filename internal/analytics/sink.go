// Package analytics owns the telemetry destination. It knows nothing about
// MCP, and it is the only package permitted to import honeybadger-go.
package analytics

// Event is one analytics event. Data must never contain customer data.
type Event struct {
	Type string
	Data map[string]any
}

// Notice is an error report. Fingerprint controls grouping; Context carries
// searchable metadata that is deliberately kept out of Event, because error
// messages can embed API response bodies.
type Notice struct {
	Err         error
	Fingerprint string
	Context     map[string]any
}

// Sink is the telemetry destination.
//
// Emit returns nothing by design: a caller must never be able to fail a tool
// call by emitting. Notify returns the notice token for correlation, which is
// the only place it is obtainable.
type Sink interface {
	Emit(Event)
	Notify(Notice) string
	Flush()
}

type nopSink struct{}

func (nopSink) Emit(Event)           {}
func (nopSink) Notify(Notice) string { return "" }
func (nopSink) Flush()               {}

// NewNopSink returns a Sink that discards everything. Used whenever analytics
// is not activated, so no *telemetry* client is constructed and nothing is
// ever sent.
//
// Note that importing honeybadger-go unavoidably constructs its package-level
// DefaultClient, seeded from ambient HONEYBADGER_API_KEY. Nothing here routes
// through it — that is the invariant worth guarding, not "no client exists".
//
// nopSink is an empty struct, so two NewNopSink() values compare equal —
// callers rely on that to assert the activation gate held.
func NewNopSink() Sink { return nopSink{} }
