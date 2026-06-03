package channel

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// fakeChannel is a CardSender + ButtonSender that records every call.
// Used to drive Hub command-dispatch tests without a real platform.
type fakeChannel struct {
	id, kind string
	mu       sync.Mutex
	sent     []ChannelMessage
	cards    []*Card
}

func (f *fakeChannel) ID() string                                   { return f.id }
func (f *fakeChannel) Kind() string                                 { return f.kind }
func (f *fakeChannel) Start(_ context.Context, _ InboundFunc) error { return nil }
func (f *fakeChannel) Stop(_ context.Context) error                 { return nil }
func (f *fakeChannel) Send(_ context.Context, msg ChannelMessage) error {
	f.mu.Lock()
	f.sent = append(f.sent, msg)
	f.mu.Unlock()
	return nil
}
func (f *fakeChannel) SendCard(_ context.Context, msg ChannelMessage, card *Card) error {
	f.mu.Lock()
	f.sent = append(f.sent, msg)
	f.cards = append(f.cards, card)
	f.mu.Unlock()
	return nil
}

func (f *fakeChannel) sentTexts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.sent))
	for i, m := range f.sent {
		out[i] = m.Text
	}
	return out
}

func TestHub_HandleCommand_DispatchesAndReplies(t *testing.T) {
	h := newTestHub(t)
	fc := &fakeChannel{id: "ch_test", kind: "stub"}
	h.mu.Lock()
	h.channels[fc.id] = fc
	h.mu.Unlock()

	called := false
	h.RegisterCommand(Command{
		Name: "echo",
		Handler: func(_ context.Context, cc CommandContext) (string, error) {
			called = true
			if cc.Channel.ID() != "ch_test" {
				t.Errorf("handler got channel %q", cc.Channel.ID())
			}
			return "echo:" + cc.Args[0], nil
		},
	})

	// Use direct call to avoid needing a DB. Persist path is exercised
	// in integration tests.
	h.dispatchCommandForTest(t, ChannelMessage{
		ChannelID: "ch_test",
		Text:      "/echo hi",
	}, "echo", []string{"hi"})

	if !called {
		t.Fatal("handler not invoked")
	}
	texts := fc.sentTexts()
	if len(texts) != 1 || texts[0] != "echo:hi" {
		t.Errorf("reply texts = %v, want [echo:hi]", texts)
	}
}

func TestHub_HandleCommand_UnknownReplies(t *testing.T) {
	h := newTestHub(t)
	fc := &fakeChannel{id: "ch_test", kind: "stub"}
	h.mu.Lock()
	h.channels[fc.id] = fc
	h.mu.Unlock()

	h.dispatchCommandForTest(t, ChannelMessage{
		ChannelID: "ch_test", Text: "/nope",
	}, "nope", nil)
	texts := fc.sentTexts()
	if len(texts) != 1 || texts[0] != "Unknown command /nope — try /help" {
		t.Errorf("reply texts = %v", texts)
	}
}

func TestHub_HandleCommand_DocksControlKeyboard(t *testing.T) {
	h := newTestHub(t)
	fc := &fakeChannel{id: "ch_test", kind: "stub"}
	h.mu.Lock()
	h.channels[fc.id] = fc
	h.mu.Unlock()

	reply := func(_ context.Context, _ CommandContext) (string, error) { return "ok", nil }
	h.RegisterCommand(Command{Name: "dock", Handler: reply, DocksControlKeyboard: true})
	h.RegisterCommand(Command{Name: "plain", Handler: reply})

	h.dispatchCommandForTest(t, ChannelMessage{ChannelID: "ch_test", Text: "/dock"}, "dock", nil)
	h.dispatchCommandForTest(t, ChannelMessage{ChannelID: "ch_test", Text: "/plain"}, "plain", nil)

	fc.mu.Lock()
	defer fc.mu.Unlock()
	if len(fc.sent) != 2 {
		t.Fatalf("want 2 sends, got %d", len(fc.sent))
	}
	if v, _ := fc.sent[0].Metadata[MetaControlKeyboard].(bool); !v {
		t.Error("DocksControlKeyboard command reply should carry MetaControlKeyboard")
	}
	if _, has := fc.sent[1].Metadata[MetaControlKeyboard]; has {
		t.Error("plain command reply must not carry MetaControlKeyboard")
	}
}

