package telegram

import (
	"sync"
	"time"

	"github.com/linivek/ntc/gateway/telegram/jsonl"
)

// MultiSelectStore holds in-flight checkbox-keyboard state keyed by
// (chatID, messageID). State is created when a PromptMultiSelect message
// is sent and drained on Submit. Entries older than TTL are lazily
// expired on every access to cap memory.
//
// Multi-round flows are supported naturally: each new prompt = new
// Telegram message = new (chatID, messageID) tuple = independent state.
type MultiSelectStore struct {
	mu    sync.Mutex
	items map[msKey]*MultiSelectState
	ttl   time.Duration
}

type msKey struct {
	ChatID    int64
	MessageID int64
}

// MultiSelectState is the per-message selection record.
type MultiSelectState struct {
	SessionID string
	Options   []jsonl.PromptOption
	Checked   map[string]bool
	CreatedAt time.Time
}

// NewMultiSelectStore returns a store with the given TTL.
func NewMultiSelectStore(ttl time.Duration) *MultiSelectStore {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &MultiSelectStore{
		items: map[msKey]*MultiSelectState{},
		ttl:   ttl,
	}
}

// Create registers a fresh selection record for a newly-sent multi-select
// message. Idempotent — a re-create with the same key overwrites.
func (s *MultiSelectStore) Create(chatID, messageID int64, sessionID string, opts []jsonl.PromptOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked()
	s.items[msKey{chatID, messageID}] = &MultiSelectState{
		SessionID: sessionID,
		Options:   opts,
		Checked:   map[string]bool{},
		CreatedAt: time.Now(),
	}
}

// Get returns a snapshot (copy of Checked map) or nil when the entry is
// missing or expired. Callers must not mutate the returned Options slice.
func (s *MultiSelectStore) Get(chatID, messageID int64) *MultiSelectState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked()
	st, ok := s.items[msKey{chatID, messageID}]
	if !ok {
		return nil
	}
	checked := make(map[string]bool, len(st.Checked))
	for k, v := range st.Checked {
		checked[k] = v
	}
	return &MultiSelectState{
		SessionID: st.SessionID,
		Options:   st.Options,
		Checked:   checked,
		CreatedAt: st.CreatedAt,
	}
}

// Toggle flips the checked flag for a given option key and returns the
// updated snapshot. Returns nil if the entry is missing / expired, or if
// key is not a known option.
func (s *MultiSelectStore) Toggle(chatID, messageID int64, key string) *MultiSelectState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked()
	st, ok := s.items[msKey{chatID, messageID}]
	if !ok {
		return nil
	}
	valid := false
	for _, o := range st.Options {
		if o.Key == key {
			valid = true
			break
		}
	}
	if !valid {
		return nil
	}
	if st.Checked[key] {
		delete(st.Checked, key)
	} else {
		st.Checked[key] = true
	}
	checked := make(map[string]bool, len(st.Checked))
	for k, v := range st.Checked {
		checked[k] = v
	}
	return &MultiSelectState{
		SessionID: st.SessionID,
		Options:   st.Options,
		Checked:   checked,
		CreatedAt: st.CreatedAt,
	}
}

// Submit atomically removes and returns the final selection record.
// Returns nil if the entry is missing / expired. After this call the
// store no longer holds the entry — any late Toggle on the same key is
// a no-op.
func (s *MultiSelectStore) Submit(chatID, messageID int64) *MultiSelectState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked()
	st, ok := s.items[msKey{chatID, messageID}]
	if !ok {
		return nil
	}
	delete(s.items, msKey{chatID, messageID})
	return st
}

// gcLocked drops entries past TTL. Cheap enough to run on every access
// at NTC's scale (dozens of concurrent prompts max).
func (s *MultiSelectStore) gcLocked() {
	if s.ttl <= 0 {
		return
	}
	cutoff := time.Now().Add(-s.ttl)
	for k, st := range s.items {
		if st.CreatedAt.Before(cutoff) {
			delete(s.items, k)
		}
	}
}
