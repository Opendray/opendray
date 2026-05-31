package cliacct

import (
	"sync"
	"time"
)

// ThrottleStore tracks which Claude accounts are currently rate-limited
// at the Anthropic side. Used by the auto-failover path (Phase 2 Tier A)
// to skip accounts whose quota is exhausted when picking the next one
// for a session whose current account just hit the limit.
//
// Pure in-memory — the throttle window is the response to a server-side
// rate limit, not a persistent attribute of the account. A gateway
// restart starts everyone "unthrottled"; the next request that hits a
// limit re-marks the account. This is intentional: a stale throttle
// marker after a long downtime would be worse than briefly trying a
// just-reset account.
type ThrottleStore struct {
	mu sync.RWMutex
	// m maps accountID → "throttled until this time (UTC)".
	// An entry with an expiry in the past is GC'd by the next access.
	m map[string]time.Time
}

// NewThrottleStore returns an empty store ready for use.
func NewThrottleStore() *ThrottleStore {
	return &ThrottleStore{m: map[string]time.Time{}}
}

// MarkThrottled records that accountID is throttled until 'until'.
// If the account is already marked with a LATER expiry, the existing
// entry wins (the more pessimistic of the two timestamps stays). If
// the new expiry is sooner, the existing entry stays — we never
// shorten a throttle, only extend.
func (t *ThrottleStore) MarkThrottled(accountID string, until time.Time) {
	if accountID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if existing, ok := t.m[accountID]; ok && existing.After(until) {
		return
	}
	t.m[accountID] = until
}

// IsThrottled reports whether accountID is currently throttled
// (i.e. has a non-expired entry). Expired entries are removed
// lazily on read.
func (t *ThrottleStore) IsThrottled(accountID string) bool {
	if accountID == "" {
		return false
	}
	now := time.Now().UTC()
	t.mu.Lock()
	defer t.mu.Unlock()
	until, ok := t.m[accountID]
	if !ok {
		return false
	}
	if !now.Before(until) {
		delete(t.m, accountID)
		return false
	}
	return true
}

// ThrottledIDs returns the ids of every account currently throttled.
// Useful for SQL exclusion when picking the next account. Removes
// expired entries as a side-effect.
func (t *ThrottleStore) ThrottledIDs() []string {
	now := time.Now().UTC()
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, 0, len(t.m))
	for id, until := range t.m {
		if now.Before(until) {
			out = append(out, id)
		} else {
			delete(t.m, id)
		}
	}
	return out
}

// Until returns the throttle expiry for accountID and whether one is
// currently set. Useful for surfacing "throttled until 10:20am UTC" in
// the UI. Returns zero time + false when not throttled.
func (t *ThrottleStore) Until(accountID string) (time.Time, bool) {
	if accountID == "" {
		return time.Time{}, false
	}
	now := time.Now().UTC()
	t.mu.RLock()
	defer t.mu.RUnlock()
	until, ok := t.m[accountID]
	if !ok || !now.Before(until) {
		return time.Time{}, false
	}
	return until, true
}

// Clear removes any throttle entry for accountID. Used when an operator
// explicitly accepts a switched-back-to account, or when an account is
// deleted.
func (t *ThrottleStore) Clear(accountID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.m, accountID)
}
