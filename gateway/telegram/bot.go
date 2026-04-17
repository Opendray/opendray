// Package telegram is the in-process Telegram bot bridge for OpenDray.
//
// One Bot instance long-polls api.telegram.org for incoming updates and
// dispatches them to command handlers (see commands.go). Notifications
// flow the other direction via Bot.Send, fed by hook subscribers
// (see notifications.go).
//
// All Telegram REST calls are plain net/http — no third-party SDK.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Config bundles the runtime parameters extracted from the plugin manifest.
type Config struct {
	BotToken       string
	AllowedChatIDs map[int64]bool
	NotifyChatID   int64
	NotifyOnIdle   bool
	NotifyOnExit   bool
	TailLines      int
	PollInterval   time.Duration
	// ExtraClaudeDirs are optional Claude config-root directories used by
	// the JSONL reader in addition to the automatic ones (session-bound
	// account, all enabled accounts, $HOME/.claude). Lets users point the
	// bridge at non-standard CLAUDE_CONFIG_DIR layouts.
	ExtraClaudeDirs []string
}

// Bot is a Telegram client + long-poll loop. Lifecycle: New → Start → Stop.
// Safe to call Send from any goroutine after Start.
type Bot struct {
	cfg     Config
	client  *http.Client
	logger  *slog.Logger
	handler UpdateHandler

	// Bot identity (filled by getMe on Start)
	username atomic.Pointer[string]

	mu          sync.Mutex
	running     bool
	cancel      context.CancelFunc
	lastUpdate  int64 // highest update_id we acknowledged
	lastPollAt  atomic.Int64
	lastError   atomic.Pointer[string]

	// Telemetry counters
	sentCount     atomic.Int64
	receivedCount atomic.Int64
}

// UpdateHandler dispatches a single Telegram update to whichever subsystem
// owns it (command parser, link-forwarder, etc.). Implemented in commands.go.
type UpdateHandler interface {
	HandleUpdate(ctx context.Context, u Update)
}

// New constructs a bot. It does NOT contact Telegram until Start is called.
func New(cfg Config, handler UpdateHandler, logger *slog.Logger) *Bot {
	if logger == nil {
		logger = slog.Default()
	}
	return &Bot{
		cfg:     cfg,
		client:  &http.Client{Timeout: 60 * time.Second},
		logger:  logger,
		handler: handler,
	}
}

// Start kicks off the long-poll goroutine. Returns immediately.
func (b *Bot) Start(parent context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running {
		return fmt.Errorf("telegram: bot already running")
	}
	if b.cfg.BotToken == "" {
		return fmt.Errorf("telegram: bot token is empty")
	}

	// Validate token + cache identity.
	me, err := b.getMe(parent)
	if err != nil {
		return fmt.Errorf("telegram: getMe: %w", err)
	}
	uname := me.Username
	b.username.Store(&uname)

	ctx, cancel := context.WithCancel(parent)
	b.cancel = cancel
	b.running = true
	go b.pollLoop(ctx)
	b.logger.Info("telegram bot started",
		"username", "@"+uname,
		"allowed_chats", len(b.cfg.AllowedChatIDs),
	)
	return nil
}

// Stop signals the poll loop to exit. Safe to call multiple times.
func (b *Bot) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.running {
		return
	}
	b.running = false
	if b.cancel != nil {
		b.cancel()
	}
}

// Username returns "@bot_username" if known, "" otherwise.
func (b *Bot) Username() string {
	u := b.username.Load()
	if u == nil {
		return ""
	}
	return "@" + *u
}

// Status snapshot for the frontend status panel.
type Status struct {
	Connected     bool   `json:"connected"`
	Username      string `json:"username,omitempty"`
	LastPollAt    int64  `json:"lastPollAt,omitempty"` // unix milli
	LastError     string `json:"lastError,omitempty"`
	AllowedChats  int    `json:"allowedChats"`
	Sent          int64  `json:"sent"`
	Received      int64  `json:"received"`
}

// Snapshot returns a status snapshot — used by the HTTP /status endpoint.
func (b *Bot) Snapshot() Status {
	st := Status{
		Connected:    b.username.Load() != nil,
		Username:     b.Username(),
		LastPollAt:   b.lastPollAt.Load(),
		AllowedChats: len(b.cfg.AllowedChatIDs),
		Sent:         b.sentCount.Load(),
		Received:     b.receivedCount.Load(),
	}
	if e := b.lastError.Load(); e != nil {
		st.LastError = *e
	}
	return st
}

// IsAllowed reports whether the given chat is in the allowlist.
func (b *Bot) IsAllowed(chatID int64) bool {
	return b.cfg.AllowedChatIDs[chatID]
}

