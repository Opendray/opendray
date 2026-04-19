package bridge

// api_events_test.go — TDD test suite for EventsAPI (M2 T11).
//
// Package declaration: internal `bridge` (not bridge_test) so we can reuse
// the fakeWS / Manager helpers that are already in manager_test.go's package.
// NOTE: fakeWS is already declared in manager_test.go — we must NOT re-declare
// it here. We share the same test binary because both files are in package bridge.
//
// Adapter note: the real plugin.HookBus does not implement HookBusLike directly
// (it lacks SubscribeByName / a generic Publish). A thin adapter
// HookBusAdapter is provided in api_events.go in this package so production
// wiring can wrap *plugin.HookBus without modifying hooks.go.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"testing"
	"time"
)

// ─────────────────────────────────────────────
// fakeBus — in-memory HookBusLike for tests
// ─────────────────────────────────────────────

type fakeSub struct {
	pattern string
	handler func(name string, data any)
	active  bool
}

type fakeBus struct {
	mu         sync.Mutex
	subs       []*fakeSub
	publishLog []struct {
		Name string
		Data any
	}
}

func (f *fakeBus) SubscribeByName(pattern string, h func(string, any)) (func(), error) {
	f.mu.Lock()
	sub := &fakeSub{pattern: pattern, handler: h, active: true}
	f.subs = append(f.subs, sub)
	f.mu.Unlock()
	return func() {
		f.mu.Lock()
		sub.active = false
		f.mu.Unlock()
	}, nil
}

func (f *fakeBus) Publish(name string, data any) {
	f.mu.Lock()
	f.publishLog = append(f.publishLog, struct {
		Name string
		Data any
	}{Name: name, Data: data})
	activeSubs := make([]*fakeSub, 0, len(f.subs))
	for _, s := range f.subs {
		if s.active {
			activeSubs = append(activeSubs, s)
		}
	}
	f.mu.Unlock()

	// Fan out to all matching active subs.
	for _, s := range activeSubs {
		if matchEventSegments(s.pattern, name) {
			s.handler(name, data)
		}
	}
}

// emit simulates an upstream event arriving on the bus (drives handlers directly).
func (f *fakeBus) emit(name string, data any) {
	f.mu.Lock()
	activeSubs := make([]*fakeSub, 0, len(f.subs))
	for _, s := range f.subs {
		if s.active {
			activeSubs = append(activeSubs, s)
		}
	}
	f.mu.Unlock()

	for _, s := range activeSubs {
		if matchEventSegments(s.pattern, name) {
			s.handler(name, data)
		}
	}
}

// matchEventSegments is a helper for fakeBus to route events to subs.
// Uses the same semantics as MatchEventPattern but for a single pattern.
func matchEventSegments(pattern, name string) bool {
	return MatchEventPattern([]string{pattern}, name)
}

// ─────────────────────────────────────────────
// gateWithEventsPerms builds a Gate whose consent reader grants the given
// events patterns for "testplugin".
// ─────────────────────────────────────────────

func gateWithEventsPerms(t *testing.T, patterns []string) *Gate {
	t.Helper()
	raw, err := json.Marshal(patterns)
	if err != nil {
		t.Fatalf("marshal events perms: %v", err)
	}
	permsJSON := []byte(fmt.Sprintf(`{"events":%s}`, string(raw)))
	cr := &fakeConsentReaderEvents{perms: permsJSON}
	return NewGate(cr, nil, slog.Default())
}

// fakeConsentReaderEvents implements ConsentReader for events tests.
type fakeConsentReaderEvents struct {
	perms []byte
}

func (f *fakeConsentReaderEvents) Load(_ context.Context, _ string) ([]byte, bool, error) {
	if f.perms == nil {
		return nil, false, nil
	}
	return f.perms, true, nil
}

// gateNoPerms returns a Gate where the plugin has a consent record but no events cap.
func gateNoPerms(t *testing.T) *Gate {
	t.Helper()
	permsJSON := []byte(`{}`)
	cr := &fakeConsentReaderEvents{perms: permsJSON}
	return NewGate(cr, nil, slog.Default())
}

