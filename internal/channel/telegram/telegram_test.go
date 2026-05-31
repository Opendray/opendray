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

// telegramServer is a minimal stand-in for api.telegram.org. Returns
// a canned getUpdates result once, then empty results, so the polling
// loop runs cleanly. Records every outbound API call by method.
type telegramServer struct {
	mu      sync.Mutex
	updates []tgUpdate
	calls   map[string][]map[string]any
	srv     *httptest.Server
	nextMID int
}

func newTelegramServer(updates []tgUpdate) *telegramServer {
	s := &telegramServer{
		updates: updates,
		calls:   make(map[string][]map[string]any),
		nextMID: 999,
	}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

func (s *telegramServer) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	method := apiMethod(r.URL.Path)

	switch method {
	case "getUpdates":
		s.mu.Lock()
		updates := s.updates
		s.updates = nil
		s.mu.Unlock()
		_ = json.NewEncoder(w).Encode(struct {
			Ok     bool       `json:"ok"`
			Result []tgUpdate `json:"result"`
		}{Ok: true, Result: updates})
	case "sendMessage":
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		s.record(method, payload)
		s.mu.Lock()
		mid := s.nextMID
		s.nextMID++
		s.mu.Unlock()
		resp := map[string]any{
			"ok":     true,
			"result": map[string]any{"message_id": mid},
		}
		_ = json.NewEncoder(w).Encode(resp)
	case "editMessageText", "sendChatAction", "answerCallbackQuery":
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		s.record(method, payload)
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	default:
		http.Error(w, "unknown method", http.StatusNotFound)
	}
}

func apiMethod(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return path
	}
	return path[idx+1:]
}

func (s *telegramServer) record(method string, payload map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls[method] = append(s.calls[method], payload)
}

func (s *telegramServer) close() { s.srv.Close() }

func (s *telegramServer) callsFor(method string) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]map[string]any, len(s.calls[method]))
	copy(out, s.calls[method])
	return out
}

// patchAPIBase rewires the package's apiBase to the test server.
func patchAPIBase(t *testing.T, base string) {
	t.Helper()
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
	defer func() { _ = tg.Stop(context.Background()) }()

	waitFor(t, 2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(received) >= 1
	})

	mu.Lock()
	if len(received) != 1 {
		mu.Unlock()
		t.Fatalf("got %d inbound, want 1", len(received))
	}
	got := received[0]
	mu.Unlock()
	if got.Text != "hi opendray" || got.ConversationID != "42" {
		t.Errorf("inbound = %+v", got)
	}
	if rc, ok := got.ReplyCtx.(ReplyCtx); !ok || rc.ChatID != 42 || rc.MessageID != 100 {
		t.Errorf("ReplyCtx = %+v, want {ChatID:42, MessageID:100}", got.ReplyCtx)
	}

	if err := tg.Send(context.Background(), channel.ChannelMessage{
		ChannelID: "ch_x", Text: "out",
	}); err != nil {
		t.Fatal(err)
	}
	if calls := srv.callsFor("sendMessage"); len(calls) != 1 || calls[0]["text"] != "out" {
		t.Errorf("sendMessage = %v", calls)
	}
}

