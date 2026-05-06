// Package feishu is opendray's bundled Feishu (Lark) channel.
//
// Feishu uses an HTTP webhook for inbound events (configured in the
// "Event Subscriptions" section of the Lark developer console) and
// the standard /open-apis/im/v1/messages endpoint for outbound — the
// tenant_access_token is refreshed lazily via app_id + app_secret.
//
// The webhook URL each channel needs is:
//
//	https://<opendray-host>/api/v1/channels/<channel_id>/webhook
//
// Configure that URL in the Lark dev portal → Event subscriptions →
// "Request URL". The first POST Feishu sends is a URL-verification
// challenge; opendray echoes the challenge back automatically.
//
// Capabilities implemented: text, card (interactive), buttons,
// reply_to_message (reply API).
package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/opendray/opendray-v2/internal/channel"
)

const (
	apiBase     = "https://open.feishu.cn"
	httpTimeout = 30 * time.Second
	tokenSlack  = 5 * time.Minute // refresh this much before expiry
)

// Test seam.
var apiBaseOverride = ""

func apiURL(path string) string {
	if apiBaseOverride != "" {
		return apiBaseOverride + path
	}
	return apiBase + path
}

func init() { channel.Register("feishu", New) }

type config struct {
	AppID             string `json:"app_id"`
	AppSecret         string `json:"app_secret"`
	VerificationToken string `json:"verification_token,omitempty"`
	ChatID            string `json:"chat_id,omitempty"`
}

// ReplyCtx routes a Feishu outbound back to the right thread.
type ReplyCtx struct {
	ChatID    string // open_chat_id of the originating chat
	MessageID string // om_xxx parent message — when set, replies via reply API
}

type Feishu struct {
	id     string
	cfg    config
	log    *slog.Logger
	client *http.Client

	mu       sync.Mutex
	cancel   context.CancelFunc
	done     chan struct{}
	inbound  channel.InboundFunc
	tokenMu  sync.Mutex
	token    string
	tokenExp time.Time
}

func New(id string, raw json.RawMessage, log *slog.Logger) (channel.Channel, error) {
	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("feishu: parse config: %w", err)
	}
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, fmt.Errorf("feishu: app_id and app_secret are required")
	}
	if log == nil {
		log = slog.Default()
	}
	return &Feishu{
		id:     id,
		cfg:    cfg,
		log:    log.With("channel", "feishu", "channel_id", id),
		client: &http.Client{Timeout: httpTimeout},
	}, nil
}

func (f *Feishu) ID() string          { return f.id }
func (f *Feishu) Kind() string        { return "feishu" }
func (f *Feishu) SupportsReply() bool { return true }

// Start records the inbound callback. There's no upstream connection
// to maintain — Feishu pushes events via the webhook.
func (f *Feishu) Start(_ context.Context, inbound channel.InboundFunc) error {
	f.mu.Lock()
	if f.cancel != nil {
		f.mu.Unlock()
		return nil
	}
	f.inbound = inbound
	_, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.done = make(chan struct{})
	close(f.done) // nothing to wait on
	f.mu.Unlock()
	f.log.Info("feishu channel started (webhook mode)")
	return nil
}

func (f *Feishu) Stop(_ context.Context) error {
	f.mu.Lock()
	cancel := f.cancel
	f.cancel = nil
	f.inbound = nil
	f.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	f.log.Info("feishu channel stopped")
	return nil
}

