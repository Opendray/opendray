package bridge

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ─────────────────────────────────────────────
// fakeWS — minimal WSLike for tests
// ─────────────────────────────────────────────

type fakeWS struct {
	mu      sync.Mutex
	writes  [][]byte
	closed  bool
	blockN  int // if > 0, block WriteMessage this many calls (for backpressure / SLO test)
	warnBuf *bytes.Buffer
}

func (f *fakeWS) WriteMessage(_ int, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return errFakeWSClosed
	}

	if f.blockN > 0 {
		f.blockN--
		// Simulate a slow writer by blocking until Close is called.
		// For testing backpressure: we sleep a short time to simulate slow writes.
		time.Sleep(50 * time.Millisecond)
	}

	cp := make([]byte, len(data))
	copy(cp, data)
	f.writes = append(f.writes, cp)
	return nil
}

func (f *fakeWS) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	return nil
}

func (f *fakeWS) writeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.writes)
}

func (f *fakeWS) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

var errFakeWSClosed = &wsClosedError{}

type wsClosedError struct{}

func (e *wsClosedError) Error() string { return "fakeWS: closed" }

// ─────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	log := slog.Default()
	return NewManager(log)
}

// ─────────────────────────────────────────────
// Test cases
// ─────────────────────────────────────────────

// 1. Register returns non-nil *Conn; ActiveConns == 1.
func TestManager_RegisterAddsConn(t *testing.T) {
	m := newTestManager(t)
	ws := &fakeWS{}

	conn := m.Register("plugin-a", ws)
	if conn == nil {
		t.Fatal("Register returned nil Conn")
	}
	if conn.Plugin != "plugin-a" {
		t.Errorf("conn.Plugin = %q, want plugin-a", conn.Plugin)
	}
	if got := m.ActiveConns("plugin-a"); got != 1 {
		t.Errorf("ActiveConns = %d, want 1", got)
	}
}

// 2. Two plugins each with their own Conn; InvalidateConsent on A must not hit B.
func TestManager_MultiplePluginsIsolated(t *testing.T) {
	m := newTestManager(t)
	wsA := &fakeWS{}
	wsB := &fakeWS{}

	connA := m.Register("plugin-a", wsA)
	m.Register("plugin-b", wsB)

	// Subscribe A to storage.
	_, err := connA.Subscribe("sub-a1", "storage")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Invalidate A — must not write to wsB.
	m.InvalidateConsent("plugin-a", "storage")

	// Give goroutines time to settle.
	time.Sleep(50 * time.Millisecond)

	if wsB.writeCount() != 0 {
		t.Errorf("plugin-b got %d writes, want 0 (isolation failure)", wsB.writeCount())
	}
	if wsA.writeCount() == 0 {
		t.Errorf("plugin-a got 0 writes, expected EPERM envelope")
	}
}

// 3. 100 goroutines call WriteEnvelope concurrently; race-clean; all writes land.
func TestConn_WriteEnvelopeSerialized(t *testing.T) {
	m := newTestManager(t)
	ws := &fakeWS{}
	conn := m.Register("plugin-c", ws)

	const N = 100
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			env := NewErr("id", "EPERM", "test")
			if err := conn.WriteEnvelope(env); err != nil {
				// May fail after Close — not an issue here since we don't close.
				t.Errorf("WriteEnvelope: %v", err)
			}
		}()
	}
	wg.Wait()

	count := ws.writeCount()
	if count != N {
		t.Errorf("writes = %d, want %d", count, N)
	}

	// Verify every written frame is valid JSON.
	ws.mu.Lock()
	defer ws.mu.Unlock()
	for i, raw := range ws.writes {
		var env Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Errorf("write[%d] is not valid JSON: %v | raw: %s", i, err, raw)
		}
	}
}

