// Package telegram is a minimal Telegram channel implementation for
// opendray. Pure net/http calls to api.telegram.org — no third-party
// SDK. Long-poll getUpdates → InboundFunc; outbound goes via
// sendMessage.
//
// v1 also shipped multi-select / question state machines, slash-command
// parsing, and forwarder logic (~3000 LOC). Per ADR 0005 those are not
// in M4. Add them as a separate package once a routing layer needs
// them.
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
	"sync"
	"time"

	"github.com/opendray/opendray-v2/internal/channel"
)

const (
	defaultAPIBase = "https://api.telegram.org/bot"
	pollTimeoutSec = 25
	httpTimeout    = 35 * time.Second
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

// Telegram implements channel.Channel.
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

func (t *Telegram) ID() string   { return t.id }
func (t *Telegram) Kind() string { return "telegram" }

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
	t.log.Info("telegram channel started")
	return nil
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

func (t *Telegram) Send(ctx context.Context, msg channel.ChannelMessage) error {
	chatID := t.cfg.ChatID
	if msg.ConversationID != "" && msg.ConversationID != "default" {
		if id, err := strconv.ParseInt(msg.ConversationID, 10, 64); err == nil {
			chatID = id
		}
	}
	if chatID == 0 {
		return fmt.Errorf("telegram: no chat_id configured")
	}
	body := map[string]any{
		"chat_id": chatID,
		"text":    msg.Text,
	}
	return t.callAPI(ctx, "sendMessage", body, nil)
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
			"offset":  t.offset,
			"timeout": pollTimeoutSec,
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
			if u.Message == nil {
				continue
			}
			msg := channel.ChannelMessage{
				ChannelID:      t.id,
				Direction:      channel.DirectionInbound,
				ConversationID: strconv.FormatInt(u.Message.Chat.ID, 10),
				Author:         u.Message.From.username(),
				Text:           u.Message.Text,
				Timestamp:      time.Unix(u.Message.Date, 0).UTC(),
				Metadata: map[string]any{
					"telegram_message_id": u.Message.MessageID,
					"chat_type":           u.Message.Chat.Type,
				},
			}
			if err := inbound(ctx, msg); err != nil {
				t.log.Error("inbound handler failed", "err", err)
			}
		}
	}
}

func minDur(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

type tgUpdate struct {
	UpdateID int64      `json:"update_id"`
	Message  *tgMessage `json:"message,omitempty"`
}

type tgMessage struct {
	MessageID int     `json:"message_id"`
	From      *tgUser `json:"from,omitempty"`
	Chat      tgChat  `json:"chat"`
	Date      int64   `json:"date"`
	Text      string  `json:"text"`
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