// NotifyChatID returns the configured notify destination, falling back to
// the first allowed chat if none is set.
func (b *Bot) NotifyChatID() int64 {
	if b.cfg.NotifyChatID != 0 {
		return b.cfg.NotifyChatID
	}
	for id := range b.cfg.AllowedChatIDs {
		return id
	}
	return 0
}

// Cfg exposes the running config (read-only) for handlers that need it.
func (b *Bot) Cfg() Config { return b.cfg }

// ── Long-poll loop ──────────────────────────────────────────────

func (b *Bot) pollLoop(ctx context.Context) {
	timeout := int(b.cfg.PollInterval.Seconds())
	if timeout < 5 {
		timeout = 25 // minimum 5s; default 25s for responsiveness without API churn
	}
	failures := 0
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		updates, err := b.getUpdates(ctx, b.lastUpdate+1, timeout)
		b.lastPollAt.Store(time.Now().UnixMilli())
		if err != nil {
			s := err.Error()
			b.lastError.Store(&s)

			// 409 Conflict = another bot instance running with the same token.
			// Wait a full 30 s (the other instance's long-poll timeout) before
			// retrying so we don't flood Telegram's API with rejected polls.
			if errors.Is(err, errConflict) {
				b.logger.Warn("telegram: another instance is polling — waiting 30 s",
					"hint", "kill the other process or wait for it to stop")
				select {
				case <-ctx.Done():
					return
				case <-time.After(30 * time.Second):
					continue
				}
			}

			failures++
			delay := time.Duration(failures) * 2 * time.Second
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			b.logger.Warn("telegram getUpdates failed", "error", err, "retry_in", delay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				continue
			}
		}
		failures = 0
		empty := ""
		b.lastError.Store(&empty)

		for _, u := range updates {
			if u.UpdateID > b.lastUpdate {
				b.lastUpdate = u.UpdateID
			}
			b.receivedCount.Add(1)
			// Authorization check up-front. Anything from a non-allowed
			// chat is dropped before the handler ever sees it.
			chatID := u.chatID()
			if chatID != 0 && !b.IsAllowed(chatID) {
				b.logger.Warn("telegram: dropped update from unauthorized chat",
					"chat_id", chatID)
				continue
			}
			if b.handler != nil {
				go b.handler.HandleUpdate(ctx, u)
			}
		}
	}
}

// ── Telegram Bot API calls ──────────────────────────────────────

// Update is a partial mirror of the Telegram Update object.
// Only the fields M1 needs are decoded; rest passes through as raw JSON.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

