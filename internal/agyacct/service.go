package agyacct

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// Service is the public surface used by HTTP handlers and the
// SessionProvider adapter. It hides the on-disk HOME plumbing.
type Service struct {
	log         *slog.Logger
	store       *store
	bus         *eventbus.Hub
	accountsDir string // root for derived ConfigDir; "" → ~/.antigravity-accounts

	// importMu serializes ImportLocal() so concurrent invocations
	// (startup scan + UI "Import local" click) don't race on the
	// GetByName/Create check-then-insert window.
	importMu sync.Mutex
}

// Option mutates Service defaults.
type Option func(*Service)

// WithAccountsDir overrides the directory used to derive default
// ConfigDir (per-account HOME) for new accounts. Empty value falls back
// to ~/.antigravity-accounts.
func WithAccountsDir(dir string) Option {
	return func(s *Service) { s.accountsDir = dir }
}

func NewService(pool *pgxpool.Pool, bus *eventbus.Hub, log *slog.Logger, opts ...Option) *Service {
	if log == nil {
		log = slog.Default()
	}
	s := &Service{
		log:   log.With("component", "agyacct"),
		store: newStore(pool),
		bus:   bus,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// resolveAccountsDir returns the configured root, falling back to
// ~/.antigravity-accounts when unset. Returns "" only when HOME is also
// unset (test environments must inject WithAccountsDir explicitly).
func (s *Service) resolveAccountsDir() string {
	if s.accountsDir != "" {
		return s.accountsDir
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".antigravity-accounts")
}

// AccountsDir is the public version of resolveAccountsDir, exposed so the
// App wiring can construct a watcher without reaching into internals.
func (s *Service) AccountsDir() string { return s.resolveAccountsDir() }

// List returns all accounts, decorated with derived fields (TokenFilled
// from the on-disk OAuth token, ActiveSessions/LastUsedAt from a single
// JOIN against sessions).
func (s *Service) List(ctx context.Context) ([]Account, error) {
	out, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	stats, err := s.store.sessionLoad(ctx)
	if err != nil {
		s.log.Warn("session-load failed; account list will lack usage signal", "err", err)
		stats = map[string]sessionStats{}
	}
	for i := range out {
		s.decorate(&out[i], stats[out[i].ID])
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, id string) (Account, error) {
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return Account{}, err
	}
	stats, _ := s.store.sessionLoad(ctx) // best-effort
	s.decorate(&a, stats[a.ID])
	return a, nil
}

// decorate fills in all derived fields on an Account in place.
func (s *Service) decorate(a *Account, stats sessionStats) {
	a.TokenFilled = accountHasCredentials(a.ConfigDir)
	a.ActiveSessions = stats.ActiveSessions
	a.LastUsedAt = stats.LastUsedAt
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Account, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Account{}, errors.New("name is required")
	}
	if _, err := s.store.GetByName(ctx, name); err == nil {
		return Account{}, ErrDuplicate
	} else if !errors.Is(err, ErrNotFound) {
		return Account{}, err
	}

	accountsDir := s.resolveAccountsDir()
	configDir := strings.TrimSpace(req.ConfigDir)
	if configDir == "" && accountsDir != "" {
		configDir = filepath.Join(accountsDir, name)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	a := Account{
		Name:        name,
		DisplayName: req.DisplayName,
		ConfigDir:   configDir,
		Description: req.Description,
		Enabled:     enabled,
	}
	created, err := s.store.Insert(ctx, a)
	if err != nil {
		return Account{}, err
	}
	s.decorate(&created, sessionStats{}) // brand-new row → no sessions yet
	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Topic: "antigravity_account.created",
			Data:  map[string]any{"id": created.ID, "name": created.Name},
		})
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Account, error) {
	cur, err := s.store.Get(ctx, id)
	if err != nil {
		return Account{}, err
	}
	if req.Name != nil {
		cur.Name = strings.TrimSpace(*req.Name)
	}
	if req.DisplayName != nil {
		cur.DisplayName = *req.DisplayName
	}
	if req.ConfigDir != nil {
		cur.ConfigDir = *req.ConfigDir
	}
	if req.Description != nil {
		cur.Description = *req.Description
	}
	if req.Enabled != nil {
		cur.Enabled = *req.Enabled
	}
	updated, err := s.store.Update(ctx, cur)
	if err != nil {
		return Account{}, err
	}
	stats, _ := s.store.sessionLoad(ctx) // best-effort
	s.decorate(&updated, stats[updated.ID])
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Topic: "antigravity_account.deleted",
			Data:  map[string]any{"id": id},
		})
	}
	return nil
}

// CheckEnabled is the upstream validator used by session handlers
// (create / switch) so a bogus or disabled id fails with 400 before the
// session row is touched.
func (s *Service) CheckEnabled(ctx context.Context, id string) error {
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return err // store wraps to ErrNotFound on missing row
	}
	if !a.Enabled {
		return ErrDisabled
	}
	return nil
}

