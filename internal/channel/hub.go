package channel

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// Hub manages the lifecycle of all configured channels in this process.
type Hub struct {
	log   *slog.Logger
	bus   *eventbus.Hub
	store *store

	mu        sync.RWMutex
	channels  map[string]Channel
	started   bool
	cancelOut context.CancelFunc
	outDone   chan struct{}
}

func NewHub(pool *pgxpool.Pool, bus *eventbus.Hub, log *slog.Logger) *Hub {
	if log == nil {
		log = slog.Default()
	}
	return &Hub{
		log:      log.With("component", "channel"),
		bus:      bus,
		store:    newStore(pool),
		channels: make(map[string]Channel),
	}
}

// Start loads enabled channels from DB, instantiates each via its
// registered factory, calls Channel.Start, and subscribes to outbound
// session.* events. Caller must call Shutdown to stop.
func (h *Hub) Start(ctx context.Context) error {
	h.mu.Lock()
	if h.started {
		h.mu.Unlock()
		return nil
	}
	h.started = true
	h.mu.Unlock()

	rows, err := h.store.List(ctx)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if !r.Enabled {
			continue
		}
		if err := h.spawn(ctx, r); err != nil {
			h.log.Error("channel start failed", "id", r.ID, "kind", r.Kind, "err", err)
		}
	}

	outCtx, cancel := context.WithCancel(context.Background())
	h.cancelOut = cancel
	h.outDone = make(chan struct{})
	go h.runOutbound(outCtx)
	return nil
}

// Shutdown stops all channels and the outbound dispatcher.
func (h *Hub) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	if !h.started {
		h.mu.Unlock()
		return nil
	}
	h.started = false
	cancel := h.cancelOut
	done := h.outDone
	chs := make([]Channel, 0, len(h.channels))
	for _, c := range h.channels {
		chs = append(chs, c)
	}
	h.channels = make(map[string]Channel)
	h.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	for _, c := range chs {
		if err := c.Stop(ctx); err != nil {
			h.log.Error("channel stop", "id", c.ID(), "err", err)
		}
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (h *Hub) spawn(ctx context.Context, r channelRow) error {
	factory := Lookup(r.Kind)
	if factory == nil {
		return fmt.Errorf("%w: %s", ErrUnknownKind, r.Kind)
	}
	ch, err := factory(r.ID, r.Config, h.log)
	if err != nil {
		return fmt.Errorf("factory: %w", err)
	}
	if err := ch.Start(ctx, h.handleInbound); err != nil {
		return fmt.Errorf("channel start: %w", err)
	}
	h.mu.Lock()
	h.channels[r.ID] = ch
	h.mu.Unlock()
	return nil
}

// handleInbound is invoked by Channel impls when a message arrives.
// Per ADR 0005 the Hub persists + publishes; routing to sessions is
// out of scope for M4.
func (h *Hub) handleInbound(ctx context.Context, msg ChannelMessage) error {
	msg.Direction = DirectionInbound
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	}
	id, err := h.store.InsertMessage(ctx, msg)
	if err != nil {
		h.log.Error("inbound persist failed", "channel", msg.ChannelID, "err", err)
		return err
	}
	h.bus.Publish(eventbus.Event{
		Topic: "channel.message_received",
		Data: map[string]any{
			"channel_id":         msg.ChannelID,
			"channel_message_id": id,
			"conversation_id":    msg.ConversationID,
			"author":             msg.Author,
			"text":               msg.Text,
		},
	})
	return nil
}

func (h *Hub) runOutbound(ctx context.Context) {
	defer close(h.outDone)
	chIdle, unsubI := h.bus.Subscribe("session.idle", 64)
	defer unsubI()
	chEnded, unsubE := h.bus.Subscribe("session.ended", 64)
	defer unsubE()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-chIdle:
			if !ok {
				return
			}
			h.dispatch(ctx, ev)
		case ev, ok := <-chEnded:
			if !ok {
				return
			}
			h.dispatch(ctx, ev)
		}
	}
}

func (h *Hub) dispatch(ctx context.Context, ev eventbus.Event) {
	h.mu.RLock()
	chs := make([]Channel, 0, len(h.channels))
	for _, c := range h.channels {
		chs = append(chs, c)
	}
	h.mu.RUnlock()

	for _, c := range chs {
		topics, err := h.notifyTopicsFor(ctx, c.ID())
		if err != nil {
			continue
		}
		if len(topics) > 0 && !contains(topics, ev.Topic) {
			continue
		}
		text := formatNotification(ev)
		msg := ChannelMessage{
			ChannelID:      c.ID(),
			Direction:      DirectionOutbound,
			ConversationID: "default",
			Text:           text,
			Timestamp:      time.Now().UTC(),
		}
		if err := c.Send(ctx, msg); err != nil {
			h.log.Error("channel send failed", "id", c.ID(), "err", err)
			continue
		}
		if _, err := h.store.InsertMessage(ctx, msg); err != nil {
			h.log.Warn("outbound persist failed", "id", c.ID(), "err", err)
		}
		h.bus.Publish(eventbus.Event{
			Topic: "channel.message_sent",
			Data: map[string]any{
				"channel_id": c.ID(),
				"topic":      ev.Topic,
			},
		})
	}
}

