// Package slack is opendray's bundled Slack channel.
//
// Authentication uses Slack's **Socket Mode** so opendray never needs a
// public webhook URL — the channel opens a WebSocket back to Slack on
// startup and receives events_api + interactive payloads through it.
//
// Two tokens required:
//   - bot_token  ("xoxb-...") — chat:write scope, used for outbound
//     chat.postMessage / chat.update.
//   - app_token  ("xapp-...") — connections:write scope, used to
//     bootstrap the Socket Mode WS via apps.connections.open.
//
// Capabilities implemented: text, card (blocks), buttons, update_message,
// reply_to_message (thread_ts).
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/opendray/opendray-v2/internal/channel"
)

const (
	apiBase        = "https://slack.com/api/"
	httpTimeout    = 30 * time.Second
	pingPeriod     = 30 * time.Second
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	maxMessageSize = 256 * 1024
)

// apiBaseOverride lets tests redirect REST calls to a stub server.
var apiBaseOverride = ""

func apiURL() string {
	if apiBaseOverride != "" {
		return apiBaseOverride
	}
	return apiBase
}

func init() {
	channel.Register("slack", New)
}

type config struct {
	BotToken  string   `json:"bot_token"`
	AppToken  string   `json:"app_token"`
	ChannelID string   `json:"channel_id"`
	NotifyOn  []string `json:"notify_on,omitempty"`
}

// ReplyCtx threads a Slack outbound back to the right conversation.
//   - ChannelID: target channel (C0123...) — wins over cfg.ChannelID.
//   - ThreadTS:  parent message ts; setting this posts as a thread reply.
type ReplyCtx struct {
	ChannelID string
	ThreadTS  string
}

// Slack implements channel.Channel + capability interfaces.
type Slack struct {
	id     string
	cfg    config
	log    *slog.Logger
	client *http.Client

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	conn    *websocket.Conn
	writeMu sync.Mutex
	inbound channel.InboundFunc
}

func New(id string, raw json.RawMessage, log *slog.Logger) (channel.Channel, error) {
	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("slack: parse config: %w", err)
	}
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("slack: bot_token is required")
	}
	if cfg.AppToken == "" {
		return nil, fmt.Errorf("slack: app_token is required")
	}
	if log == nil {
		log = slog.Default()
	}
	return &Slack{
		id:     id,
		cfg:    cfg,
		log:    log.With("channel", "slack", "channel_id", id),
		client: &http.Client{Timeout: httpTimeout},
	}, nil
}

func (s *Slack) ID() string          { return s.id }
func (s *Slack) Kind() string        { return "slack" }
func (s *Slack) SupportsReply() bool { return true }

func (s *Slack) Start(_ context.Context, inbound channel.InboundFunc) error {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return nil
	}
	s.inbound = inbound
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	s.mu.Unlock()
	go s.runLoop(ctx)
	s.log.Info("slack channel started")
	return nil
}

func (s *Slack) Stop(ctx context.Context) error {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	conn := s.conn
	s.cancel = nil
	s.done = nil
	s.conn = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if conn != nil {
		_ = conn.Close()
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.log.Info("slack channel stopped")
	return nil
}

// runLoop opens a Socket Mode WS, handles frames, and reconnects with
// exponential backoff if the WS dies.
func (s *Slack) runLoop(ctx context.Context) {
	defer close(s.done)
	backoff := time.Second
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		wssURL, err := s.openConnection(ctx)
		if err != nil {
			s.log.Warn("slack apps.connections.open failed; backing off", "err", err, "wait", backoff)
			if !sleep(ctx, backoff) {
				return
			}
			backoff = minDur(backoff*2, 30*time.Second)
			continue
		}
		backoff = time.Second
		if err := s.serveWS(ctx, wssURL); err != nil {
			s.log.Warn("slack ws disconnected", "err", err)
			if !sleep(ctx, time.Second) {
				return
			}
		}
	}
}

func (s *Slack) openConnection(ctx context.Context) (string, error) {
	var resp struct {
		OK    bool   `json:"ok"`
		URL   string `json:"url"`
		Error string `json:"error"`
	}
	if err := s.callAPI(ctx, s.cfg.AppToken, "apps.connections.open", nil, &resp); err != nil {
		return "", err
	}
	if !resp.OK {
		return "", fmt.Errorf("apps.connections.open: %s", resp.Error)
	}
	return resp.URL, nil
}

