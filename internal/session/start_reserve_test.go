package session

import (
	"sync"
	"testing"
)

// newBareManager builds a Manager with only the fields tryReserveStart
// touches, so the reservation guard can be exercised without a DB or PTY.
func newBareManager() *Manager {
	return &Manager{
		sessions: make(map[string]*runningSession),
		starting: make(map[string]struct{}),
	}
}

// TestTryReserveStartExclusive is the core of the double-spawn fix: when N
// goroutines race to resume the same terminal row, exactly one may hold
// the reservation at a time. Without release, the rest are rejected — so a
// concurrent auto-resume + operator Restart cannot both spawn a process
// against the same cwd.
func TestTryReserveStartExclusive(t *testing.T) {
	m := newBareManager()
	const id = "ses_race"
	const goroutines = 64

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		granted int
	)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // release all at once to maximise contention
			if release, _, ok := m.tryReserveStart(id); ok {
				mu.Lock()
				granted++
				mu.Unlock()
				// Hold the reservation; do NOT release. Simulates a spawn
				// in flight — every other racer must be rejected.
				_ = release
			}
		}()
	}
	close(start)
	wg.Wait()

	if granted != 1 {
		t.Fatalf("tryReserveStart granted %d concurrent reservations, want exactly 1", granted)
	}
}

// TestTryReserveStartReleaseAllowsReuse verifies the reservation is not a
// permanent lock: once the in-flight spawn releases, the id can be started
// again (a later legitimate Restart after the first resume finished).
func TestTryReserveStartReleaseAllowsReuse(t *testing.T) {
	m := newBareManager()
	const id = "ses_reuse"

	release, _, ok := m.tryReserveStart(id)
	if !ok {
		t.Fatal("first reservation should succeed")
	}
	if _, _, ok := m.tryReserveStart(id); ok {
		t.Fatal("second concurrent reservation must be rejected while first is in flight")
	}
	release()
	if _, _, ok := m.tryReserveStart(id); !ok {
		t.Fatal("reservation should be grantable again after release")
	}
}

// TestTryReserveStartRejectsLiveSession pins that a session already live in
// the map (non-terminal) is rejected with its real state, mirroring the
// matrix guard — a resume of a running/idle session is ErrAlreadyRunning.
func TestTryReserveStartRejectsLiveSession(t *testing.T) {
	for _, state := range []State{StateRunning, StateIdle, StatePending} {
		m := newBareManager()
		const id = "ses_live"
		m.sessions[id] = &runningSession{sess: Session{ID: id, State: state}}

		release, blocked, ok := m.tryReserveStart(id)
		if ok {
			t.Fatalf("state %q: reservation must be rejected for a live session", state)
		}
		if blocked != state {
			t.Errorf("state %q: blockedState = %q, want %q", state, blocked, state)
		}
		if release != nil {
			t.Errorf("state %q: rejected reservation must not return a release func", state)
		}
	}
}

// TestTryReserveStartAllowsTerminalSession pins that a terminal row (the
// normal resume case) is allowed through and reserved.
func TestTryReserveStartAllowsTerminalSession(t *testing.T) {
	for _, state := range []State{StateStopped, StateEnded, StateInterrupted} {
		m := newBareManager()
		const id = "ses_terminal"
		m.sessions[id] = &runningSession{sess: Session{ID: id, State: state}}

		release, _, ok := m.tryReserveStart(id)
		if !ok {
			t.Fatalf("state %q: terminal session should be startable", state)
		}
		if release == nil {
			t.Fatalf("state %q: granted reservation must return a release func", state)
		}
		release()
	}
}