// ─────────────────────────────────────────────
// newTestConn builds a Conn backed by a fakeWS for testing.
// fakeWS is declared in manager_test.go.
// ─────────────────────────────────────────────

func newTestConnForEvents(t *testing.T, plugin string) (*Conn, *fakeWS) {
	t.Helper()
	mgr := NewManager(slog.Default())
	ws := &fakeWS{}
	conn := mgr.Register(plugin, ws)
	return conn, ws
}

// decodeWritten parses the nth written envelope from fakeWS.
func decodeWritten(t *testing.T, ws *fakeWS, n int) Envelope {
	t.Helper()
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if n >= len(ws.writes) {
		t.Fatalf("expected at least %d write(s), got %d", n+1, len(ws.writes))
	}
	var env Envelope
	if err := json.Unmarshal(ws.writes[n], &env); err != nil {
		t.Fatalf("decode envelope[%d]: %v", n, err)
	}
	return env
}

func writeCountForEvents(ws *fakeWS) int {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return len(ws.writes)
}

// ─────────────────────────────────────────────
// 1. TestMatchEventPattern_Exact
// ─────────────────────────────────────────────

func TestMatchEventPattern_Exact(t *testing.T) {
	if !MatchEventPattern([]string{"session.idle"}, "session.idle") {
		t.Fatal("exact match should return true")
	}
	if MatchEventPattern([]string{"session.idle"}, "session.output") {
		t.Fatal("non-matching exact should return false")
	}
}

// ─────────────────────────────────────────────
// 2. TestMatchEventPattern_StarWithinSegment
// ─────────────────────────────────────────────

func TestMatchEventPattern_StarWithinSegment(t *testing.T) {
	// "session.*" matches session.idle and session.output
	if !MatchEventPattern([]string{"session.*"}, "session.idle") {
		t.Fatal("session.* should match session.idle")
	}
	if !MatchEventPattern([]string{"session.*"}, "session.output") {
		t.Fatal("session.* should match session.output")
	}
	// "session.*" must NOT match session.x.y (dot crosses segment boundary)
	if MatchEventPattern([]string{"session.*"}, "session.x.y") {
		t.Fatal("session.* must not match session.x.y (crosses segment boundary)")
	}
	// "session.*" must NOT match just "session"
	if MatchEventPattern([]string{"session.*"}, "session") {
		t.Fatal("session.* must not match bare 'session'")
	}
}

// ─────────────────────────────────────────────
// 3. TestMatchEventPattern_DoubleStar
// ─────────────────────────────────────────────

func TestMatchEventPattern_DoubleStar(t *testing.T) {
	// "session.**" matches across segments
	if !MatchEventPattern([]string{"session.**"}, "session.x.y") {
		t.Fatal("session.** should match session.x.y")
	}
	if !MatchEventPattern([]string{"session.**"}, "session.idle") {
		t.Fatal("session.** should match session.idle")
	}
	// Should not match an unrelated prefix
	if MatchEventPattern([]string{"session.**"}, "plugin.session.idle") {
		t.Fatal("session.** should not match plugin.session.idle")
	}
}

// ─────────────────────────────────────────────
// 4. TestMatchEventPattern_EmptyGranted
// ─────────────────────────────────────────────

func TestMatchEventPattern_EmptyGranted(t *testing.T) {
	if MatchEventPattern([]string{}, "session.idle") {
		t.Fatal("empty granted should always return false")
	}
	if MatchEventPattern(nil, "session.idle") {
		t.Fatal("nil granted should always return false")
	}
}

// ─────────────────────────────────────────────
// 5. TestMatchEventPattern_LoneStar
// ─────────────────────────────────────────────

