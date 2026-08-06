package analytics

import (
	honeybadger "github.com/honeybadger-io/honeybadger-go"
)

type hbSink struct {
	client *honeybadger.Client
}

// NewHoneybadgerSink builds a dedicated client. It never touches
// honeybadger.DefaultClient, which is seeded from ambient HONEYBADGER_API_KEY
// at package init and could otherwise carry a customer's own key.
func NewHoneybadgerSink(apiKey, env string) Sink {
	return newHoneybadgerSink(honeybadger.Configuration{
		APIKey: apiKey,
		Env:    env,
	})
}

func newHoneybadgerSink(cfg honeybadger.Configuration) *hbSink {
	return &hbSink{client: honeybadger.New(cfg)}
}

func (s *hbSink) Emit(e Event) {
	// Swallowed by design. With the async worker this returns nil anyway;
	// delivery failures surface through the library's own drop logging.
	_ = s.client.Event(e.Type, e.Data)
}

func (s *hbSink) Notify(n Notice) string {
	extra := []any{honeybadger.Fingerprint{Content: n.Fingerprint}}
	if len(n.Context) > 0 {
		extra = append(extra, honeybadger.Context(n.Context))
	}
	token, _ := s.client.Notify(n.Err, extra...)
	return token
}

func (s *hbSink) Flush() { s.client.Flush() }