func (u Update) chatID() int64 {
	if u.Message != nil {
		return u.Message.Chat.ID
	}
	if u.CallbackQuery != nil && u.CallbackQuery.Message != nil {
		return u.CallbackQuery.Message.Chat.ID
	}
	return 0
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from,omitempty"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data"` // payload we set in the button
}

type Message struct {
	MessageID      int64   `json:"message_id"`
	Date           int64   `json:"date"`
	Chat           Chat    `json:"chat"`
	From           *User   `json:"from,omitempty"`
	Text           string  `json:"text,omitempty"`
	ReplyToMessage *Message `json:"reply_to_message,omitempty"`
}

type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
}

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	IsBot     bool   `json:"is_bot,omitempty"`
}

type meResponse struct {
	OK     bool `json:"ok"`
	Result User `json:"result"`
}

func (b *Bot) getMe(ctx context.Context) (User, error) {
	var resp meResponse
	if err := b.call(ctx, "getMe", nil, &resp); err != nil {
		return User{}, err
	}
	if !resp.OK {
		return User{}, fmt.Errorf("telegram: getMe returned ok=false")
	}
	return resp.Result, nil
}

type updatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

// errConflict is returned by getUpdates when another bot instance is polling
// with the same token. The caller should wait ≥30 s before retrying.
var errConflict = fmt.Errorf("telegram: another bot instance is using the same token — kill the other process or wait")

func (b *Bot) getUpdates(ctx context.Context, offset int64, timeout int) ([]Update, error) {
	q := url.Values{}
	q.Set("offset", strconv.FormatInt(offset, 10))
	q.Set("timeout", strconv.Itoa(timeout))
	q.Set("allowed_updates", `["message","callback_query"]`)
	var resp updatesResponse
	if err := b.call(ctx, "getUpdates?"+q.Encode(), nil, &resp); err != nil {
		// Detect the 409 Conflict ("terminated by other getUpdates request")
		// and return a sentinel so the poll loop can wait a long time before
		// retrying instead of exponential-spamming Telegram.
		if strings.Contains(err.Error(), "409") {
			return nil, errConflict
		}
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram: getUpdates ok=false")
	}
	return resp.Result, nil
}

// SendOpts mirrors the optional fields of sendMessage we care about.
type SendOpts struct {
	ParseMode        string      // "" | "Markdown" | "HTML" | "MarkdownV2"
	ReplyToMessageID int64
	DisablePreview   bool
	ReplyMarkup      interface{} // *InlineKeyboardMarkup or nil
}

// InlineKeyboardMarkup is the Telegram inline-keyboard structure.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// Send delivers a text message to chatID. Returns the sent message ID or
// 0 on failure. Long messages are split at 4000 chars to stay under
// Telegram's 4096 limit with a safety margin.
func (b *Bot) Send(ctx context.Context, chatID int64, text string, opts *SendOpts) (int64, error) {
	if chatID == 0 {
		return 0, fmt.Errorf("telegram: chat id is zero")
	}
	if text == "" {
		return 0, nil
	}
	const maxLen = 4000
	chunks := splitForTelegram(text, maxLen)

	var lastID int64
	for i, chunk := range chunks {
		body := map[string]any{
			"chat_id": chatID,
			"text":    chunk,
		}
		if opts != nil {
			if opts.ParseMode != "" {
				body["parse_mode"] = opts.ParseMode
			}
			if opts.ReplyToMessageID != 0 && i == 0 {
				body["reply_to_message_id"] = opts.ReplyToMessageID
			}
			if opts.DisablePreview {
				body["disable_web_page_preview"] = true
			}
			if opts.ReplyMarkup != nil && i == len(chunks)-1 {
				body["reply_markup"] = opts.ReplyMarkup
			}
		}
		var resp struct {
			OK     bool `json:"ok"`
			Result struct {
				MessageID int64 `json:"message_id"`
			} `json:"result"`
			Description string `json:"description,omitempty"`
		}
		if err := b.call(ctx, "sendMessage", body, &resp); err != nil {
			return lastID, err
		}
		if !resp.OK {
			return lastID, fmt.Errorf("telegram: sendMessage: %s", resp.Description)
		}
		lastID = resp.Result.MessageID
		b.sentCount.Add(1)
	}
	return lastID, nil
}

// AnswerCallbackQuery acknowledges a button press. Telegram shows a loading
// spinner on the button until this is called.
func (b *Bot) AnswerCallbackQuery(ctx context.Context, id string, text string) {
	body := map[string]any{"callback_query_id": id}
	if text != "" {
		body["text"] = text
		body["show_alert"] = false
	}
	_ = b.call(ctx, "answerCallbackQuery", body, nil)
}

// EditMessageReplyMarkup swaps the inline keyboard on an existing
// message. Used to flip a checkbox in place without spamming the chat.
// A nil kb clears all buttons.
func (b *Bot) EditMessageReplyMarkup(ctx context.Context, chatID, messageID int64, kb *InlineKeyboardMarkup) error {
	body := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	if kb != nil {
		body["reply_markup"] = kb
	} else {
		body["reply_markup"] = map[string]any{"inline_keyboard": [][]InlineKeyboardButton{}}
	}
	var resp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description,omitempty"`
	}
	if err := b.call(ctx, "editMessageReplyMarkup", body, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram: editMessageReplyMarkup: %s", resp.Description)
	}
	return nil
}

// EditMessageText replaces the body text of an existing message. Used to
// convert a multi-select prompt into a "✅ already submitted" receipt
// once the user commits their choice.
func (b *Bot) EditMessageText(ctx context.Context, chatID, messageID int64, text string, opts *SendOpts) error {
	body := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
	}
	if opts != nil {
		if opts.ParseMode != "" {
			body["parse_mode"] = opts.ParseMode
		}
		if opts.DisablePreview {
			body["disable_web_page_preview"] = true
		}
		if opts.ReplyMarkup != nil {
			body["reply_markup"] = opts.ReplyMarkup
		}
	}
	var resp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description,omitempty"`
	}
	if err := b.call(ctx, "editMessageText", body, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram: editMessageText: %s", resp.Description)
	}
	return nil
}

// call performs a Telegram Bot API request. body=nil uses GET; otherwise POST JSON.
func (b *Bot) call(ctx context.Context, method string, body any, into any) error {
	endpoint := "https://api.telegram.org/bot" + b.cfg.BotToken + "/" + method

	var req *http.Request
	var err error
	if body == nil {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	} else {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	if err != nil {
		return err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if into != nil {
		if err := json.Unmarshal(raw, into); err != nil {
			return fmt.Errorf("telegram: decode: %w", err)
		}
	}
	return nil
}

// splitForTelegram chops `text` at line boundaries when possible, never
// exceeding maxLen runes per chunk.
func splitForTelegram(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}
	var out []string
	remaining := text
	for len(remaining) > maxLen {
		// Prefer to split on a newline within the window.
		split := strings.LastIndex(remaining[:maxLen], "\n")
		if split <= 0 {
			split = maxLen
		}
		out = append(out, remaining[:split])
		remaining = strings.TrimPrefix(remaining[split:], "\n")
	}
	if len(remaining) > 0 {
		out = append(out, remaining)
	}
	return out
}
