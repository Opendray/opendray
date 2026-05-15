// Package telegram is a Telegram channel implementation for opendray.
//
// Pure net/http calls to api.telegram.org — no third-party SDK.
// Long-poll getUpdates → InboundFunc; outbound goes via sendMessage,
// editMessageText, and sendChatAction.
//
// Capabilities implemented (M5):
//   - Card → inline_keyboard rendering (CardSender)
//   - Standalone button rows (ButtonSender)
//   - Edit-in-place message updates (MessageUpdater)
//   - "typing…" indicator (TypingIndicator)
//   - Reply-to-message routing via ReplyCtx (ReplyCapable)
//   - callback_query → inbound action delivery
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/opendray/opendray-v2/internal/channel"
)

const (
	defaultAPIBase = "https://api.telegram.org/bot"
	pollTimeoutSec = 25
	httpTimeout    = 35 * time.Second
	typingPeriod   = 4 * time.Second
)

// apiBaseOverride is set by tests to redirect API calls to a stub
// server. Empty in production.
var apiBaseOverride = ""

func apiBase() string {
	if apiBaseOverride != "" {
		return apiBaseOverride
	}
	return defaultAPIBase
}

func init() {
	channel.Register("telegram", New)
}

type config struct {
	BotToken string   `json:"bot_token"`
	ChatID   int64    `json:"chat_id"`
	NotifyOn []string `json:"notify_on,omitempty"`
}

// Telegram implements channel.Channel + several capability interfaces.
type Telegram struct {
	id     string
	cfg    config
	log    *slog.Logger
	client *http.Client

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
	offset int64
}

// ReplyCtx carries the platform-routing data needed to send a message
// as a reply to a specific inbound message. Stored on
// ChannelMessage.ReplyCtx by the inbound path; used by Send / SendCard
// to populate reply_to_message_id and chat_id.
type ReplyCtx struct {
	ChatID    int64
	MessageID int
}

// New is the registered factory for kind="telegram".
func New(id string, raw json.RawMessage, log *slog.Logger) (channel.Channel, error) {
	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("telegram: parse config: %w", err)
	}
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("telegram: bot_token is required")
	}
	if log == nil {
		log = slog.Default()
	}
	return &Telegram{
		id:     id,
		cfg:    cfg,
		log:    log.With("channel", "telegram", "channel_id", id),
		client: &http.Client{Timeout: httpTimeout},
	}, nil
}

func (t *Telegram) ID() string          { return t.id }
func (t *Telegram) Kind() string        { return "telegram" }
func (t *Telegram) SupportsReply() bool { return true }

func (t *Telegram) Start(_ context.Context, inbound channel.InboundFunc) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		return nil
	}
	pollCtx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.done = make(chan struct{})
	go t.poll(pollCtx, inbound)
	// Best-effort autocomplete registration — Telegram caches the
	// command list per-bot and serves it on the "/" picker in every
	// chat. Errors here are non-fatal (the bot still works without
	// autocomplete) so we just log and proceed.
	go t.publishCommands(pollCtx)
	t.log.Info("telegram channel started")
	return nil
}

// publishCommands tells Telegram which slash commands the bot
// exposes, so the chat client's "/" picker autocompletes them. The
// list is hardcoded against the channel package's actual registry
// because the Channel interface doesn't (yet) carry a hook for the
// hub to push command metadata down into transports — small,
// stable set, easier to mirror by hand than to plumb through.
//
// If we ever extend the command set, update both this list and
// the registrations in channel.Hub.registerBuiltinCommands +
// internal/app/channel_commands.go.
func (t *Telegram) publishCommands(ctx context.Context) {
	type tgCmd struct {
		Command     string `json:"command"`
		Description string `json:"description"`
	}
	cmds := []tgCmd{
		{Command: "help", Description: "List available commands"},
		{Command: "list", Description: "List active sessions"},
		{Command: "end", Description: "End a session: /end <session_id>"},
		{Command: "resume", Description: "Resume a stopped session: /resume <session_id>"},
		{Command: "notify", Description: "Toggle notifications: /notify on|off"},
	}
	body := map[string]any{"commands": cmds}
	var resp struct {
		Ok bool `json:"ok"`
	}
	if err := t.callAPI(ctx, "setMyCommands", body, &resp); err != nil {
		t.log.Warn("telegram setMyCommands failed", "err", err)
		return
	}
	if !resp.Ok {
		t.log.Warn("telegram setMyCommands rejected", "resp", resp)
	}
}