// ResolveSpawnHome returns the HOME directory to inject when spawning
// `agy` for account id. The directory must exist and contain a logged-in
// OAuth token; otherwise we error with a guided-login hint rather than
// spawning a session that would immediately demand interactive auth.
// Used at session spawn time (catalog adapter); not exposed over HTTP.
func (s *Service) ResolveSpawnHome(ctx context.Context, id string) (string, error) {
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if !a.Enabled {
		return "", ErrDisabled
	}
	return selectSpawnHome(a.Name, a.ConfigDir)
}

// selectSpawnHome is the pure-filesystem half of ResolveSpawnHome: given an
// account's name + HOME dir it validates the dir is set and holds a logged-in
// agy OAuth token, returning the HOME to inject or a guided-login error. No
// DB access, so it's unit-testable without a store (mirrors cliacct's
// selectSpawnCreds).
func selectSpawnHome(name, home string) (string, error) {
	if home == "" {
		return "", fmt.Errorf("antigravity account %q has no HOME directory configured", name)
	}
	if !accountHasCredentials(home) {
		return "", fmt.Errorf(
			"antigravity account %q is not logged in: no %s under %s — run `HOME=%s agy` on the host and complete the Google sign-in",
			name, agyTokenRelPath, home, home)
	}
	return home, nil
}

// discoveredAccount is one local account candidate found on disk.
type discoveredAccount struct {
	name        string
	displayName string
	configDir   string // explicit when non-empty; otherwise Create derives
}

// discoverLocalAccounts returns every antigravity account that should be
// surfaced in the panel, in discovery order. Two sources:
//
//  1. ~/ (the gateway user's real HOME) — the primary `agy` login, used
//     when no antigravity_account_id is pinned to a session. Yielded as
//     a synthetic "default" entry so the operator sees it like the
//     named accounts. Only emitted when its OAuth token actually exists.
//  2. <accountsDir>/<name>/ — a per-account HOME created via the guided
//     `HOME=<dir> agy` login. Only dirs that already contain the OAuth
//     token are emitted (a half-set-up dir isn't a usable account yet).
//
// Symlinks are rejected at every step; a missing dir is not an error.
func discoverLocalAccounts(accountsDir string) ([]discoveredAccount, error) {
	var out []discoveredAccount
	seen := map[string]bool{}
	emit := func(d discoveredAccount) {
		if d.name == "" || seen[d.name] {
			return
		}
		seen[d.name] = true
		out = append(out, d)
	}

	// 1) Synthetic "default" — the gateway user's own HOME.
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if accountHasCredentials(home) {
			emit(discoveredAccount{
				name:        "default",
				displayName: "Default (~)",
				configDir:   home,
			})
		}
	}

	// 2) Per-account HOMEs under accountsDir.
	dirEntries, err := os.ReadDir(accountsDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", accountsDir, err)
	}
	for _, e := range dirEntries {
		if !e.IsDir() {
			continue
		}
		// Reject symlinked account dirs so a malicious symlink can't
		// feed an arbitrary HOME to a spawn.
		if e.Type()&os.ModeSymlink != 0 {
			continue
		}
		if !accountHasCredentials(filepath.Join(accountsDir, e.Name())) {
			continue // not a logged-in agy HOME
		}
		emit(discoveredAccount{name: e.Name()})
	}

	return out, nil
}

// ImportLocal scans the gateway host for antigravity account HOMEs and
// creates a row for each one not already known. Idempotent; returns the
// accounts it created this run.
func (s *Service) ImportLocal(ctx context.Context) ([]Account, error) {
	s.importMu.Lock()
	defer s.importMu.Unlock()

	discovered, err := discoverLocalAccounts(s.resolveAccountsDir())
	if err != nil {
		return nil, err
	}
	var created []Account
	for _, d := range discovered {
		if _, err := s.store.GetByName(ctx, d.name); err == nil {
			continue // already known
		} else if !errors.Is(err, ErrNotFound) {
			return created, err
		}
		acc, err := s.Create(ctx, CreateRequest{
			Name:        d.name,
			DisplayName: d.displayName,
			ConfigDir:   d.configDir,
		})
		if err != nil {
			if errors.Is(err, ErrDuplicate) {
				continue
			}
			return created, err
		}
		created = append(created, acc)
	}
	return created, nil
}

// accountHasCredentials reports whether the account HOME contains a
// logged-in agy OAuth token. Uses Lstat so a symlinked token (pointing
// outside the account tree) is rejected.
func accountHasCredentials(home string) bool {
	if home == "" {
		return false
	}
	return fileExists(filepath.Join(home, agyTokenRelPath))
}

// fileExists reports whether path exists and is a regular file. Uses
// Lstat so symlinks return false — defense in depth against an attacker
// who can write under the accounts dir.
func fileExists(path string) bool {
	st, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return st.Mode().IsRegular() && st.Size() > 0
}
