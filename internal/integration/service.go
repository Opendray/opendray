package integration

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// Service is the integration registry's primary facade. Construct via
// NewService and pass into the combined auth middleware + handlers.
type Service struct {
	log   *slog.Logger
	store *store
	bus   *eventbus.Hub

	tokenMu    sync.RWMutex
	tokenCache map[string]string // plaintext token → integration ID
}

func NewService(pool *pgxpool.Pool, bus *eventbus.Hub, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		log:        log.With("component", "integration"),
		store:      newStore(pool),
		bus:        bus,
		tokenCache: make(map[string]string),
	}
}

// RegisterRequest is the body for POST /integrations.
//
// IsSystem is internal-only — never honoured for requests coming
// from the admin HTTP handler (validateRegister rejects it). Set
// it from inside the process when bootstrapping opendray-managed
// integrations.
type RegisterRequest struct {
	Name        string   `json:"name"`
	BaseURL     string   `json:"base_url"`
	RoutePrefix string   `json:"route_prefix"`
	Scopes      []string `json:"scopes,omitempty"`
	Version     string   `json:"version,omitempty"`
	IsSystem    bool     `json:"-"`
	// MemoryPolicy declares how the capture pipeline treats sessions
	// this integration creates (none|quarantine|full). Empty defaults
	// to quarantine — safe by default.
	MemoryPolicy MemoryPolicy `json:"memory_policy,omitempty"`

	// DefaultProviderID / DefaultModel / DefaultClaudeAccountID seed the
	// spawn defaults for sessions this integration creates. All optional;
	// empty means "no default". See Integration for semantics.
	DefaultProviderID      string `json:"default_provider_id,omitempty"`
	DefaultModel           string `json:"default_model,omitempty"`
	DefaultClaudeAccountID string `json:"default_claude_account_id,omitempty"`

	// MCPServers / SystemPrompt / PermissionMode seed the provider-agnostic
	// spawn profile for sessions this integration creates. All optional;
	// empty means "no injection" / "default". See Integration for semantics.
	MCPServers     json.RawMessage `json:"mcp_servers,omitempty"`
	SystemPrompt   string          `json:"system_prompt,omitempty"`
	PermissionMode PermissionMode  `json:"permission_mode,omitempty"`
	// AgentID is the reserved forward-compat slot (see Integration.AgentID).
	AgentID string `json:"agent_id,omitempty"`
}

// RegisterResult bundles the persisted integration with the one-time
// plaintext API key. Plaintext is never stored.
type RegisterResult struct {
	Integration Integration `json:"integration"`
	APIKey      string      `json:"api_key"`
}

// defaultScopes from design §8.6 — read-only session view + session.*
// event subscription. Admin can widen at registration time.
var defaultScopes = []string{"session:read", "event:subscribe:session.*"}

// Register provisions a new integration row + a one-time API key.
//
// Consumer-only integrations (empty base_url + empty route_prefix)
// are stored with a synthesized internal prefix derived from the
// new ID — needed because the DB has UNIQUE NOT NULL on
// route_prefix. This synthetic value never appears in JSON
// responses (we re-blank it before serialising) so the UI sees a
// clean "no proxy" state.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (RegisterResult, error) {
	if err := validateRegister(req); err != nil {
		return RegisterResult{}, err
	}
	consumerOnly := req.BaseURL == "" && req.RoutePrefix == ""
	if !consumerOnly {
		if isReservedPrefix(req.RoutePrefix) {
			return RegisterResult{}, ErrReservedPrefix
		}
		if _, err := s.store.GetByPrefix(ctx, req.RoutePrefix); err == nil {
			return RegisterResult{}, ErrPrefixTaken
		} else if !errors.Is(err, ErrNotFound) {
			return RegisterResult{}, err
		}
	}

	token, hash, err := generateAPIKey()
	if err != nil {
		return RegisterResult{}, err
	}

	id := newID()
	storedPrefix := req.RoutePrefix
	if consumerOnly {
		// Synthesize a non-collidable prefix; the consumer-only
		// status is tracked by `BaseURL == ""` everywhere else.
		storedPrefix = "_consumer_" + id
	}
	policy := req.MemoryPolicy
	if policy == "" {
		// Integration-created sessions are isolated by default: their
		// transcripts are never captured into the memory store. Third-party
		// consumers may serve many end users, so their session content is
		// private by default. An operator can opt a trusted integration into
		// quarantine/full explicitly.
		policy = MemoryPolicyNone
	}
	i := Integration{
		ID:           id,
		Name:         req.Name,
		BaseURL:      strings.TrimRight(req.BaseURL, "/"),
		RoutePrefix:  storedPrefix,
		Scopes:       req.Scopes,
		Version:      req.Version,
		Enabled:      true,
		HealthStatus: HealthUnknown,
		CreatedAt:    time.Now().UTC(),
		IsSystem:     req.IsSystem,
		MemoryPolicy: policy,

		DefaultProviderID:      req.DefaultProviderID,
		DefaultModel:           req.DefaultModel,
		DefaultClaudeAccountID: req.DefaultClaudeAccountID,

		MCPServers:     req.MCPServers,
		SystemPrompt:   req.SystemPrompt,
		PermissionMode: NormalizePermissionMode(req.PermissionMode),
		AgentID:        req.AgentID,

		apiKeyHash: hash,
	}
	if len(i.Scopes) == 0 {
		i.Scopes = append([]string{}, defaultScopes...)
	}

	if err := s.store.Insert(ctx, i); err != nil {
		if isUniqueViolation(err) {
			return RegisterResult{}, ErrNameTaken // either name or prefix collision
		}
		return RegisterResult{}, err
	}

	s.bus.Publish(eventbus.Event{
		Topic: "integration.registered",
		Data: map[string]any{
			"integration_id": i.ID,
			"name":           i.Name,
			"route_prefix":   i.RoutePrefix,
			"scopes":         i.Scopes,
		},
	})
	return RegisterResult{Integration: i, APIKey: token}, nil
}

