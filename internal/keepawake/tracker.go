package keepawake

import (
	"net/http"
	"sync/atomic"
	"time"
)

// recentWindow is how long after a request completes the Tracker still
// reports activity. The keeper samples on a poll grid, so without this
// window a burst of fast requests — a phone loading a screen — could
// start and finish entirely between two samples and never earn an
// assertion, leaving the dark-woken host to sleep mid-browse.
const recentWindow = 45 * time.Second

// Tracker derives the gateway's HTTP activity signal for ModeOnDemand.
// Wrap the root handler once; Active is then true while any request is
// in flight or one completed within the recent window. WebSocket and
// SSE handlers block inside ServeHTTP for the connection's lifetime, so
// an attached live terminal or event feed counts as in flight too —
// exactly right: a client streaming a PTY must keep the host up.
type Tracker struct {
	recent   time.Duration
	inflight atomic.Int64
	lastNano atomic.Int64
}

// NewTracker returns a Tracker with the default recent window.
func NewTracker() *Tracker {
	return &Tracker{recent: recentWindow}
}

// Wrap counts next's requests towards the activity signal.
func (t *Tracker) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.inflight.Add(1)
		defer func() {
			t.lastNano.Store(time.Now().UnixNano())
			t.inflight.Add(-1)
		}()
		next.ServeHTTP(w, r)
	})
}

// Active reports whether the gateway is serving now or did within the
// recent window.
func (t *Tracker) Active() bool {
	if t.inflight.Load() > 0 {
		return true
	}
	last := t.lastNano.Load()
	return last != 0 && time.Since(time.Unix(0, last)) < t.recent
}
