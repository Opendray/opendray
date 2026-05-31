package cliacct

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
// SessionProvider adapter. It hides the on-disk token plumbing.
type Service struct {
	log         *slog.Logger
	store       *store
	bus         *eventbus.Hub
	accountsDir string // root for default ConfigDir/TokenPath; "" → ~/.claude-accounts

	// importMu serializes ImportLocal() so concurrent invocations
	// (startup scan + fsnotify watcher event + UI "Import local" click)
	// don't race on the GetByName/Create check-then-insert window.
	// Held only for the duration of one scan; UI requests still queue
	// quickly because each scan is O(accounts on disk).
	importMu sync.Mutex
}

// Option mutates Service defaults.
type Option func(*Service)

// WithAccountsDir overrides the directory used to derive default
// ConfigDir / TokenPath for new accounts. Empty value falls back
// to ~/.claude-accounts (the historical hardcoded default).
func WithAccountsDir(dir string) Option {
	return func(s *Service) { s.accountsDir = dir }
}

func NewService(pool *pgxpool.Pool, bus *eventbus.Hub, log *slog.Logger, opts ...Option) *Service {
	if log == nil {
		log = slog.Default()
	}
	s := &Service{
		log:   log.With("component", "cliacct"),
		store: newStore(pool),
		bus:   bus,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// resolveAccountsDir returns the configured root, falling back to
// ~/.claude-accounts when unset. Returns "" only when HOME is also
// unset (test environments must inject WithAccountsDir explicitly).
func (s *Service) resolveAccountsDir() string {
	if s.accountsDir != "" {
		return s.accountsDir
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".claude-accounts")
}

// AccountsDir is the public version of resolveAccountsDir, exposed so
// the cliacct.Watcher (constructed in App.New) can be wired without
// reaching into Service internals.
func (s *Service) AccountsDir() string { return s.resolveAccountsDir() }

// List returns all accounts, with TokenFilled set per account.
func (s *Service) List(ctx context.Context) ([]Account, error) {
	out, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].TokenFilled = tokenFileFilled(out[i].TokenPath)
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, id string) (Account, error) {
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return Account{}, err
	}
	a.TokenFilled = tokenFileFilled(a.TokenPath)
	return a, nil
}

// Create inserts a new account. ConfigDir/TokenPath default to the
// claude-acc convention so manually-created accounts can coexist
// with `claude-acc login --name <name>` runs on the same host.
func (s *Service) Create(ctx context.Context, req CreateRequest) (Account, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Account{}, errors.New("name is required")
	}
	if existing, err := s.store.GetByName(ctx, name); err == nil {
		_ = existing
		return Account{}, ErrDuplicate
	} else if !errors.Is(err, ErrNotFound) {
		return Account{}, err
	}

	accountsDir := s.resolveAccountsDir()
	configDir := strings.TrimSpace(req.ConfigDir)
	if configDir == "" && accountsDir != "" {
		configDir = filepath.Join(accountsDir, name)
	}
	tokenPath := strings.TrimSpace(req.TokenPath)
	if tokenPath == "" && accountsDir != "" {
		tokenPath = filepath.Join(accountsDir, "tokens", name+".token")
	}

	if req.Token != "" {
		if err := writeToken(tokenPath, req.Token); err != nil {
			return Account{}, fmt.Errorf("write token: %w", err)
		}
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	a := Account{
		Name:        name,
		DisplayName: req.DisplayName,
		ConfigDir:   configDir,
		TokenPath:   tokenPath,
		Description: req.Description,
		Enabled:     enabled,
	}
	created, err := s.store.Insert(ctx, a)
	if err != nil {
		return Account{}, err
	}
	created.TokenFilled = tokenFileFilled(created.TokenPath)
	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Topic: "claude_account.created",
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
	if req.TokenPath != nil {
		cur.TokenPath = *req.TokenPath
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
	updated.TokenFilled = tokenFileFilled(updated.TokenPath)
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Topic: "claude_account.deleted",
			Data:  map[string]any{"id": id},
		})
	}
	return nil
}

// SetToken writes/overwrites the token file at TokenPath. The DB row
// is unchanged, but the public Account view will report TokenFilled=true.
func (s *Service) SetToken(ctx context.Context, id, token string) error {
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if a.TokenPath == "" {
		return errors.New("account has no token_path set")
	}
	return writeToken(a.TokenPath, token)
}

// ImportLocal registers an account row for every Claude account found
// on the gateway host that doesn't already have one. It looks in two
// places under the accounts dir (default ~/.claude-accounts):
//
//  1. Per-account CONFIG_DIRs — <accountsDir>/<name>/.credentials.json,
//     produced by the documented `CLAUDE_CONFIG_DIR=<dir> claude login`
//     flow. This is the primary, self-refreshing layout the provider
//     panel instructs operators to use.
//  2. Legacy flat tokens — <accountsDir>/tokens/<name>.token, produced
//     by the older `claude-acc` tool.
//
// A missing directory is not an error (an operator may use only one
// layout, or none yet) — the result is simply empty. Returns the list
// of newly-created accounts.
func (s *Service) ImportLocal(ctx context.Context) ([]Account, error) {
	s.importMu.Lock()
	defer s.importMu.Unlock()

	accountsDir := s.resolveAccountsDir()
	if accountsDir == "" {
		return nil, fmt.Errorf("resolve accounts dir: HOME unset and no accounts_dir configured")
	}

	names, err := discoverLocalAccountNames(accountsDir)
	if err != nil {
		return nil, err
	}

	created := []Account{}
	for _, name := range names {
		if _, err := s.store.GetByName(ctx, name); err == nil {
			continue // already registered
		} else if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		// Best-effort: a single bad entry logs and is skipped rather
		// than failing the whole import.
		acct, err := s.Create(ctx, CreateRequest{Name: name})
		if err != nil {
			s.log.Warn("import-local: create failed", "name", name, "err", err)
			continue
		}
		created = append(created, acct)
	}
	return created, nil
}

