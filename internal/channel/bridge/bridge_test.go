package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/opendray/opendray-v2/internal/channel"
)

// fixture returns (broker, bridge, ws-client-url, cleanup).
func fixture(t *testing.T, cfg Config) (*Broker, *Bridge, string, func()) {
	t.Helper()
	broker := NewBroker()
	prevBroker := defaultBroker
	defaultBroker = broker
	t.Cleanup(func() { defaultBroker = prevBroker })

	raw, _ := json.Marshal(cfg)
	ch, err := Factory("ch_b", raw, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	br := ch.(*Bridge)

	hh := NewHandlers(broker, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r := chi.NewRouter()
	hh.Mount(r)
	srv := httptest.NewServer(r)
	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1) + "/channels/bridge/ws"
	cleanup := func() {
		_ = br.Stop(context.Background())
		srv.Close()
	}
	return broker, br, wsURL, cleanup
}

func dialAndRegister(t *testing.T, br *Bridge, wsURL, token, platform string, caps []channel.Capability) *websocket.Conn {
	t.Helper()
	header := http.Header{}
	header.Set("X-Bridge-Token", token)
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			t.Fatalf("dial: %v (status=%d)", err, resp.StatusCode)
		}
		t.Fatalf("dial: %v", err)
	}
	reg := map[string]any{
		"type":         "register",
		"platform":     platform,
		"capabilities": caps,
	}
	raw, _ := json.Marshal(reg)
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write register: %v", err)
	}
	// Bound the ack wait so a dropped/lost ack on a loaded CI runner
	// fails this helper in seconds rather than hanging the whole
	// `go test -timeout=5m` budget.
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, ackRaw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	// Reset the deadline so the caller can read further frames
	// without tripping over our short ack window.
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		t.Fatalf("clear read deadline: %v", err)
	}
	var ack map[string]any
	_ = json.Unmarshal(ackRaw, &ack)
	if ack["ok"] != true {
		t.Fatalf("register failed: %v", ack)
	}
	// The ack confirms the broker received our register frame, but
	// applying capabilities to the bridge's state map happens on a
	// different goroutine. Block until every declared capability
	// has propagated so callers can immediately call Send/SendCard
	// without racing the broker. 2s is generous on every runner we
	// observe; local typically lands in < 5ms.
	waitFor(t, 2*time.Second, func() bool {
		got := channel.Capabilities(br)
		for _, c := range caps {
			if !containsCap(got, c) {
				return false
			}
		}
		return true
	})
	return conn
}

func TestRegister_BadTokenRejected(t *testing.T) {
	_, br, wsURL, cleanup := fixture(t, Config{Name: "wechat", Token: "secret"})
	defer cleanup()
	if err := br.Start(context.Background(), nopInbound); err != nil {
		t.Fatal(err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?token=wrong", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	reg, _ := json.Marshal(map[string]any{
		"type": "register", "platform": "wechat",
	})
	_ = conn.WriteMessage(websocket.TextMessage, reg)
	_, ackRaw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	var ack map[string]any
	_ = json.Unmarshal(ackRaw, &ack)
	if ack["ok"] != false {
		t.Errorf("expected register_ack ok=false, got %v", ack)
	}
	if ack["error"] != "invalid token" {
		t.Errorf("error reason = %v", ack["error"])
	}
}

func TestRegister_AttachesBridgeAndDeliversInbound(t *testing.T) {
	_, br, wsURL, cleanup := fixture(t, Config{Name: "wechat", Token: "secret"})
	defer cleanup()

	var (
		mu       sync.Mutex
		received []channel.ChannelMessage
	)
	inbound := func(_ context.Context, msg channel.ChannelMessage) error {
		mu.Lock()
		received = append(received, msg)
		mu.Unlock()
		return nil
	}
	if err := br.Start(context.Background(), inbound); err != nil {
		t.Fatal(err)
	}

	conn := dialAndRegister(t, br, wsURL, "secret", "wechat",
		[]channel.Capability{channel.CapText, channel.CapCard, channel.CapButtons})
	defer conn.Close()

	// Text message
	msg, _ := json.Marshal(map[string]any{
		"type":            "message",
		"session_key":     "wechat:gid123:user42",
		"conversation_id": "gid123",
		"user_id":         "u42",
		"user_name":       "Alice",
		"text":            "hello opendray",
		"reply_ctx":       "wx-msg-ref-001",
	})
	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatal(err)
	}
	// card_action
	act, _ := json.Marshal(map[string]any{
		"type":            "card_action",
		"session_key":     "wechat:gid123:user42",
		"conversation_id": "gid123",
		"user_id":         "u42",
		"user_name":       "Alice",
		"action":          "cmd:/cancel sess1",
		"reply_ctx":       "wx-msg-ref-002",
	})
	if err := conn.WriteMessage(websocket.TextMessage, act); err != nil {
		t.Fatal(err)
	}

	waitFor(t, 2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(received) >= 2
	})

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("got %d inbound, want 2", len(received))
	}
	if received[0].Text != "hello opendray" || received[0].Author != "Alice" {
		t.Errorf("first inbound = %+v", received[0])
	}
	if rc, _ := received[0].ReplyCtx.(string); rc != "wx-msg-ref-001" {
		t.Errorf("reply_ctx = %v", received[0].ReplyCtx)
	}
	action, ok := channel.DecodeAction(received[1].Text)
	if !ok || action != "cmd:/cancel sess1" {
		t.Errorf("DecodeAction(%q) = (%q, %v)", received[1].Text, action, ok)
	}
}