func (t *Telegram) Stop(ctx context.Context) error {
	t.mu.Lock()
	cancel := t.cancel
	done := t.done
	t.cancel = nil
	t.done = nil
	t.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	t.log.Info("telegram channel stopped")
	return nil
}

// Send pushes a plain-text message. Honours ReplyCtx if present.
func (t *Telegram) Send(ctx context.Context, msg channel.ChannelMessage) error {
	chatID, replyTo := t.routing(msg)
	if chatID == 0 {
		return fmt.Errorf("telegram: no chat_id configured")
	}
	body := map[string]any{
		"chat_id": chatID,
		"text":    msg.Text,
	}
	if replyTo != 0 {
		body["reply_parameters"] = map[string]any{"message_id": replyTo}
	}
	var resp tgSendResp
	if err := t.callAPI(ctx, "sendMessage", body, &resp); err != nil {
		return err
	}
	t.recordHandle(&msg, resp.Result.MessageID)
	return nil
}

// SendCard renders a Card as one or more sendMessage calls.
//
// Pipeline:
//  1. renderCard → markdown text body
//  2. formatForTelegram → Telegram-flavoured HTML (tables → vertical
//     "Header: value", code fences → <pre>, **bold** → <b>, …)
//  3. splitForTelegram → chunks ≤3800 chars, line-aligned
//  4. rebalanceHTMLChunks → close/reopen <pre> across chunk
//     boundaries so each chunk is independently valid HTML
//
// Inline keyboard attaches only to the *last* chunk so the reader
// scrolls through full content before seeing the action buttons.
func (t *Telegram) SendCard(ctx context.Context, msg channel.ChannelMessage, card *channel.Card) error {
	chatID, replyTo := t.routing(msg)
	if chatID == 0 {
		return fmt.Errorf("telegram: no chat_id configured")
	}
	text, keyboard := renderCard(card)
	htmlBody := formatForTelegram(text)
	chunks := splitForTelegram(htmlBody)
	chunks = rebalanceHTMLChunks(chunks)

	var lastID int
	for i, chunk := range chunks {
		body := map[string]any{
			"chat_id":    chatID,
			"text":       chunk,
			"parse_mode": "HTML",
		}
		// Only the first chunk is rendered as a "reply to" the
		// originating message — chained replies on every chunk just
		// produce noise. Buttons only on the last chunk.
		if i == 0 && replyTo != 0 {
			body["reply_parameters"] = map[string]any{"message_id": replyTo}
		}
		if i == len(chunks)-1 && len(keyboard) > 0 {
			body["reply_markup"] = map[string]any{"inline_keyboard": keyboard}
		}
		var resp tgSendResp
		if err := t.callAPI(ctx, "sendMessage", body, &resp); err != nil {
			return fmt.Errorf("telegram chunk %d/%d: %w", i+1, len(chunks), err)
		}
		lastID = resp.Result.MessageID
	}
	t.recordHandle(&msg, lastID)
	return nil
}

// telegramChunkSize is the conservative per-message char ceiling.
// Telegram's hard limit is 4096; we keep headroom for the parse-mode
// header and any escaping the API might inject.
const telegramChunkSize = 3800

