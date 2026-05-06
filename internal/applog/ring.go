// Package applog provides an in-memory ring buffer for slog records
// plus a custom slog.Handler that fans every log line out to:
//
//   - stderr (or whichever underlying handler the operator chose)
//   - the ring buffer (so /admin/logs/tail and the WS stream can
//     replay or live-tail recent activity)
//   - an optional rotating file (when [log].file is set)
//
// The ring is sized so that ~2,000 records (default) cost ~1-2 MB
// of RSS — small enough to leave on at all times and large enough
// to cover several minutes of busy traffic.
package applog

import (
	"log/slog"
	"sync"
	"time"
)

// Record is one log line as captured for in-process display.
// Self-contained: Text holds a pre-rendered single-line preview so
// the WS stream and tail endpoint don't need access to the original
// slog handler.
type Record struct {
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"` // "DEBUG"/"INFO"/"WARN"/"ERROR"
	Message string         `json:"message"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Text    string         `json:"text"` // rendered "13:48:22 INFO foo=bar baz" preview
}

// Buffer is a thread-safe ring of the most recent N Records plus a
// publish/subscribe channel set so subscribers can live-tail. Push
// is O(1); Snapshot copies the current items in chronological order.
type Buffer struct {
	mu       sync.RWMutex
	capacity int
	items    []Record
	head     int  // next write index
	full     bool // wrapped at least once

	subsMu sync.Mutex
	subs   map[chan Record]struct{}
}

// NewBuffer allocates a ring with the given capacity. capacity <= 0
// falls back to 2000.
func NewBuffer(capacity int) *Buffer {
	if capacity <= 0 {
		capacity = 2000
	}
	return &Buffer{
		capacity: capacity,
		items:    make([]Record, capacity),
		subs:     make(map[chan Record]struct{}),
	}
}

// Push appends r to the ring (overwriting the oldest entry once
// full) and broadcasts to every subscriber. Non-blocking: if a
// subscriber's channel is full the record is dropped for that
// subscriber rather than stalling the logger.
func (b *Buffer) Push(r Record) {
	b.mu.Lock()
	b.items[b.head] = r
	b.head++
	if b.head >= b.capacity {
		b.head = 0
		b.full = true
	}
	b.mu.Unlock()

	b.subsMu.Lock()
	for ch := range b.subs {
		select {
		case ch <- r:
		default:
			// Slow consumer; drop the record for that subscriber.
		}
	}
	b.subsMu.Unlock()
}

// Snapshot returns up to n most-recent records in chronological
// order (oldest first). n <= 0 returns the whole buffer.
func (b *Buffer) Snapshot(n int) []Record {
	b.mu.RLock()
	defer b.mu.RUnlock()
	total := b.capacity
	if !b.full {
		total = b.head
	}
	if n <= 0 || n > total {
		n = total
	}
	out := make([]Record, 0, n)
	// Walk from oldest to newest. When `full`, oldest is at b.head;
	// when not, oldest is at index 0.
	start := 0
	if b.full {
		start = b.head
	}
	skip := total - n // skip older items if capping
	for i := 0; i < total; i++ {
		idx := (start + i) % b.capacity
		if i < skip {
			continue
		}
		out = append(out, b.items[idx])
	}
	return out
}

// Subscribe registers a buffered channel for live-tail. Returns the
// channel and an unsubscribe function. Caller must drain promptly
// or accept dropped records.
func (b *Buffer) Subscribe() (<-chan Record, func()) {
	ch := make(chan Record, 64)
	b.subsMu.Lock()
	b.subs[ch] = struct{}{}
	b.subsMu.Unlock()
	unsub := func() {
		b.subsMu.Lock()
		if _, ok := b.subs[ch]; ok {
			delete(b.subs, ch)
			close(ch)
		}
		b.subsMu.Unlock()
	}
	return ch, unsub
}

// LevelFromSlog maps a slog.Level to its short upper-case name
// (e.g. slog.LevelInfo → "INFO"). Centralised so handler.go and
// formatters in package settings agree on spelling.
func LevelFromSlog(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARN"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}
