package opencodeacct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// FieldCipher wraps a short secret at rest — the live backup cipher
// satisfies it (same shape dbtool / channel / summarizer consume).
// Declared here, where it is consumed.
type FieldCipher interface {
	EncryptField(plain string) (string, error)
	DecryptField(envelope string) (string, error)
}

// Service is the public surface used by HTTP handlers, the session
// handler (validation) and the catalog adapter (spawn-time injection).
type Service struct {
	log    *slog.Logger
	repo   repo
	bus    *eventbus.Hub
	cipher FieldCipher

	// localAuthPath overrides where ImportLocal reads opencode's on-disk
	// auth.json; "" → localAuthPath().
	localAuth string

	importMu sync.Mutex
}

// Option mutates Service defaults.
type Option func(*Service)

// WithLocalAuthPath overrides the on-disk auth.json ImportLocal reads.
func WithLocalAuthPath(p string) Option { return func(s *Service) { s.localAuth = p } }

// NewService builds the service. The at-rest cipher is attached later via
// SetCipher (it's backed by the live backup subsystem, wired after the
// account services); until then credential writes are refused.
func NewService(pool *pgxpool.Pool, bus *eventbus.Hub, log *slog.Logger, opts ...Option) *Service {
	return newService(newStore(pool), bus, nil, log, opts...)
}

// SetCipher installs the at-rest cipher for credential bundles. Called
// once during startup, before routes are served.
func (s *Service) SetCipher(c FieldCipher) { s.cipher = c }

func newService(r repo, bus *eventbus.Hub, cipher FieldCipher, log *slog.Logger, opts ...Option) *Service {
	if log == nil {
		log = slog.Default()
	}
	s := &Service{log: log.With("component", "opencodeacct"), repo: r, bus: bus, cipher: cipher}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// List returns all accounts with masked credential summaries and usage.
func (s *Service) List(ctx context.Context) ([]Account, error) {
	out, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	stats, err := s.repo.sessionLoad(ctx)
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
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		return Account{}, err
	}
	stats, _ := s.repo.sessionLoad(ctx) // best-effort
	s.decorate(&a, stats[a.ID])
	return a, nil
}

// decorate fills derived, display-safe fields. The decrypted bundle is
// used only to build masked summaries and is dropped immediately.
func (s *Service) decorate(a *Account, stats sessionStats) {
	a.ActiveSessions = stats.ActiveSessions
	a.LastUsedAt = stats.LastUsedAt
	a.Credentials = []CredentialSummary{}
	if b, err := s.decryptBundle(a.credentialsEnc); err == nil && len(b) > 0 {
		a.Credentials = summarize(b)
		a.TokenFilled = true
	}
}

// encryptBundle serializes + encrypts a bundle. Refuses to store
// plaintext: no cipher, or an unarmed one, yields ErrCipherRequired.
func (s *Service) encryptBundle(b map[string]any) (string, error) {
	if s.cipher == nil {
		return "", ErrCipherRequired
	}
	raw, err := json.Marshal(b)
	if err != nil {
		return "", fmt.Errorf("marshal credentials: %w", err)
	}
	enc, err := s.cipher.EncryptField(string(raw))
	if err != nil || enc == "" {
		return "", ErrCipherRequired
	}
	return enc, nil
}

