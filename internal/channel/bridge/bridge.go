// Package bridge implements opendray's external-adapter channel kind.
//
// One bridge channel row holds (token, name, accepted_capabilities)
// in its config JSON. An external adapter (Python WeChat bot,
// Node.js custom-platform script, ...) opens a WebSocket to
// /api/v1/channels/bridge/ws, presents the token, and declares its
// real capabilities — at which point opendray treats it as a regular
// Channel implementation and routes session.* notifications and
// commands through it like any other channel.
//
// Wire protocol — see ./protocol.md (also doc-commented in protocol.go).
package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/opendray/opendray-v2/internal/channel"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	registerWait   = 10 * time.Second
	maxMessageSize = 256 * 1024 // 256KB — comfortably above typical card payloads
)

// Config is the channel-row config JSON shape for kind="bridge".
type Config struct {
	// Name is a human label shown in the admin UI ("wechat", "discord-custom").
	Name string `json:"name"`
	// Token is the shared secret an adapter must present on connect.
	Token string `json:"token"`
	// AcceptCapabilities optionally restricts which capabilities the
	// adapter is allowed to claim on register. Empty = accept whatever
	// it declares.
	AcceptCapabilities []channel.Capability `json:"accept_capabilities,omitempty"`
}

// Broker maps tokens to attached Bridge channels. The HTTP handler
// looks an incoming connection up by token and hands it to the
// matching Bridge.
type Broker struct {
	mu      sync.RWMutex
	byToken map[string]*Bridge
	byID    map[string]*Bridge
}

// NewBroker returns a fresh in-memory broker. One per process.
func NewBroker() *Broker {
	return &Broker{
		byToken: make(map[string]*Bridge),
		byID:    make(map[string]*Bridge),
	}
}

func (b *Broker) register(token, id string, br *Bridge) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.byToken[token] = br
	b.byID[id] = br
}

func (b *Broker) deregister(token, id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.byToken, token)
	delete(b.byID, id)
}

// LookupByToken returns the bridge channel guarding the given token,
// or nil when none matches. Used by the WS handler.
func (b *Broker) LookupByToken(token string) *Bridge {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.byToken[token]
}

// defaultBroker is the package-level singleton wired by the Factory.
// Tests may swap it via SetBroker.
var defaultBroker = NewBroker()

// SetBroker replaces the package-level broker. Intended for tests so
// they can run isolated brokers in parallel.
func SetBroker(b *Broker) { defaultBroker = b }

// DefaultBroker returns the active broker.
func DefaultBroker() *Broker { return defaultBroker }

// Factory builds a Bridge channel from its DB row. Registered with
// the channel package via init() in factory.go so the Hub can spawn
// it like any other kind.
func Factory(id string, raw json.RawMessage, log *slog.Logger) (channel.Channel, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("bridge: parse config: %w", err)
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("bridge: token is required")
	}
	if log == nil {
		log = slog.Default()
	}
	return &Bridge{
		id:     id,
		cfg:    cfg,
		log:    log.With("channel", "bridge", "channel_id", id, "bridge_name", cfg.Name),
		broker: defaultBroker,
	}, nil
}

// Bridge implements channel.Channel + every optional capability
// interface. SendCard / SendImage / etc. call ErrNotSupported when the
// attached adapter did not claim the matching capability.
type Bridge struct {
	id     string
	cfg    Config
	log    *slog.Logger
	broker *Broker

	mu         sync.Mutex
	conn       *websocket.Conn
	writeMu    sync.Mutex // serialises conn writes
	caps       map[channel.Capability]bool
	platform   string // adapter-declared platform name ("wechat", ...)
	inbound    channel.InboundFunc
	cancel     context.CancelFunc
	done       chan struct{}
	registered bool
}

func (b *Bridge) ID() string   { return b.id }
func (b *Bridge) Kind() string { return "bridge" }

// SupportsReply mirrors whatever the adapter declared via the
// reply_to_message capability.
func (b *Bridge) SupportsReply() bool { return b.hasCap(channel.CapReplyToMessage) }

// DeclaredCapabilities returns whatever the connected adapter
// declared on register, or just CapText when no adapter is attached.
// Always includes CapText.
func (b *Bridge) DeclaredCapabilities() []channel.Capability {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := []channel.Capability{channel.CapText}
	for k, v := range b.caps {
		if v && k != channel.CapText {
			out = append(out, k)
		}
	}
	return out
}

func (b *Bridge) hasCap(c channel.Capability) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.caps != nil && b.caps[c]
}

// Start registers this bridge with the broker so an adapter
// connecting with a matching token can attach. There's no upstream
// connection to make — adapters connect to us.
func (b *Bridge) Start(_ context.Context, inbound channel.InboundFunc) error {
	b.mu.Lock()
	if b.registered {
		b.mu.Unlock()
		return nil
	}
	b.inbound = inbound
	b.registered = true
	b.mu.Unlock()
	b.broker.register(b.cfg.Token, b.id, b)
	b.log.Info("bridge registered", "name", b.cfg.Name)
	return nil
}