func TestSendCard_GatedByCapability(t *testing.T) {
	_, br, wsURL, cleanup := fixture(t, Config{Name: "wechat", Token: "secret"})
	defer cleanup()
	if err := br.Start(context.Background(), nopInbound); err != nil {
		t.Fatal(err)
	}

	// Adapter only declares "text" — no card capability.
	conn := dialAndRegister(t, br, wsURL, "secret", "wechat",
		[]channel.Capability{channel.CapText})
	defer conn.Close()

	cs := channel.CardSender(br)
	err := cs.SendCard(context.Background(), channel.ChannelMessage{ChannelID: br.ID()}, &channel.Card{})
	if !errors.Is(err, channel.ErrNotSupported) {
		t.Errorf("SendCard without cap = %v, want ErrNotSupported", err)
	}
}

func TestSendCard_WhenSupportedShipsFrame(t *testing.T) {
	_, br, wsURL, cleanup := fixture(t, Config{Name: "wechat", Token: "secret"})
	defer cleanup()
	if err := br.Start(context.Background(), nopInbound); err != nil {
		t.Fatal(err)
	}

	conn := dialAndRegister(t, br, wsURL, "secret", "wechat",
		[]channel.Capability{channel.CapText, channel.CapCard})
	defer conn.Close()

	// Drain async frames from the adapter side.
	frames := make(chan map[string]any, 4)
	go func() {
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				close(frames)
				return
			}
			var f map[string]any
			_ = json.Unmarshal(raw, &f)
			frames <- f
		}
	}()

	card := &channel.Card{
		Header: &channel.CardHeader{Title: "Hi"},
		Elements: []channel.CardElement{
			channel.CardMarkdown{Content: "From bridge"},
		},
	}
	cs := channel.CardSender(br)
	if err := cs.SendCard(context.Background(), channel.ChannelMessage{
		ChannelID: br.ID(), ConversationID: "gid", ReplyCtx: "ref-1",
	}, card); err != nil {
		t.Fatal(err)
	}

	select {
	case f, ok := <-frames:
		if !ok {
			t.Fatal("frame channel closed before send_card")
		}
		if f["type"] != "send_card" {
			t.Errorf("frame type = %v, want send_card", f["type"])
		}
		if f["reply_ctx"] != "ref-1" {
			t.Errorf("reply_ctx = %v", f["reply_ctx"])
		}
		if f["session_key"] != "bridge:gid:gid" {
			t.Errorf("session_key = %v", f["session_key"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for send_card frame")
	}
}

func TestDeclaredCapabilities_OnlyWhatAdapterClaimed(t *testing.T) {
	_, br, wsURL, cleanup := fixture(t, Config{Name: "wechat", Token: "secret"})
	defer cleanup()
	if err := br.Start(context.Background(), nopInbound); err != nil {
		t.Fatal(err)
	}

	// Pre-attach: only CapText.
	if got := channel.Capabilities(br); !containsCap(got, channel.CapText) || containsCap(got, channel.CapCard) {
		t.Errorf("pre-attach caps = %v, want only [text]", got)
	}

	conn := dialAndRegister(t, br, wsURL, "secret", "wechat",
		[]channel.Capability{channel.CapText, channel.CapButtons})
	defer conn.Close()

	got := channel.Capabilities(br)
	if !containsCap(got, channel.CapButtons) || containsCap(got, channel.CapCard) {
		t.Errorf("post-attach caps = %v, want includes 'buttons' but not 'card'", got)
	}
}

func TestExtractToken_Sources(t *testing.T) {
	cases := []struct {
		name   string
		mut    func(*http.Request)
		expect string
	}{
		{
			name:   "query",
			mut:    func(r *http.Request) { r.URL.RawQuery = "token=q1" },
			expect: "q1",
		},
		{
			name:   "header X-Bridge-Token",
			mut:    func(r *http.Request) { r.Header.Set("X-Bridge-Token", "h1") },
			expect: "h1",
		},
		{
			name:   "Authorization Bearer",
			mut:    func(r *http.Request) { r.Header.Set("Authorization", "Bearer b1") },
			expect: "b1",
		},
		{
			name:   "none",
			mut:    func(*http.Request) {},
			expect: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u, _ := url.Parse("http://x/y")
			r := &http.Request{URL: u, Header: http.Header{}}
			c.mut(r)
			if got := extractToken(r); got != c.expect {
				t.Errorf("extractToken = %q, want %q", got, c.expect)
			}
		})
	}
}

func nopInbound(_ context.Context, _ channel.ChannelMessage) error { return nil }

func containsCap(caps []channel.Capability, target channel.Capability) bool {
	for _, c := range caps {
		if c == target {
			return true
		}
	}
	return false
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("waitFor timed out after %s", timeout)
}