func TestMatchEventPattern_LoneStar(t *testing.T) {
	// A lone "*" should match any single-segment name
	if !MatchEventPattern([]string{"*"}, "session") {
		t.Fatal("* should match single-segment name 'session'")
	}
	if !MatchEventPattern([]string{"*"}, "idle") {
		t.Fatal("* should match single-segment name 'idle'")
	}
	// "*" should NOT match multi-segment names
	if MatchEventPattern([]string{"*"}, "session.idle") {
		t.Fatal("* should not match multi-segment 'session.idle'")
	}
}

// ─────────────────────────────────────────────
// 6. TestEvents_SubscribeCapGate
// ─────────────────────────────────────────────

func TestEvents_SubscribeCapGate(t *testing.T) {
	bus := &fakeBus{}
	gate := gateWithEventsPerms(t, []string{"session.idle"})
	api := NewEventsAPI(bus, gate)
	conn, _ := newTestConnForEvents(t, "testplugin")

	ctx := context.Background()

	// Request a name NOT in the grant → should get EPERM
	argsOutput, _ := json.Marshal(map[string]string{"name": "session.output"})
	_, err := api.Dispatch(ctx, "testplugin", "subscribe", argsOutput, conn)
	if err == nil {
		t.Fatal("expected error for denied event name, got nil")
	}
	pe, ok := err.(*PermError)
	if !ok {
		t.Fatalf("expected *PermError, got %T: %v", err, err)
	}
	if pe.Code != "EPERM" {
		t.Fatalf("expected EPERM, got %q", pe.Code)
	}

	// Request an allowed name → should succeed
	argsIdle, _ := json.Marshal(map[string]string{"name": "session.idle"})
	result, err2 := api.Dispatch(ctx, "testplugin", "subscribe", argsIdle, conn)
	if err2 != nil {
		t.Fatalf("subscribe session.idle should succeed, got: %v", err2)
	}
	m, ok2 := result.(map[string]string)
	if !ok2 {
		t.Fatalf("expected map[string]string result, got %T", result)
	}
	if m["subId"] == "" {
		t.Fatal("expected non-empty subId in result")
	}
}

// ─────────────────────────────────────────────
// 7. TestEvents_PublishRewritesPrefix
// ─────────────────────────────────────────────

