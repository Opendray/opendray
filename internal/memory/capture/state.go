package capture

import (
	"sync"
	"time"
)

// ruleState is the per-rule cursor: how far we've already
// summarised. Phase A keeps it purely in-memory — restart resets
// every cursor to "we've seen 0 messages", which means the next
// tick treats every existing message as new and may produce a
// big initial batch. That's acceptable: dedup search guards
// against duplicate facts being stored, and the first-tick batch
// is only proportional to the most recent N messages we limit
// the History call to.
//
// A future iteration can persist cursors in DB if "fire exactly
// once across restarts" matters more than the simplicity payoff.
type ruleState struct {
	// LastSeenIndex is the index in the most-recent History entry
	// list that we already summarised. -1 = nothing summarised yet.
	LastSeenIndex int
	// LastFiredAt records when we last successfully ran the
	// summarizer for this rule — used by /diagnostics endpoints
	// (Phase C). Phase A doesn't act on it.
	LastFiredAt time.Time
	// FailureStreak counts consecutive provider_unavailable /
	// failed runs. >=3 → engine pauses the rule for 1h via
	// PauseUntil. Resets on success.
	FailureStreak int
	// PauseUntil — when set, engine skips this rule until that
	// timestamp. Used to back off after consecutive failures so a
	// chronically-broken provider doesn't burn cycles every tick.
	PauseUntil time.Time
}

// stateMap is the engine's in-memory cursor table, keyed by
// rule.ID + session.ID composite (each session has its own cursor
// even when sharing a rule).
type stateMap struct {
	mu sync.Mutex
	m  map[string]*ruleState
}

func newStateMap() *stateMap {
	return &stateMap{m: map[string]*ruleState{}}
}

// stateKey isolates per-session cursors under one rule.
func stateKey(ruleID, sessionID string) string {
	return ruleID + "@" + sessionID
}

// Get returns (or initialises) the state for a rule × session pair.
// Init = LastSeenIndex -1 (nothing seen yet).
func (s *stateMap) Get(ruleID, sessionID string) *ruleState {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := stateKey(ruleID, sessionID)
	st, ok := s.m[k]
	if !ok {
		st = &ruleState{LastSeenIndex: -1}
		s.m[k] = st
	}
	return st
}

// MarkFired updates LastSeenIndex + LastFiredAt + resets failure
// streak. Caller passes the index of the LAST message included in
// the summarizer batch.
func (s *stateMap) MarkFired(ruleID, sessionID string, lastIndex int) {
	st := s.Get(ruleID, sessionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	st.LastSeenIndex = lastIndex
	st.LastFiredAt = time.Now().UTC()
	st.FailureStreak = 0
	st.PauseUntil = time.Time{}
}

// MarkFailure increments the streak; pauses for 1h after 3 consecutive
// failures so a broken provider doesn't burn cycles.
func (s *stateMap) MarkFailure(ruleID, sessionID string) {
	st := s.Get(ruleID, sessionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	st.FailureStreak++
	if st.FailureStreak >= 3 {
		st.PauseUntil = time.Now().Add(time.Hour)
	}
}

// IsPaused returns true when the cursor is currently in a backoff
// window. Engine.tick checks this before evaluating triggers.
func (s *stateMap) IsPaused(ruleID, sessionID string) bool {
	st := s.Get(ruleID, sessionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if st.PauseUntil.IsZero() {
		return false
	}
	return time.Now().Before(st.PauseUntil)
}

// Forget removes the cursor for a rule × session — used when the
// rule or session is deleted. Frees memory but, more importantly,
// ensures a re-created rule with the same ID gets a fresh cursor
// rather than inheriting a stale "we already saw 50 messages"
// state.
func (s *stateMap) Forget(ruleID, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, stateKey(ruleID, sessionID))
}