func TestSend_HonoursReplyCtx(t *testing.T) {
	srv := newTelegramServer(nil)
	defer srv.close()
	patchAPIBase(t, srv.srv.URL)

	tg, _ := New("ch_x", json.RawMessage(`{"bot_token":"t","chat_id":1}`), nil)
	err := tg.Send(context.Background(), channel.ChannelMessage{
		Text:     "echo",
		ReplyCtx: ReplyCtx{ChatID: 99, MessageID: 555},
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := srv.callsFor("sendMessage")
	if len(calls) != 1 {
		t.Fatalf("got %d sendMessage calls", len(calls))
	}
	if got := toInt64(calls[0]["chat_id"]); got != 99 {
		t.Errorf("chat_id = %v, want 99", calls[0]["chat_id"])
	}
	rp, ok := calls[0]["reply_parameters"].(map[string]any)
	if !ok || toInt64(rp["message_id"]) != 555 {
		t.Errorf("reply_parameters = %v, want {message_id:555}", calls[0]["reply_parameters"])
	}
}

func TestSendCard_ProducesInlineKeyboard(t *testing.T) {
	srv := newTelegramServer(nil)
	defer srv.close()
	patchAPIBase(t, srv.srv.URL)

	tg, _ := New("ch_x", json.RawMessage(`{"bot_token":"t","chat_id":1}`), nil)
	cardSender, ok := tg.(channel.CardSender)
	if !ok {
		t.Fatal("Telegram does not implement CardSender")
	}
	card := &channel.Card{
		Header: &channel.CardHeader{Title: "Session idle"},
		Elements: []channel.CardElement{
			channel.CardMarkdown{Content: "Session abc went idle."},
			channel.CardActions{Buttons: [][]channel.ButtonOption{{
				{Text: "Resume", Value: "cmd:/resume abc"},
				{Text: "End", Value: "cmd:/cancel abc", Style: "danger"},
			}}},
		},
	}
	err := cardSender.SendCard(context.Background(), channel.ChannelMessage{
		ChannelID: "ch_x", ConversationID: "1",
	}, card)
	if err != nil {
		t.Fatal(err)
	}
	calls := srv.callsFor("sendMessage")
	if len(calls) != 1 {
		t.Fatalf("sendMessage calls = %d", len(calls))
	}
	text, _ := calls[0]["text"].(string)
	for _, want := range []string{"Session idle", "Session abc went idle."} {
		if !strings.Contains(text, want) {
			t.Errorf("card text missing %q\n--full--\n%s", want, text)
		}
	}
	rm, ok := calls[0]["reply_markup"].(map[string]any)
	if !ok {
		t.Fatalf("reply_markup missing: %v", calls[0])
	}
	rows, _ := rm["inline_keyboard"].([]any)
	if len(rows) != 1 {
		t.Fatalf("inline_keyboard rows = %d, want 1", len(rows))
	}
	row, _ := rows[0].([]any)
	if len(row) != 2 {
		t.Fatalf("button row = %d, want 2", len(row))
	}
	first, _ := row[0].(map[string]any)
	if first["text"] != "Resume" || first["callback_data"] != "cmd:/resume abc" {
		t.Errorf("first button = %v", first)
	}
}

// When the hub flags a message for the persistent control keyboard, the
// card's inline row is replaced by a ReplyKeyboardMarkup (keyboard +
// is_persistent), and the labels mirror channel.ControlKeyboardLayout.
func TestSendCard_ControlKeyboardReplacesInline(t *testing.T) {
	srv := newTelegramServer(nil)
	defer srv.close()
	patchAPIBase(t, srv.srv.URL)

	tg, _ := New("ch_x", json.RawMessage(`{"bot_token":"t","chat_id":1}`), nil)
	cs := tg.(channel.CardSender)
	card := &channel.Card{
		Elements: []channel.CardElement{
			channel.CardMarkdown{Content: "agent reply text"},
			// An inline row the flag must suppress.
			channel.CardActions{Buttons: [][]channel.ButtonOption{{
				{Text: "⏸ Stop", Value: "cmd:/confirm stop abc"},
			}}},
		},
	}
	err := cs.SendCard(context.Background(), channel.ChannelMessage{
		ChannelID: "ch_x", ConversationID: "1",
		Metadata: map[string]any{channel.MetaControlKeyboard: true},
	}, card)
	if err != nil {
		t.Fatal(err)
	}
	calls := srv.callsFor("sendMessage")
	if len(calls) != 1 {
		t.Fatalf("sendMessage calls = %d", len(calls))
	}
	rm, ok := calls[0]["reply_markup"].(map[string]any)
	if !ok {
		t.Fatalf("reply_markup missing: %v", calls[0])
	}
	if _, isInline := rm["inline_keyboard"]; isInline {
		t.Error("inline_keyboard should be suppressed when control keyboard is requested")
	}
	if persistent, _ := rm["is_persistent"].(bool); !persistent {
		t.Errorf("expected is_persistent=true, got %v", rm["is_persistent"])
	}
	rows, _ := rm["keyboard"].([]any)
	if len(rows) != len(channel.ControlKeyboardLayout()) {
		t.Fatalf("keyboard rows = %d, want %d", len(rows), len(channel.ControlKeyboardLayout()))
	}
	first, _ := rows[0].([]any)
	btn, _ := first[0].(map[string]any)
	if btn["text"] != "⏸ Stop" {
		t.Errorf("first key = %v, want label %q", btn, "⏸ Stop")
	}
}

func TestCallbackQuery_DeliveredAsAction(t *testing.T) {
	srv := newTelegramServer([]tgUpdate{
		{UpdateID: 7, CallbackQuery: &tgCallbackQuery{
			ID:   "cb-1",
			From: &tgUser{Username: "alice"},
			Message: &tgMessage{
				MessageID: 50, Date: time.Now().Unix(),
				Chat: tgChat{ID: 77, Type: "private"},
			},
			Data: "cmd:/cancel abc",
		}},
	})
	defer srv.close()
	patchAPIBase(t, srv.srv.URL)

	tg, _ := New("ch_x", json.RawMessage(`{"bot_token":"t","chat_id":77}`), nil)
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
	defer func() { _ = tg.Stop(context.Background()) }()

	waitFor(t, 2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(received) >= 1
	})

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("got %d inbound, want 1", len(received))
	}
	got := received[0]
	if action, ok := channel.DecodeAction(got.Text); !ok || action != "cmd:/cancel abc" {
		t.Errorf("DecodeAction(%q) = (%q, %v)", got.Text, action, ok)
	}
	if got.Metadata["callback_query_id"] != "cb-1" {
		t.Errorf("callback_query_id = %v", got.Metadata["callback_query_id"])
	}
}