// List returns all integrations (admin view; api_key_hash never leaks).
func (s *Service) List(ctx context.Context) ([]Integration, error) {
	return s.store.List(ctx)
}

func (s *Service) Get(ctx context.Context, id string) (Integration, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) GetByPrefix(ctx context.Context, prefix string) (Integration, error) {
	return s.store.GetByPrefix(ctx, prefix)
}

func (s *Service) Update(ctx context.Context, id string, patch UpdatePatch) (Integration, error) {
	if patch.MemoryPolicy != nil && *patch.MemoryPolicy != "" && !ValidMemoryPolicy(*patch.MemoryPolicy) {
		return Integration{}, fmt.Errorf("memory_policy must be none|quarantine|full, got %q", *patch.MemoryPolicy)
	}
	if patch.PermissionMode != nil && *patch.PermissionMode != "" && !ValidPermissionMode(*patch.PermissionMode) {
		return Integration{}, fmt.Errorf("permission_mode must be default|bypass, got %q", *patch.PermissionMode)
	}
	if patch.MCPServers != nil {
		if err := validateMCPServers(*patch.MCPServers); err != nil {
			return Integration{}, err
		}
	}
	if err := s.store.Update(ctx, id, patch); err != nil {
		return Integration{}, err
	}
	if patch.Enabled != nil && !*patch.Enabled {
		s.clearTokenCache()
	}
	return s.store.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	s.clearTokenCache()
	s.bus.Publish(eventbus.Event{
		Topic: "integration.deregistered",
		Data:  map[string]any{"integration_id": id},
	})
	return nil
}

// RotateKey issues a fresh API key, invalidates the old hash, and
// clears the token cache.
func (s *Service) RotateKey(ctx context.Context, id string) (RegisterResult, error) {
	token, hash, err := generateAPIKey()
	if err != nil {
		return RegisterResult{}, err
	}
	if err := s.store.UpdateAPIKey(ctx, id, hash); err != nil {
		return RegisterResult{}, err
	}
	s.clearTokenCache()
	i, err := s.store.Get(ctx, id)
	if err != nil {
		return RegisterResult{}, err
	}
	s.bus.Publish(eventbus.Event{
		Topic: "integration.key_rotated",
		Data: map[string]any{
			"integration_id": id,
			"name":           i.Name,
		},
	})
	return RegisterResult{Integration: i, APIKey: token}, nil
}

