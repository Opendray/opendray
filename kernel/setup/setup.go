// Package setup drives OpenDray's first-run configuration flow.
//
// When [Manager] reports Active() true, the gateway serves only
// /api/setup/* (guarded by the bootstrap token) plus static assets;
// everything else returns 503 "setup required". The Flutter wizard
// walks the user through:
//
//  1. Choose database mode (embedded / external)
//  2. Set admin credentials
//  3. Optionally install agent CLIs (Phase 5)
//  4. Finalize — writes config.toml, flips Active() false, router rebuilds
//
// Endpoints must require a header X-Setup-Token or ?token= query param
// matching [Manager.BootstrapToken] — this prevents a LAN attacker from
// racing to configure an unclaimed binary.
package setup

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"sync"
	"sync/atomic"

	"github.com/opendray/opendray/kernel/config"
)

// Manager is shared between cmd/opendray/main.go (which owns the gateway
// transition) and gateway/setup_handlers.go (which serves the API).
type Manager struct {
	token     string
	done      atomic.Bool
	onFinish  func()
	mu        sync.Mutex
	draft     config.Config // accumulated across wizard steps
	dbTested  bool          // /api/setup/db/test succeeded at least once
	dbChoice  string        // "embedded" | "external" set by /api/setup/db/commit
}

// New returns a Manager seeded with a fresh bootstrap token and the
// defaults as the draft. Caller registers onFinish to trigger the router
// rebuild after /api/setup/finalize writes the config file.
func New(onFinish func()) (*Manager, error) {
	tok, err := generateToken()
	if err != nil {
		return nil, err
	}
	m := &Manager{
		token:    tok,
		onFinish: onFinish,
		draft:    config.Defaults(),
	}
	return m, nil
}

// BootstrapToken is the shared secret the wizard must present on every
// /api/setup/* call. Printed to stderr on startup by main.go and written
// alongside config at ~/.opendray/setup-token so the wizard UI can
// retrieve it via a same-origin fetch if the user follows a plain /setup
// URL without the token query param.
func (m *Manager) BootstrapToken() string { return m.token }

// Active reports whether setup mode is still engaged. Flipped false by
// [Manager.Finalize].
func (m *Manager) Active() bool { return !m.done.Load() }

// ValidateToken does a constant-time comparison against the bootstrap
// token. Returns true iff provided matches.
func (m *Manager) ValidateToken(provided string) bool {
	if provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(m.token)) == 1
}

// Draft returns a copy of the in-progress config, safe to inspect from
// handlers without holding the lock.
func (m *Manager) Draft() config.Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.draft
}

// UpdateDraft applies a mutator under the lock. The mutator sees a
// pointer to the live draft; any mutation survives.
func (m *Manager) UpdateDraft(mutate func(*config.Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mutate(&m.draft)
}

// MarkDBTested is called by /api/setup/db/test on success so finalize
// knows the user actually verified connectivity.
func (m *Manager) MarkDBTested(mode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dbTested = true
	m.dbChoice = mode
}

// DBTested reports whether a successful test has run for the current
// choice. Finalize refuses to commit until this is true OR the user
// picked embedded (no remote to test).
func (m *Manager) DBTested() (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dbTested, m.dbChoice
}

// Finalize writes the draft config, flips Active to false, and triggers
// the onFinish callback so the gateway can rebuild its router.
func (m *Manager) Finalize() error {
	m.mu.Lock()
	cfg := m.draft
	m.mu.Unlock()

	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	m.done.Store(true)
	if m.onFinish != nil {
		m.onFinish()
	}
	return nil
}

// Step identifies where the wizard currently is. Clients read GET
// /api/setup/status to resume mid-flow.
type Step string

const (
	StepWelcome   Step = "welcome"
	StepDB        Step = "db"
	StepAdmin     Step = "admin"
	StepCLI       Step = "cli"
	StepFinalize  Step = "finalize"
	StepCompleted Step = "completed"
)

// Status is the GET /api/setup/status response body.
type Status struct {
	NeedsSetup      bool   `json:"needsSetup"`
	Step            Step   `json:"step"`
	DBMode          string `json:"dbMode,omitempty"`
	DBTested        bool   `json:"dbTested"`
	AdminConfigured bool   `json:"adminConfigured"`
	SchemaVersion   int    `json:"schemaVersion"`
}

// Status snapshots the current wizard state. The AdminConfigured flag is
// derived from whether a non-empty admin_bootstrap_password has been
// staged in the draft — actual persistence to the admin_auth table
// happens in /api/setup/admin.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	step := StepWelcome
	switch {
	case m.done.Load():
		step = StepCompleted
	case m.draft.DB.External.Host != "" || m.dbChoice == "embedded":
		if m.draft.Auth.AdminBootstrapPassword != "" {
			step = StepFinalize
		} else {
			step = StepAdmin
		}
	default:
		step = StepDB
	}
	return Status{
		NeedsSetup:      !m.done.Load(),
		Step:            step,
		DBMode:          m.dbChoice,
		DBTested:        m.dbTested,
		AdminConfigured: m.draft.Auth.AdminBootstrapPassword != "",
		SchemaVersion:   config.SchemaVersion,
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