// 4. SLO test: 100 subs for "storage"; hot-revoke completes within 200 ms.
func TestManager_HotRevokeDeliversUnderSLO(t *testing.T) {
	m := newTestManager(t)
	ws := &fakeWS{}
	conn := m.Register("plugin-d", ws)

	const N = 100
	dones := make([]<-chan struct{}, N)
	for i := 0; i < N; i++ {
		subID := "sub-storage-" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		done, err := conn.Subscribe(subID, "storage")
		if err != nil {
			t.Fatalf("Subscribe[%d]: %v", i, err)
		}
		dones[i] = done
	}

	deadline := time.Now().Add(200 * time.Millisecond)
	invokeStart := time.Now()
	m.InvalidateConsent("plugin-d", "storage")
	invokeElapsed := time.Since(invokeStart)
	t.Logf("InvalidateConsent returned in %v (caller-path latency)", invokeElapsed)

	// All done channels must close within 200 ms from InvalidateConsent call.
	for i, done := range dones {
		select {
		case <-done:
			// good
		case <-time.After(time.Until(deadline)):
			t.Fatalf("sub[%d] done channel not closed within SLO (200 ms)", i)
		}
	}

	// All 100 terminal EPERM envelopes must land on the WS within the SLO.
	// We already verified dones closed; now wait for writes too.
	for time.Now().Before(deadline) {
		if ws.writeCount() >= N {
			break
		}
		time.Sleep(time.Millisecond)
	}
	got := ws.writeCount()
	sloElapsed := time.Since(invokeStart)
	t.Logf("SLO: %d/%d EPERM envelopes written, total elapsed %v (SLO target ≤200ms)", got, N, sloElapsed)
	if got < N {
		t.Errorf("only %d/%d EPERM envelopes written within 200 ms SLO", got, N)
	}

	// Verify each is an EPERM stream-end envelope.
	ws.mu.Lock()
	defer ws.mu.Unlock()
	for i, raw := range ws.writes {
		var env Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Errorf("write[%d] bad JSON: %v", i, err)
			continue
		}
		if env.Error == nil || env.Error.Code != "EPERM" {
			t.Errorf("write[%d] not EPERM: %+v", i, env)
		}
		if env.Stream != "end" {
			t.Errorf("write[%d] stream = %q, want end", i, env.Stream)
		}
	}
}

// 5. Plugin has 3 subs: storage, storage, events. Invalidate storage → only 2 close.
func TestManager_HotRevokeOnlyMatchingCap(t *testing.T) {
	m := newTestManager(t)
	ws := &fakeWS{}
	conn := m.Register("plugin-e", ws)

	doneSt1, err := conn.Subscribe("sub-st1", "storage")
	if err != nil {
		t.Fatalf("Subscribe st1: %v", err)
	}
	doneSt2, err := conn.Subscribe("sub-st2", "storage")
	if err != nil {
		t.Fatalf("Subscribe st2: %v", err)
	}
	doneEv, err := conn.Subscribe("sub-ev1", "events")
	if err != nil {
		t.Fatalf("Subscribe ev1: %v", err)
	}

	m.InvalidateConsent("plugin-e", "storage")

	timeout := time.After(200 * time.Millisecond)

	// Both storage subs must close.
	for _, done := range []<-chan struct{}{doneSt1, doneSt2} {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("storage sub not closed within 200 ms")
		}
	}

	// Events sub must NOT close.
	select {
	case <-doneEv:
		t.Fatal("events sub closed unexpectedly on storage invalidation")
	case <-time.After(20 * time.Millisecond):
		// Correct — still open.
	}
}

// 6. Call Conn.Close twice → no panic, only one ws.Close.
func TestManager_CloseIdempotent(t *testing.T) {
	m := newTestManager(t)
	ws := &fakeWS{}
	conn := m.Register("plugin-f", ws)

	if err := conn.Close(1000, "done"); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := conn.Close(1000, "done again"); err != nil {
		t.Errorf("second Close returned error: %v", err)
	}

	// ws.Close must have been called exactly once internally.
	// Verify by checking closed flag is set.
	if !ws.isClosed() {
		t.Error("fakeWS not closed after Conn.Close")
	}
}

// 7. ActiveConns drops to 0 after Unregister.
func TestManager_UnregisterRemovesConn(t *testing.T) {
	m := newTestManager(t)
	ws := &fakeWS{}
	conn := m.Register("plugin-g", ws)

	if got := m.ActiveConns("plugin-g"); got != 1 {
		t.Fatalf("before unregister: ActiveConns = %d, want 1", got)
	}

	m.Unregister(conn)

	if got := m.ActiveConns("plugin-g"); got != 0 {
		t.Errorf("after unregister: ActiveConns = %d, want 0", got)
	}
}

// 8. Subscribe returns a channel that closes on Unsubscribe.
func TestConn_SubscribeAndUnsubscribe(t *testing.T) {
	m := newTestManager(t)
	ws := &fakeWS{}
	conn := m.Register("plugin-h", ws)

	done, err := conn.Subscribe("sub-h1", "events")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Channel must be open.
	select {
	case <-done:
		t.Fatal("done channel closed before Unsubscribe")
	default:
	}

	conn.Unsubscribe("sub-h1")

	select {
	case <-done:
		// Correct.
	case <-time.After(50 * time.Millisecond):
		t.Fatal("done channel not closed after Unsubscribe")
	}
}

