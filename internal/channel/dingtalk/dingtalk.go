// Package dingtalk is opendray's bundled DingTalk (钉钉) channel.
//
// The current implementation supports the most common deployment
// pattern: a **custom group robot** with the optional HMAC-SHA256
// signature. Outbound: text + markdown + actionCard. Inbound is
// out-of-scope for this minimal version (DingTalk's stream/callback
// API requires the app-platform credential set; group robots do not
// surface callbacks).
//
// Provisioning summary: in the DingTalk group → ⋯ → Robot management →
// Add a custom robot → "Sign" security → copy the secret + webhook URL.
//
// Capabilities implemented: text, card (markdown + actionCard buttons
// rendered as link rows — DingTalk's group-robot cards do not support
// callback buttons, so the buttons are rendered as `[label](value)`
// markdown links and only `nav:` style values make sense).
package dingtalk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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

const httpTimeout = 30 * time.Second

func init() { channel.Register("dingtalk", New) }

type config struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret,omitempty"`
}

type DingTalk struct {
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
		return nil, fmt.Errorf("dingtalk: parse config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("dingtalk: webhook_url is required")
	}
	if log == nil {
		log = slog.Default()
	}
	return &DingTalk{
		id:     id,
		cfg:    cfg,
		log:    log.With("channel", "dingtalk", "channel_id", id),
		client: &http.Client{Timeout: httpTimeout},
	}, nil
}

func (d *DingTalk) ID() string   { return d.id }
func (d *DingTalk) Kind() string { return "dingtalk" }

func (d *DingTalk) Start(_ context.Context, inbound channel.InboundFunc) error {
	d.mu.Lock()
	d.inbound = inbound
	if d.cancel != nil {
		d.mu.Unlock()
		return nil
	}
	_, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.mu.Unlock()
	d.log.Info("dingtalk channel started (outbound-only)")
	return nil
}

func (d *DingTalk) Stop(_ context.Context) error {
	d.mu.Lock()
	cancel := d.cancel
	d.cancel = nil
	d.inbound = nil
	d.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	d.log.Info("dingtalk channel stopped")
	return nil
}

// Send posts a plain text message via the group robot webhook.
func (d *DingTalk) Send(ctx context.Context, msg channel.ChannelMessage) error {
	body := map[string]any{
		"msgtype": "text",
		"text":    map[string]string{"content": msg.Text},
	}
	return d.post(ctx, body)
}

// SendCard renders Card → DingTalk actionCard / markdown.
//
// DingTalk's "actionCard" supports a title + markdown body + N stacked
// "btns" each with a title + actionURL (must be http(s)). Since
// callback buttons are not available without the app-platform setup,
// `cmd:`-prefixed values are dropped (logged) and `nav:`-prefixed
// ones are turned into clickable buttons that open the URL after
// stripping the prefix.
func (d *DingTalk) SendCard(ctx context.Context, _ channel.ChannelMessage, card *channel.Card) error {
	title, body, btns := renderCard(card)
	if len(btns) > 0 {
		return d.post(ctx, map[string]any{
			"msgtype": "actionCard",
			"actionCard": map[string]any{
				"title":          title,
				"text":           body,
				"btnOrientation": "0",
				"btns":           btns,
			},
		})
	}
	return d.post(ctx, map[string]any{
		"msgtype":  "markdown",
		"markdown": map[string]any{"title": title, "text": body},
	})
}

// post wraps WebhookURL with a sign + timestamp when cfg.Secret is set
// and POSTs the JSON body.
func (d *DingTalk) post(ctx context.Context, body any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	endpoint := d.cfg.WebhookURL
	if d.cfg.Secret != "" {
		endpoint = signedURL(endpoint, d.cfg.Secret, time.Now())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
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
		return fmt.Errorf("dingtalk webhook: HTTP %d: %s", resp.StatusCode, respBody)
	}
	var probe struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(respBody, &probe)
	if probe.ErrCode != 0 {
		return fmt.Errorf("dingtalk webhook: errcode=%d errmsg=%s", probe.ErrCode, probe.ErrMsg)
	}
	return nil
}

// signedURL adds the timestamp+sign query parameters required when the
// custom robot is configured with the "Sign" security mode.
//
//	stringToSign = "{ts}\n{secret}"
//	sign         = base64(hmac-sha256(stringToSign, secret))
func signedURL(base, secret string, now time.Time) string {
	ts := strconv.FormatInt(now.UnixMilli(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "\n" + secret))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + "timestamp=" + ts + "&sign=" + url.QueryEscape(sign)
}

func renderCard(card *channel.Card) (title string, body string, btns []map[string]string) {
	if card == nil {
		return "", "", nil
	}
	if card.Header != nil {
		title = card.Header.Title
	}
	if title == "" {
		title = "OpenDray"
	}
	var lines []string
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
						btns = append(btns, map[string]string{"title": b.Text, "actionURL": u})
					}
					// cmd:* dropped silently — group robot can't fire callbacks
				}
			}
		case channel.CardListItem:
			lines = append(lines, v.Text)
			if u, ok := navURL(v.Button.Value); ok {
				btns = append(btns, map[string]string{"title": v.Button.Text, "actionURL": u})
			}
		case channel.CardSelect:
			// Render as plain markdown — selectable elements aren't supported.
			lines = append(lines, v.Placeholder)
		case channel.CardNote:
			lines = append(lines, "> "+v.Text)
		}
	}
	body = strings.Join(filterEmpty(lines), "\n\n")
	if body == "" {
		body = title
	}
	return title, body, btns
}

// navURL strips the "nav:" prefix used by opendray when a card button
// should open a URL. cc-connect uses bare URLs for the same purpose;
// we accept either form.
func navURL(v string) (string, bool) {
	switch {
	case strings.HasPrefix(v, "nav:"):
		u := strings.TrimPrefix(v, "nav:")
		if strings.HasPrefix(u, "/") {
			// relative path — caller wanted opendray UI navigation; the
			// group robot can't reach it without the admin host. Drop.
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