// discoverLocalAccountNames returns the unique account names found on
// disk under accountsDir, config-dir layout first then legacy tokens,
// preserving discovery order. Missing directories yield no entries (not
// an error). Pure filesystem — no DB — so it's unit-testable.
func discoverLocalAccountNames(accountsDir string) ([]string, error) {
	var names []string
	seen := map[string]bool{}
	addName := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}

	// 1) Per-account CONFIG_DIRs — <accountsDir>/<name>/.credentials.json
	//    (the documented `CLAUDE_CONFIG_DIR=<dir> claude login` flow).
	dirEntries, err := os.ReadDir(accountsDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", accountsDir, err)
	}
	for _, e := range dirEntries {
		if !e.IsDir() || e.Name() == "tokens" {
			continue
		}
		// Reject symlinked account dirs: a malicious symlink at
		// ~/.claude-accounts/foo → /etc would otherwise let the
		// watcher feed arbitrary paths to selectSpawnCreds.
		// fs.DirEntry.Type() returns the type bits *without* following
		// symlinks, so this is the right check.
		if e.Type()&os.ModeSymlink != 0 {
			continue
		}
		if !fileExists(filepath.Join(accountsDir, e.Name(), ".credentials.json")) {
			continue // not a Claude Code config dir
		}
		addName(e.Name())
	}

	// 2) Legacy <accountsDir>/tokens/*.token (the older claude-acc tool).
	tokensDir := filepath.Join(accountsDir, "tokens")
	tokEntries, err := os.ReadDir(tokensDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", tokensDir, err)
	}
	for _, e := range tokEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".token") {
			continue
		}
		addName(strings.TrimSuffix(e.Name(), ".token"))
	}

	return names, nil
}

func tokenFileFilled(path string) bool {
	if path == "" {
		return false
	}
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return st.Size() > 0
}

// writeToken writes the OAuth token to path with chmod 600. The
// containing dir is created with 0o700 if missing.
func writeToken(path, token string) error {
	if path == "" {
		return errors.New("token path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("mkdir token parent: %w", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimRight(token, "\n")+"\n"), 0o600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}
	return nil
}

// CheckClaudeAccountEnabled implements session.ClaudeAccountChecker —
// the upstream validator used by session handlers (create / switch)
// so a bogus or disabled id fails with 400 before the session row is
// touched. Returns ErrNotFound if the account is missing or ErrDisabled
// if present-but-toggled-off; callers map both to a clean error.
func (s *Service) CheckClaudeAccountEnabled(ctx context.Context, id string) error {
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return err // store wraps to ErrNotFound on missing row
	}
	if !a.Enabled {
		return ErrDisabled
	}
	return nil
}

// ResolveSpawnCreds returns the credentials to inject when spawning a
// process for account id:
//
//   - configDir → CLAUDE_CONFIG_DIR, the account's persistent dir where
//     Claude Code reads and *refreshes* .credentials.json itself.
//   - token → CLAUDE_CODE_OAUTH_TOKEN, a static OAuth token, set ONLY
//     for legacy accounts that carry a token file. For the documented
//     config-dir flow it is intentionally empty: pinning a static token
//     would expire in ~1h, whereas the config dir self-refreshes.
//
// Errors when the account is disabled or has neither a non-empty token
// file nor a config dir containing .credentials.json. Used at session
// spawn time (catalog adapter + memory worker); not exposed over HTTP.
func (s *Service) ResolveSpawnCreds(ctx context.Context, id string) (configDir, token string, err error) {
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return "", "", err
	}
	if !a.Enabled {
		return "", "", ErrDisabled
	}
	return selectSpawnCreds(a.Name, a.ConfigDir, a.TokenPath)
}

// selectSpawnCreds is the pure filesystem half of ResolveSpawnCreds: it
// reads the legacy token file (if any) and validates the config dir's
// credentials, without touching the DB. Returns the static token only
// when a token file is present; config-dir accounts get an empty token
// and rely on CLAUDE_CONFIG_DIR.
func selectSpawnCreds(name, configDir, tokenPath string) (string, string, error) {
	token := ""
	if tokenPath != "" {
		// Lstat first so a symlink at tokenPath doesn't trick us into
		// reading some other file the opendray user can reach. Pair
		// with fileExists() which also rejects symlinks. Defense in
		// depth: a path that survived ImportLocal's symlink check
		// could still be substituted later (delete-rename race), and
		// catching it here means we never spawn with a token sourced
		// from outside the accounts tree.
		if st, err := os.Lstat(tokenPath); err == nil && st.Mode().IsRegular() {
			if body, err := os.ReadFile(tokenPath); err == nil {
				token = strings.TrimSpace(string(body))
			}
		}
	}
	if token == "" {
		if configDir == "" || !fileExists(filepath.Join(configDir, ".credentials.json")) {
			return "", "", fmt.Errorf(
				"account %q has no usable credentials: no token file at %q and no %s/.credentials.json — run `CLAUDE_CONFIG_DIR=%s claude login` on the host",
				name, tokenPath, configDir, configDir)
		}
	}
	return configDir, token, nil
}

// fileExists reports whether path exists and is a regular file. Uses
// Lstat so symlinks (even those pointing at real files) return false —
// callers want to reach exactly the file at `path`, not whatever the
// symlink resolves to. Defense in depth against an attacker who can
// write under the accounts dir.
func fileExists(path string) bool {
	st, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return st.Mode().IsRegular()
}