func TestEvents_PublishRewritesPrefix(t *testing.T) {
	bus := &fakeBus{}
	gate := gateNoPerms(t) // publish doesn't need events cap
	api := NewEventsAPI(bus, gate)
	conn, _ := newTestConnForEvents(t, "kanban")

	ctx := context.Background()
	args, _ := json.Marshal(map[string]any{"name": "board.update", "data": map[string]string{"key": "val"}})
	_, err := api.Dispatch(ctx, "kanban", "publish", args, conn)
	if err != nil {
		t.Fatalf("publish should succeed: %v", err)
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()
	if len(bus.publishLog) != 1 {
		t.Fatalf("expected 1 publish record, got %d", len(bus.publishLog))
	}
	got := bus.publishLog[0].Name
	want := "plugin.kanban.board.update"
	if got != want {
		t.Fatalf("publish name: got %q, want %q", got, want)
	}
	// Original name must not appear
	if got == "board.update" {
		t.Fatal("original name must be rewritten")
	}
}

// ─────────────────────────────────────────────
// 8. TestEvents_PublishIgnoresCap
// ─────────────────────────────────────────────

func TestEvents_PublishIgnoresCap(t *testing.T) {
	bus := &fakeBus{}
	// Gate with no events permission at all
	cr := &fakeConsentReaderEvents{perms: []byte(`{}`)}
	gate := NewGate(cr, nil, slog.Default())
	api := NewEventsAPI(bus, gate)
	conn, _ := newTestConnForEvents(t, "myplugin")

	ctx := context.Background()
	args, _ := json.Marshal(map[string]any{"name": "task.done", "data": nil})
	_, err := api.Dispatch(ctx, "myplugin", "publish", args, conn)
	if err != nil {
		t.Fatalf("publish should succeed even without events cap: %v", err)
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()
	if len(bus.publishLog) == 0 {
		t.Fatal("expected publish to go through")
	}
}

// ─────────────────────────────────────────────
// 9. TestEvents_SubscribeDeliversChunkEnvelope
// ─────────────────────────────────────────────

func TestEvents_SubscribeDeliversChunkEnvelope(t *testing.T) {
	bus := &fakeBus{}
	gate := gateWithEventsPerms(t, []string{"session.*"})
	api := NewEventsAPI(bus, gate)
	conn, ws := newTestConnForEvents(t, "testplugin")

	ctx := context.Background()
	args, _ := json.Marshal(map[string]string{"name": "session.idle"})
	result, err := api.Dispatch(ctx, "testplugin", "subscribe", args, conn)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	subId := result.(map[string]string)["subId"]

	// Emit an event from the bus
	bus.emit("session.idle", map[string]string{"sessionId": "s1"})

	// Give the pump goroutine time to write
	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		if writeCountForEvents(ws) > 0 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	count := writeCountForEvents(ws)
	if count == 0 {
		t.Fatal("expected at least one envelope written to conn")
	}

	env := decodeWritten(t, ws, 0)
	if env.Stream != "chunk" {
		t.Fatalf("expected stream=chunk, got %q", env.Stream)
	}
	if env.ID != subId {
		t.Fatalf("expected envelope ID=%q, got %q", subId, env.ID)
	}
	if len(env.Data) == 0 {
		t.Fatal("expected non-empty data in chunk envelope")
	}
}

// ─────────────────────────────────────────────
// 10. TestEvents_UnsubscribeStopsDelivery
// ─────────────────────────────────────────────

func TestEvents_UnsubscribeStopsDelivery(t *testing.T) {
	bus := &fakeBus{}
	gate := gateWithEventsPerms(t, []string{"session.*"})
	api := NewEventsAPI(bus, gate)
	conn, ws := newTestConnForEvents(t, "testplugin")

	ctx := context.Background()
	subArgs, _ := json.Marshal(map[string]string{"name": "session.idle"})
	result, err := api.Dispatch(ctx, "testplugin", "subscribe", subArgs, conn)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	subId := result.(map[string]string)["subId"]

	// Unsubscribe
	unsubArgs, _ := json.Marshal(map[string]string{"subId": subId})
	_, err = api.Dispatch(ctx, "testplugin", "unsubscribe", unsubArgs, conn)
	if err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}

	// Give the goroutine time to observe the done channel
	time.Sleep(20 * time.Millisecond)

	countBefore := writeCountForEvents(ws)

	// Emit an event — should NOT be delivered
	bus.emit("session.idle", map[string]string{"sessionId": "s2"})
	time.Sleep(30 * time.Millisecond)

	countAfter := writeCountForEvents(ws)
	if countAfter != countBefore {
		t.Fatalf("expected no new envelopes after unsubscribe, but count went %d→%d", countBefore, countAfter)
	}
}

// ─────────────────────────────────────────────
// 11. TestEvents_NoGoroutineLeakAfterConnClose
// ─────────────────────────────────────────────

func TestEvents_NoGoroutineLeakAfterConnClose(t *testing.T) {
	bus := &fakeBus{}
	gate := gateWithEventsPerms(t, []string{"session.*"})
	api := NewEventsAPI(bus, gate)
	conn, _ := newTestConnForEvents(t, "testplugin")

	// Baseline goroutine count
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	before := runtime.NumGoroutine()

	ctx := context.Background()
	subArgs, _ := json.Marshal(map[string]string{"name": "session.idle"})
	_, err := api.Dispatch(ctx, "testplugin", "subscribe", subArgs, conn)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	// Close the conn — this should drain all subs and close their done channels.
	if err := conn.Close(1000, "test done"); err != nil {
		t.Logf("conn.Close: %v (non-fatal for fakeWS)", err)
	}

	// Wait up to 200ms for the pump goroutine to exit.
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		runtime.GC()
		current := runtime.NumGoroutine()
		if current <= before+2 {
			return // goroutines cleaned up
		}
		time.Sleep(5 * time.Millisecond)
	}

	runtime.GC()
	after := runtime.NumGoroutine()
	if after > before+2 {
		t.Fatalf("goroutine leak: before=%d, after conn.Close=%d (delta=%d, allowed ≤2)",
			before, after, after-before)
	}
}

