// Package eventbus implements opendray's in-process pub/sub.
//
// Subscribers receive events on a buffered channel; if the buffer is full,
// the event for that subscriber is dropped and a warning is logged. The bus
// never blocks publishers — slow subscribers are their own problem.
//
// Topic syntax is dot-namespaced (e.g. "session.output", "integration.health").
// Subscribe patterns may end with ".*" for prefix matching, or be a literal
// topic for exact matching.
package eventbus

import (
	"log/slog"
	"strings"
	"sync"
	"time"
)

type Event struct {
	Topic string    `json:"topic"`
	Data  any       `json:"data,omitempty"`
	Time  time.Time `json:"ts"`
}

type subscription struct {
	pattern string
	ch      chan Event
}

type Hub struct {
	mu     sync.RWMutex
	subs   []*subscription
	log    *slog.Logger
	closed bool
}

func New(log *slog.Logger) *Hub {
	if log == nil {
		log = slog.Default()
	}
	return &Hub{log: log.With("component", "eventbus")}
}

// Publish delivers ev to every matching subscriber. Non-blocking: a full
// subscriber buffer causes that one event to be dropped (logged) without
// affecting other subscribers.
func (h *Hub) Publish(ev Event) {
	if ev.Time.IsZero() {
		ev.Time = time.Now()
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.closed {
		return
	}
	for _, s := range h.subs {
		if !match(s.pattern, ev.Topic) {
			continue
		}
		select {
		case s.ch <- ev:
		default:
			h.log.Warn("dropped event: subscriber buffer full",
				"topic", ev.Topic, "pattern", s.pattern)
		}
	}
}

// Subscribe returns a receive-only channel and an unsubscribe function.
// Buffer must be >= 1.
func (h *Hub) Subscribe(pattern string, buffer int) (<-chan Event, func()) {
	if buffer < 1 {
		buffer = 1
	}
	s := &subscription{pattern: pattern, ch: make(chan Event, buffer)}
	h.mu.Lock()
	h.subs = append(h.subs, s)
	h.mu.Unlock()
	return s.ch, func() { h.unsubscribe(s) }
}

func (h *Hub) unsubscribe(target *subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, s := range h.subs {
		if s == target {
			h.subs = append(h.subs[:i], h.subs[i+1:]...)
			close(s.ch)
			return
		}
	}
}

// Close stops accepting publishes and closes all subscriber channels.
// Idempotent.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for _, s := range h.subs {
		close(s.ch)
	}
	h.subs = nil
}

func match(pattern, topic string) bool {
	if pattern == topic {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(topic, prefix)
	}
	if pattern == "*" {
		return true
	}
	return false
}