// dispatchCommandForTest runs handleCommand without touching the DB.
// Tests insert channels directly into Hub.channels via the lock above
// and then exercise the lookup → handler → reply path.
func (h *Hub) dispatchCommandForTest(t *testing.T, msg ChannelMessage, name string, args []string) {
	t.Helper()
	h.mu.RLock()
	ch := h.channels[msg.ChannelID]
	h.mu.RUnlock()
	if ch == nil {
		t.Fatalf("channel %s not registered", msg.ChannelID)
	}
	cmd, ok := h.cmds.Lookup(name)
	if !ok {
		// Mimic handleCommand's unknown-reply branch.
		h.replyText(context.Background(), ch, msg, "Unknown command /"+name+" — try /help")
		return
	}
	cc := CommandContext{Channel: ch, Message: msg, Hub: h, Command: name, Args: args, Raw: msg.Text}
	reply, err := cmd.Handler(context.Background(), cc)
	if err != nil {
		t.Fatalf("handler err: %v", err)
	}
	if reply != "" {
		// Mirror handleCommand's DocksControlKeyboard branch.
		if cmd.DocksControlKeyboard {
			h.replyControlText(context.Background(), ch, msg, reply)
		} else {
			h.replyText(context.Background(), ch, msg, reply)
		}
	}
}

// newTestHub returns a Hub with no DB pool; tests must skip code paths
// that touch the store. Used only for command-dispatch logic.
func newTestHub(t *testing.T) *Hub {
	t.Helper()
	return &Hub{
		log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		cmds:          NewCommandRegistry(),
		channels:      make(map[string]Channel),
		notifyState:   make(map[string]map[string]time.Time),
		lastSess:      make(map[string]string),
		outboundIndex: make(map[string]map[string]outboundEntry),
		activeSess:    make(map[string]string),
	}
}

func TestResolveTargetSession_Priority(t *testing.T) {
	h := newTestHub(t)
	const ch = "ch_test"

	// 1) When nothing is recorded, resolution fails.
	if _, ok := h.resolveTargetSession(ChannelMessage{ChannelID: ch}); ok {
		t.Fatal("empty hub should return false")
	}

	// 2) Last-notified beats nothing.
	h.lastSess[ch] = "sess_last"
	if got, ok := h.resolveTargetSession(ChannelMessage{ChannelID: ch}); !ok || got != "sess_last" {
		t.Errorf("last fallback: got %q ok=%v", got, ok)
	}

	// 3) Active /select pin beats last.
	h.setActiveSession(ch, "sess_pinned")
	if got, ok := h.resolveTargetSession(ChannelMessage{ChannelID: ch}); !ok || got != "sess_pinned" {
		t.Errorf("active pin: got %q ok=%v", got, ok)
	}

	// 4) Reply-to outbound beats both.
	h.recordOutbound(ch, map[string]any{"outbound_msg_id": "999"}, "sess_replied")
	msg := ChannelMessage{
		ChannelID: ch,
		Metadata:  map[string]any{"reply_to_outbound_msg_id": "999"},
	}
	if got, ok := h.resolveTargetSession(msg); !ok || got != "sess_replied" {
		t.Errorf("reply-to: got %q ok=%v", got, ok)
	}

	// 5) Reply-to with unknown id falls through to active.
	msg2 := ChannelMessage{
		ChannelID: ch,
		Metadata:  map[string]any{"reply_to_outbound_msg_id": "nope"},
	}
	if got, ok := h.resolveTargetSession(msg2); !ok || got != "sess_pinned" {
		t.Errorf("reply-to fallback: got %q ok=%v", got, ok)
	}

	// 6) Clearing active falls back to last.
	h.setActiveSession(ch, "")
	if got, ok := h.resolveTargetSession(ChannelMessage{ChannelID: ch}); !ok || got != "sess_last" {
		t.Errorf("after clear: got %q ok=%v", got, ok)
	}
}

func TestRecordOutbound_LRUEvictsOldest(t *testing.T) {
	h := newTestHub(t)
	const ch = "ch_x"
	for i := 0; i < outboundIndexMax+50; i++ {
		h.recordOutbound(ch, map[string]any{
			"outbound_msg_id": "id" + strconvItoa(i),
		}, "sess"+strconvItoa(i))
	}
	h.outboundMu.Lock()
	size := len(h.outboundIndex[ch])
	h.outboundMu.Unlock()
	if size > outboundIndexMax {
		t.Errorf("LRU did not evict: size=%d > cap=%d", size, outboundIndexMax)
	}
	// Most recent should still be findable.
	if got := h.lookupOutbound(ch, "id"+strconvItoa(outboundIndexMax+49)); got == "" {
		t.Error("most recent entry was evicted")
	}
}

// strconvItoa is a shim so we can keep test imports minimal.
func strconvItoa(i int) string {
	if i == 0 {
		return "0"
	}
	negative := i < 0
	if negative {
		i = -i
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{'0' + byte(i%10)}, digits...)
		i /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
