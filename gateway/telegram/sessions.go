package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/linivek/ntc/kernel/hub"
	"github.com/linivek/ntc/kernel/store"
)

// SessionResolver caches a numbered list of sessions per chat so the user
// can type `/link 1` instead of `/link 0c3b9f2a-...`. The cache is
// refreshed on every /sessions or /status call and whenever a command
// needs to resolve an argument.
//
// Resolution order (first match wins):
//   1. Integer index → nth entry from the last /sessions listing
//   2. Exact session name (case-insensitive)
//   3. Session ID prefix (≥4 chars)
//   4. Exact full session ID (fallback)
type SessionResolver struct {
	hub *hub.Hub
	mu  sync.RWMutex
	// chatID → ordered list of session IDs from the last /sessions call
	cache map[int64][]string
}

func NewSessionResolver(h *hub.Hub) *SessionResolver {
	return &SessionResolver{
		hub:   h,
		cache: map[int64][]string{},
	}
}

// Refresh loads the running session list for this chat and returns it.
// Also caches the ordering for subsequent Resolve calls.
func (r *SessionResolver) Refresh(ctx context.Context, chatID int64) ([]store.Session, error) {
	all, err := r.hub.List(ctx)
	if err != nil {
		return nil, err
	}
	var running []store.Session
	for _, s := range all {
		if s.Status == "running" {
			running = append(running, s)
		}
	}

	ids := make([]string, len(running))
	for i, s := range running {
		ids[i] = s.ID
	}
	r.mu.Lock()
	r.cache[chatID] = ids
	r.mu.Unlock()

	return running, nil
}

// Resolve turns a user-typed argument into a full session ID.
// Accepts: "1" (number), "ntc-dev" (name), "0c3b" (prefix), or full UUID.
func (r *SessionResolver) Resolve(ctx context.Context, chatID int64, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", fmt.Errorf("session argument is empty — try /sessions to see the list")
	}

	// 1. Try as an integer index (1-based)
	if idx, err := strconv.Atoi(arg); err == nil {
		r.mu.RLock()
		ids := r.cache[chatID]
		r.mu.RUnlock()
		if len(ids) == 0 {
			// Cache empty — refresh and retry
			if _, err := r.Refresh(ctx, chatID); err == nil {
				r.mu.RLock()
				ids = r.cache[chatID]
				r.mu.RUnlock()
			}
		}
		if idx >= 1 && idx <= len(ids) {
			return ids[idx-1], nil
		}
		return "", fmt.Errorf("no session #%d — run /sessions to see the list (%d running)", idx, len(ids))
	}

	// Need the session list for name/prefix matching
	sessions, err := r.hub.List(ctx)
	if err != nil {
		return "", err
	}

	// 2. Exact name match (case-insensitive)
	argLower := strings.ToLower(arg)
	for _, s := range sessions {
		if strings.ToLower(s.Name) == argLower {
			return s.ID, nil
		}
	}

	// 3. Session type match (e.g. "claude", "terminal")
	for _, s := range sessions {
		if strings.ToLower(s.SessionType) == argLower && s.Status == "running" {
			return s.ID, nil
		}
	}

	// 4. ID prefix match (≥4 chars)
	if len(arg) >= 4 {
		for _, s := range sessions {
			if strings.HasPrefix(s.ID, arg) {
				return s.ID, nil
			}
		}
	}

	// 5. Full ID exact match
	for _, s := range sessions {
		if s.ID == arg {
			return s.ID, nil
		}
	}

	return "", fmt.Errorf("no session matching %q — run /sessions to see the list", arg)
}

// FormatSessionList builds a user-friendly numbered list for /sessions.
func FormatSessionList(sessions []store.Session) string {
	if len(sessions) == 0 {
		return "No running sessions."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("*%d running*\n\n", len(sessions)))
	for i, s := range sessions {
		name := s.Name
		if name == "" {
			name = s.SessionType
		}
		shortID := s.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		b.WriteString(fmt.Sprintf(
			"`%d` · *%s* (%s)\n   `%s` · %s\n\n",
			i+1, name, s.SessionType, shortID, shortenPath(s.CWD),
		))
	}
	b.WriteString("_Use the number to link:_ `/link 1`")
	return b.String()
}