func (s *Slack) serveWS(ctx context.Context, wssURL string) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wssURL, nil)
	if err != nil {
		return fmt.Errorf("dial socket-mode ws: %w", err)
	}
	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		if s.conn == conn {
			s.conn = nil
		}
		s.mu.Unlock()
		_ = conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	go s.pingPump(ctx, conn)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			s.log.Warn("slack frame parse failed", "err", err)
			continue
		}
		switch env.Type {
		case "hello":
			s.log.Info("slack socket-mode connected")
		case "events_api":
			s.handleEvent(ctx, env)
			s.ack(conn, env.EnvelopeID, nil)
		case "interactive":
			s.handleInteractive(ctx, env)
			s.ack(conn, env.EnvelopeID, nil)
		case "slash_commands":
			s.handleSlashCommand(ctx, env)
			s.ack(conn, env.EnvelopeID, nil)
		case "disconnect":
			return fmt.Errorf("slack disconnect: %s", env.Reason)
		}
	}
}

func (s *Slack) pingPump(ctx context.Context, conn *websocket.Conn) {
	t := time.NewTicker(pingPeriod)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.writeMu.Lock()
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := conn.WriteMessage(websocket.PingMessage, nil)
			s.writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

func (s *Slack) ack(conn *websocket.Conn, envelopeID string, payload map[string]any) {
	if envelopeID == "" {
		return
	}
	body := map[string]any{"envelope_id": envelopeID}
	if payload != nil {
		body["payload"] = payload
	}
	raw, _ := json.Marshal(body)
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	_ = conn.WriteMessage(websocket.TextMessage, raw)
}

func (s *Slack) handleEvent(ctx context.Context, env envelope) {
	var payload struct {
		Event struct {
			Type     string `json:"type"`
			Channel  string `json:"channel"`
			User     string `json:"user"`
			Text     string `json:"text"`
			TS       string `json:"ts"`
			ThreadTS string `json:"thread_ts"`
			BotID    string `json:"bot_id"`
			SubType  string `json:"subtype"`
		} `json:"event"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		s.log.Warn("slack events_api decode failed", "err", err)
		return
	}
	ev := payload.Event
	if ev.Type != "message" || ev.BotID != "" || ev.SubType == "bot_message" || ev.SubType == "message_changed" {
		return
	}
	rc := ReplyCtx{ChannelID: ev.Channel, ThreadTS: orFirst(ev.ThreadTS, ev.TS)}
	msg := channel.ChannelMessage{
		ChannelID:      s.id,
		Direction:      channel.DirectionInbound,
		ConversationID: ev.Channel,
		Author:         ev.User,
		Text:           ev.Text,
		Timestamp:      tsToTime(ev.TS),
		ReplyCtx:       rc,
		Metadata: map[string]any{
			"slack_ts":        ev.TS,
			"slack_thread_ts": ev.ThreadTS,
		},
	}
	s.mu.Lock()
	inbound := s.inbound
	s.mu.Unlock()
	if inbound == nil {
		return
	}
	if err := inbound(ctx, msg); err != nil {
		s.log.Error("slack inbound dispatch failed", "err", err)
	}
}

func (s *Slack) handleInteractive(ctx context.Context, env envelope) {
	// Slack wraps the actual interactive payload in a string-encoded
	// "payload" field on classic webhooks but Socket Mode delivers it as
	// a JSON object on the envelope's payload. We accept either.
	raw := env.Payload
	if len(raw) > 0 && raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			raw = json.RawMessage(s)
		}
	}
	var payload struct {
		Type    string              `json:"type"`
		User    struct{ ID string } `json:"user"`
		Channel struct{ ID string } `json:"channel"`
		Message struct {
			TS       string `json:"ts"`
			ThreadTS string `json:"thread_ts"`
		} `json:"message"`
		Actions []struct {
			ActionID string `json:"action_id"`
			Value    string `json:"value"`
		} `json:"actions"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		s.log.Warn("slack interactive decode failed", "err", err)
		return
	}
	if len(payload.Actions) == 0 {
		return
	}
	action := payload.Actions[0].Value
	rc := ReplyCtx{ChannelID: payload.Channel.ID, ThreadTS: orFirst(payload.Message.ThreadTS, payload.Message.TS)}
	msg := channel.ChannelMessage{
		ChannelID:      s.id,
		Direction:      channel.DirectionInbound,
		ConversationID: payload.Channel.ID,
		Author:         payload.User.ID,
		Text:           channel.EncodeAction(action),
		Timestamp:      time.Now().UTC(),
		ReplyCtx:       rc,
		Metadata: map[string]any{
			"action":          action,
			"action_id":       payload.Actions[0].ActionID,
			"slack_ts":        payload.Message.TS,
			"slack_thread_ts": payload.Message.ThreadTS,
		},
	}
	s.mu.Lock()
	inbound := s.inbound
	s.mu.Unlock()
	if inbound == nil {
		return
	}
	if err := inbound(ctx, msg); err != nil {
		s.log.Error("slack interactive dispatch failed", "err", err)
	}
}

func (s *Slack) handleSlashCommand(ctx context.Context, env envelope) {
	var payload struct {
		Command   string `json:"command"`
		Text      string `json:"text"`
		UserID    string `json:"user_id"`
		ChannelID string `json:"channel_id"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return
	}
	full := payload.Command
	if payload.Text != "" {
		full = payload.Command + " " + payload.Text
	}
	rc := ReplyCtx{ChannelID: payload.ChannelID}
	msg := channel.ChannelMessage{
		ChannelID:      s.id,
		Direction:      channel.DirectionInbound,
		ConversationID: payload.ChannelID,
		Author:         payload.UserID,
		Text:           full,
		Timestamp:      time.Now().UTC(),
		ReplyCtx:       rc,
	}
	s.mu.Lock()
	inbound := s.inbound
	s.mu.Unlock()
	if inbound != nil {
		_ = inbound(ctx, msg)
	}
}

// Send posts a plain text message via chat.postMessage.
func (s *Slack) Send(ctx context.Context, msg channel.ChannelMessage) error {
	chID, threadTS := s.routing(msg)
	if chID == "" {
		return fmt.Errorf("slack: no channel_id configured")
	}
	body := map[string]any{"channel": chID, "text": msg.Text}
	if threadTS != "" {
		body["thread_ts"] = threadTS
	}
	return s.callAPI(ctx, s.cfg.BotToken, "chat.postMessage", body, nil)
}

// SendCard renders Card → Slack blocks.
func (s *Slack) SendCard(ctx context.Context, msg channel.ChannelMessage, card *channel.Card) error {
	chID, threadTS := s.routing(msg)
	if chID == "" {
		return fmt.Errorf("slack: no channel_id configured")
	}
	text, blocks := renderCard(card)
	body := map[string]any{
		"channel": chID,
		"text":    text, // fallback for notifications
		"blocks":  blocks,
	}
	if threadTS != "" {
		body["thread_ts"] = threadTS
	}
	return s.callAPI(ctx, s.cfg.BotToken, "chat.postMessage", body, nil)
}

// SendWithButtons posts text + a single actions block.
func (s *Slack) SendWithButtons(ctx context.Context, msg channel.ChannelMessage, buttons [][]channel.ButtonOption) error {
	chID, threadTS := s.routing(msg)
	if chID == "" {
		return fmt.Errorf("slack: no channel_id configured")
	}
	blocks := []map[string]any{
		{"type": "section", "text": map[string]any{"type": "mrkdwn", "text": msg.Text}},
	}
	for _, row := range buttons {
		blocks = append(blocks, map[string]any{
			"type":     "actions",
			"elements": btnRow(row),
		})
	}
	body := map[string]any{"channel": chID, "text": msg.Text, "blocks": blocks}
	if threadTS != "" {
		body["thread_ts"] = threadTS
	}
	return s.callAPI(ctx, s.cfg.BotToken, "chat.postMessage", body, nil)
}

// UpdateMessage edits a previously-sent message via chat.update.
// previewHandle is the slack ts.
func (s *Slack) UpdateMessage(ctx context.Context, msg channel.ChannelMessage, previewHandle, newText string) error {
	chID, _ := s.routing(msg)
	if chID == "" || previewHandle == "" {
		return fmt.Errorf("slack: chat.update needs channel + ts")
	}
	body := map[string]any{"channel": chID, "ts": previewHandle, "text": newText}
	return s.callAPI(ctx, s.cfg.BotToken, "chat.update", body, nil)
}

func (s *Slack) routing(msg channel.ChannelMessage) (string, string) {
	if rc, ok := msg.ReplyCtx.(ReplyCtx); ok && rc.ChannelID != "" {
		return rc.ChannelID, rc.ThreadTS
	}
	if rc, ok := msg.ReplyCtx.(*ReplyCtx); ok && rc != nil && rc.ChannelID != "" {
		return rc.ChannelID, rc.ThreadTS
	}
	if msg.ConversationID != "" && msg.ConversationID != "default" {
		return msg.ConversationID, ""
	}
	return s.cfg.ChannelID, ""
}

func renderCard(card *channel.Card) (string, []map[string]any) {
	if card == nil {
		return "", nil
	}
	var blocks []map[string]any
	textParts := []string{}
	if card.Header != nil && card.Header.Title != "" {
		blocks = append(blocks, map[string]any{
			"type": "header",
			"text": map[string]any{"type": "plain_text", "text": card.Header.Title},
		})
		textParts = append(textParts, card.Header.Title)
	}
	for _, el := range card.Elements {
		switch v := el.(type) {
		case channel.CardMarkdown:
			blocks = append(blocks, map[string]any{
				"type": "section",
				"text": map[string]any{"type": "mrkdwn", "text": v.Content},
			})
			textParts = append(textParts, v.Content)
		case channel.CardDivider:
			blocks = append(blocks, map[string]any{"type": "divider"})
		case channel.CardActions:
			for _, row := range v.Buttons {
				blocks = append(blocks, map[string]any{
					"type":     "actions",
					"elements": btnRow(row),
				})
			}
		case channel.CardListItem:
			blocks = append(blocks, map[string]any{
				"type":      "section",
				"text":      map[string]any{"type": "mrkdwn", "text": v.Text},
				"accessory": singleBtn(v.Button),
			})
			textParts = append(textParts, v.Text)
		case channel.CardSelect:
			elements := []map[string]any{{
				"type":        "static_select",
				"placeholder": map[string]any{"type": "plain_text", "text": v.Placeholder},
				"options":     selectOptions(v),
				"action_id":   "select",
			}}
			blocks = append(blocks, map[string]any{"type": "actions", "elements": elements})
		case channel.CardNote:
			blocks = append(blocks, map[string]any{
				"type":     "context",
				"elements": []map[string]any{{"type": "mrkdwn", "text": v.Text}},
			})
		}
	}
	return joinNonEmpty(textParts, "\n"), blocks
}

func btnRow(row []channel.ButtonOption) []map[string]any {
	out := make([]map[string]any, 0, len(row))
	for _, b := range row {
		out = append(out, singleBtn(b))
	}
	return out
}

func singleBtn(b channel.ButtonOption) map[string]any {
	out := map[string]any{
		"type":      "button",
		"text":      map[string]any{"type": "plain_text", "text": b.Text},
		"value":     b.Value,
		"action_id": b.Value,
	}
	switch b.Style {
	case "primary":
		out["style"] = "primary"
	case "danger":
		out["style"] = "danger"
	}
	return out
}

func selectOptions(s channel.CardSelect) []map[string]any {
	out := make([]map[string]any, 0, len(s.Options))
	for _, o := range s.Options {
		out = append(out, map[string]any{
			"text":  map[string]any{"type": "plain_text", "text": o.Text},
			"value": s.CallbackPrefix + o.Value,
		})
	}
	return out
}

// envelope is the common Socket Mode envelope shape.
type envelope struct {
	Type       string          `json:"type"`
	EnvelopeID string          `json:"envelope_id"`
	Reason     string          `json:"reason"`
	Payload    json.RawMessage `json:"payload"`
}

// callAPI posts a JSON body to the given Slack Web API method using the
// provided bearer token. Decodes into out (if non-nil). Surfaces
// `ok:false` responses as errors.
func (s *Slack) callAPI(ctx context.Context, token, method string, body any, out any) error {
	var raw []byte
	var err error
	if body != nil {
		raw, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	endpoint := apiURL() + url.PathEscape(method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("slack %s: HTTP %d: %s", method, resp.StatusCode, respBody)
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("slack %s: decode: %w", method, err)
		}
		return nil
	}
	// Probe ok:false even when caller didn't ask for the body.
	var probe struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(respBody, &probe)
	if !probe.OK && probe.Error != "" {
		return fmt.Errorf("slack %s: %s", method, probe.Error)
	}
	return nil
}

func tsToTime(ts string) time.Time {
	// Slack ts looks like "1700000000.000123". Microsecond precision; we
	// only need wall-clock so int seconds is enough.
	if len(ts) >= 10 {
		var sec, usec int64
		_, _ = fmt.Sscanf(ts, "%d.%d", &sec, &usec)
		if sec > 0 {
			return time.Unix(sec, usec*1000).UTC()
		}
	}
	return time.Now().UTC()
}

func orFirst(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func joinNonEmpty(parts []string, sep string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return ""
	}
	s := out[0]
	for _, p := range out[1:] {
		s += sep + p
	}
	return s
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

func minDur(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
