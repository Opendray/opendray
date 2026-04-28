package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/opendray/opendray-v2/internal/channel"
)

// telegramServer is a minimal stand-in for api.telegram.org. It returns
// canned getUpdates results once, then empty results, so the polling
// loop runs cleanly.
type telegramServer struct {
	mu       sync.Mutex
	updates  []tgUpdate
	sent     []map[string]any
	srv      *httptest.Server
}

func newTelegramServer(updates []tgUpdate) *telegramServer {
	s := &telegramServer{updates: updates}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

func (s *telegramServer) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	if strings.Contains(r.URL.Path, "/getUpdates") {
		s.mu.Lock()
		updates := s.updates
		s.updates = nil
		s.mu.Unlock()
		_ = json.NewEncoder(w).Encode(struct {
			Ok     bool       `json:"ok"`
			Result []tgUpdate `json:"result"`
		}{Ok: true, Result: updates})
		return
	}
	if strings.Contains(r.URL.Path, "/sendMessage") {
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		s.mu.Lock()
		s.sent = append(s.sent, payload)
		s.mu.Unlock()
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
		return
	}
	http.Error(w, "unknown method", http.StatusNotFound)
}

func (s *telegramServer) close()                 { s.srv.Close() }
func (s *telegramServer) outbound() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]map[string]any, len(s.sent))
	copy(out, s.sent)
	return out
}

// patchAPIBase rewires the package's apiBase to the test server. Tests
// run sequentially; restored via t.Cleanup.
func patchAPIBase(t *testing.T, base string) {
	t.Helper()
	// our impl uses package-level const; we cheat with a thin wrapper
	// since tests live in the same package.
	prev := apiBaseOverride
	apiBaseOverride = base + "/bot"
	t.Cleanup(func() { apiBaseOverride = prev })
}

func TestNew_RequiresBotToken(t *testing.T) {
	if _, err := New("ch_x", json.RawMessage(`{}`), nil); err == nil {
		t.Fatal("expected error for missing bot_token")
	}
}

func TestSend_NoChatID(t *testing.T) {
	tg, err := New("ch_x", json.RawMessage(`{"bot_token":"t"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	err = tg.Send(context.Background(), channel.ChannelMessage{Text: "hi"})
	if err == nil || !strings.Contains(err.Error(), "chat_id") {
		t.Fatalf("err=%v", err)
	}
}

func TestStartPollAndSend(t *testing.T) {
	srv := newTelegramServer([]tgUpdate{
		{UpdateID: 1, Message: &tgMessage{
			MessageID: 100, Date: time.Now().Unix(),
			Chat: tgChat{ID: 42, Type: "private"},
			From: &tgUser{Username: "alice"},
			Text: "hi opendray",
		}},
	})
	defer srv.close()
	patchAPIBase(t, srv.srv.URL)

	cfg := json.RawMessage(`{"bot_token":"abc","chat_id":42}`)
	tg, err := New("ch_x", cfg, nil)
	if err != nil {
		t.Fatal(err)
	}

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

	if err := tg.Start(context.Background(), inbound); err != nil {
		t.Fatal(err)
	}
	defer tg.Stop(context.Background())

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		got := len(received)
		mu.Unlock()
		if got >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("got %d inbound, want 1", len(received))
	}
	if received[0].Text != "hi opendray" || received[0].ConversationID != "42" {
		t.Errorf("inbound = %+v", received[0])
	}

	if err := tg.Send(context.Background(), channel.ChannelMessage{
		ChannelID: "ch_x", Text: "out",
	}); err != nil {
		t.Fatal(err)
	}
	if got := srv.outbound(); len(got) != 1 || got[0]["text"] != "out" {
		t.Errorf("outbound = %v", got)
	}
}