// HandleWebhook is the entry point the Hub-mounted public route hits
// for every POST to /api/v1/channels/{id}/webhook. Feishu's first
// request is a URL-verification challenge; subsequent ones are events.
func (f *Feishu) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var probe struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
		Token     string `json:"token"`
		Schema    string `json:"schema"`
		Header    struct {
			Token     string `json:"token"`
			EventType string `json:"event_type"`
		} `json:"header"`
		Event json.RawMessage `json:"event"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// URL verification (legacy v1 envelope).
	if probe.Type == "url_verification" && probe.Challenge != "" {
		if f.cfg.VerificationToken != "" && probe.Token != f.cfg.VerificationToken {
			http.Error(w, "bad verification token", http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"challenge": probe.Challenge})
		return
	}

	// v2 event envelope (schema: "2.0")
	token := probe.Header.Token
	if token == "" {
		token = probe.Token
	}
	if f.cfg.VerificationToken != "" && token != f.cfg.VerificationToken {
		http.Error(w, "bad verification token", http.StatusUnauthorized)
		return
	}

	// Acknowledge fast — Feishu retries if we don't 200 within a few seconds.
	w.WriteHeader(http.StatusOK)

	if probe.Header.EventType == "im.message.receive_v1" {
		f.dispatchMessage(r.Context(), probe.Event)
	}
}

func (f *Feishu) dispatchMessage(ctx context.Context, raw json.RawMessage) {
	var ev struct {
		Sender struct {
			SenderID struct {
				OpenID string `json:"open_id"`
				UserID string `json:"user_id"`
			} `json:"sender_id"`
		} `json:"sender"`
		Message struct {
			MessageID   string `json:"message_id"`
			ChatID      string `json:"chat_id"`
			ChatType    string `json:"chat_type"`
			MessageType string `json:"message_type"`
			Content     string `json:"content"` // JSON-encoded string
			CreateTime  string `json:"create_time"`
		} `json:"message"`
	}
	if err := json.Unmarshal(raw, &ev); err != nil {
		f.log.Warn("feishu event decode failed", "err", err)
		return
	}
	text := extractText(ev.Message.MessageType, ev.Message.Content)
	rc := ReplyCtx{ChatID: ev.Message.ChatID, MessageID: ev.Message.MessageID}
	msg := channel.ChannelMessage{
		ChannelID:      f.id,
		Direction:      channel.DirectionInbound,
		ConversationID: ev.Message.ChatID,
		Author:         ev.Sender.SenderID.OpenID,
		Text:           text,
		Timestamp:      time.Now().UTC(),
		ReplyCtx:       rc,
		Metadata: map[string]any{
			"feishu_message_id":   ev.Message.MessageID,
			"feishu_message_type": ev.Message.MessageType,
		},
	}
	f.mu.Lock()
	inbound := f.inbound
	f.mu.Unlock()
	if inbound == nil {
		return
	}
	if err := inbound(ctx, msg); err != nil {
		f.log.Error("feishu inbound dispatch failed", "err", err)
	}
}

func extractText(msgType, content string) string {
	if msgType != "text" || content == "" {
		return content
	}
	var c struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(content), &c); err != nil {
		return content
	}
	return c.Text
}

// Send posts a plain-text message.
func (f *Feishu) Send(ctx context.Context, msg channel.ChannelMessage) error {
	target := f.routing(msg)
	if target.ChatID == "" {
		return fmt.Errorf("feishu: no chat_id configured")
	}
	body := map[string]string{"text": msg.Text}
	raw, _ := json.Marshal(body)
	return f.send(ctx, target, "text", string(raw))
}

// SendCard renders a Card as an interactive card v2.
func (f *Feishu) SendCard(ctx context.Context, msg channel.ChannelMessage, card *channel.Card) error {
	target := f.routing(msg)
	if target.ChatID == "" {
		return fmt.Errorf("feishu: no chat_id configured")
	}
	rendered := renderCard(card)
	raw, _ := json.Marshal(rendered)
	return f.send(ctx, target, "interactive", string(raw))
}

// SendWithButtons sends a card with text + a single action row.
func (f *Feishu) SendWithButtons(ctx context.Context, msg channel.ChannelMessage, buttons [][]channel.ButtonOption) error {
	card := &channel.Card{
		Elements: []channel.CardElement{
			channel.CardMarkdown{Content: msg.Text},
			channel.CardActions{Buttons: buttons},
		},
	}
	return f.SendCard(ctx, msg, card)
}

func (f *Feishu) routing(msg channel.ChannelMessage) ReplyCtx {
	if rc, ok := msg.ReplyCtx.(ReplyCtx); ok && rc.ChatID != "" {
		return rc
	}
	if rc, ok := msg.ReplyCtx.(*ReplyCtx); ok && rc != nil && rc.ChatID != "" {
		return *rc
	}
	if msg.ConversationID != "" && msg.ConversationID != "default" {
		return ReplyCtx{ChatID: msg.ConversationID}
	}
	return ReplyCtx{ChatID: f.cfg.ChatID}
}

// send posts to the appropriate IM endpoint. When ReplyCtx carries a
// MessageID, uses the reply endpoint so the message threads under the
// original message.
func (f *Feishu) send(ctx context.Context, target ReplyCtx, msgType, content string) error {
	token, err := f.accessToken(ctx)
	if err != nil {
		return err
	}
	if target.MessageID != "" {
		path := "/open-apis/im/v1/messages/" + target.MessageID + "/reply"
		body := map[string]any{"msg_type": msgType, "content": content}
		return f.callJSON(ctx, http.MethodPost, path, token, body, nil)
	}
	path := "/open-apis/im/v1/messages?receive_id_type=chat_id"
	body := map[string]any{
		"receive_id": target.ChatID,
		"msg_type":   msgType,
		"content":    content,
	}
	return f.callJSON(ctx, http.MethodPost, path, token, body, nil)
}

// accessToken lazily refreshes the tenant_access_token. Cached until
// 5 minutes before the server-provided expiry.
func (f *Feishu) accessToken(ctx context.Context) (string, error) {
	f.tokenMu.Lock()
	defer f.tokenMu.Unlock()
	if f.token != "" && time.Now().Before(f.tokenExp) {
		return f.token, nil
	}
	body := map[string]string{
		"app_id":     f.cfg.AppID,
		"app_secret": f.cfg.AppSecret,
	}
	var resp struct {
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
		Token  string `json:"tenant_access_token"`
		Expire int    `json:"expire"`
	}
	if err := f.callJSON(ctx, http.MethodPost, "/open-apis/auth/v3/tenant_access_token/internal", "", body, &resp); err != nil {
		return "", err
	}
	if resp.Code != 0 {
		return "", fmt.Errorf("feishu auth: code=%d msg=%s", resp.Code, resp.Msg)
	}
	f.token = resp.Token
	f.tokenExp = time.Now().Add(time.Duration(resp.Expire) * time.Second).Add(-tokenSlack)
	return f.token, nil
}

func (f *Feishu) callJSON(ctx context.Context, method, path, token string, body any, out any) error {
	var raw []byte
	var err error
	if body != nil {
		raw, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, apiURL(path), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("feishu %s %s: HTTP %d: %s", method, path, resp.StatusCode, respBody)
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("feishu decode: %w", err)
		}
		return nil
	}
	var probe struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	_ = json.Unmarshal(respBody, &probe)
	if probe.Code != 0 {
		return fmt.Errorf("feishu %s: code=%d %s", path, probe.Code, probe.Msg)
	}
	return nil
}

// renderCard turns our Card into a Feishu interactive Card v2 JSON.
// Reference: https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/feishu-cards
func renderCard(card *channel.Card) map[string]any {
	if card == nil {
		return map[string]any{"schema": "2.0", "body": map[string]any{}}
	}
	out := map[string]any{
		"schema": "2.0",
		"config": map[string]any{"update_multi": true},
	}
	if card.Header != nil && card.Header.Title != "" {
		header := map[string]any{
			"title": map[string]any{"tag": "plain_text", "content": card.Header.Title},
		}
		if t := feishuTemplate(card.Header.Color); t != "" {
			header["template"] = t
		}
		out["header"] = header
	}
	elements := []map[string]any{}
	for _, el := range card.Elements {
		switch v := el.(type) {
		case channel.CardMarkdown:
			elements = append(elements, map[string]any{
				"tag":  "div",
				"text": map[string]any{"tag": "lark_md", "content": v.Content},
			})
		case channel.CardDivider:
			elements = append(elements, map[string]any{"tag": "hr"})
		case channel.CardActions:
			for _, row := range v.Buttons {
				elements = append(elements, map[string]any{
					"tag":     "action",
					"actions": btnRow(row),
				})
			}
		case channel.CardListItem:
			elements = append(elements, map[string]any{
				"tag":   "div",
				"text":  map[string]any{"tag": "lark_md", "content": v.Text},
				"extra": singleBtn(v.Button),
			})
		case channel.CardSelect:
			elements = append(elements, map[string]any{
				"tag": "action",
				"actions": []map[string]any{{
					"tag":         "select_static",
					"placeholder": map[string]any{"tag": "plain_text", "content": v.Placeholder},
					"options":     selectOptions(v),
					"value":       map[string]any{"key": "select"},
				}},
			})
		case channel.CardNote:
			elements = append(elements, map[string]any{
				"tag":      "note",
				"elements": []map[string]any{{"tag": "lark_md", "content": v.Text}},
			})
		}
	}
	out["body"] = map[string]any{"elements": elements}
	return out
}

func btnRow(row []channel.ButtonOption) []map[string]any {
	out := make([]map[string]any, 0, len(row))
	for _, b := range row {
		out = append(out, singleBtn(b))
	}
	return out
}

func singleBtn(b channel.ButtonOption) map[string]any {
	t := "default"
	switch b.Style {
	case "primary":
		t = "primary"
	case "danger":
		t = "danger"
	}
	return map[string]any{
		"tag":   "button",
		"text":  map[string]any{"tag": "plain_text", "content": b.Text},
		"type":  t,
		"value": map[string]any{"action": b.Value},
	}
}

func selectOptions(s channel.CardSelect) []map[string]any {
	out := make([]map[string]any, 0, len(s.Options))
	for _, o := range s.Options {
		out = append(out, map[string]any{
			"text":  map[string]any{"tag": "plain_text", "content": o.Text},
			"value": s.CallbackPrefix + o.Value,
		})
	}
	return out
}

// feishuTemplate maps our generic colour names to Feishu's header templates.
func feishuTemplate(color string) string {
	switch color {
	case "blue":
		return "blue"
	case "green":
		return "green"
	case "red":
		return "red"
	case "orange":
		return "orange"
	case "yellow":
		return "yellow"
	case "violet", "indigo":
		return "purple"
	case "turquoise":
		return "turquoise"
	}
	return ""
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