// splitForTelegram breaks `text` into chunks ≤telegramChunkSize.
// Splits prefer line boundaries (so code-block fencing stays
// readable). When a single line exceeds the cap it gets hard-cut
// into byte-bounded slices that respect UTF-8 rune boundaries.
func splitForTelegram(text string) []string {
	if len([]rune(text)) <= telegramChunkSize {
		return []string{text}
	}
	lines := strings.Split(text, "\n")
	chunks := make([]string, 0, 4)
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			chunks = append(chunks, b.String())
			b.Reset()
		}
	}
	addLine := func(line string) {
		// Would this line overflow the current chunk?
		if b.Len() > 0 && runesIn(b.String())+runesIn(line)+1 > telegramChunkSize {
			flush()
		}
		if runesIn(line) > telegramChunkSize {
			flush()
			chunks = append(chunks, hardSplitRunes(line, telegramChunkSize)...)
			return
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(line)
	}
	for _, l := range lines {
		addLine(l)
	}
	flush()
	return chunks
}

func runesIn(s string) int { return len([]rune(s)) }

// hardSplitRunes slices s into chunks of `cap` runes each, respecting
// UTF-8 rune boundaries. Used as a last resort when a single line is
// longer than the chunk ceiling.
func hardSplitRunes(s string, cap int) []string {
	if cap <= 0 || s == "" {
		return []string{s}
	}
	runes := []rune(s)
	out := make([]string, 0, (len(runes)/cap)+1)
	for i := 0; i < len(runes); i += cap {
		end := i + cap
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[i:end]))
	}
	return out
}

// SendWithButtons posts a text message attached with inline buttons.
func (t *Telegram) SendWithButtons(ctx context.Context, msg channel.ChannelMessage, buttons [][]channel.ButtonOption) error {
	chatID, replyTo := t.routing(msg)
	if chatID == 0 {
		return fmt.Errorf("telegram: no chat_id configured")
	}
	body := map[string]any{
		"chat_id":      chatID,
		"text":         msg.Text,
		"reply_markup": map[string]any{"inline_keyboard": buttonsToKeyboard(buttons)},
	}
	if replyTo != 0 {
		body["reply_parameters"] = map[string]any{"message_id": replyTo}
	}
	var resp tgSendResp
	if err := t.callAPI(ctx, "sendMessage", body, &resp); err != nil {
		return err
	}
	t.recordHandle(&msg, resp.Result.MessageID)
	return nil
}

// UpdateMessage edits the text of a previously-sent message.
// previewHandle is the message_id the original send returned.
func (t *Telegram) UpdateMessage(ctx context.Context, msg channel.ChannelMessage, previewHandle string, newText string) error {
	chatID, _ := t.routing(msg)
	mid, err := strconv.Atoi(previewHandle)
	if err != nil || chatID == 0 || mid == 0 {
		return fmt.Errorf("telegram: invalid preview handle %q (chat=%d)", previewHandle, chatID)
	}
	body := map[string]any{
		"chat_id":    chatID,
		"message_id": mid,
		"text":       newText,
	}
	return t.callAPI(ctx, "editMessageText", body, nil)
}

// StartTyping spins a goroutine that posts sendChatAction every ~4s
// (Telegram clears the indicator after 5s of silence). The returned
// stop func cancels the loop.
func (t *Telegram) StartTyping(ctx context.Context, msg channel.ChannelMessage) func() {
	chatID, _ := t.routing(msg)
	if chatID == 0 {
		return func() {}
	}
	typeCtx, cancel := context.WithCancel(ctx)
	go func() {
		send := func() {
			body := map[string]any{"chat_id": chatID, "action": "typing"}
			_ = t.callAPI(typeCtx, "sendChatAction", body, nil)
		}
		send()
		ticker := time.NewTicker(typingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-typeCtx.Done():
				return
			case <-ticker.C:
				send()
			}
		}
	}()
	return cancel
}

