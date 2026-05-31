package cliacct

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// identityStore persists a per-account "first-seen" OAuth email so the
// UI can flag identity drift — the case where the operator runs
// `claude login` (no CLAUDE_CONFIG_DIR) and silently replaces the
// default account's underlying identity. The Anthropic-side identity
// is in <configDir>/.claude.json (oauthAccount.emailAddress), but
// nothing in OpenDray remembered it, so a silent swap looked the same
// as "nothing changed."
//
// The store is a single JSON file at <stateDir>/cliacct-identity.json,
// chmod 0600. Schema: {"<account_id>": {"email": "...", "first_seen_at": "..."}}.
// Created on first decorate; never deleted automatically (the operator
// can wipe entries by deleting the file or by hitting the per-account
// "accept new identity" endpoint).
//
// We intentionally keep this small: no SQL, no migration. The state
// file lives next to other gateway state under ~/.opendray/.
//
// Concurrency model: every Known/Record/Accept/Forget call reads the
// file fresh from disk under s.mu, then (for mutating ops) rewrites
// it atomically via tmp+rename. There is intentionally NO in-memory
// cache — the file is small (≤1 KB per account, ≤a few accounts),
// the read cost is negligible, and the lack of caching means
// out-of-band tampering with the state file (e.g. an operator
// editing it directly to reset a baseline) propagates on the next
// API call instead of being shadowed by stale memory.
type identityStore struct {
	path string
	mu   sync.Mutex
}

type identityEntry struct {
	Email       string    `json:"email"`
	FirstSeenAt time.Time `json:"first_seen_at"`
}

// newIdentityStore returns an identityStore rooted at the given state
// dir. The file is not touched until Known/Record/Accept/Forget is
// called.
func newIdentityStore(stateDir string) *identityStore {
	return &identityStore{path: filepath.Join(stateDir, "cliacct-identity.json")}
}

// readLocked returns the current on-disk map. A missing or corrupt
// file yields an empty map (and no error) so a single bad write or
// an out-of-band edit can't crash the whole accounts API. Must be
// called with s.mu held.
func (s *identityStore) readLocked() map[string]identityEntry {
	out := map[string]identityEntry{}
	body, err := os.ReadFile(s.path)
	if err != nil {
		return out
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return map[string]identityEntry{}
	}
	return out
}

// persistLocked atomically rewrites the state file with the given
// snapshot. Must be called with s.mu held. tmp+rename so a concurrent
// reader never sees a half-written file.
func (s *identityStore) persistLocked(snapshot map[string]identityEntry) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Known returns the previously-recorded email for an account id and
// whether it had ever been recorded. Reads fresh from disk on every
// call so an out-of-band edit propagates immediately.
func (s *identityStore) Known(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.readLocked()[id]
	if !ok {
		return "", false
	}
	return e.Email, true
}

// Record stores the email as the known identity for account id. If the
// account already has a recorded email this is a no-op — the first
// observation wins, so a subsequent identity change still shows up
// as drift.
func (s *identityStore) Record(id, email string) error {
	if id == "" || email == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.readLocked()
	if _, ok := cur[id]; ok {
		return nil
	}
	cur[id] = identityEntry{Email: email, FirstSeenAt: time.Now().UTC()}
	return s.persistLocked(cur)
}

// Forget removes the recorded entry, used by Delete(account) so a
// recreate-then-relogin cycle starts fresh rather than carrying stale
// drift across a deletion.
func (s *identityStore) Forget(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.readLocked()
	if _, ok := cur[id]; !ok {
		return nil
	}
	delete(cur, id)
	return s.persistLocked(cur)
}

// Accept replaces the recorded email with the new one — used by the
// operator-visible "I know, this swap is intentional" action. Returns
// ErrNotFound when the account isn't tracked yet (caller can call
// Record instead).
func (s *identityStore) Accept(id, email string) error {
	if id == "" || email == "" {
		return errors.New("id and email required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.readLocked()
	e, ok := cur[id]
	if !ok {
		return ErrNotFound
	}
	e.Email = email
	cur[id] = e
	return s.persistLocked(cur)
}
