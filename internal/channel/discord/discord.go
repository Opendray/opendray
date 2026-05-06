// Package discord is opendray's bundled Discord channel.
//
// Inbound flow uses the Discord Gateway WebSocket: opendray fetches
// wss://gateway.discord.gg via /gateway, identifies with the bot
// token, and listens for MESSAGE_CREATE + INTERACTION_CREATE events.
// Outbound uses the REST API (channels/{id}/messages and the
// interactions callback endpoint).
//
// Required: a Discord application + bot, with the
// "MESSAGE CONTENT INTENT" enabled on the dev portal. The bot must
// also be invited to the server with "Send Messages" + "Embed Links".
//
// Capabilities implemented: text, card (embeds + buttons),
// buttons, update_message, reply_to_message.
package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/opendray/opendray-v2/internal/channel"
)

const (
	apiBase = "https://discord.com/api/v10"

	httpTimeout    = 30 * time.Second
	writeWait      = 10 * time.Second
	maxMessageSize = 4 * 1024 * 1024 // gateway frames can be large
)

// Default intents: GUILD_MESSAGES | DIRECT_MESSAGES | MESSAGE_CONTENT.
// Without MESSAGE_CONTENT (privileged) every message arrives with
// content="" — useless for our purposes.
const defaultIntents = (1 << 9) | (1 << 12) | (1 << 15)

// Test seam.
var apiBaseOverride = ""

func apiURL(path string) string {
	if apiBaseOverride != "" {
		return apiBaseOverride + path
	}
	return apiBase + path
}

func init() { channel.Register("discord", New) }

type config struct {
	BotToken  string `json:"bot_token"`
	ChannelID string `json:"channel_id"`
	Intents   int    `json:"intents,omitempty"`
}

// ReplyCtx routes a Discord outbound to the right channel + message.
//   - ChannelID: target channel.
//   - MessageID: when set, the outbound is sent as a reply (via
//     message_reference).
type ReplyCtx struct {
	ChannelID string
	MessageID string
}

type Discord struct {
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
	seq     int64
	hbAck   chan struct{}
}

func New(id string, raw json.RawMessage, log *slog.Logger) (channel.Channel, error) {
	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("discord: parse config: %w", err)
	}
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("discord: bot_token is required")
	}
	if cfg.Intents == 0 {
		cfg.Intents = defaultIntents
	}
	if log == nil {
		log = slog.Default()
	}
	return &Discord{
		id:     id,
		cfg:    cfg,
		log:    log.With("channel", "discord", "channel_id", id),
		client: &http.Client{Timeout: httpTimeout},
	}, nil
}

func (d *Discord) ID() string          { return d.id }
func (d *Discord) Kind() string        { return "discord" }
func (d *Discord) SupportsReply() bool { return true }

func (d *Discord) Start(_ context.Context, inbound channel.InboundFunc) error {
	d.mu.Lock()
	if d.cancel != nil {
		d.mu.Unlock()
		return nil
	}
	d.inbound = inbound
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.done = make(chan struct{})
	d.mu.Unlock()
	go d.runLoop(ctx)
	d.log.Info("discord channel started")
	return nil
}

func (d *Discord) Stop(ctx context.Context) error {
	d.mu.Lock()
	cancel := d.cancel
	done := d.done
	conn := d.conn
	d.cancel = nil
	d.done = nil
	d.conn = nil
	d.mu.Unlock()
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
	d.log.Info("discord channel stopped")
	return nil
}

func (d *Discord) runLoop(ctx context.Context) {
	defer close(d.done)
	backoff := time.Second
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		gateway, err := d.fetchGateway(ctx)
		if err != nil {
			d.log.Warn("discord gateway lookup failed", "err", err, "wait", backoff)
			if !sleep(ctx, backoff) {
				return
			}
			backoff = minDur(backoff*2, 30*time.Second)
			continue
		}
		backoff = time.Second
		if err := d.serveWS(ctx, gateway); err != nil {
			d.log.Warn("discord ws disconnected", "err", err)
			if !sleep(ctx, time.Second) {
				return
			}
		}
	}
}

func (d *Discord) fetchGateway(ctx context.Context) (string, error) {
	var resp struct {
		URL string `json:"url"`
	}
	if err := d.callJSON(ctx, http.MethodGet, "/gateway", nil, &resp); err != nil {
		return "", err
	}
	if resp.URL == "" {
		return "", errors.New("discord: empty gateway URL")
	}
	return resp.URL + "/?v=10&encoding=json", nil
}

