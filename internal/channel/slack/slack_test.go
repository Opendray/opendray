package slack

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/opendray/opendray-v2/internal/channel"
)

// slackServer mimics the subset of slack.com/api we use.
type slackServer struct {
	mu    sync.Mutex
	calls map[string][]map[string]any
	srv   *httptest.Server
}

func newSlackServer() *slackServer {
	s := &slackServer{calls: make(map[string][]map[string]any)}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

func (s *slackServer) handle(w http.ResponseWriter, r *http.Request) {
	method := strings.TrimPrefix(r.URL.Path, "/")
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	auth := r.Header.Get("Authorization")
	var payload map[string]any
	if len(body) > 0 {
		_ = json.Unmarshal(body, &payload)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payload["_auth"] = auth
	s.mu.Lock()
	s.calls[method] = append(s.calls[method], payload)
	s.mu.Unlock()

	switch method {
	case "apps.connections.open":
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "url": "wss://test/"})
	case "chat.postMessage", "chat.update":
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"channel": payload["channel"],
			"ts":      "1700000000.000001",
		})
	default:
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "method_not_supported"})
	}
}

func (s *slackServer) close() { s.srv.Close() }
func (s *slackServer) callsFor(m string) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]map[string]any, len(s.calls[m]))
	copy(out, s.calls[m])
	return out
}

func patchAPI(t *testing.T, base string) {
	t.Helper()
	prev := apiBaseOverride
	apiBaseOverride = base + "/"
	t.Cleanup(func() { apiBaseOverride = prev })
}

func TestNew_RequiresTokens(t *testing.T) {
	if _, err := New("ch_x", json.RawMessage(`{}`), nil); err == nil {
		t.Fatal("expected error for missing tokens")
	}
	if _, err := New("ch_x", json.RawMessage(`{"bot_token":"x"}`), nil); err == nil {
		t.Fatal("expected error for missing app_token")
	}
}

func TestSend_PostsToChannel(t *testing.T) {
	srv := newSlackServer()
	defer srv.close()
	patchAPI(t, srv.srv.URL)

	c, err := New("ch_x", json.RawMessage(`{"bot_token":"xoxb-bot","app_token":"xapp-app","channel_id":"C111"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Send(context.Background(), channel.ChannelMessage{Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	calls := srv.callsFor("chat.postMessage")
	if len(calls) != 1 {
		t.Fatalf("got %d posts", len(calls))
	}
	if calls[0]["channel"] != "C111" || calls[0]["text"] != "hi" {
		t.Errorf("payload = %v", calls[0])
	}
	if !strings.Contains(calls[0]["_auth"].(string), "xoxb-bot") {
		t.Errorf("auth header = %v", calls[0]["_auth"])
	}
}

func TestSend_HonoursReplyCtx(t *testing.T) {
	srv := newSlackServer()
	defer srv.close()
	patchAPI(t, srv.srv.URL)

	c, _ := New("ch_x", json.RawMessage(`{"bot_token":"xoxb","app_token":"xapp","channel_id":"C111"}`), nil)
	err := c.Send(context.Background(), channel.ChannelMessage{
		Text:     "echo",
		ReplyCtx: ReplyCtx{ChannelID: "C999", ThreadTS: "1700000000.000001"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := srv.callsFor("chat.postMessage")[0]
	if got["channel"] != "C999" || got["thread_ts"] != "1700000000.000001" {
		t.Errorf("routing = %v", got)
	}
}

func TestSendCard_RendersBlocks(t *testing.T) {
	srv := newSlackServer()
	defer srv.close()
	patchAPI(t, srv.srv.URL)

	c, _ := New("ch_x", json.RawMessage(`{"bot_token":"xoxb","app_token":"xapp","channel_id":"C111"}`), nil)
	cs := c.(channel.CardSender)

	card := &channel.Card{
		Header: &channel.CardHeader{Title: "Session idle"},
		Elements: []channel.CardElement{
			channel.CardMarkdown{Content: "Session abc went idle."},
			channel.CardDivider{},
			channel.CardActions{Buttons: [][]channel.ButtonOption{{
				{Text: "Resume", Value: "cmd:/resume abc", Style: "primary"},
				{Text: "End", Value: "cmd:/cancel abc", Style: "danger"},
			}}},
		},
	}
	if err := cs.SendCard(context.Background(), channel.ChannelMessage{}, card); err != nil {
		t.Fatal(err)
	}
	call := srv.callsFor("chat.postMessage")[0]
	blocks, ok := call["blocks"].([]any)
	if !ok || len(blocks) != 4 {
		t.Fatalf("blocks = %v", call["blocks"])
	}
	header, _ := blocks[0].(map[string]any)
	if header["type"] != "header" {
		t.Errorf("header block = %v", header)
	}
	actions, _ := blocks[3].(map[string]any)
	if actions["type"] != "actions" {
		t.Errorf("actions block = %v", actions)
	}
	elements, _ := actions["elements"].([]any)
	if len(elements) != 2 {
		t.Fatalf("button count = %d", len(elements))
	}
	first, _ := elements[0].(map[string]any)
	if first["value"] != "cmd:/resume abc" || first["style"] != "primary" {
		t.Errorf("button = %v", first)
	}
}

func TestUpdateMessage_CallsChatUpdate(t *testing.T) {
	srv := newSlackServer()
	defer srv.close()
	patchAPI(t, srv.srv.URL)

	c, _ := New("ch_x", json.RawMessage(`{"bot_token":"xoxb","app_token":"xapp","channel_id":"C111"}`), nil)
	mu := c.(channel.MessageUpdater)
	if err := mu.UpdateMessage(context.Background(), channel.ChannelMessage{}, "1700000000.000001", "edited"); err != nil {
		t.Fatal(err)
	}
	call := srv.callsFor("chat.update")[0]
	if call["channel"] != "C111" || call["ts"] != "1700000000.000001" || call["text"] != "edited" {
		t.Errorf("update payload = %v", call)
	}
}

func TestCapabilities(t *testing.T) {
	c, _ := New("ch_x", json.RawMessage(`{"bot_token":"xoxb","app_token":"xapp","channel_id":"C111"}`), nil)
	caps := channel.Capabilities(c)
	want := map[channel.Capability]bool{
		channel.CapText:           true,
		channel.CapCard:           true,
		channel.CapButtons:        true,
		channel.CapUpdateMessage:  true,
		channel.CapReplyToMessage: true,
	}
	got := map[channel.Capability]bool{}
	for _, k := range caps {
		got[k] = true
	}
	for k := range want {
		if !got[k] {
			t.Errorf("missing capability %s", k)
		}
	}
}

func TestRouting_Fallback(t *testing.T) {
	c, _ := New("ch_x", json.RawMessage(`{"bot_token":"xoxb","app_token":"xapp","channel_id":"C111"}`), nil)
	s := c.(*Slack)
	cases := []struct {
		name        string
		msg         channel.ChannelMessage
		wantChannel string
		wantThread  string
	}{
		{"falls back to cfg", channel.ChannelMessage{}, "C111", ""},
		{"conversation_id wins", channel.ChannelMessage{ConversationID: "C222"}, "C222", ""},
		{"replyctx wins", channel.ChannelMessage{ConversationID: "C222", ReplyCtx: ReplyCtx{ChannelID: "C333", ThreadTS: "t1"}}, "C333", "t1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotCh, gotThread := s.routing(c.msg)
			if gotCh != c.wantChannel || gotThread != c.wantThread {
				t.Errorf("got (%q, %q), want (%q, %q)", gotCh, gotThread, c.wantChannel, c.wantThread)
			}
		})
	}
}
