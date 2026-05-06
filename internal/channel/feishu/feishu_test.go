package feishu

import (
	"bytes"
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

type feishuServer struct {
	mu    sync.Mutex
	calls map[string][]map[string]any
	srv   *httptest.Server
}

func newFeishuServer() *feishuServer {
	s := &feishuServer{calls: make(map[string][]map[string]any)}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

func (s *feishuServer) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	var payload map[string]any
	if len(body) > 0 {
		_ = json.Unmarshal(body, &payload)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payload["_path"] = r.URL.Path + "?" + r.URL.RawQuery
	payload["_auth"] = r.Header.Get("Authorization")
	s.mu.Lock()
	s.calls[r.URL.Path] = append(s.calls[r.URL.Path], payload)
	s.mu.Unlock()

	switch r.URL.Path {
	case "/open-apis/auth/v3/tenant_access_token/internal":
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":                0,
			"msg":                 "ok",
			"tenant_access_token": "t-abc-123",
			"expire":              7200,
		})
	default:
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok"})
	}
}

func (s *feishuServer) close() { s.srv.Close() }
func (s *feishuServer) callsFor(p string) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]map[string]any, len(s.calls[p]))
	copy(out, s.calls[p])
	return out
}

func patchAPI(t *testing.T, base string) {
	t.Helper()
	prev := apiBaseOverride
	apiBaseOverride = base
	t.Cleanup(func() { apiBaseOverride = prev })
}

func TestNew_RequiresCredentials(t *testing.T) {
	if _, err := New("ch_x", json.RawMessage(`{}`), nil); err == nil {
		t.Fatal("expected error for missing app_id/app_secret")
	}
}