func (d *Discord) serveWS(ctx context.Context, gateway string) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, gateway, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	d.mu.Lock()
	d.conn = conn
	d.hbAck = make(chan struct{}, 1)
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		if d.conn == conn {
			d.conn = nil
		}
		d.mu.Unlock()
		_ = conn.Close()
	}()
	conn.SetReadLimit(maxMessageSize)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var ev gatewayPayload
		if err := json.Unmarshal(raw, &ev); err != nil {
			d.log.Warn("discord frame parse failed", "err", err)
			continue
		}
		if ev.S != 0 {
			d.mu.Lock()
			d.seq = ev.S
			d.mu.Unlock()
		}
		switch ev.Op {
		case 10: // Hello
			var hello struct {
				HeartbeatInterval int `json:"heartbeat_interval"`
			}
			if err := json.Unmarshal(ev.D, &hello); err != nil {
				return fmt.Errorf("hello decode: %w", err)
			}
			go d.heartbeatPump(ctx, conn, time.Duration(hello.HeartbeatInterval)*time.Millisecond)
			if err := d.identify(conn); err != nil {
				return fmt.Errorf("identify: %w", err)
			}
		case 11: // Heartbeat ACK
			d.mu.Lock()
			ack := d.hbAck
			d.mu.Unlock()
			select {
			case ack <- struct{}{}:
			default:
			}
		case 0: // Dispatch
			d.handleDispatch(ctx, ev.T, ev.D)
		case 7: // Reconnect
			return errors.New("discord: server requested reconnect")
		case 9: // Invalid Session
			return errors.New("discord: invalid session")
		}
	}
}

func (d *Discord) identify(conn *websocket.Conn) error {
	body := map[string]any{
		"op": 2,
		"d": map[string]any{
			"token":   d.cfg.BotToken,
			"intents": d.cfg.Intents,
			"properties": map[string]any{
				"os":      "linux",
				"browser": "opendray",
				"device":  "opendray",
			},
		},
	}
	return d.writeFrame(conn, body)
}

func (d *Discord) heartbeatPump(ctx context.Context, conn *websocket.Conn, interval time.Duration) {
	// Discord wants the first heartbeat after a randomised fraction of
	// the interval (jitter); subsequent ones at full cadence.
	jitter := time.Duration(rand.Int64N(interval.Nanoseconds()))
	if !sleep(ctx, jitter) {
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		d.mu.Lock()
		seq := d.seq
		d.mu.Unlock()
		body := map[string]any{"op": 1, "d": seq}
		if err := d.writeFrame(conn, body); err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (d *Discord) writeFrame(conn *websocket.Conn, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteMessage(websocket.TextMessage, raw)
}

type gatewayPayload struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d"`
	S  int64           `json:"s"`
	T  string          `json:"t"`
}

func (d *Discord) handleDispatch(ctx context.Context, name string, data json.RawMessage) {
	switch name {
	case "MESSAGE_CREATE":
		d.handleMessage(ctx, data)
	case "INTERACTION_CREATE":
		d.handleInteraction(ctx, data)
	}
}

func (d *Discord) handleMessage(ctx context.Context, data json.RawMessage) {
	var ev struct {
		ID        string `json:"id"`
		ChannelID string `json:"channel_id"`
		Content   string `json:"content"`
		Author    struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Bot      bool   `json:"bot"`
		} `json:"author"`
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		return
	}
	if ev.Author.Bot {
		return
	}
	rc := ReplyCtx{ChannelID: ev.ChannelID, MessageID: ev.ID}
	msg := channel.ChannelMessage{
		ChannelID:      d.id,
		Direction:      channel.DirectionInbound,
		ConversationID: ev.ChannelID,
		Author:         ev.Author.Username,
		Text:           ev.Content,
		Timestamp:      time.Now().UTC(),
		ReplyCtx:       rc,
		Metadata: map[string]any{
			"discord_message_id": ev.ID,
			"discord_user_id":    ev.Author.ID,
		},
	}
	d.mu.Lock()
	inbound := d.inbound
	d.mu.Unlock()
	if inbound == nil {
		return
	}
	if err := inbound(ctx, msg); err != nil {
		d.log.Error("discord inbound dispatch failed", "err", err)
	}
}

