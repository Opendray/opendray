package channel

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

func TestSuppressByCooldown_PureLogic(t *testing.T) {
	// notifyCooldown's value comes from the DB store; we exercise the
	// in-memory map by hand-installing a cooldown via a stub Hub that
	// bypasses the store lookup.
	h := &Hub{
		log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		notifyState: make(map[string]map[string]time.Time),
	}
	const channelID = "ch_test"
	const topic = "session.idle"
	const sessionID = "sess_a"

	cooldown := 200 * time.Millisecond
	tick := func(now time.Time) bool {
		return suppress(h, channelID, topic, sessionID, now, cooldown)
	}

	t0 := time.Now()
	if tick(t0) {
		t.Fatal("first call should not be suppressed")
	}
	if !tick(t0.Add(50 * time.Millisecond)) {
		t.Error("within-cooldown call should be suppressed")
	}
	if tick(t0.Add(cooldown + 1*time.Millisecond)) {
		t.Error("after-cooldown call should fire again")
	}
}

func TestSuppressByCooldown_PerSession(t *testing.T) {
	h := &Hub{
		log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		notifyState: make(map[string]map[string]time.Time),
	}
	now := time.Now()
	cooldown := time.Hour
	if suppress(h, "c1", "session.idle", "A", now, cooldown) {
		t.Fatal("session A first")
	}
	if suppress(h, "c1", "session.idle", "B", now, cooldown) {
		t.Fatal("session B should be independent of A")
	}
	if !suppress(h, "c1", "session.idle", "A", now, cooldown) {
		t.Error("session A repeat within window should be suppressed")
	}
}

func TestForgetNotifyState_ClearsCooldown(t *testing.T) {
	h := &Hub{
		log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		notifyState: make(map[string]map[string]time.Time),
	}
	now := time.Now()
	cooldown := time.Hour
	suppress(h, "c1", "session.idle", "A", now, cooldown)
	if !suppress(h, "c1", "session.idle", "A", now, cooldown) {
		t.Fatal("setup: should be in cooldown")
	}
	h.forgetNotifyState("c1")
	if suppress(h, "c1", "session.idle", "A", now, cooldown) {
		t.Error("forgetNotifyState should reset state")
	}
}

// forgetNotifyAllForSession (driven by the session.input signal) must
// re-arm a session across EVERY channel, while leaving other sessions
// untouched — this is what makes a web/CLI reply count like a chat reply.
func TestForgetNotifyAllForSession_ClearsAcrossChannels(t *testing.T) {
	h := &Hub{
		log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		notifyState: make(map[string]map[string]time.Time),
	}
	now := time.Now()
	cd := time.Hour
	// Two channels notified about session A; c1 also about B.
	suppress(h, "c1", "session.idle", "A", now, cd)
	suppress(h, "c2", "session.idle", "A", now, cd)
	suppress(h, "c1", "session.idle", "B", now, cd)

	h.forgetNotifyAllForSession("A")

	if suppress(h, "c1", "session.idle", "A", now, cd) {
		t.Error("c1/A should be re-armed across channels")
	}
	if suppress(h, "c2", "session.idle", "A", now, cd) {
		t.Error("c2/A should be re-armed across channels")
	}
	if !suppress(h, "c1", "session.idle", "B", now, cd) {
		t.Error("session B must be unaffected by re-arming session A")
	}
}

func TestSuppressByCooldown_RaceFree(t *testing.T) {
	h := &Hub{
		log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		notifyState: make(map[string]map[string]time.Time),
	}
	now := time.Now()
	cooldown := time.Hour
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			suppress(h, "c1", "session.idle", "A", now.Add(time.Duration(i)*time.Microsecond), cooldown)
		}(i)
	}
	wg.Wait()
	// No assertion beyond "race detector did not trip".
}

// suppress mirrors Hub.suppressByCooldown but bypasses the store
// lookup by accepting cooldown directly. Used in tests that don't
// have a real DB pool.
func suppress(h *Hub, channelID, topic, sessionID string, now time.Time, cooldown time.Duration) bool {
	if cooldown <= 0 {
		return false
	}
	key := topic + "|" + sessionID
	h.notifyMu.Lock()
	defer h.notifyMu.Unlock()
	chState := h.notifyState[channelID]
	if chState == nil {
		chState = make(map[string]time.Time)
		h.notifyState[channelID] = chState
	}
	if last, ok := chState[key]; ok && now.Sub(last) < cooldown {
		return true
	}
	chState[key] = now
	cutoff := now.Add(-2 * cooldown)
	for k, t := range chState {
		if t.Before(cutoff) {
			delete(chState, k)
		}
	}
	return false
}

// Compile-time check: ctx is unused here; tests do not exercise the
// store-backed entry points (covered by integration via dispatch).
var _ = context.Background