// ─────────────────────────────────────────────
// 12. TestEvents_UnknownMethodReturnsEUNAVAIL
// ─────────────────────────────────────────────

func TestEvents_UnknownMethodReturnsEUNAVAIL(t *testing.T) {
	bus := &fakeBus{}
	gate := gateNoPerms(t)
	api := NewEventsAPI(bus, gate)
	conn, _ := newTestConnForEvents(t, "testplugin")

	ctx := context.Background()
	_, err := api.Dispatch(ctx, "testplugin", "frobnicate", json.RawMessage(`{}`), conn)
	if err == nil {
		t.Fatal("expected error for unknown method")
	}
	we, ok := err.(*WireError)
	if !ok {
		t.Fatalf("expected *WireError, got %T: %v", err, err)
	}
	if we.Code != "EUNAVAIL" {
		t.Fatalf("expected EUNAVAIL, got %q", we.Code)
	}
}

// ─────────────────────────────────────────────
// 13. TestEvents_MalformedArgsReturnsEINVAL
// ─────────────────────────────────────────────

func TestEvents_MalformedArgsReturnsEINVAL(t *testing.T) {
	bus := &fakeBus{}
	gate := gateWithEventsPerms(t, []string{"session.*"})
	api := NewEventsAPI(bus, gate)
	conn, _ := newTestConnForEvents(t, "testplugin")

	ctx := context.Background()

	// Malformed JSON for subscribe
	_, err := api.Dispatch(ctx, "testplugin", "subscribe", json.RawMessage(`{bad json`), conn)
	if err == nil {
		t.Fatal("expected error for malformed args")
	}
	we, ok := err.(*WireError)
	if !ok {
		t.Fatalf("expected *WireError, got %T: %v", err, err)
	}
	if we.Code != "EINVAL" {
		t.Fatalf("expected EINVAL, got %q", we.Code)
	}
}

// ─────────────────────────────────────────────
// Bonus: verify unsubscribe of unknown subId is a no-op
// ─────────────────────────────────────────────

func TestEvents_UnsubscribeUnknownSubIdNoOp(t *testing.T) {
	bus := &fakeBus{}
	gate := gateNoPerms(t)
	api := NewEventsAPI(bus, gate)
	conn, _ := newTestConnForEvents(t, "testplugin")

	ctx := context.Background()
	args, _ := json.Marshal(map[string]string{"subId": "nonexistent"})
	result, err := api.Dispatch(ctx, "testplugin", "unsubscribe", args, conn)
	if err != nil {
		t.Fatalf("unsubscribe unknown subId should be a no-op, got: %v", err)
	}
	if result != nil {
		t.Fatalf("unsubscribe should return nil, got %v", result)
	}
}

// ─────────────────────────────────────────────
// Bonus: verify publish returns nil result
// ─────────────────────────────────────────────

func TestEvents_PublishReturnsNil(t *testing.T) {
	bus := &fakeBus{}
	gate := gateNoPerms(t)
	api := NewEventsAPI(bus, gate)
	conn, _ := newTestConnForEvents(t, "testplugin")

	ctx := context.Background()
	args, _ := json.Marshal(map[string]any{"name": "evt", "data": "hello"})
	result, err := api.Dispatch(ctx, "testplugin", "publish", args, conn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("publish should return nil, got %v", result)
	}
}

// ─────────────────────────────────────────────
// checkEventsCap: no consent row
// ─────────────────────────────────────────────