func (s *Service) decryptBundle(enc string) (map[string]any, error) {
	if enc == "" {
		return nil, ErrNoCredentials
	}
	if s.cipher == nil {
		return nil, ErrCipherRequired
	}
	plain, err := s.cipher.DecryptField(enc)
	if err != nil {
		return nil, ErrCipherRequired
	}
	var b map[string]any
	if err := json.Unmarshal([]byte(plain), &b); err != nil {
		return nil, ErrNoCredentials
	}
	return b, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Account, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Account{}, errors.New("name is required")
	}
	bundle, err := buildBundle(req.Credentials, req.ProviderID, req.APIKey)
	if err != nil {
		return Account{}, err
	}
	if bundle == nil {
		return Account{}, fmt.Errorf("%w: credentials are required", ErrInvalidCredentials)
	}
	if _, err := s.repo.GetByName(ctx, name); err == nil {
		return Account{}, ErrDuplicate
	} else if !errors.Is(err, ErrNotFound) {
		return Account{}, err
	}
	enc, err := s.encryptBundle(bundle)
	if err != nil {
		return Account{}, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	created, err := s.repo.Insert(ctx, Account{
		Name:           name,
		DisplayName:    req.DisplayName,
		Description:    req.Description,
		Enabled:        enabled,
		Providers:      providerIDs(bundle),
		credentialsEnc: enc,
	})
	if err != nil {
		return Account{}, err
	}
	s.decorate(&created, sessionStats{})
	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Topic: "opencode_account.created",
			Data:  map[string]any{"id": created.ID, "name": created.Name},
		})
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Account, error) {
	cur, err := s.repo.Get(ctx, id)
	if err != nil {
		return Account{}, err
	}
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if n == "" {
			return Account{}, errors.New("name is required")
		}
		cur.Name = n
	}
	if req.DisplayName != nil {
		cur.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		cur.Description = *req.Description
	}
	if req.Enabled != nil {
		cur.Enabled = *req.Enabled
	}
	bundle, err := buildBundle(req.Credentials, req.ProviderID, req.APIKey)
	if err != nil {
		return Account{}, err
	}
	if bundle != nil {
		enc, err := s.encryptBundle(bundle)
		if err != nil {
			return Account{}, err
		}
		cur.credentialsEnc = enc
		cur.Providers = providerIDs(bundle)
	}
	updated, err := s.repo.Update(ctx, cur)
	if err != nil {
		return Account{}, err
	}
	stats, _ := s.repo.sessionLoad(ctx) // best-effort
	s.decorate(&updated, stats[updated.ID])
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(eventbus.Event{
			Topic: "opencode_account.deleted",
			Data:  map[string]any{"id": id},
		})
	}
	return nil
}

// CheckEnabled returns nil when id is an existing, enabled account.
func (s *Service) CheckEnabled(ctx context.Context, id string) error {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if !a.Enabled {
		return ErrDisabled
	}
	return nil
}

// CheckUsable is stricter: existing, enabled AND its bundle decrypts to
// at least one credential. Guards a switch so a working session is never
// stopped to respawn under an unusable account.
func (s *Service) CheckUsable(ctx context.Context, id string) error {
	_, err := s.ResolveSpawnAuth(ctx, id)
	return err
}

// ResolveSpawnAuth returns the OPENCODE_AUTH_CONTENT value (the decrypted
// auth.json-shaped bundle) for account id. Spawn-time only; never exposed
// over HTTP and never logged.
func (s *Service) ResolveSpawnAuth(ctx context.Context, id string) (string, error) {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if !a.Enabled {
		return "", ErrDisabled
	}
	b, err := s.decryptBundle(a.credentialsEnc)
	if err != nil {
		return "", err
	}
	if len(b) == 0 {
		return "", ErrNoCredentials
	}
	raw, err := json.Marshal(b)
	if err != nil {
		return "", fmt.Errorf("marshal credentials: %w", err)
	}
	return string(raw), nil
}

// ImportLocal registers the gateway user's current on-disk opencode
// credential set (auth.json) as an account named req.Name ("local" by
// default). Returns ErrDuplicate when that name is already taken, so a
// repeat click is harmless.
func (s *Service) ImportLocal(ctx context.Context, req ImportLocalRequest) (Account, error) {
	s.importMu.Lock()
	defer s.importMu.Unlock()

	path := s.localAuth
	if path == "" {
		path = localAuthPath()
	}
	if path == "" {
		return Account{}, errors.New("cannot locate opencode auth.json (HOME unset)")
	}
	bundle, err := readBundleFile(path)
	if err != nil {
		return Account{}, fmt.Errorf("read opencode auth.json: %w", err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "local"
	}
	return s.Create(ctx, CreateRequest{
		Name:        name,
		Description: "Imported from " + path,
		Credentials: bundle,
	})
}
