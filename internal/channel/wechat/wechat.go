// Package wechat is opendray's bundled personal-WeChat (个人微信)
// channel.
//
// Personal WeChat has no official open API, so this implementation
// uses **WxPusher** (https://wxpusher.zjiecode.com), a free push
// service: a user follows a public WeChat account once, and any
// caller with the App Token can push notifications to their phone.
// This is currently the lowest-friction, lowest-risk way to deliver
// session.* events to personal WeChat.
//
// Outbound only (push services do not relay user replies). For
// bidirectional personal-WeChat scenarios, use the bridge channel
// with a WeChaty / iPad-protocol adapter.
//
// Capabilities implemented: text · card (markdown).
//
// Provisioning:
//
//  1. Visit https://wxpusher.zjiecode.com → 应用管理 → 创建应用 →
//     copy the **APP_TOKEN** (starts with `AT_`).
//  2. Open the QR code from 用户管理 in WeChat to subscribe; copy
//     your **UID** (starts with `UID_`). Or define a 主题 (topic)
//     and use its numeric topicId so anyone subscribed to the topic
//     receives the push.
package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/opendray/opendray-v2/internal/channel"
)

const (
	apiBase     = "https://wxpusher.zjiecode.com"
	httpTimeout = 30 * time.Second
)

// Test seam.
var apiBaseOverride = ""

func apiURL(path string) string {
	if apiBaseOverride != "" {
		return apiBaseOverride + path
	}
	return apiBase + path
}

func init() { channel.Register("wechat", New) }

// contentType values from WxPusher docs:
//
//	1 = plain text
//	2 = HTML
//	3 = Markdown
const (
	wxpContentText     = 1
	wxpContentMarkdown = 3
)

type config struct {
	AppToken string   `json:"app_token"`
	UIDs     []string `json:"uids,omitempty"`
	TopicIDs []int    `json:"topic_ids,omitempty"`
	URL      string   `json:"url,omitempty"` // optional landing URL on tap
}

type WeChat struct {
	id     string
	cfg    config
	log    *slog.Logger
	client *http.Client

	mu      sync.Mutex
	cancel  context.CancelFunc
	inbound channel.InboundFunc
}

func New(id string, raw json.RawMessage, log *slog.Logger) (channel.Channel, error) {
	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("wechat: parse config: %w", err)
	}
	if cfg.AppToken == "" {
		return nil, fmt.Errorf("wechat: app_token is required")
	}
	if len(cfg.UIDs) == 0 && len(cfg.TopicIDs) == 0 {
		return nil, fmt.Errorf("wechat: at least one of uids or topic_ids is required")
	}
	if log == nil {
		log = slog.Default()
	}
	return &WeChat{
		id:     id,
		cfg:    cfg,
		log:    log.With("channel", "wechat", "channel_id", id),
		client: &http.Client{Timeout: httpTimeout},
	}, nil
}

func (w *WeChat) ID() string   { return w.id }
func (w *WeChat) Kind() string { return "wechat" }

func (w *WeChat) Start(_ context.Context, inbound channel.InboundFunc) error {
	w.mu.Lock()
	w.inbound = inbound
	if w.cancel != nil {
		w.mu.Unlock()
		return nil
	}
	_, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.mu.Unlock()
	w.log.Info("wechat channel started (push-only via wxpusher)")
	return nil
}

func (w *WeChat) Stop(_ context.Context) error {
	w.mu.Lock()
	cancel := w.cancel
	w.cancel = nil
	w.inbound = nil
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	w.log.Info("wechat channel stopped")
	return nil
}

// Send pushes a plain-text notification.
func (w *WeChat) Send(ctx context.Context, msg channel.ChannelMessage) error {
	return w.push(ctx, summary(msg.Text), msg.Text, wxpContentText)
}

// SendCard renders Card → markdown push. Buttons are rendered as
// `[text](url)` links when their value points to a clickable URL,
// otherwise dropped (push services have no callback channel).
func (w *WeChat) SendCard(ctx context.Context, _ channel.ChannelMessage, card *channel.Card) error {
	body := renderMarkdown(card)
	title := summary(body)
	if card != nil && card.Header != nil && card.Header.Title != "" {
		title = card.Header.Title
	}
	return w.push(ctx, title, body, wxpContentMarkdown)
}

func (w *WeChat) push(ctx context.Context, summaryStr, content string, contentType int) error {
	body := map[string]any{
		"appToken":    w.cfg.AppToken,
		"content":     content,
		"summary":     summaryStr,
		"contentType": contentType,
	}
	if len(w.cfg.UIDs) > 0 {
		body["uids"] = w.cfg.UIDs
	}
	if len(w.cfg.TopicIDs) > 0 {
		body["topicIds"] = w.cfg.TopicIDs
	}
	if w.cfg.URL != "" {
		body["url"] = w.cfg.URL
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		apiURL("/api/send/message"), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("wxpusher: HTTP %d: %s", resp.StatusCode, respBody)
	}
	var probe struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	_ = json.Unmarshal(respBody, &probe)
	if probe.Code != 1000 { // wxpusher uses 1000 for success
		return fmt.Errorf("wxpusher: code=%d msg=%s", probe.Code, probe.Msg)
	}
	return nil
}

// summary returns a one-line preview suitable for the WeChat
// notification banner. WxPusher caps the field at ~20 chars so we
// trim aggressively.
func summary(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "OpenDray"
	}
	// Strip newlines.
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) > 20 {
		return string(runes[:20]) + "…"
	}
	return s
}

// renderMarkdown turns our Card into WxPusher-flavoured Markdown.
// WxPusher uses standard CommonMark; the WeChat client renders most
// of it (headings, lists, links). Buttons whose value resolves to a
// URL become `[label](url)` rows at the bottom.
func renderMarkdown(card *channel.Card) string {
	if card == nil {
		return ""
	}
	var lines []string
	if card.Header != nil && card.Header.Title != "" {
		lines = append(lines, "## "+card.Header.Title)
	}
	var links []string
	for _, el := range card.Elements {
		switch v := el.(type) {
		case channel.CardMarkdown:
			lines = append(lines, v.Content)
		case channel.CardDivider:
			lines = append(lines, "---")
		case channel.CardActions:
			for _, row := range v.Buttons {
				for _, b := range row {
					if u, ok := navURL(b.Value); ok {
						links = append(links, fmt.Sprintf("[%s](%s)", b.Text, u))
					}
				}
			}
		case channel.CardListItem:
			line := "- " + v.Text
			if u, ok := navURL(v.Button.Value); ok {
				line += fmt.Sprintf("  [%s](%s)", v.Button.Text, u)
			}
			lines = append(lines, line)
		case channel.CardSelect:
			lines = append(lines, v.Placeholder)
		case channel.CardNote:
			lines = append(lines, "> "+v.Text)
		}
	}
	if len(links) > 0 {
		lines = append(lines, strings.Join(links, "  ·  "))
	}
	return strings.Join(filterEmpty(lines), "\n\n")
}

func navURL(v string) (string, bool) {
	switch {
	case strings.HasPrefix(v, "nav:"):
		u := strings.TrimPrefix(v, "nav:")
		if strings.HasPrefix(u, "/") {
			return "", false
		}
		return u, true
	case strings.HasPrefix(v, "http://"), strings.HasPrefix(v, "https://"):
		return v, true
	}
	return "", false
}

func filterEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}
