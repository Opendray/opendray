package channel

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// recordingInputter captures every Input call with a timestamp so
// tests can assert both the byte split (text vs Enter) and the
// ordering / gap between them.
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
	// Copy data because callers may reuse the slice.
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

// The headline regression: forwarding "hello" to a session must
// produce TWO PTY writes — body first, Enter byte second — with
// the body alone in the first write. A single combined "hello\r"
// write is what made Gemini swallow the Enter.
func TestSubmitToSession_TwoWritesWithEnterAlone(t *testing.T) {
	h := newTestHub(t)
	rec := &recordingInputter{}
	h.SetSessionInput(rec)

	if err := h.submitToSession(context.Background(), "ses_x", "hello"); err != nil {
		t.Fatal(err)
	}
	got := rec.snapshot()
	if len(got) != 2 {
		t.Fatalf("want 2 PTY writes, got %d: %v", len(got), got)
	}
	if string(got[0].data) != "hello" {
		t.Errorf("first write should be body only, got %q", got[0].data)
	}
	if len(got[1].data) != 1 || got[1].data[0] != '\r' {
		t.Errorf("second write should be \\r alone, got %q", got[1].data)
	}
	if got[0].sid != "ses_x" || got[1].sid != "ses_x" {
		t.Errorf("both writes should target the same session: %v", got)
	}
}

// The pause between the two writes is what gives the Ink input
// handler time to process the body as keystrokes before the Enter
// byte arrives. Verify it's actually sleeping at least the
// declared delay (with a small fudge for scheduling jitter).
func TestSubmitToSession_PausesBetweenWrites(t *testing.T) {
	h := newTestHub(t)
	rec := &recordingInputter{}
	h.SetSessionInput(rec)

	if err := h.submitToSession(context.Background(), "ses_x", "hi"); err != nil {
		t.Fatal(err)
	}
	got := rec.snapshot()
	if len(got) != 2 {
		t.Fatalf("want 2 writes, got %d", len(got))
	}
	gap := got[1].ts.Sub(got[0].ts)
	// Allow 5 ms slack — submitDelay is 30 ms, so even a noisy
	// scheduler should comfortably clear 25 ms.
	if gap < submitDelay-5*time.Millisecond {
		t.Errorf("write gap %v shorter than submitDelay %v; submit may be batched",
			gap, submitDelay)
	}
}

// Empty body should skip the body write but still emit the Enter
// alone — useful for "tap Enter" gestures that the chat platform
// might surface as empty-text submissions.
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

// A cancelled ctx during the pause should bail out — we already
// wrote the body, so the test asserts (body sent, no Enter, error
// returned). The caller surfaces this to the chat as a delivery
// failure.
func TestSubmitToSession_CancelledDuringPause(t *testing.T) {
	h := newTestHub(t)
	rec := &recordingInputter{}
	h.SetSessionInput(rec)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call so the select's ctx.Done path fires

	err := h.submitToSession(ctx, "ses_x", "hello")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
	got := rec.snapshot()
	if len(got) != 1 || string(got[0].data) != "hello" {
		t.Errorf("body should have been sent before cancel: %v", got)
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
	// h.input is nil
	err := h.submitToSession(context.Background(), "ses_x", "hi")
	if err == nil {
		t.Error("want error when no SessionInputter wired, got nil")
	}
}