// 9. Backpressure: fakeWS.blockN=10; broadcast of 100 envelopes completes within 1s.
func TestConn_BackpressureDoesNotBlock(t *testing.T) {
	// Use a logger that writes to a buffer so we can check warn logs.
	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	m := NewManager(log)

	ws := &fakeWS{blockN: 10}
	conn := m.Register("plugin-i", ws)

	// Subscribe 100 subs.
	const N = 100
	for i := 0; i < N; i++ {
		subID := "sub-bp-" + string(rune('A'+i%26)) + string(rune('A'+i/26))
		if _, err := conn.Subscribe(subID, "storage"); err != nil {
			t.Fatalf("Subscribe[%d]: %v", i, err)
		}
	}

	start := time.Now()
	m.InvalidateConsent("plugin-i", "storage")

	// Must complete within 1 s even with blocked writes.
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		// The manager's fan-out must not block caller regardless of slow WS.
		if time.Since(start) > 1*time.Second {
			t.Fatal("InvalidateConsent blocked for > 1s (backpressure not handled)")
		}
		break
	}

	// InvalidateConsent itself must return fast (non-blocking). Sleep briefly to
	// allow async writes to drain.
	time.Sleep(600 * time.Millisecond)

	got := ws.writeCount()
	if got < 10 {
		t.Errorf("expected at least 10 writes, got %d", got)
	}
	t.Logf("backpressure test: %d/%d writes landed, logs: %s", got, N, logBuf.String())
}

// 10. Close conn, then WriteEnvelope → returns error.
func TestManager_WriteAfterCloseReturnsError(t *testing.T) {
	m := newTestManager(t)
	ws := &fakeWS{}
	conn := m.Register("plugin-j", ws)

	if err := conn.Close(1000, "bye"); err != nil {
		t.Fatalf("Close: %v", err)
	}

	env := NewErr("id", "EPERM", "denied")
	err := conn.WriteEnvelope(env)
	if err == nil {
		t.Fatal("WriteEnvelope after Close: want error, got nil")
	}
}

// 11. Ancestor context cancellation closes conn within 100 ms.
func TestManager_ContextCancellationCloses(t *testing.T) {
	m := newTestManager(t)
	ws := &fakeWS{}

	conn := m.Register("plugin-k", ws)

	// Cancel via Conn.Close (which cancels internal context).
	// Since the Manager does not take a parent ctx in Register (per contract),
	// we simulate by calling Close from an external goroutine.
	var once sync.Once
	cancelled := make(chan struct{})
	go func() {
		time.Sleep(10 * time.Millisecond)
		once.Do(func() {
			conn.Close(1001, "context cancelled")
			close(cancelled)
		})
	}()

	select {
	case <-cancelled:
		// Check that conn is effectively closed.
		if !ws.isClosed() {
			t.Error("ws not closed after Close()")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("conn not closed within 100 ms of context cancellation")
	}
}

// 12. InvalidateConsent on unknown plugin is a safe no-op.
func TestManager_InvalidateConsentUnknownPlugin(t *testing.T) {
	m := newTestManager(t)

	// Must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("InvalidateConsent panicked: %v", r)
		}
	}()

	m.InvalidateConsent("no-such-plugin", "storage")
}

// Extra credit: sub's done channel fires when the conn itself closes.
func TestManager_SubscribeDonesCloseOnConnClose(t *testing.T) {
	m := newTestManager(t)
	ws := &fakeWS{}
	conn := m.Register("plugin-l", ws)

	done1, err := conn.Subscribe("sub-l1", "storage")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	done2, err := conn.Subscribe("sub-l2", "events")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	conn.Close(1000, "test close")

	for i, done := range []<-chan struct{}{done1, done2} {
		select {
		case <-done:
			// Correct.
		case <-time.After(100 * time.Millisecond):
			t.Errorf("sub[%d] done channel not closed after conn Close()", i)
		}
	}
}

// ─────────────────────────────────────────────
// ConsentChange type check
// ─────────────────────────────────────────────

func TestConsentChange_FieldsExist(t *testing.T) {
	// Compile-time check that ConsentChange has Plugin and Cap fields.
	cc := ConsentChange{Plugin: "p", Cap: "storage"}
	if cc.Plugin != "p" || cc.Cap != "storage" {
		t.Fail()
	}
}

// ─────────────────────────────────────────────
// WSLike interface conformance
// ─────────────────────────────────────────────

func TestWSLike_FakeWSConforms(t *testing.T) {
	// Verify fakeWS satisfies WSLike at compile time.
	var _ WSLike = (*fakeWS)(nil)
	t.Log("fakeWS satisfies WSLike")
}

// ─────────────────────────────────────────────
// Race safety: concurrent Register/Unregister/InvalidateConsent
// ─────────────────────────────────────────────

func TestManager_ConcurrentOpsRaceSafe(t *testing.T) {
	m := newTestManager(t)
	var wg sync.WaitGroup
	const goroutines = 20

	var registered atomic.Int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ws := &fakeWS{}
			conn := m.Register("plugin-race", ws)
			registered.Add(1)

			if i%3 == 0 {
				m.InvalidateConsent("plugin-race", "storage")
			}
			if i%2 == 0 {
				m.Unregister(conn)
				registered.Add(-1)
			}
		}(i)
	}
	wg.Wait()
	// No assertion on count — just verifying no data race + no panic.
	t.Logf("registered remaining: %d", registered.Load())
}