func TestEvents_SubscribeNoConsentRow(t *testing.T) {
	bus := &fakeBus{}
	// Gate where plugin has no consent row at all
	cr := &fakeConsentReaderEvents{perms: nil} // nil → not found
	gate := NewGate(cr, nil, slog.Default())
	api := NewEventsAPI(bus, gate)
	conn, _ := newTestConnForEvents(t, "testplugin")

	ctx := context.Background()
	args, _ := json.Marshal(map[string]string{"name": "session.idle"})
	_, err := api.Dispatch(ctx, "testplugin", "subscribe", args, conn)
	if err == nil {
		t.Fatal("expected error when no consent row")
	}
	pe, ok := err.(*PermError)
	if !ok {
		t.Fatalf("expected *PermError, got %T: %v", err, err)
	}
	if pe.Code != "EPERM" {
		t.Fatalf("expected EPERM, got %q", pe.Code)
	}
}

// ─────────────────────────────────────────────
// publish: missing name returns EINVAL
// ─────────────────────────────────────────────

func TestEvents_PublishMissingNameEINVAL(t *testing.T) {
	bus := &fakeBus{}
	gate := gateNoPerms(t)
	api := NewEventsAPI(bus, gate)
	conn, _ := newTestConnForEvents(t, "testplugin")

	ctx := context.Background()
	args, _ := json.Marshal(map[string]any{"name": "", "data": nil})
	_, err := api.Dispatch(ctx, "testplugin", "publish", args, conn)
	if err == nil {
		t.Fatal("expected EINVAL for empty name")
	}
	we, ok := err.(*WireError)
	if !ok {
		t.Fatalf("expected *WireError, got %T", err)
	}
	if we.Code != "EINVAL" {
		t.Fatalf("expected EINVAL, got %q", we.Code)
	}
}

// ─────────────────────────────────────────────
// subscribe: empty name returns EINVAL
// ─────────────────────────────────────────────

func TestEvents_SubscribeMissingNameEINVAL(t *testing.T) {
	bus := &fakeBus{}
	gate := gateWithEventsPerms(t, []string{"session.*"})
	api := NewEventsAPI(bus, gate)
	conn, _ := newTestConnForEvents(t, "testplugin")

	ctx := context.Background()
	args, _ := json.Marshal(map[string]string{"name": ""})
	_, err := api.Dispatch(ctx, "testplugin", "subscribe", args, conn)
	if err == nil {
		t.Fatal("expected EINVAL for empty name")
	}
	we, ok := err.(*WireError)
	if !ok {
		t.Fatalf("expected *WireError, got %T", err)
	}
	if we.Code != "EINVAL" {
		t.Fatalf("expected EINVAL, got %q", we.Code)
	}
}

// ─────────────────────────────────────────────
// MatchEventPattern: suffix wildcard within segment
// ─────────────────────────────────────────────

func TestMatchEventPattern_SegmentSuffixWildcard(t *testing.T) {
	// "session.out*" matches "session.output" but not "session.idle"
	if !MatchEventPattern([]string{"session.out*"}, "session.output") {
		t.Fatal("session.out* should match session.output")
	}
	if MatchEventPattern([]string{"session.out*"}, "session.idle") {
		t.Fatal("session.out* should not match session.idle")
	}
}

// ─────────────────────────────────────────────
// MatchEventPattern: right-anchored segment wildcard
// ─────────────────────────────────────────────

func TestMatchEventPattern_SegmentPrefixWildcard(t *testing.T) {
	// "*idle" should match "session.idle" segment "idle"
	if !MatchEventPattern([]string{"session.*idle"}, "session.idle") {
		t.Fatal("session.*idle should match session.idle")
	}
	if MatchEventPattern([]string{"session.*idle"}, "session.output") {
		t.Fatal("session.*idle should not match session.output")
	}
}

// ─────────────────────────────────────────────
// WireError.Error()
// ─────────────────────────────────────────────

func TestWireError_ErrorMethod(t *testing.T) {
	we := &WireError{Code: "EPERM", Message: "no access"}
	got := we.Error()
	want := "EPERM: no access"
	if got != want {
		t.Fatalf("WireError.Error() = %q, want %q", got, want)
	}
}
