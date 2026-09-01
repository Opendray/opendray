package grokacct

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// Service is the public surface used by HTTP handlers and the
// SessionProvider adapter. It hides the on-disk GROK_HOME plumbing.
type Service struct {
	log         *slog.Logger
	store       *store
	bus         *eventbus.Hub
	accountsDir string // root for derived ConfigDir; "" → ~/.grok-accounts

	// importMu serializes ImportLocal() so concurrent invocations
	// (startup scan + UI "Import local" click) don't race on the
	// GetByName/Create check-then-insert window.
	importMu sync.Mutex
}

// Option mutates Service defaults.
type Option func(*Service)

// WithAccountsDir overrides the directory used to derive default
// ConfigDir (per-account GROK_HOME) for new accounts. Empty value falls
// back to ~/.grok-accounts.
func WithAccountsDir(dir string) Option {
	return func(s *Service) { s.accountsDir = dir }
}

func NewService(pool *pgxpool.Pool, bus *eventbus.Hub, log *slog.Logger, opts ...Option) *Service {
	if log == nil {
		log = slog.Default()
	}
	s := &Service{
		log:   log.With("component", "grokacct"),
		store: newStore(pool),
		bus:   bus,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// resolveAccountsDir returns the configured root, falling back to
// ~/.grok-accounts when unset. Returns "" only when HOME is also unset
// (test environments must inject WithAccountsDir explicitly).
func (s *Service) resolveAccountsDir() string {
	if s.accountsDir != "" {
		return s.accountsDir
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".grok-accounts")
}

// AccountsDir is the public version of resolveAccountsDir, exposed so the
// App wiring can construct a watcher without reaching into internals.
func (s *Service) AccountsDir() string { return s.resolveAccountsDir() }

// List returns all accounts, decorated with derived fields (TokenFilled
// from the on-disk login token, ActiveSessions/LastUsedAt from a single
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
			Topic: "grok_account.created",
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
			Topic: "grok_account.deleted",
			Data:  map[string]any{"id": id},
		})
	}
	return nil
}

// CheckEnabled is the upstream validator used by session handlers (create
// / switch) so a bogus or disabled id fails with 400 before the session
// row is touched.
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

// ResolveSpawnHome returns the GROK_HOME directory to inject when spawning
// `grok` for account id. The directory must exist and contain a logged-in
// token; otherwise we error with a guided-login hint rather than spawning
// a session that would immediately demand interactive auth. Used at
// session spawn time (catalog adapter); not exposed over HTTP.
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

// ImportLocal scans the accounts dir (and the gateway user's own grok
// home) for logged-in accounts and creates a metadata row for any not yet
// known. Idempotent; safe to call on startup and from the UI.
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