// Stop deregisters from the broker and tears down any attached conn.
func (b *Bridge) Stop(ctx context.Context) error {
	b.broker.deregister(b.cfg.Token, b.id)
	b.mu.Lock()
	conn := b.conn
	cancel := b.cancel
	done := b.done
	b.conn = nil
	b.cancel = nil
	b.done = nil
	b.registered = false
	b.caps = nil
	b.mu.Unlock()
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
	b.log.Info("bridge stopped")
	return nil
}

// attach hands an authenticated WS connection over to this Bridge.
// Capabilities filtered by AcceptCapabilities (when non-empty).
// Replaces any prior connection.
func (b *Bridge) attach(conn *websocket.Conn, declared []channel.Capability, platform string) {
	caps := make(map[channel.Capability]bool, len(declared))
	allowed := acceptedSet(b.cfg.AcceptCapabilities)
	caps[channel.CapText] = true
	for _, c := range declared {
		if allowed != nil && !allowed[c] {
			continue
		}
		caps[c] = true
	}

	b.mu.Lock()
	if b.conn != nil {
		_ = b.conn.Close()
	}
	if b.cancel != nil {
		b.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	b.conn = conn
	b.cancel = cancel
	b.done = done
	b.caps = caps
	b.platform = platform
	b.mu.Unlock()

	// Pass `done` as a captured local so readPump's deferred close
	// still fires even if Stop() (which clears b.done) wins the race
	// for b.mu. Reading b.done from inside readPump would race with
	// that clear and silently leak a never-closed channel — Stop
	// would then block forever on its own done receive.
	go b.readPump(ctx, conn, done)
	go b.pingPump(ctx, conn)
	b.log.Info("bridge adapter attached", "platform", platform, "caps", declared)
}

func acceptedSet(allowed []channel.Capability) map[channel.Capability]bool {
	if len(allowed) == 0 {
		return nil
	}
	out := make(map[channel.Capability]bool, len(allowed))
	for _, c := range allowed {
		out[c] = true
	}
	return out
}

// Send writes a "send" frame (text only). When no adapter is attached
// returns ErrNotSupported so the Hub can short-circuit.
func (b *Bridge) Send(_ context.Context, msg channel.ChannelMessage) error {
	return b.writeFrame(map[string]any{
		"type":            "send",
		"session_key":     msg.SessionKey("bridge"),
		"conversation_id": msg.ConversationID,
		"reply_ctx":       msg.ReplyCtx,
		"text":            msg.Text,
	})
}

// SendCard ships a structured Card frame. Returns ErrNotSupported
// when the adapter did not claim CapCard on register.
func (b *Bridge) SendCard(_ context.Context, msg channel.ChannelMessage, card *channel.Card) error {
	if !b.hasCap(channel.CapCard) {
		return channel.ErrNotSupported
	}
	return b.writeFrame(map[string]any{
		"type":            "send_card",
		"session_key":     msg.SessionKey("bridge"),
		"conversation_id": msg.ConversationID,
		"reply_ctx":       msg.ReplyCtx,
		"card":            card,
	})
}

// SendWithButtons ships a text + button-row frame.
func (b *Bridge) SendWithButtons(_ context.Context, msg channel.ChannelMessage, buttons [][]channel.ButtonOption) error {
	if !b.hasCap(channel.CapButtons) {
		return channel.ErrNotSupported
	}
	return b.writeFrame(map[string]any{
		"type":            "send_buttons",
		"session_key":     msg.SessionKey("bridge"),
		"conversation_id": msg.ConversationID,
		"reply_ctx":       msg.ReplyCtx,
		"text":            msg.Text,
		"buttons":         buttons,
	})
}

// UpdateMessage edits a previously-sent message. The adapter is
// expected to know how to interpret previewHandle.
func (b *Bridge) UpdateMessage(_ context.Context, msg channel.ChannelMessage, previewHandle, newText string) error {
	if !b.hasCap(channel.CapUpdateMessage) {
		return channel.ErrNotSupported
	}
	return b.writeFrame(map[string]any{
		"type":            "update_message",
		"session_key":     msg.SessionKey("bridge"),
		"conversation_id": msg.ConversationID,
		"preview_handle":  previewHandle,
		"text":            newText,
	})
}

// SendImage sends an image attachment. Adapter resolves Path or URL.
func (b *Bridge) SendImage(_ context.Context, msg channel.ChannelMessage, img channel.ImageAttachment) error {
	if !b.hasCap(channel.CapImage) {
		return channel.ErrNotSupported
	}
	return b.writeFrame(map[string]any{
		"type":            "send_image",
		"session_key":     msg.SessionKey("bridge"),
		"conversation_id": msg.ConversationID,
		"reply_ctx":       msg.ReplyCtx,
		"image":           img,
	})
}

// SendFile sends a file attachment.
func (b *Bridge) SendFile(_ context.Context, msg channel.ChannelMessage, file channel.FileAttachment) error {
	if !b.hasCap(channel.CapFile) {
		return channel.ErrNotSupported
	}
	return b.writeFrame(map[string]any{
		"type":            "send_file",
		"session_key":     msg.SessionKey("bridge"),
		"conversation_id": msg.ConversationID,
		"reply_ctx":       msg.ReplyCtx,
		"file":            file,
	})
}

// StartTyping notifies the adapter to display a typing indicator.
// Returns a stop func that fires the matching stop_typing frame.
func (b *Bridge) StartTyping(_ context.Context, msg channel.ChannelMessage) func() {
	if !b.hasCap(channel.CapTyping) {
		return func() {}
	}
	_ = b.writeFrame(map[string]any{
		"type":        "start_typing",
		"session_key": msg.SessionKey("bridge"),
		"reply_ctx":   msg.ReplyCtx,
	})
	stopped := false
	var once sync.Once
	return func() {
		once.Do(func() {
			stopped = true
			_ = b.writeFrame(map[string]any{
				"type":        "stop_typing",
				"session_key": msg.SessionKey("bridge"),
				"reply_ctx":   msg.ReplyCtx,
			})
		})
		_ = stopped // appease linters
	}
}

// writeFrame serialises one JSON object to the WS conn. Returns
// ErrNotSupported when no adapter is currently attached.
func (b *Bridge) writeFrame(payload map[string]any) error {
	b.mu.Lock()
	conn := b.conn
	b.mu.Unlock()
	if conn == nil {
		return channel.ErrNotSupported
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("bridge: marshal frame: %w", err)
	}
	b.writeMu.Lock()
	defer b.writeMu.Unlock()
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, raw)
}

