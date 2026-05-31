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
// "accept new identity" endpoint — TODO).
//
// We intentionally keep this small: no SQL, no migration. The state
// file lives next to other gateway state under ~/.opendray/.
type identityStore struct {
	path  string
	mu    sync.Mutex
	cache map[string]identityEntry // lazily loaded on first Get/Set
	loaded bool
}

type identityEntry struct {
	Email       string    `json:"email"`
	FirstSeenAt time.Time `json:"first_seen_at"`
}

// newIdentityStore returns an identityStore rooted at the given state
// dir. The file is not touched until Get or Set is called.
func newIdentityStore(stateDir string) *identityStore {
	return &identityStore{
		path:  filepath.Join(stateDir, "cliacct-identity.json"),
		cache: map[string]identityEntry{},
	}
}

// Known returns the previously-recorded email for an account id and
// whether it had ever been recorded. A missing or unreadable state
// file is treated as "nothing recorded yet" so decorate() degrades
// to "no drift" rather than failing the whole list.
func (s *identityStore) Known(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	e, ok := s.cache[id]
	if !ok {
		return "", false
	}
	return e.Email, true
}

// Record stores the email as the known identity for account id. If the
// account already has a recorded email this is a no-op (callers use
// Known() to decide what to record); the first observation wins, so a
// subsequent identity change still shows up as drift.
func (s *identityStore) Record(id, email string) error {
	if id == "" || email == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	if _, ok := s.cache[id]; ok {
		return nil // first observation wins
	}
	s.cache[id] = identityEntry{Email: email, FirstSeenAt: time.Now().UTC()}
	return s.persistLocked()
}

// Forget removes the recorded entry, used by Delete(account) so a
// recreate-then-relogin cycle starts fresh rather than carrying stale
// drift across a deletion.
func (s *identityStore) Forget(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	if _, ok := s.cache[id]; !ok {
		return nil
	}
	delete(s.cache, id)
	return s.persistLocked()
}

// Accept replaces the recorded email with the new one — used by an
// "I know, this swap is intentional" admin action. Returns ErrNotFound
// when the account isn't tracked yet (caller can call Record instead).
func (s *identityStore) Accept(id, email string) error {
	if id == "" || email == "" {
		return errors.New("id and email required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	e, ok := s.cache[id]
	if !ok {
		return ErrNotFound
	}
	e.Email = email
	s.cache[id] = e
	return s.persistLocked()
}

// loadLocked reads the state file into cache. Must be called with s.mu
// held. Idempotent: only reads disk on the first call. A missing file
// is fine (cache starts empty); a corrupt file logs nothing and is
// also treated as empty so a single bad write doesn't kill startup.
func (s *identityStore) loadLocked() {
	if s.loaded {
		return
	}
	s.loaded = true
	body, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var disk map[string]identityEntry
	if err := json.Unmarshal(body, &disk); err != nil {
		return
	}
	s.cache = disk
}

// persistLocked atomically rewrites the state file. Must be called
// with s.mu held. Uses the standard "write to .tmp + os.Rename"
// pattern so a concurrent reader never sees a half-written file.
func (s *identityStore) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(s.cache, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