func (h *Hub) notifyTopicsFor(ctx context.Context, channelID string) ([]string, error) {
	row, err := h.store.Get(ctx, channelID)
	if err != nil {
		return nil, err
	}
	var cfg struct {
		NotifyOn []string `json:"notify_on"`
	}
	_ = json.Unmarshal(row.Config, &cfg)
	return cfg.NotifyOn, nil
}

// CreateChannel registers a new channel and starts it if enabled.
func (h *Hub) CreateChannel(ctx context.Context, kind string, config json.RawMessage, enabled bool) (string, error) {
	if Lookup(kind) == nil {
		return "", fmt.Errorf("%w: %s", ErrUnknownKind, kind)
	}
	id := newID()
	if err := h.store.Insert(ctx, id, kind, config, enabled); err != nil {
		return "", err
	}
	if enabled && h.isStarted() {
		if err := h.spawn(ctx, channelRow{ID: id, Kind: kind, Config: config, Enabled: true}); err != nil {
			return "", err
		}
	}
	return id, nil
}

// UpdateChannel persists changes and restarts the impl when running.
// Pass nil for any unchanged field.
func (h *Hub) UpdateChannel(ctx context.Context, id string, config json.RawMessage, enabled *bool) error {
	if err := h.store.Update(ctx, id, config, enabled); err != nil {
		return err
	}
	row, err := h.store.Get(ctx, id)
	if err != nil {
		return err
	}
	h.mu.Lock()
	existing, running := h.channels[id]
	delete(h.channels, id)
	h.mu.Unlock()
	if running {
		_ = existing.Stop(ctx)
	}
	if row.Enabled && h.isStarted() {
		return h.spawn(ctx, row)
	}
	return nil
}

// DeleteChannel stops the running impl (if any) and removes the row.
func (h *Hub) DeleteChannel(ctx context.Context, id string) error {
	h.mu.Lock()
	ch, ok := h.channels[id]
	delete(h.channels, id)
	h.mu.Unlock()
	if ok {
		_ = ch.Stop(ctx)
	}
	return h.store.Delete(ctx, id)
}

// SendTest pushes a fixed text message via channel.Send.
func (h *Hub) SendTest(ctx context.Context, id string) error {
	h.mu.RLock()
	ch, ok := h.channels[id]
	h.mu.RUnlock()
	if !ok {
		return ErrNotFound
	}
	return ch.Send(ctx, ChannelMessage{
		ChannelID:      id,
		Direction:      DirectionOutbound,
		ConversationID: "default",
		Text:           "OpenDray channel test ✓",
		Timestamp:      time.Now().UTC(),
	})
}

// List returns the persisted channels along with a "running" flag.
func (h *Hub) List(ctx context.Context) ([]ChannelView, error) {
	rows, err := h.store.List(ctx)
	if err != nil {
		return nil, err
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]ChannelView, 0, len(rows))
	for _, r := range rows {
		_, running := h.channels[r.ID]
		out = append(out, ChannelView{
			ID:      r.ID,
			Kind:    r.Kind,
			Config:  r.Config,
			Enabled: r.Enabled,
			Running: running,
		})
	}
	return out, nil
}

// Get returns one channel view.
func (h *Hub) Get(ctx context.Context, id string) (ChannelView, error) {
	r, err := h.store.Get(ctx, id)
	if err != nil {
		return ChannelView{}, err
	}
	h.mu.RLock()
	_, running := h.channels[id]
	h.mu.RUnlock()
	return ChannelView{
		ID:      r.ID,
		Kind:    r.Kind,
		Config:  r.Config,
		Enabled: r.Enabled,
		Running: running,
	}, nil
}

// ChannelView is the public wire shape for REST.
type ChannelView struct {
	ID      string          `json:"id"`
	Kind    string          `json:"kind"`
	Config  json.RawMessage `json:"config"`
	Enabled bool            `json:"enabled"`
	Running bool            `json:"running"`
}

func (h *Hub) isStarted() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.started
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// formatNotification turns an event payload into the text we send to
// channels. Kept terse — channel UX preference is short, mobile-line
// notifications.
func formatNotification(ev eventbus.Event) string {
	data, _ := ev.Data.(map[string]any)
	sid, _ := data["session_id"].(string)
	switch ev.Topic {
	case "session.idle":
		ms := toInt64(data["idle_for_ms"])
		return fmt.Sprintf("Session %s went idle (silent for %ds).", sid, ms/1000)
	case "session.ended":
		exit := toInt64(data["exit_code"])
		return fmt.Sprintf("Session %s ended with exit_code=%d.", sid, exit)
	}
	return fmt.Sprintf("%s: %s", ev.Topic, sid)
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	}
	return 0
}

func newID() string {
	var b [9]byte
	_, _ = rand.Read(b[:])
	return "ch_" + base64.RawURLEncoding.EncodeToString(b[:])
}