// waitFor polls until cond returns true or timeout elapses.
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

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	}
	return 0
}

func TestSplitForTelegram_ShortMessage_NoSplit(t *testing.T) {
	out := splitForTelegram("hello world")
	if len(out) != 1 || out[0] != "hello world" {
		t.Errorf("short msg should not split: %v", out)
	}
}

func TestSplitForTelegram_LineBoundarySplit(t *testing.T) {
	// Build something just over the chunk size, with line boundaries.
	line := strings.Repeat("a", 1000)
	parts := []string{line, line, line, line, line} // 5 lines × 1000 = 5000 chars (+ separators)
	out := splitForTelegram(strings.Join(parts, "\n"))
	if len(out) < 2 {
		t.Fatalf("want ≥2 chunks, got %d", len(out))
	}
	for i, c := range out {
		if runesIn(c) > telegramChunkSize {
			t.Errorf("chunk %d size %d exceeds cap %d", i, runesIn(c), telegramChunkSize)
		}
	}
}

func TestSplitForTelegram_HardSplitMonsterLine(t *testing.T) {
	// Single very long line forces hard split.
	monster := strings.Repeat("漢", telegramChunkSize*2+50) // > 2 chunks
	out := splitForTelegram(monster)
	if len(out) < 3 {
		t.Fatalf("want ≥3 chunks, got %d", len(out))
	}
	rejoined := strings.Join(out, "")
	if rejoined != monster {
		t.Error("hard split lost or duplicated content")
	}
}

func TestSendCard_ChunksLongBody(t *testing.T) {
	srv := newTelegramServer(nil)
	defer srv.close()
	patchAPIBase(t, srv.srv.URL)

	tg, _ := New("ch_x", json.RawMessage(`{"bot_token":"t","chat_id":1}`), nil)
	cs := tg.(channel.CardSender)

	bigBody := strings.Repeat("一段很长的助手回复内容\n", 500) // ~6000+ runes
	card := &channel.Card{
		Header: &channel.CardHeader{Title: "Idle"},
		Elements: []channel.CardElement{
			channel.CardMarkdown{Content: bigBody},
			channel.CardActions{Buttons: [][]channel.ButtonOption{{
				{Text: "Resume", Value: "cmd:/resume abc"},
			}}},
		},
	}
	if err := cs.SendCard(context.Background(), channel.ChannelMessage{}, card); err != nil {
		t.Fatal(err)
	}
	calls := srv.callsFor("sendMessage")
	if len(calls) < 2 {
		t.Fatalf("expected multi-message split, got %d", len(calls))
	}
	last := calls[len(calls)-1]
	if last["reply_markup"] == nil {
		t.Error("buttons should be on the LAST chunk")
	}
	for i := 0; i < len(calls)-1; i++ {
		if calls[i]["reply_markup"] != nil {
			t.Errorf("chunk %d should not carry buttons", i)
		}
	}
}
