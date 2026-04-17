package telegram

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Link binds one Telegram chat to one OpenDray session. While linked, plain
// (non-slash) messages from the chat get forwarded as session input, and
// new session output is pushed back to the chat (coalesced into 2-second
// windows by Forwarder — see forwarder.go).
//
// At most one session per chat (re-linking replaces the previous binding).
// A session may have multiple chats listening (broadcast). LinkedAt is
// used in /links output for "linked 5m ago" affordance.
type Link struct {
	ChatID    int64     `json:"chatId"`
	SessionID string    `json:"sessionId"`
	LinkedAt  time.Time `json:"linkedAt"`
}

// LinkStore persists chat ↔ session bindings to a JSON file alongside the
// other runtime state. We don't reach for Postgres here because:
//   • the data is tiny (handful of rows)
//   • losing it on crash is graceful — the user just runs /link again
//   • Postgres add-row would need a migration which is over-engineering for v1
//
// Concurrent-safe; load-on-construct, save-after-write.
type LinkStore struct {
	path  string
	mu    sync.RWMutex
	links map[int64]Link // chat_id → link
	// pendingMsgs maps "telegram message_id we sent (notification)" →
	// session_id. When a user *replies* to that notification message,
	// the dispatcher routes the reply text to that session even if no
	// /link is in force. Cleared after 24 h.
	pendingMsgs map[int64]pending
}

type pending struct {
	sessionID string
	at        time.Time
}

// NewLinkStore loads from disk if present, otherwise starts empty.
// path defaults to {tmp}/opendray-telegram-links.json when "" is passed.
func NewLinkStore(path string) *LinkStore {
	if path == "" {
		path = filepath.Join(os.TempDir(), "opendray-telegram-links.json")
	}
	s := &LinkStore{
		path:        path,
		links:       map[int64]Link{},
		pendingMsgs: map[int64]pending{},
	}
	s.load()
	return s
}

func (s *LinkStore) load() {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return // first run — no file yet
	}
	var saved struct {
		Links []Link `json:"links"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		return
	}
	for _, l := range saved.Links {
		s.links[l.ChatID] = l
	}
}

// saveLocked persists the current link map. Caller must hold s.mu.
func (s *LinkStore) saveLocked() {
	out := struct {
		Links []Link `json:"links"`
	}{Links: make([]Link, 0, len(s.links))}
	for _, l := range s.links {
		out.Links = append(out.Links, l)
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, s.path)
}

// Set creates / replaces a link. Returns the prior session id (or "")
// for callers that want to log the replacement.
func (s *LinkStore) Set(chatID int64, sessionID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	prior := ""
	if existing, ok := s.links[chatID]; ok {
		prior = existing.SessionID
	}
	s.links[chatID] = Link{
		ChatID: chatID, SessionID: sessionID, LinkedAt: time.Now(),
	}
	s.saveLocked()
	return prior
}

// Remove deletes a link. Returns the unlinked session ID, or "" if none.
func (s *LinkStore) Remove(chatID int64) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	link, ok := s.links[chatID]
	if !ok {
		return ""
	}
	delete(s.links, chatID)
	s.saveLocked()
	return link.SessionID
}

// Get returns the session ID linked to chatID, or "" if none.
func (s *LinkStore) Get(chatID int64) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.links[chatID].SessionID
}

// ChatsFor returns every chat ID currently linked to sessionID. Used by
// Forwarder to fan-out output to every listener of a session.
func (s *LinkStore) ChatsFor(sessionID string) []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []int64
	for chatID, l := range s.links {
		if l.SessionID == sessionID {
			out = append(out, chatID)
		}
	}
	return out
}

// All returns a copy of every link. For panel display.
func (s *LinkStore) All() []Link {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Link, 0, len(s.links))
	for _, l := range s.links {
		out = append(out, l)
	}
	return out
}

// RememberNotification stores msg_id → session_id for reply-to-routing.
// Called immediately after the bot sends an idle notification.
func (s *LinkStore) RememberNotification(messageID int64, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingMsgs[messageID] = pending{sessionID: sessionID, at: time.Now()}
	// Cheap GC: drop entries older than 24 h on every write.
	cutoff := time.Now().Add(-24 * time.Hour)
	for k, v := range s.pendingMsgs {
		if v.at.Before(cutoff) {
			delete(s.pendingMsgs, k)
		}
	}
}

// ResolveReply checks whether a Telegram message-id was a notification we
// sent, and returns the session ID it referred to (or "").
func (s *LinkStore) ResolveReply(messageID int64) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pendingMsgs[messageID].sessionID
}

// ── Errors ─────────────────────────────────────────────────────

var (
	errLinkSessionNotRunning = errors.New("session is not running — start it first")
	errLinkUnknownSession    = errors.New("unknown session id")
)

// ValidateSessionExists is a small helper so commands.go can return a
// useful error before we mutate the store.
func ValidateSessionExists(found bool, running bool) error {
	if !found {
		return errLinkUnknownSession
	}
	if !running {
		return errLinkSessionNotRunning
	}
	return nil
}

// ShortLinkSummary formats a link list for /links responses.
func ShortLinkSummary(links []Link) string {
	if len(links) == 0 {
		return "(no active links)"
	}
	out := ""
	for _, l := range links {
		out += fmt.Sprintf("• chat `%d` → session `%s`\n", l.ChatID, l.SessionID)
	}
	return out
}