func (d *Discord) handleInteraction(ctx context.Context, data json.RawMessage) {
	var ev struct {
		ID        string `json:"id"`
		Token     string `json:"token"`
		Type      int    `json:"type"`
		ChannelID string `json:"channel_id"`
		Member    struct {
			User struct {
				ID       string `json:"id"`
				Username string `json:"username"`
			} `json:"user"`
		} `json:"member"`
		User struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"user"`
		Message struct {
			ID string `json:"id"`
		} `json:"message"`
		Data struct {
			CustomID string `json:"custom_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		return
	}
	// type=3 is MESSAGE_COMPONENT (button click etc.)
	if ev.Type != 3 || ev.Data.CustomID == "" {
		return
	}
	// ACK with DEFERRED_UPDATE_MESSAGE so the spinner stops; the actual
	// command flow runs through the inbound handler asynchronously.
	go d.ackInteraction(ev.ID, ev.Token)

	username := ev.Member.User.Username
	if username == "" {
		username = ev.User.Username
	}
	userID := ev.Member.User.ID
	if userID == "" {
		userID = ev.User.ID
	}
	rc := ReplyCtx{ChannelID: ev.ChannelID, MessageID: ev.Message.ID}
	msg := channel.ChannelMessage{
		ChannelID:      d.id,
		Direction:      channel.DirectionInbound,
		ConversationID: ev.ChannelID,
		Author:         username,
		Text:           channel.EncodeAction(ev.Data.CustomID),
		Timestamp:      time.Now().UTC(),
		ReplyCtx:       rc,
		Metadata: map[string]any{
			"action":             ev.Data.CustomID,
			"discord_message_id": ev.Message.ID,
			"discord_user_id":    userID,
		},
	}
	d.mu.Lock()
	inbound := d.inbound
	d.mu.Unlock()
	if inbound == nil {
		return
	}
	if err := inbound(ctx, msg); err != nil {
		d.log.Error("discord interaction dispatch failed", "err", err)
	}
}

func (d *Discord) ackInteraction(interactionID, token string) {
	body := map[string]any{"type": 6} // DEFERRED_UPDATE_MESSAGE
	path := fmt.Sprintf("/interactions/%s/%s/callback", interactionID, token)
	if err := d.callJSON(context.Background(), http.MethodPost, path, body, nil); err != nil {
		d.log.Warn("discord interaction ack failed", "err", err)
	}
}

// Send posts a plain text message.
func (d *Discord) Send(ctx context.Context, msg channel.ChannelMessage) error {
	chID, replyTo := d.routing(msg)
	if chID == "" {
		return fmt.Errorf("discord: no channel_id configured")
	}
	body := map[string]any{"content": msg.Text}
	if replyTo != "" {
		body["message_reference"] = map[string]any{"message_id": replyTo}
	}
	return d.callJSON(ctx, http.MethodPost, "/channels/"+chID+"/messages", body, nil)
}

// SendCard renders Card → embed + components.
func (d *Discord) SendCard(ctx context.Context, msg channel.ChannelMessage, card *channel.Card) error {
	chID, replyTo := d.routing(msg)
	if chID == "" {
		return fmt.Errorf("discord: no channel_id configured")
	}
	embed, components, fallback := renderCard(card)
	body := map[string]any{
		"content":    "",
		"embeds":     []map[string]any{embed},
		"components": components,
	}
	if fallback != "" {
		body["content"] = fallback
	}
	if replyTo != "" {
		body["message_reference"] = map[string]any{"message_id": replyTo}
	}
	return d.callJSON(ctx, http.MethodPost, "/channels/"+chID+"/messages", body, nil)
}

// SendWithButtons posts text + a single action row.
func (d *Discord) SendWithButtons(ctx context.Context, msg channel.ChannelMessage, buttons [][]channel.ButtonOption) error {
	chID, replyTo := d.routing(msg)
	if chID == "" {
		return fmt.Errorf("discord: no channel_id configured")
	}
	body := map[string]any{
		"content":    msg.Text,
		"components": rowsToComponents(buttons),
	}
	if replyTo != "" {
		body["message_reference"] = map[string]any{"message_id": replyTo}
	}
	return d.callJSON(ctx, http.MethodPost, "/channels/"+chID+"/messages", body, nil)
}

// UpdateMessage edits a previously-sent message; previewHandle is the
// Discord message id.
func (d *Discord) UpdateMessage(ctx context.Context, msg channel.ChannelMessage, previewHandle, newText string) error {
	chID, _ := d.routing(msg)
	if chID == "" || previewHandle == "" {
		return fmt.Errorf("discord: edit needs channel + message id")
	}
	path := "/channels/" + chID + "/messages/" + previewHandle
	body := map[string]any{"content": newText}
	return d.callJSON(ctx, http.MethodPatch, path, body, nil)
}

