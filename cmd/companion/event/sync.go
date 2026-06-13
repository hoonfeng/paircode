package event

import "sync"

// Sync wraps a Sink so concurrent Emit calls are serialized. The base Sink
// contract assumes serial emission — the agent's run loop emits one event at a
// time. Background jobs (cmd/companion/jobs) emit from their own goroutines, which can
// overlap a running turn's emission; wrapping the session sink once in Sync keeps
// the serial-Emit invariant every sink relies on (a chat bridge, an SSE writer)
// without each having to lock. A nil sink yields Discard.
func Sync(s Sink) Sink {
	if isNil(s) {
		return Discard
	}
	return &syncSink{inner: s}
}

type syncSink struct {
	mu    sync.Mutex
	inner Sink
}

func (s *syncSink) Emit(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inner.Emit(e)
}

// isNil checks if an interface value is truly nil (both type and value nil).
func isNil(s Sink) bool {
	switch v := any(s).(type) {
	case nil:
		return true
	case FuncSink:
		return v == nil
	default:
		return false
	}
}