// routing pulls (chat_id, reply_to_message_id) from ReplyCtx if
// present, falling back to ConversationID and the configured chat_id.
func (t *Telegram) routing(msg channel.ChannelMessage) (int64, int) {
	if rc, ok := msg.ReplyCtx.(ReplyCtx); ok && rc.ChatID != 0 {
		return rc.ChatID, rc.MessageID
	}
	if rc, ok := msg.ReplyCtx.(*ReplyCtx); ok && rc != nil && rc.ChatID != 0 {
		return rc.ChatID, rc.MessageID
	}
	chatID := t.cfg.ChatID
	if msg.ConversationID != "" && msg.ConversationID != "default" {
		if id, err := strconv.ParseInt(msg.ConversationID, 10, 64); err == nil {
			chatID = id
		}
	}
	return chatID, 0
}

// recordHandle stashes the Telegram message_id into msg.Metadata so
// callers can later pass it back as previewHandle for UpdateMessage,
// and so Hub.dispatch can index outbound IDs for reply-to-message
// session routing. Both the platform-specific key and the generic
// "outbound_msg_id" key are populated so the Hub doesn't need to
// know the kind.
func (t *Telegram) recordHandle(msg *channel.ChannelMessage, mid int) {
	if mid == 0 {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = map[string]any{}
	}
	msg.Metadata["telegram_message_id"] = mid
	msg.Metadata["outbound_msg_id"] = strconv.Itoa(mid)
}

func (t *Telegram) poll(ctx context.Context, inbound channel.InboundFunc) {
	defer close(t.done)
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		body := map[string]any{
			"offset":          t.offset,
			"timeout":         pollTimeoutSec,
			"allowed_updates": []string{"message", "callback_query"},
		}
		var resp struct {
			Ok     bool       `json:"ok"`
			Result []tgUpdate `json:"result"`
		}
		err := t.callAPI(ctx, "getUpdates", body, &resp)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			t.log.Warn("getUpdates failed; backing off", "err", err, "wait", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = minDur(backoff*2, 30*time.Second)
			continue
		}
		backoff = time.Second

		for _, u := range resp.Result {
			if u.UpdateID >= t.offset {
				t.offset = u.UpdateID + 1
			}
			switch {
			case u.Message != nil:
				t.deliverMessage(ctx, inbound, u.Message)
			case u.CallbackQuery != nil:
				t.deliverCallback(ctx, inbound, u.CallbackQuery)
			}
		}
	}
}

func (t *Telegram) deliverMessage(ctx context.Context, inbound channel.InboundFunc, m *tgMessage) {
	rc := ReplyCtx{ChatID: m.Chat.ID, MessageID: m.MessageID}
	meta := map[string]any{
		"telegram_message_id": m.MessageID,
		"chat_type":           m.Chat.Type,
	}
	// When this message is a reply to one of our outbound notifications,
	// surface the original message_id so the Hub can route the reply
	// to the *specific* session that notification was about, instead
	// of falling back to "last notified" routing.
	if m.ReplyToMessage != nil && m.ReplyToMessage.MessageID != 0 {
		meta["reply_to_outbound_msg_id"] = strconv.Itoa(m.ReplyToMessage.MessageID)
	}
	msg := channel.ChannelMessage{
		ChannelID:      t.id,
		Direction:      channel.DirectionInbound,
		ConversationID: strconv.FormatInt(m.Chat.ID, 10),
		Author:         m.From.username(),
		Text:           m.Text,
		Timestamp:      time.Unix(m.Date, 0).UTC(),
		ReplyCtx:       rc,
		Metadata:       meta,
	}
	if err := inbound(ctx, msg); err != nil {
		t.log.Error("inbound handler failed", "err", err)
	}
}

func (t *Telegram) deliverCallback(ctx context.Context, inbound channel.InboundFunc, cq *tgCallbackQuery) {
	// Acknowledge the click so Telegram clears the spinner.
	go func() {
		_ = t.callAPI(context.Background(), "answerCallbackQuery",
			map[string]any{"callback_query_id": cq.ID}, nil)
	}()
	if cq.Message == nil {
		return
	}
	rc := ReplyCtx{ChatID: cq.Message.Chat.ID, MessageID: cq.Message.MessageID}
	msg := channel.ChannelMessage{
		ChannelID:      t.id,
		Direction:      channel.DirectionInbound,
		ConversationID: strconv.FormatInt(cq.Message.Chat.ID, 10),
		Author:         cq.From.username(),
		Text:           channel.EncodeAction(cq.Data),
		Timestamp:      time.Now().UTC(),
		ReplyCtx:       rc,
		Metadata: map[string]any{
			"callback_query_id":   cq.ID,
			"telegram_message_id": cq.Message.MessageID,
			"action":              cq.Data,
		},
	}
	if err := inbound(ctx, msg); err != nil {
		t.log.Error("inbound callback handler failed", "err", err)
	}
}