func (d *Discord) routing(msg channel.ChannelMessage) (string, string) {
	if rc, ok := msg.ReplyCtx.(ReplyCtx); ok && rc.ChannelID != "" {
		return rc.ChannelID, rc.MessageID
	}
	if rc, ok := msg.ReplyCtx.(*ReplyCtx); ok && rc != nil && rc.ChannelID != "" {
		return rc.ChannelID, rc.MessageID
	}
	if msg.ConversationID != "" && msg.ConversationID != "default" {
		return msg.ConversationID, ""
	}
	return d.cfg.ChannelID, ""
}

func renderCard(card *channel.Card) (map[string]any, []map[string]any, string) {
	if card == nil {
		return map[string]any{}, nil, ""
	}
	embed := map[string]any{}
	if card.Header != nil && card.Header.Title != "" {
		embed["title"] = card.Header.Title
	}
	if card.Header != nil && card.Header.Color != "" {
		if c, ok := colorMap[card.Header.Color]; ok {
			embed["color"] = c
		}
	}
	var description []string
	var fields []map[string]any
	var components []map[string]any
	var footer string
	for _, el := range card.Elements {
		switch v := el.(type) {
		case channel.CardMarkdown:
			description = append(description, v.Content)
		case channel.CardDivider:
			description = append(description, "—")
		case channel.CardActions:
			for _, row := range v.Buttons {
				components = append(components, map[string]any{
					"type":       1,
					"components": btnRow(row),
				})
			}
		case channel.CardListItem:
			fields = append(fields, map[string]any{
				"name":  v.Text,
				"value": "[" + v.Button.Text + "]",
			})
			components = append(components, map[string]any{
				"type":       1,
				"components": btnRow([]channel.ButtonOption{v.Button}),
			})
		case channel.CardSelect:
			components = append(components, map[string]any{
				"type": 1,
				"components": []map[string]any{{
					"type":        3, // string select
					"custom_id":   "select",
					"placeholder": v.Placeholder,
					"options":     selectOptions(v),
				}},
			})
		case channel.CardNote:
			footer = v.Text
		}
	}
	if len(description) > 0 {
		embed["description"] = joinNonEmpty(description, "\n")
	}
	if len(fields) > 0 {
		embed["fields"] = fields
	}
	if footer != "" {
		embed["footer"] = map[string]any{"text": footer}
	}
	fallback := ""
	if title, ok := embed["title"].(string); ok {
		fallback = title
	}
	return embed, components, fallback
}

func btnRow(row []channel.ButtonOption) []map[string]any {
	out := make([]map[string]any, 0, len(row))
	for _, b := range row {
		out = append(out, singleBtn(b))
	}
	return out
}

func singleBtn(b channel.ButtonOption) map[string]any {
	style := 2 // secondary
	switch b.Style {
	case "primary":
		style = 1
	case "danger":
		style = 4
	}
	return map[string]any{
		"type":      2, // button
		"label":     b.Text,
		"style":     style,
		"custom_id": b.Value,
	}
}

func rowsToComponents(rows [][]channel.ButtonOption) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{"type": 1, "components": btnRow(row)})
	}
	return out
}

func selectOptions(s channel.CardSelect) []map[string]any {
	out := make([]map[string]any, 0, len(s.Options))
	for _, o := range s.Options {
		out = append(out, map[string]any{
			"label": o.Text,
			"value": s.CallbackPrefix + o.Value,
		})
	}
	return out
}

// Discord embed.color is a 24-bit RGB int. Map opendray's named
// palette to roughly equivalent shades.
var colorMap = map[string]int{
	"blue":      0x3b82f6,
	"green":     0x22c55e,
	"red":       0xef4444,
	"orange":    0xf97316,
	"yellow":    0xeab308,
	"grey":      0x6b7280,
	"turquoise": 0x14b8a6,
	"violet":    0x8b5cf6,
	"indigo":    0x6366f1,
}

// callJSON does an authenticated REST call.
func (d *Discord) callJSON(ctx context.Context, method, path string, body any, out any) error {
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
	req.Header.Set("Authorization", "Bot "+d.cfg.BotToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "opendray/1.0 (+https://github.com/opendray)")
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("discord %s %s: HTTP %d: %s", method, path, resp.StatusCode, respBody)
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("discord decode: %w", err)
		}
	}
	return nil
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