// Verify checks a bearer token against all enabled integrations. The
// first match is cached so repeat verifications skip bcrypt.
func (s *Service) Verify(ctx context.Context, token string) (Integration, []string, error) {
	if !looksLikeAPIKey(token) {
		return Integration{}, nil, ErrInvalidAPIKey
	}

	s.tokenMu.RLock()
	cachedID, hit := s.tokenCache[token]
	s.tokenMu.RUnlock()
	if hit {
		i, err := s.store.Get(ctx, cachedID)
		if err == nil && i.Enabled {
			return i, i.Scopes, nil
		}
		// stale cache entry; fall through and re-verify.
		s.tokenMu.Lock()
		delete(s.tokenCache, token)
		s.tokenMu.Unlock()
	}

	rows, err := s.store.ListEnabled(ctx)
	if err != nil {
		return Integration{}, nil, err
	}
	for _, i := range rows {
		if verifyAPIKey(i.apiKeyHash, token) == nil {
			s.tokenMu.Lock()
			s.tokenCache[token] = i.ID
			s.tokenMu.Unlock()
			return i, i.Scopes, nil
		}
	}
	return Integration{}, nil, ErrInvalidAPIKey
}

// SetHealth records a probe outcome (used by the health checker
// goroutine) and emits integration.health_changed when the status
// transitions.
func (s *Service) SetHealth(ctx context.Context, id string, prev, next HealthStatus, payload map[string]any) error {
	if err := s.store.UpdateHealth(ctx, id, next, payload, time.Now().UTC()); err != nil {
		return err
	}
	if prev != next {
		s.bus.Publish(eventbus.Event{
			Topic: "integration.health_changed",
			Data: map[string]any{
				"integration_id": id,
				"from":           string(prev),
				"to":             string(next),
				"payload":        payload,
			},
		})
	}
	return nil
}

// HasScope returns true if `want` is granted by `granted`. Supports
// exact match and "event:subscribe:prefix.*" wildcard.
func HasScope(granted []string, want string) bool {
	for _, g := range granted {
		if g == want {
			return true
		}
		if strings.HasSuffix(g, ".*") {
			if strings.HasPrefix(want, strings.TrimSuffix(g, "*")) {
				return true
			}
		}
	}
	return false
}

func (s *Service) clearTokenCache() {
	s.tokenMu.Lock()
	s.tokenCache = make(map[string]string)
	s.tokenMu.Unlock()
}

func validateRegister(req RegisterRequest) error {
	if req.Name == "" {
		return errors.New("name is required")
	}
	// base_url + route_prefix are now optional but go together:
	// either both set (reverse-proxy integration) or both empty
	// (consumer-only — third-party app that calls opendray's API
	// but doesn't expose its own service to be proxied).
	hasURL := req.BaseURL != ""
	hasPrefix := req.RoutePrefix != ""
	if hasURL != hasPrefix {
		return errors.New("base_url and route_prefix must be set together (or both empty for a consumer-only integration)")
	}
	if req.MemoryPolicy != "" && !ValidMemoryPolicy(req.MemoryPolicy) {
		return fmt.Errorf("memory_policy must be none|quarantine|full, got %q", req.MemoryPolicy)
	}
	if req.PermissionMode != "" && !ValidPermissionMode(req.PermissionMode) {
		return fmt.Errorf("permission_mode must be default|bypass, got %q", req.PermissionMode)
	}
	if err := validateMCPServers(req.MCPServers); err != nil {
		return err
	}
	if hasURL {
		if u, err := url.Parse(req.BaseURL); err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("base_url is invalid: %s", req.BaseURL)
		}
		if strings.ContainsAny(req.RoutePrefix, "/?#") {
			return fmt.Errorf("route_prefix may not contain /?#")
		}
	}
	return nil
}

// validateMCPServers rejects a malformed spawn-profile mcp_servers value
// at the API boundary (Register / Update) rather than silently dropping it
// at spawn time. Empty / "[]" / "null" are fine (no injection). Otherwise
// it MUST be a JSON array of objects each carrying a non-empty "name".
func validateMCPServers(raw json.RawMessage) error {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "[]" || s == "null" {
		return nil
	}
	var servers []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &servers); err != nil {
		return fmt.Errorf("mcp_servers must be a JSON array of server objects: %w", err)
	}
	for i, srv := range servers {
		var name string
		if err := json.Unmarshal(srv["name"], &name); err != nil || strings.TrimSpace(name) == "" {
			return fmt.Errorf("mcp_servers[%d] is missing a non-empty \"name\"", i)
		}
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pe interface{ SQLState() string }
	if errors.As(err, &pe) {
		return pe.SQLState() == "23505"
	}
	return false
}

func newID() string {
	var b [9]byte
	_, _ = rand.Read(b[:])
	return "int_" + base64.RawURLEncoding.EncodeToString(b[:])
}