func renderCard(card *channel.Card) (string, [][]map[string]any) {
	if card == nil {
		return "", nil
	}
	var parts []string
	if card.Header != nil && card.Header.Title != "" {
		// Markdown heading — formatForTelegram turns "# Title" into
		// "<b>Title</b>" for the HTML parse_mode pipeline. (When
		// upstream changes drop HTML, the literal "# " is harmless
		// chrome.)
		parts = append(parts, "# "+card.Header.Title)
	}
	var keyboard [][]map[string]any
	for _, el := range card.Elements {
		switch v := el.(type) {
		case channel.CardMarkdown:
			parts = append(parts, v.Content)
		case channel.CardDivider:
			parts = append(parts, "──────────")
		case channel.CardActions:
			keyboard = append(keyboard, buttonsToKeyboard(v.Buttons)...)
		case channel.CardListItem:
			parts = append(parts, v.Text)
			keyboard = append(keyboard, []map[string]any{btnToTG(v.Button)})
		case channel.CardSelect:
			parts = append(parts, v.Placeholder)
			row := make([]map[string]any, 0, len(v.Options))
			for _, o := range v.Options {
				row = append(row, map[string]any{
					"text":          o.Text,
					"callback_data": v.CallbackPrefix + o.Value,
				})
			}
			if len(row) > 0 {
				keyboard = append(keyboard, row)
			}
		case channel.CardNote:
			parts = append(parts, "_"+v.Text+"_")
		}
	}
	return joinNonEmpty(parts, "\n"), keyboard
}

func buttonsToKeyboard(rows [][]channel.ButtonOption) [][]map[string]any {
	out := make([][]map[string]any, 0, len(rows))
	for _, row := range rows {
		r := make([]map[string]any, 0, len(row))
		for _, b := range row {
			r = append(r, btnToTG(b))
		}
		out = append(out, r)
	}
	return out
}

func btnToTG(b channel.ButtonOption) map[string]any {
	return map[string]any{
		"text":          b.Text,
		"callback_data": b.Value,
	}
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

func minDur(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

type tgUpdate struct {
	UpdateID      int64            `json:"update_id"`
	Message       *tgMessage       `json:"message,omitempty"`
	CallbackQuery *tgCallbackQuery `json:"callback_query,omitempty"`
}

type tgMessage struct {
	MessageID      int        `json:"message_id"`
	From           *tgUser    `json:"from,omitempty"`
	Chat           tgChat     `json:"chat"`
	Date           int64      `json:"date"`
	Text           string     `json:"text"`
	ReplyToMessage *tgMessage `json:"reply_to_message,omitempty"`
}

type tgCallbackQuery struct {
	ID      string     `json:"id"`
	From    *tgUser    `json:"from,omitempty"`
	Message *tgMessage `json:"message,omitempty"`
	Data    string     `json:"data"`
}

type tgUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
}

func (u *tgUser) username() string {
	if u == nil {
		return ""
	}
	if u.Username != "" {
		return "@" + u.Username
	}
	return u.FirstName
}

type tgChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type tgSendResp struct {
	Ok     bool      `json:"ok"`
	Result tgMessage `json:"result"`
}

func (t *Telegram) callAPI(ctx context.Context, method string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	apiURL := apiBase() + url.PathEscape(t.cfg.BotToken) + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("telegram %s: HTTP %d: %s", method, resp.StatusCode, respBody)
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("telegram %s: decode: %w", method, err)
		}
	}
	return nil
}
