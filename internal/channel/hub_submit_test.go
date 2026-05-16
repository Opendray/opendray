package channel

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// recordingInputter captures every Input call with a timestamp so
// tests can assert both the byte split (one write per rune + one
// for Enter) and the inter-key timing.
type recordingInputter struct {
	mu    sync.Mutex
	calls []recordedInput
	err   error // optional injection: returned on Input call
}

type recordedInput struct {
	sid  string
	data []byte
	ts   time.Time
}

func (r *recordingInputter) Input(_ context.Context, sid string, data []byte) error {
	if r.err != nil {
		return r.err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	r.calls = append(r.calls, recordedInput{sid: sid, data: cp, ts: time.Now()})
	return nil
}

func (r *recordingInputter) snapshot() []recordedInput {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]recordedInput, len(r.calls))
	copy(out, r.calls)
	return out
}

// The headline regression: forwarding "hi" must produce three
// PTY writes — 'h', 'i', '\r' — each on its own. A single combined
// write made Gemini classify the burst as a paste and swallow the
// trailing Enter. xterm.js emits one write per real keystroke; we
// mirror that.
func TestSubmitToSession_RuneByRuneThenEnter(t *testing.T) {
	h := newTestHub(t)
	rec := &recordingInputter{}
	h.SetSessionInput(rec)

	if err := h.submitToSession(context.Background(), "ses_x", "hi"); err != nil {
		t.Fatal(err)
	}
	got := rec.snapshot()
	if len(got) != 3 {
		t.Fatalf("want 3 PTY writes (h, i, \\r), got %d: %v", len(got), got)
	}
	if string(got[0].data) != "h" {
		t.Errorf("write[0] should be 'h' alone, got %q", got[0].data)
	}
	if string(got[1].data) != "i" {
		t.Errorf("write[1] should be 'i' alone, got %q", got[1].data)
	}
	if len(got[2].data) != 1 || got[2].data[0] != '\r' {
		t.Errorf("write[2] should be \\r alone, got %q", got[2].data)
	}
	for i, c := range got {
		if c.sid != "ses_x" {
			t.Errorf("write[%d] sid = %q, want ses_x", i, c.sid)
		}
	}
}

// UTF-8 must be split by RUNE, not byte. A Chinese character is
// 3 bytes — sending one byte at a time would deliver three garbled
// bytes to the CLI's UTF-8 decoder. Each rune is one keystroke.
func TestSubmitToSession_UTF8SplitByRune(t *testing.T) {
	h := newTestHub(t)
	rec := &recordingInputter{}
	h.SetSessionInput(rec)

	if err := h.submitToSession(context.Background(), "ses_x", "你好"); err != nil {
		t.Fatal(err)
	}
	got := rec.snapshot()
	// 2 runes + 1 Enter = 3 writes
	if len(got) != 3 {
		t.Fatalf("want 3 writes (你, 好, \\r), got %d", len(got))
	}
	if string(got[0].data) != "你" {
		t.Errorf("first rune mangled: %q", got[0].data)
	}
	if string(got[1].data) != "好" {
		t.Errorf("second rune mangled: %q", got[1].data)
	}
	if got[2].data[0] != '\r' {
		t.Errorf("expected Enter last, got %q", got[2].data)
	}
}

// Between every adjacent pair of writes there must be a delay —
// either perRuneDelay (between two runes / between last rune and
// Enter pause start) or at least submitDelay (between last rune
// and Enter). We don't need to nail the exact value, just verify
// the writes aren't all bunched in the same millisecond (which is
// what a single PTY write or zero-delay loop would produce).
func TestSubmitToSession_HasInterKeyDelay(t *testing.T) {
	h := newTestHub(t)
	rec := &recordingInputter{}
	h.SetSessionInput(rec)

	if err := h.submitToSession(context.Background(), "ses_x", "ab"); err != nil {
		t.Fatal(err)
	}
	got := rec.snapshot()
	if len(got) != 3 {
		t.Fatalf("want 3 writes, got %d", len(got))
	}
	betweenRunes := got[1].ts.Sub(got[0].ts)
	// Allow generous slack: perRuneDelay is 5ms, but a noisy
	// scheduler might still come close to zero. Require at least
	// 1ms so single-write or zero-delay regressions can't pass.
	if betweenRunes < time.Millisecond {
		t.Errorf("no delay between runes: %v — keystrokes may be batched", betweenRunes)
	}
	beforeEnter := got[2].ts.Sub(got[1].ts)
	// Final pause should be the larger submitDelay (30ms).
	if beforeEnter < submitDelay-5*time.Millisecond {
		t.Errorf("settle pause before Enter %v shorter than submitDelay %v",
			beforeEnter, submitDelay)
	}
}

// Empty body skips the typing loop but still emits the Enter —
// useful for chat platforms that might surface tap-Enter gestures
// as empty-text submissions.
func TestSubmitToSession_EmptyBodyOnlySendsEnter(t *testing.T) {
	h := newTestHub(t)
	rec := &recordingInputter{}
	h.SetSessionInput(rec)

	if err := h.submitToSession(context.Background(), "ses_x", ""); err != nil {
		t.Fatal(err)
	}
	got := rec.snapshot()
	if len(got) != 1 {
		t.Fatalf("want 1 write (Enter only), got %d", len(got))
	}
	if len(got[0].data) != 1 || got[0].data[0] != '\r' {
		t.Errorf("expected \\r only, got %q", got[0].data)
	}
}

// Cancelled context aborts cleanly between keystrokes — whatever
// runes already arrived at the PTY are not "undone", but the
// caller gets context.Canceled so it can report failure to the
// chat and won't keep typing into a defunct session.
func TestSubmitToSession_CancelledDuringTyping(t *testing.T) {
	h := newTestHub(t)
	rec := &recordingInputter{}
	h.SetSessionInput(rec)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call so the very first select fires

	err := h.submitToSession(ctx, "ses_x", "hello")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
	// At least the first rune should have been written before the
	// post-rune select observed the cancel.
	got := rec.snapshot()
	if len(got) == 0 {
		t.Error("expected at least the first rune to have been written")
	}
	for _, c := range got {
		if len(c.data) > 0 && c.data[0] == '\r' {
			t.Error("Enter should not have been emitted after cancel")
		}
	}
}

// Underlying PTY write failure surfaces as an error and stops
// further writes — caller decides whether to retry.
func TestSubmitToSession_PropagatesPtyWriteError(t *testing.T) {
	h := newTestHub(t)
	boom := errors.New("pty closed")
	rec := &recordingInputter{err: boom}
	h.SetSessionInput(rec)

	err := h.submitToSession(context.Background(), "ses_x", "hi")
	if !errors.Is(err, boom) {
		t.Errorf("want %v, got %v", boom, err)
	}
}

// nil input means SetSessionInput was never called — the surface
// should refuse cleanly instead of nil-derefing.
func TestSubmitToSession_NoInputterReturnsError(t *testing.T) {
	h := newTestHub(t)
	err := h.submitToSession(context.Background(), "ses_x", "hi")
	if err == nil {
		t.Error("want error when no SessionInputter wired, got nil")
	}
}
