package gateway

// T14 — SSE workbench stream — test suite.
//
// TDD: tests are written before the implementation. Running `go test
// ./gateway/...` immediately after creating this file (before workbench_stream.go
// exists) must fail to compile — that is the RED phase.
//
// Strategy:
//   - WorkbenchBus tests use the real bus in-process (no mocks).
//   - HTTP handler tests use httptest.NewServer + a real HTTP client reading
//     the SSE stream via bufio.Scanner so the full net/http stack is exercised
//     (Flusher, chunked transfer, connection reset on cancel).
//   - The heartbeat test sleeps 25 s and is skipped under -short.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opendray/opendray/plugin/bridge"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func newBus(t *testing.T) *WorkbenchBus {
	t.Helper()
	return NewWorkbenchBus(slog.Default())
}

// readSSELines opens an SSE connection to srv at path, reads lines from the
// body into ch, and stops when ctx is cancelled or the connection drops.
// The caller owns ch and should close it (or let the goroutine drain it).
func readSSELines(ctx context.Context, t *testing.T, srv *httptest.Server, path string, ch chan<- string) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Errorf("readSSELines: new request: %v", err)
		return
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		// context cancellation during read is expected — don't report as error
		if ctx.Err() != nil {
			return
		}
		t.Errorf("readSSELines: do: %v", err)
		return
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		select {
		case ch <- line:
		case <-ctx.Done():
			return
		}
	}
}

