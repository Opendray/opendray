package install

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/opendray/opendray/plugin"
)

// PendingInstall is a staged install awaiting user consent. Token is a
// 32-byte random hex string (64 chars). The Installer keeps exactly one
// pending entry per token; confirming consumes + deletes the entry.
type PendingInstall struct {
	Token        string
	Name         string
	Version      string
	ManifestHash string
	Perms        plugin.PermissionsV1
	StagedPath   string
	ExpiresAt    time.Time
}

// pendingStore is the in-memory token table. It is intentionally
// unexported because the Installer is the only sanctioned consumer —
// external packages talk to the Installer, never directly to the store.
// The janitor reuses newPendingStore for its unit tests.
type pendingStore struct {
	mu      sync.Mutex
	byToken map[string]*PendingInstall
	ttl     time.Duration
	now     func() time.Time // injectable for tests
}

// newPendingStore constructs a pendingStore. now may be nil — it falls
// back to time.Now. ttl may be zero; the caller (Installer) is expected
// to provide a sensible default.
func newPendingStore(ttl time.Duration, now func() time.Time) *pendingStore {
	if now == nil {
		now = time.Now
	}
	return &pendingStore{
		byToken: make(map[string]*PendingInstall),
		ttl:     ttl,
		now:     now,
	}
}

// put records p under p.Token. Overwrites any existing entry with the
// same token (caller's responsibility to generate unique tokens — see
// newToken).
func (s *pendingStore) put(p *PendingInstall) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byToken[p.Token] = p
}

// take atomically fetches-and-deletes the pending entry for token. Returns
// (entry, true) on hit; (nil, false) if the token is missing OR expired.
// Expired entries are also deleted as a side-effect so the caller's
// subsequent reap call won't double-count them.
func (s *pendingStore) take(token string) (*PendingInstall, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.byToken[token]
	if !ok {
		return nil, false
	}
	delete(s.byToken, token)
	if s.now().After(p.ExpiresAt) {
		return nil, false
	}
	return p, true
}

// reap drops every expired entry and returns the list of staged paths
// whose owners were reaped. The caller (Installer.janitor) is responsible
// for os.RemoveAll on each path — keeping the filesystem effect outside
// the mutex avoids long-held locks under heavy reap load.
func (s *pendingStore) reap() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	var toClean []string
	for tk, p := range s.byToken {
		if now.After(p.ExpiresAt) {
			toClean = append(toClean, p.StagedPath)
			delete(s.byToken, tk)
		}
	}
	return toClean
}

// count returns the current number of pending entries. Test-only helper.
func (s *pendingStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.byToken)
}

// newToken generates a 32-byte hex-encoded random string (64 chars).
// Uses crypto/rand; panics on failure because the platform's entropy
// source is considered critical for security — we would rather crash
// than issue a predictable token.
func newToken() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		// A crypto/rand failure is catastrophic; surfacing it as a panic
		// ensures the installer is never tricked into using a fallback.
		panic("install: crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}