// readPump reads one frame at a time and dispatches it to the right
// inbound path. Closes the supplied done channel on exit so Stop()
// can wait for the goroutine to actually finish.
func (b *Bridge) readPump(ctx context.Context, conn *websocket.Conn, done chan struct{}) {
	defer func() {
		close(done)
		_ = conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				if ce, ok := err.(*websocket.CloseError); ok {
					b.log.Info("bridge ws closed", "code", ce.Code)
				} else {
					b.log.Warn("bridge read", "err", err)
				}
			}
			return
		}
		var frame map[string]any
		if err := json.Unmarshal(raw, &frame); err != nil {
			b.log.Warn("bridge frame parse failed", "err", err)
			continue
		}
		b.handleFrame(ctx, frame)
	}
}

func (b *Bridge) pingPump(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.writeMu.Lock()
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := conn.WriteMessage(websocket.PingMessage, nil)
			b.writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// handleFrame dispatches one inbound frame from the adapter.
func (b *Bridge) handleFrame(ctx context.Context, frame map[string]any) {
	t, _ := frame["type"].(string)
	switch t {
	case "message":
		b.deliverMessage(ctx, frame, false)
	case "card_action":
		b.deliverMessage(ctx, frame, true)
	case "ping":
		// adapter-level ping; reply with pong over text frame so non-WS
		// transports also work in future.
		_ = b.writeFrame(map[string]any{"type": "pong"})
	default:
		b.log.Debug("bridge unknown frame", "type", t)
	}
}

func (b *Bridge) deliverMessage(ctx context.Context, frame map[string]any, asAction bool) {
	text, _ := frame["text"].(string)
	if asAction {
		action, _ := frame["action"].(string)
		text = channel.EncodeAction(action)
	}
	conv, _ := frame["conversation_id"].(string)
	if conv == "" {
		// fall back to session_key middle field
		if sk, ok := frame["session_key"].(string); ok {
			conv = sk
		}
	}
	author, _ := frame["user_id"].(string)
	if name, ok := frame["user_name"].(string); ok && name != "" {
		author = name
	}
	msg := channel.ChannelMessage{
		ChannelID:      b.id,
		Direction:      channel.DirectionInbound,
		ConversationID: conv,
		Author:         author,
		Text:           text,
		Timestamp:      time.Now().UTC(),
		ReplyCtx:       frame["reply_ctx"],
		Metadata: map[string]any{
			"bridge_platform": b.platform,
			"frame_type":      frameTypeName(asAction),
		},
	}
	b.mu.Lock()
	inbound := b.inbound
	b.mu.Unlock()
	if inbound == nil {
		return
	}
	if err := inbound(ctx, msg); err != nil {
		b.log.Error("bridge inbound dispatch failed", "err", err)
	}
}

func frameTypeName(asAction bool) string {
	if asAction {
		return "card_action"
	}
	return "message"
}