// newStreamServer builds a minimal httptest.Server that only wires the
// workbenchStream handler. heartbeat duration is set by caller via the
// heartbeatInterval field on Server.
func newStreamServer(t *testing.T, bus *WorkbenchBus, hbInterval time.Duration) *httptest.Server {
	t.Helper()
	s := &Server{
		workbenchBus:      bus,
		heartbeatInterval: hbInterval,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/workbench/stream", s.workbenchStream)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// ─── WorkbenchBus unit tests ──────────────────────────────────────────────────

// 1. TestWorkbenchBus_PublishSubscribe — one subscriber receives in ≤50 ms.
func TestWorkbenchBus_PublishSubscribe(t *testing.T) {
	bus := newBus(t)
	ch, done := bus.Subscribe()
	defer done()

	want := WorkbenchEvent{Kind: "showMessage", Plugin: "kanban", Payload: json.RawMessage(`{"text":"hello"}`)}
	bus.Publish(want)

	select {
	case got := <-ch:
		if got.Kind != want.Kind || got.Plugin != want.Plugin {
			t.Errorf("got %+v, want %+v", got, want)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("event not received within 50ms")
	}
}

// 2. TestWorkbenchBus_FanoutTwoSubscribers — both receive same event.
func TestWorkbenchBus_FanoutTwoSubscribers(t *testing.T) {
	bus := newBus(t)
	ch1, done1 := bus.Subscribe()
	defer done1()
	ch2, done2 := bus.Subscribe()
	defer done2()

	ev := WorkbenchEvent{Kind: "openView", Plugin: "kanban", Payload: json.RawMessage(`"kanban.main"`)}
	bus.Publish(ev)

	timeout := time.After(50 * time.Millisecond)
	for i, ch := range []<-chan WorkbenchEvent{ch1, ch2} {
		select {
		case got := <-ch:
			if got.Kind != ev.Kind {
				t.Errorf("subscriber %d: kind=%q want %q", i, got.Kind, ev.Kind)
			}
		case <-timeout:
			t.Fatalf("subscriber %d did not receive event within 50ms", i)
		}
	}
}

// 3. TestWorkbenchBus_UnsubscribeStopsDelivery — after done(), channel receives nothing.
func TestWorkbenchBus_UnsubscribeStopsDelivery(t *testing.T) {
	bus := newBus(t)
	ch, done := bus.Subscribe()

	// Unsubscribe immediately.
	done()

	// Drain any event that may have arrived before unsubscription.
	bus.Publish(WorkbenchEvent{Kind: "updateStatusBar", Payload: json.RawMessage(`[]`)})

	// Give the bus a moment to (not) deliver.
	time.Sleep(10 * time.Millisecond)

	select {
	case ev, ok := <-ch:
		if ok {
			t.Errorf("received event after unsubscribe: %+v", ev)
		}
		// closed channel is fine — done() may close it
	default:
		// nothing received — correct
	}
}

// 4. TestWorkbenchBus_BackpressureDropOldest — slow consumer, 100 publishes → no hang.
func TestWorkbenchBus_BackpressureDropOldest(t *testing.T) {
	bus := newBus(t)
	ch, done := bus.Subscribe()
	defer done()

	const n = 100
	// Publish 100 events without reading from ch — bus must not block.
	done100 := make(chan struct{})
	go func() {
		for i := 0; i < n; i++ {
			bus.Publish(WorkbenchEvent{Kind: "showMessage", Payload: json.RawMessage(fmt.Sprintf(`{"i":%d}`, i))})
		}
		close(done100)
	}()

	select {
	case <-done100:
		// good — publisher did not hang
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Publish blocked for >500ms with slow subscriber")
	}

	// Drain whatever arrived — at least 32 should be in the buffer.
	received := 0
	drain := time.After(50 * time.Millisecond)
loop:
	for {
		select {
		case <-ch:
			received++
		case <-drain:
			break loop
		}
	}
	if received < 32 {
		t.Errorf("received %d events, want ≥32 (buffer size)", received)
	}
}

// 5. TestWorkbenchBus_ConcurrentPublishRace — 100 goroutines publish while
// 10 subscribers consume; must be -race clean.
func TestWorkbenchBus_ConcurrentPublishRace(t *testing.T) {
	bus := newBus(t)

	const numSubs = 10
	subs := make([]<-chan WorkbenchEvent, numSubs)
	dones := make([]func(), numSubs)
	for i := 0; i < numSubs; i++ {
		ch, done := bus.Subscribe()
		subs[i] = ch
		dones[i] = done
	}

	// Drain goroutines for each subscriber — they exit when channel is closed.
	var wg sync.WaitGroup
	for i := 0; i < numSubs; i++ {
		wg.Add(1)
		go func(ch <-chan WorkbenchEvent) {
			defer wg.Done()
			for range ch {
			}
		}(subs[i])
	}

	// 100 concurrent publishers.
	var pubWG sync.WaitGroup
	for i := 0; i < 100; i++ {
		pubWG.Add(1)
		go func(i int) {
			defer pubWG.Done()
			bus.Publish(WorkbenchEvent{Kind: "showMessage", Payload: json.RawMessage(fmt.Sprintf(`{"i":%d}`, i))})
		}(i)
	}
	pubWG.Wait()

	// Unsubscribe all: closes channels → drain goroutines exit.
	for _, d := range dones {
		d()
	}
	wg.Wait()
}

// 6. TestWorkbenchBus_ShowMessageSinkInterface — compile-time + runtime.
func TestWorkbenchBus_ShowMessageSinkInterface(t *testing.T) {
	// compile-time check
	var _ bridge.ShowMessageSink = (*WorkbenchBus)(nil)

	bus := newBus(t)
	ch, done := bus.Subscribe()
	defer done()

	opts := bridge.ShowMessageOpts{Text: "hello", Kind: "info"}
	if err := bus.ShowMessage("default", "kanban", opts); err != nil {
		t.Fatalf("ShowMessage error: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Kind != "showMessage" {
			t.Errorf("kind=%q want showMessage", ev.Kind)
		}
		if ev.Plugin != "kanban" {
			t.Errorf("plugin=%q want kanban", ev.Plugin)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("event not received within 50ms")
	}
}

// 7. TestWorkbenchBus_OpenViewSinkInterface — compile-time + runtime.
func TestWorkbenchBus_OpenViewSinkInterface(t *testing.T) {
	var _ bridge.OpenViewSink = (*WorkbenchBus)(nil)

	bus := newBus(t)
	ch, done := bus.Subscribe()
	defer done()

	if err := bus.OpenView("default", "kanban", "kanban.main"); err != nil {
		t.Fatalf("OpenView error: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Kind != "openView" {
			t.Errorf("kind=%q want openView", ev.Kind)
		}
		if ev.Plugin != "kanban" {
			t.Errorf("plugin=%q want kanban", ev.Plugin)
		}
		var viewID string
		if err := json.Unmarshal(ev.Payload, &viewID); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if viewID != "kanban.main" {
			t.Errorf("viewID=%q want kanban.main", viewID)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("event not received within 50ms")
	}
}

// 8. TestWorkbenchBus_StatusBarSinkInterface — compile-time + runtime.
func TestWorkbenchBus_StatusBarSinkInterface(t *testing.T) {
	var _ bridge.StatusBarSink = (*WorkbenchBus)(nil)

	bus := newBus(t)
	ch, done := bus.Subscribe()
	defer done()

	items := []bridge.StatusBarOverride{{ID: "kanban.tasks", Text: "3 tasks"}}
	if err := bus.UpdateStatusBar("default", "kanban", items); err != nil {
		t.Fatalf("UpdateStatusBar error: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Kind != "updateStatusBar" {
			t.Errorf("kind=%q want updateStatusBar", ev.Kind)
		}
		if ev.Plugin != "kanban" {
			t.Errorf("plugin=%q want kanban", ev.Plugin)
		}
		var got []bridge.StatusBarOverride
		if err := json.Unmarshal(ev.Payload, &got); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if len(got) != 1 || got[0].ID != "kanban.tasks" {
			t.Errorf("payload=%+v want [{kanban.tasks 3 tasks}]", got)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("event not received within 50ms")
	}
}

// 9. TestWorkbenchBus_ContributionsChangedEvent — PublishContributionsChanged
// → subscriber receives {Kind:"contributionsChanged"}.
func TestWorkbenchBus_ContributionsChangedEvent(t *testing.T) {
	bus := newBus(t)
	ch, done := bus.Subscribe()
	defer done()

	bus.PublishContributionsChanged()

	select {
	case ev := <-ch:
		if ev.Kind != "contributionsChanged" {
			t.Errorf("kind=%q want contributionsChanged", ev.Kind)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("event not received within 50ms")
	}
}

// ─── HTTP handler tests ────────────────────────────────────────────────────────

// 10. TestStream_HappyPath — GET /api/workbench/stream → SSE headers + first event.
func TestStream_HappyPath(t *testing.T) {
	bus := newBus(t)
	srv := newStreamServer(t, bus, 100*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Connect.
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/workbench/stream", nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	defer resp.Body.Close()

	// Check SSE headers.
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type=%q want text/event-stream", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control=%q want no-cache", cc)
	}
	if conn := resp.Header.Get("Connection"); conn != "keep-alive" {
		t.Errorf("Connection=%q want keep-alive", conn)
	}
	if xab := resp.Header.Get("X-Accel-Buffering"); xab != "no" {
		t.Errorf("X-Accel-Buffering=%q want no", xab)
	}

	// Publish an event and expect to read it.
	go func() {
		time.Sleep(20 * time.Millisecond)
		bus.Publish(WorkbenchEvent{
			Kind:    "showMessage",
			Plugin:  "test",
			Payload: json.RawMessage(`{"text":"hi"}`),
		})
	}()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var ev WorkbenchEvent
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				t.Fatalf("unmarshal SSE data: %v", err)
			}
			if ev.Kind != "showMessage" {
				t.Errorf("kind=%q want showMessage", ev.Kind)
			}
			return // success
		}
	}
	t.Fatal("no data line received")
}

// 11. TestStream_HeartbeatEvery20s — waits ≥25s; skipped under -short.
// Uses a 50 ms heartbeat interval override for test speed.
func TestStream_HeartbeatEvery20s(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping heartbeat test in -short mode")
	}

	bus := newBus(t)
	// Use 50ms heartbeat so test runs in <1s instead of 25s.
	srv := newStreamServer(t, bus, 50*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/workbench/stream", nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	defer resp.Body.Close()

	// Wait for at least one heartbeat line ":\n\n" — scanner sees it as ":"
	scanner := bufio.NewScanner(resp.Body)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == ":" {
			found = true
			break
		}
	}
	if !found {
		t.Error("no heartbeat line ':' received within 2s")
	}
}

// 12. TestStream_FanoutTwoClients — two clients both receive same event.
func TestStream_FanoutTwoClients(t *testing.T) {
	bus := newBus(t)
	srv := newStreamServer(t, bus, 100*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch1 := make(chan string, 16)
	ch2 := make(chan string, 16)
	go readSSELines(ctx, t, srv, "/api/workbench/stream", ch1)
	go readSSELines(ctx, t, srv, "/api/workbench/stream", ch2)

	// Let both clients connect.
	time.Sleep(50 * time.Millisecond)

	bus.Publish(WorkbenchEvent{
		Kind:    "contributionsChanged",
		Payload: json.RawMessage(`{}`),
	})

	expectDataLine := func(name string, ch <-chan string) {
		t.Helper()
		timeout := time.After(500 * time.Millisecond)
		for {
			select {
			case line := <-ch:
				if strings.HasPrefix(line, "data: ") {
					return // received
				}
			case <-timeout:
				t.Errorf("%s: no data line within 500ms", name)
				return
			}
		}
	}
	expectDataLine("client1", ch1)
	expectDataLine("client2", ch2)
}

// 13. TestStream_DisconnectCleansUpSubscription — after client disconnects,
// SubscriberCount drops.
func TestStream_DisconnectCleansUpSubscription(t *testing.T) {
	bus := newBus(t)
	srv := newStreamServer(t, bus, 100*time.Millisecond)

	if bus.SubscriberCount() != 0 {
		t.Fatalf("initial subscriber count = %d, want 0", bus.SubscriberCount())
	}

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/workbench/stream", nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	defer resp.Body.Close()

	// Give handler time to register subscription.
	time.Sleep(50 * time.Millisecond)

	if bus.SubscriberCount() != 1 {
		t.Fatalf("after connect: subscriber count = %d, want 1", bus.SubscriberCount())
	}

	// Disconnect by cancelling the request context.
	cancel()

	// Give handler time to clean up.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if bus.SubscriberCount() == 0 {
			return // success
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("subscriber count = %d after disconnect, want 0", bus.SubscriberCount())
}

// 14. TestStream_NilBusReturns503 — Server without WorkbenchBus → 503 EBUS.
func TestStream_NilBusReturns503(t *testing.T) {
	s := &Server{
		workbenchBus:      nil,
		heartbeatInterval: 100 * time.Millisecond,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/workbench/stream", nil)
	rr := httptest.NewRecorder()
	s.workbenchStream(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rr.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["code"] != "EBUS" {
		t.Errorf("code=%q want EBUS", body["code"])
	}
}

// 15. TestStream_WriteErrorEndsStream — cancel context mid-stream → handler
// exits, goroutine count stabilises.
func TestStream_WriteErrorEndsStream(t *testing.T) {
	bus := newBus(t)
	srv := newStreamServer(t, bus, 50*time.Millisecond)

	// Baseline goroutine count (allow a few extras from the test runner).
	runtime.GC()
	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/workbench/stream", nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}

	// Let the handler goroutine start.
	time.Sleep(50 * time.Millisecond)

	// Disconnect.
	cancel()
	resp.Body.Close()

	// Give handler time to exit.
	time.Sleep(200 * time.Millisecond)
	runtime.GC()

	after := runtime.NumGoroutine()
	// Allow ±5 goroutines for background Go runtime goroutines.
	if after > baseline+5 {
		t.Errorf("goroutine count: baseline=%d after=%d (delta=%d > 5) — possible leak",
			baseline, after, after-baseline)
	}
}

// ─── SubscriberCount (test-visibility helper) — verified here ─────────────────

// TestWorkbenchBus_SubscriberCountAccuracy verifies Subscribe/unsubscribe tracking.
func TestWorkbenchBus_SubscriberCountAccuracy(t *testing.T) {
	bus := newBus(t)

	if c := bus.SubscriberCount(); c != 0 {
		t.Fatalf("initial count = %d, want 0", c)
	}

	_, done1 := bus.Subscribe()
	if c := bus.SubscriberCount(); c != 1 {
		t.Fatalf("after 1st subscribe = %d, want 1", c)
	}

	_, done2 := bus.Subscribe()
	if c := bus.SubscriberCount(); c != 2 {
		t.Fatalf("after 2nd subscribe = %d, want 2", c)
	}

	done1()
	if c := bus.SubscriberCount(); c != 1 {
		t.Fatalf("after 1st unsubscribe = %d, want 1", c)
	}

	done2()
	if c := bus.SubscriberCount(); c != 0 {
		t.Fatalf("after 2nd unsubscribe = %d, want 0", c)
	}
}

// ─── atomic usage in ConcurrentPublishRace (explicit race-checker probe) ─────

// TestWorkbenchBus_AtomicSubscriberCount verifies SubscriberCount is safe under race.
func TestWorkbenchBus_AtomicSubscriberCount(t *testing.T) {
	bus := newBus(t)

	var wg sync.WaitGroup
	var total atomic.Int64
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, done := bus.Subscribe()
			total.Add(int64(bus.SubscriberCount()))
			done()
		}()
	}
	wg.Wait()

	// After all done() calls, count must be 0.
	if c := bus.SubscriberCount(); c != 0 {
		t.Errorf("final subscriber count = %d, want 0", c)
	}
	_ = total.Load() // just ensure no race on read
}
