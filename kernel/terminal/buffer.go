// Package terminal provides the PTY engine for OpenDray.
package terminal

import (
	"sync"
)

// RingBuffer is a thread-safe circular byte buffer with pub/sub support.
type RingBuffer struct {
	mu   sync.RWMutex
	buf  []byte
	cap  int
	head int // next write position
	full bool

	subMu sync.RWMutex
	subs  map[uint64]chan []byte
	subID uint64
}

// NewRingBuffer creates a buffer with the given byte capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		buf:  make([]byte, capacity),
		cap:  capacity,
		subs: make(map[uint64]chan []byte),
	}
}

// Write appends data to the buffer and notifies all subscribers.
func (rb *RingBuffer) Write(data []byte) {
	if len(data) == 0 {
		return
	}

	rb.mu.Lock()
	for i, b := range data {
		rb.buf[(rb.head+i)%rb.cap] = b
	}
	rb.head = (rb.head + len(data)) % rb.cap
	if !rb.full && len(data) >= rb.cap {
		rb.full = true
	} else if !rb.full {
		// check if we wrapped
		oldHead := (rb.head - len(data) + rb.cap) % rb.cap
		if rb.head <= oldHead && len(data) > 0 {
			rb.full = true
		}
	}
	rb.mu.Unlock()

	// copy for subscribers
	chunk := make([]byte, len(data))
	copy(chunk, data)

	rb.subMu.RLock()
	for _, ch := range rb.subs {
		select {
		case ch <- chunk:
		default:
			// subscriber too slow, drop
		}
	}
	rb.subMu.RUnlock()
}

// Snapshot returns a chronological copy of the entire buffer contents.
func (rb *RingBuffer) Snapshot() []byte {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if !rb.full {
		out := make([]byte, rb.head)
		copy(out, rb.buf[:rb.head])
		return out
	}

	out := make([]byte, rb.cap)
	// oldest data starts at head (since buffer is full)
	n := copy(out, rb.buf[rb.head:])
	copy(out[n:], rb.buf[:rb.head])
	return out
}

// Subscribe returns a channel that receives new data chunks.
// The returned ID must be passed to Unsubscribe when done.
func (rb *RingBuffer) Subscribe() (uint64, <-chan []byte) {
	ch := make(chan []byte, 256)
	rb.subMu.Lock()
	rb.subID++
	id := rb.subID
	rb.subs[id] = ch
	rb.subMu.Unlock()
	return id, ch
}

// Unsubscribe removes a subscriber and closes its channel.
func (rb *RingBuffer) Unsubscribe(id uint64) {
	rb.subMu.Lock()
	if ch, ok := rb.subs[id]; ok {
		delete(rb.subs, id)
		close(ch)
	}
	rb.subMu.Unlock()
}
