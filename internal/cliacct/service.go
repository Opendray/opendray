package cliacct

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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

// ImportLocal scans ~/.claude-accounts/tokens/*.token on the gateway
// host and registers a row for any token file that doesn't already
// have one. Returns the list of newly-created accounts.
//
// Mirrors v1's behavior: zero-arg adoption flow for operators who set
// up `claude-acc` before plugging into the gateway.
func (s *Service) ImportLocal(ctx context.Context) ([]Account, error) {
	accountsDir := s.resolveAccountsDir()
	if accountsDir == "" {
		return nil, fmt.Errorf("resolve accounts dir: HOME unset and no accounts_dir configured")
	}
	tokensDir := filepath.Join(accountsDir, "tokens")
	entries, err := os.ReadDir(tokensDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no %s — run `claude-acc init` on the host first", tokensDir)
		}
		return nil, fmt.Errorf("read %s: %w", tokensDir, err)
	}

	created := []Account{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".token") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".token")
		if _, err := s.store.GetByName(ctx, name); err == nil {
			continue
		} else if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		acct, err := s.Create(ctx, CreateRequest{
			Name: name,
			// Description left blank so the operator can rename
			// later without us guessing intent.
		})
		if err != nil {
			s.log.Warn("import-local: create failed", "name", name, "err", err)
			continue
		}
		created = append(created, acct)
	}
	return created, nil
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

// ReadToken loads the OAuth token for an account. Used at session
// spawn time by SessionProvider; not exposed over HTTP.
func (s *Service) ReadToken(ctx context.Context, id string) (Account, string, error) {
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return Account{}, "", err
	}
	if !a.Enabled {
		return a, "", ErrDisabled
	}
	if a.TokenPath == "" {
		return a, "", errors.New("account has no token_path")
	}
	body, err := os.ReadFile(a.TokenPath)
	if err != nil {
		return a, "", fmt.Errorf("read %s: %w", a.TokenPath, err)
	}
	tok := strings.TrimSpace(string(body))
	if tok == "" {
		return a, "", fmt.Errorf("token file %s is empty", a.TokenPath)
	}
	return a, tok, nil
}
