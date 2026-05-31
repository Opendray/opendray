package cliacct

import (
	"context"
	"encoding/json"
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

// List returns all accounts, fully decorated with on-the-fly fields:
// TokenFilled (creds present), SubscriptionType/RateLimitTier (from
// .credentials.json), and ActiveSessions/LastUsedAt (from a single
// JOIN against the sessions table). Cheap enough to run on every
// 5s panel poll: two SQL queries + one tiny fs read per account.
func (s *Service) List(ctx context.Context) ([]Account, error) {
	out, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	stats, err := s.store.sessionLoad(ctx)
	if err != nil {
		// Don't fail the whole list on a stats hiccup — the operator
		// still wants to see the accounts. Log and degrade gracefully.
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
// Centralized so List/Get/Create/Update stay in sync — there's only
// one place to remember to add a new computed field.
func (s *Service) decorate(a *Account, stats sessionStats) {
	a.TokenFilled = accountHasCredentials(a.ConfigDir, a.TokenPath)
	if sub, tier := readCredentialsMeta(a.ConfigDir); sub != "" || tier != "" {
		a.SubscriptionType = sub
		a.RateLimitTier = tier
	}
	a.ActiveSessions = stats.ActiveSessions
	a.LastUsedAt = stats.LastUsedAt
}

// readCredentialsMeta pulls (subscriptionType, rateLimitTier) out of
// the account's <configDir>/.credentials.json claudeAiOauth block.
// Returns empty strings for either field that is missing; never
// errors (a malformed file just means we can't show that signal).
// Tokens and any other sensitive fields are NOT returned by this
// helper — the caller can't accidentally leak them through Account.
func readCredentialsMeta(configDir string) (subscriptionType, rateLimitTier string) {
	if configDir == "" {
		return "", ""
	}
	p := filepath.Join(configDir, ".credentials.json")
	// Lstat first so a symlinked credentials file doesn't get us to
	// open and parse some other file — matches the safety pattern in
	// selectSpawnCreds.
	st, err := os.Lstat(p)
	if err != nil || !st.Mode().IsRegular() {
		return "", ""
	}
	body, err := os.ReadFile(p)
	if err != nil {
		return "", ""
	}
	// The file is small (a few KB at most). Parse only the bit we need.
	var doc struct {
		ClaudeAiOauth struct {
			SubscriptionType string `json:"subscriptionType"`
			RateLimitTier    string `json:"rateLimitTier"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", ""
	}
	return doc.ClaudeAiOauth.SubscriptionType, doc.ClaudeAiOauth.RateLimitTier
}

// PickAutoAssignClaudeAccount returns the id of the enabled account
// with the fewest active sessions (lexical-name tiebreaker). Used by
// the session handler when POST /sessions sends provider=claude with
// no claude_account_id pinned and ≥2 enabled accounts exist. Returns
// "" + nil when there are <2 enabled accounts (no point picking).
func (s *Service) PickAutoAssignClaudeAccount(ctx context.Context) (string, error) {
	// Count enabled accounts first; with 0 or 1 the assignment is
	// either invalid or trivially the only choice, so we let the
	// caller's fallback (empty id → CLI default) handle it.
	rows, err := s.store.List(ctx)
	if err != nil {
		return "", err
	}
	enabled := 0
	for _, a := range rows {
		if a.Enabled {
			enabled++
		}
	}
	if enabled < 2 {
		return "", nil
	}
	id, err := s.store.pickLeastLoaded(ctx)
	if errors.Is(err, ErrNotFound) {
		return "", nil // none enabled (shouldn't happen given count above, but be safe)
	}
	return id, err
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
	s.decorate(&created, sessionStats{}) // brand-new row → no sessions yet
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

	discovered, err := discoverLocalAccounts(accountsDir)
	if err != nil {
		return nil, err
	}

	created := []Account{}
	for _, d := range discovered {
		if _, err := s.store.GetByName(ctx, d.name); err == nil {
			continue // already registered
		} else if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		// Best-effort: a single bad entry logs and is skipped rather
		// than failing the whole import. The d.configDir / d.tokenPath
		// are passed explicitly so the virtual "default" entry
		// (pointing at ~/.claude rather than ~/.claude-accounts/default)
		// is honored; for named accounts both are empty and Create()
		// derives the standard claude-acc layout from the name.
		req := CreateRequest{
			Name:        d.name,
			DisplayName: d.displayName,
			ConfigDir:   d.configDir,
			TokenPath:   d.tokenPath,
		}
		acct, err := s.Create(ctx, req)
		if err != nil {
			s.log.Warn("import-local: create failed", "name", d.name, "err", err)
			continue
		}
		created = append(created, acct)
	}
	return created, nil
}

// discoveredAccount carries the minimum metadata needed to register
// one account row found on disk. Named accounts under accountsDir
// leave configDir/tokenPath empty so Create() applies its standard
// derivation; the synthetic "default" entry that surfaces the Claude
// CLI's own ~/.claude/ home sets configDir explicitly because its
// path does NOT match the accountsDir/<name> layout.
type discoveredAccount struct {
	name        string
	displayName string
	configDir   string // explicit when non-empty; otherwise Create derives
	tokenPath   string // same semantics
}

// discoverLocalAccounts returns every account that should be surfaced
// in the Claude Accounts panel, in discovery order. Three sources:
//
//  1. ~/.claude/.credentials.json — the Claude CLI's default config
//     dir. We yield it as a synthetic entry named "default" so the
//     operator can see the primary account (the one used when no
//     claude_account_id is set on a session) the same way they see
//     named accounts. This is the gap that 'info@paygear.io is
//     authenticated, why isn't it in the panel?' was hitting.
//  2. <accountsDir>/<name>/.credentials.json — the documented
//     `CLAUDE_CONFIG_DIR=<dir> claude login` flow (config-dir layout).
//  3. <accountsDir>/tokens/<name>.token — legacy `claude-acc` flow.
//
// Symlinks are rejected at every step. A missing dir is not an error
// (fresh installs yield zero entries).
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

	// 1) Synthetic "default" — the Claude CLI's own home. Only emitted
	//    when ~/.claude/.credentials.json actually exists, so a fresh
	//    install (no `claude login` ever run) doesn't surface a dead
	//    row. The path is rooted at HOME/.claude, not accountsDir, so
	//    we pass it explicitly so Create() does not derive a wrong
	//    accountsDir/default path.
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		defaultDir := filepath.Join(home, ".claude")
		if fileExists(filepath.Join(defaultDir, ".credentials.json")) {
			emit(discoveredAccount{
				name:        "default",
				displayName: "Default (~/.claude)",
				configDir:   defaultDir,
				// no tokenPath: the .credentials.json file self-refreshes
				// and is read by Claude Code via CLAUDE_CONFIG_DIR.
			})
		}
	}

	// 2) Per-account CONFIG_DIRs — <accountsDir>/<name>/.credentials.json
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
		emit(discoveredAccount{name: e.Name()})
	}

	// 3) Legacy <accountsDir>/tokens/*.token (the older claude-acc tool).
	tokensDir := filepath.Join(accountsDir, "tokens")
	tokEntries, err := os.ReadDir(tokensDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", tokensDir, err)
	}
	for _, e := range tokEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".token") {
			continue
		}
		emit(discoveredAccount{name: strings.TrimSuffix(e.Name(), ".token")})
	}

	return out, nil
}

// accountHasCredentials reports whether an account has usable
// credentials on disk via EITHER form opendray supports:
//
//   - a legacy <accountsDir>/tokens/<name>.token file (the
//     `claude-acc` flow; static OAuth token, expires ~1h)
//   - a config-dir <configDir>/.credentials.json file (the
//     documented `CLAUDE_CONFIG_DIR=… claude login` flow; Claude
//     Code self-refreshes)
//
// The public Account.TokenFilled JSON field reflects this — the name
// is historical (predates the config-dir flow); semantically it now
// means "has usable creds." The UI uses it to decide whether to show
// the "no token yet" badge, so config-dir accounts (which have no
// legacy token file but do have a working .credentials.json) must
// also return true here. Uses Lstat for the same symlink-safety
// reasons as fileExists.
func accountHasCredentials(configDir, tokenPath string) bool {
	if tokenPath != "" {
		if st, err := os.Lstat(tokenPath); err == nil && st.Mode().IsRegular() && st.Size() > 0 {
			return true
		}
	}
	if configDir != "" {
		if fileExists(filepath.Join(configDir, ".credentials.json")) {
			return true
		}
	}
	return false
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