func TestSend_TextToChat(t *testing.T) {
	srv := newFeishuServer()
	defer srv.close()
	patchAPI(t, srv.srv.URL)

	c, err := New("ch_x", json.RawMessage(`{"app_id":"A","app_secret":"S","chat_id":"oc_abc"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Send(context.Background(), channel.ChannelMessage{Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	calls := srv.callsFor("/open-apis/im/v1/messages")
	if len(calls) != 1 {
		t.Fatalf("got %d sends", len(calls))
	}
	got := calls[0]
	if got["receive_id"] != "oc_abc" || got["msg_type"] != "text" {
		t.Errorf("payload = %v", got)
	}
	if !strings.Contains(got["content"].(string), "\"text\":\"hi\"") {
		t.Errorf("content = %v", got["content"])
	}
	if !strings.HasPrefix(got["_auth"].(string), "Bearer t-abc-123") {
		t.Errorf("auth = %v", got["_auth"])
	}
}

func TestSend_RepliesViaReplyEndpoint(t *testing.T) {
	srv := newFeishuServer()
	defer srv.close()
	patchAPI(t, srv.srv.URL)

	c, _ := New("ch_x", json.RawMessage(`{"app_id":"A","app_secret":"S","chat_id":"oc_abc"}`), nil)
	err := c.Send(context.Background(), channel.ChannelMessage{
		Text:     "ack",
		ReplyCtx: ReplyCtx{ChatID: "oc_abc", MessageID: "om_xyz"},
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := srv.callsFor("/open-apis/im/v1/messages/om_xyz/reply")
	if len(calls) != 1 {
		t.Fatalf("expected reply call, got %v", calls)
	}
}

func TestSendCard_RendersInteractiveCard(t *testing.T) {
	srv := newFeishuServer()
	defer srv.close()
	patchAPI(t, srv.srv.URL)

	c, _ := New("ch_x", json.RawMessage(`{"app_id":"A","app_secret":"S","chat_id":"oc_abc"}`), nil)
	cs := c.(channel.CardSender)
	card := &channel.Card{
		Header: &channel.CardHeader{Title: "Idle", Color: "yellow"},
		Elements: []channel.CardElement{
			channel.CardMarkdown{Content: "Session abc went idle."},
			channel.CardActions{Buttons: [][]channel.ButtonOption{{
				{Text: "Resume", Value: "cmd:/resume", Style: "primary"},
				{Text: "End", Value: "cmd:/cancel", Style: "danger"},
			}}},
		},
	}
	if err := cs.SendCard(context.Background(), channel.ChannelMessage{}, card); err != nil {
		t.Fatal(err)
	}
	got := srv.callsFor("/open-apis/im/v1/messages")[0]
	if got["msg_type"] != "interactive" {
		t.Fatalf("msg_type = %v", got["msg_type"])
	}
	var card2 map[string]any
	_ = json.Unmarshal([]byte(got["content"].(string)), &card2)
	if card2["schema"] != "2.0" {
		t.Errorf("schema = %v", card2["schema"])
	}
	header, _ := card2["header"].(map[string]any)
	if header == nil || header["template"] != "yellow" {
		t.Errorf("header = %v", header)
	}
	body, _ := card2["body"].(map[string]any)
	elements, _ := body["elements"].([]any)
	if len(elements) < 2 {
		t.Fatalf("elements = %v", elements)
	}
}

func TestHandleWebhook_URLVerification(t *testing.T) {
	c, _ := New("ch_x", json.RawMessage(`{"app_id":"A","app_secret":"S","verification_token":"vt"}`), nil)
	fc := c.(*Feishu)
	body := []byte(`{"type":"url_verification","challenge":"chal-1","token":"vt"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/channels/ch_x/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()
	fc.HandleWebhook(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["challenge"] != "chal-1" {
		t.Errorf("challenge = %v", resp)
	}
}

func TestHandleWebhook_RejectsBadToken(t *testing.T) {
	c, _ := New("ch_x", json.RawMessage(`{"app_id":"A","app_secret":"S","verification_token":"vt"}`), nil)
	fc := c.(*Feishu)
	body := []byte(`{"type":"url_verification","challenge":"x","token":"WRONG"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/channels/ch_x/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()
	fc.HandleWebhook(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestHandleWebhook_DispatchesMessageEvent(t *testing.T) {
	c, _ := New("ch_x", json.RawMessage(`{"app_id":"A","app_secret":"S"}`), nil)
	fc := c.(*Feishu)
	var (
		mu       sync.Mutex
		received []channel.ChannelMessage
	)
	if err := fc.Start(context.Background(), func(_ context.Context, msg channel.ChannelMessage) error {
		mu.Lock()
		received = append(received, msg)
		mu.Unlock()
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	envelope := map[string]any{
		"schema": "2.0",
		"header": map[string]any{
			"event_type": "im.message.receive_v1",
		},
		"event": map[string]any{
			"sender": map[string]any{
				"sender_id": map[string]any{"open_id": "ou_alice", "user_id": "u_alice"},
			},
			"message": map[string]any{
				"message_id":   "om_123",
				"chat_id":      "oc_abc",
				"chat_type":    "p2p",
				"message_type": "text",
				"content":      "{\"text\":\"hello opendray\"}",
				"create_time":  "1700000000",
			},
		},
	}
	raw, _ := json.Marshal(envelope)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/channels/ch_x/webhook", bytes.NewReader(raw))
	w := httptest.NewRecorder()
	fc.HandleWebhook(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("got %d inbound, want 1", len(received))
	}
	got := received[0]
	if got.Text != "hello opendray" || got.ConversationID != "oc_abc" || got.Author != "ou_alice" {
		t.Errorf("inbound = %+v", got)
	}
	rc, ok := got.ReplyCtx.(ReplyCtx)
	if !ok || rc.ChatID != "oc_abc" || rc.MessageID != "om_123" {
		t.Errorf("ReplyCtx = %+v", got.ReplyCtx)
	}
}

func TestCapabilities(t *testing.T) {
	c, _ := New("ch_x", json.RawMessage(`{"app_id":"A","app_secret":"S"}`), nil)
	caps := channel.Capabilities(c)
	want := map[channel.Capability]bool{
		channel.CapText:           true,
		channel.CapCard:           true,
		channel.CapButtons:        true,
		channel.CapReplyToMessage: true,
	}
	got := map[channel.Capability]bool{}
	for _, k := range caps {
		got[k] = true
	}
	for k := range want {
		if !got[k] {
			t.Errorf("missing %s", k)
		}
	}
}
