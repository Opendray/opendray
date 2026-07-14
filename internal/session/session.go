package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"
)

// State enumerates the lifecycle of a session. Persisted as TEXT in
// sessions.state.
type State string

const (
	StatePending State = "pending"
	StateRunning State = "running"
	StateIdle    State = "idle"
	// StateStopped — process was terminated by an explicit user
	// action (Manager.Stop / DELETE was used as "stop"). The DB row
	// is preserved so the user can Restart it.
	StateStopped State = "stopped"
	// StateEnded — process exited on its own (clean exit or crash).
	// Row preserved; user can Restart or Remove.
	StateEnded State = "ended"
	// StateInterrupted — the session was live when the gateway process
	// exited (e.g. a self-update restart), so its PTY died with no
	// clean exit of its own. Distinct from 'ended' so startup
	// reconciliation can tell "the daemon killed this" apart from "the
	// agent exited", and auto-resume it. Terminal for Restart/resume.
	StateInterrupted State = "interrupted"
)

// IsTerminal reports whether the session is no longer running. Stopped,
// ended, and interrupted are all terminal (the PTY is gone); each can
// be re-spawned via Start.
func (s State) IsTerminal() bool {
	return s == StateStopped || s == StateEnded || s == StateInterrupted
}

// classifyExitState decides which terminal state to record for a session
// whose process just exited. Precedence: an explicit user stop wins;
// otherwise a gateway shutdown makes it 'interrupted' (so startup
// reconciliation resumes it); a spontaneous exit is a normal 'ended'.
//
// This now delegates to the transition matrix (Next) via TerminationEvent
// so the precedence lives in exactly one place; TestTerminationEventPrecedence
// pins the two in lock-step. A thin shim is kept for the existing call site.
func classifyExitState(stopRequested, closing bool) State {
	// A running session always has a legal terminal transition, so the
	// error is unreachable; fall back to the prior explicit precedence
	// defensively rather than propagate it into the exit path.
	state, err := Next(StateRunning, TerminationEvent(stopRequested, closing))
	if err != nil {
		switch {
		case stopRequested:
			return StateStopped
		case closing:
			return StateInterrupted
		default:
			return StateEnded
		}
	}
	return state
}

// Session is the public view of a PTY-backed CLI session. Runtime
// resources (PTY fd, ring buffer, subscribers) live on the Manager's
// internal struct, not here.
type Session struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	ProviderID string `json:"provider_id"`
	// Model pins the model this session spawns against, applied at
	// spawn via the provider's model flag. Empty falls back to the
	// provider config default. See CreateRequest.Model.
	Model string   `json:"model,omitempty"`
	Cwd   string   `json:"cwd"`
	Args  []string `json:"args"`
	// Theme is the operator's applied opendray theme ("light"/"dark") at
	// spawn time. Advertised to the CLI via COLORFGBG so a TUI can pick a
	// matching palette. Empty = unknown; the CLI keeps its own default.
	Theme           string `json:"theme,omitempty"`
	State           State  `json:"state"`
	PID             int    `json:"pid,omitempty"`
	ClaudeAccountID string `json:"claude_account_id,omitempty"`
	ClaudeSessionID string `json:"claude_session_id,omitempty"`
	// AntigravityAccountID is the agyacct account this session is pinned
	// to (provider "antigravity"). Empty means the CLI's default HOME
	// (~/.gemini). Mirrors ClaudeAccountID for the agy provider, whose
	// accounts are isolated by HOME rather than CLAUDE_CONFIG_DIR.
	AntigravityAccountID string `json:"antigravity_account_id,omitempty"`
	// ParentSessionID links a session spawned on behalf of another
	// (e.g. the Inspector's Tasks tab spawns shell children of an
	// AI session). Empty for top-level sessions. Used purely for UI
	// grouping — children are independent processes.
	ParentSessionID string `json:"parent_session_id,omitempty"`
	// Origin records who created the session — derived from the
	// authenticated principal at create time, never client-supplied.
	// The memory capture pipeline routes on it (Cortex Phase 2) so
	// third-party temp sessions can't pollute durable memory.
	Origin Origin `json:"origin,omitempty"`
	// IntegrationID is set when Origin == OriginIntegration: the id
	// of the integration whose API key created the session.
	IntegrationID string     `json:"integration_id,omitempty"`
	StartedAt     time.Time  `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty"`
	ExitCode      *int       `json:"exit_code,omitempty"`
}

// Origin identifies the kind of principal that created a session.
type Origin string

const (
	// OriginOperator — the human operator (admin token; web/mobile UI).
	OriginOperator Origin = "operator"
	// OriginIntegration — a third-party app via a scoped API key.
	OriginIntegration Origin = "integration"
	// OriginCLI — the local opendray CLI.
	OriginCLI Origin = "cli"
)

// CreateRequest is the JSON body for POST /api/v1/sessions.
type CreateRequest struct {
	Name       string `json:"name"`
	ProviderID string `json:"provider_id"`
	// Model optionally pins the model for this session (e.g. an "opus"
	// id for the claude provider). Empty means "use the provider config
	// default". Applied at spawn via the provider's model flag, so a
	// `--model` in Args still wins (the manager dedups overridden flags).
	Model                string   `json:"model,omitempty"`
	ClaudeAccountID      string   `json:"claude_account_id,omitempty"`
	AntigravityAccountID string   `json:"antigravity_account_id,omitempty"`
	ParentSessionID      string   `json:"parent_session_id,omitempty"`
	Cwd                  string   `json:"cwd"`
	Args                 []string `json:"args"`
	// Theme is the client's applied theme ("light"/"dark"). Optional: an
	// older client or an API caller may omit it, in which case opendray
	// advertises nothing and the CLI keeps its own default.
	Theme string `json:"theme,omitempty"`

	// origin/integrationID are unexported on purpose: they are derived
	// from the authenticated principal by the HTTP handler (SetOrigin)
	// and must never be settable from the JSON body.
	origin        Origin
	integrationID string
}

// SetOrigin stamps the provenance derived from the authenticated
// principal. integrationID is only meaningful for OriginIntegration.
func (r *CreateRequest) SetOrigin(o Origin, integrationID string) {
	r.origin = o
	r.integrationID = integrationID
}

func (r CreateRequest) Validate() error {
	if r.ProviderID == "" {
		return errors.New("provider_id is required")
	}
	if r.Cwd == "" {
		return errors.New("cwd is required")
	}
	if r.Theme != "" && r.Theme != "light" && r.Theme != "dark" {
		return errors.New(`theme must be "light" or "dark"`)
	}
	return nil
}

// InputRequest is the JSON body for POST /api/v1/sessions/{id}/input.
// `Data` is treated as raw bytes (not base64-decoded) and sent verbatim
// to the PTY's stdin.
type InputRequest struct {
	Data string `json:"data"`
}

// ResizeRequest is the JSON body for POST /api/v1/sessions/{id}/resize.
type ResizeRequest struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// SwitchAccountRequest is the body for PATCH
// /api/v1/sessions/{id}/claude-account. An empty AccountID clears the
// binding, falling back to the CLI's default credential.
type SwitchAccountRequest struct {
	AccountID string `json:"account_id"`
	// CarryContext, when true, seeds the new account's fresh session
	// with a recap of the prior conversation (read from the old
	// transcript, injected via --append-system-prompt). Default false
	// preserves the clean-slate switch. Note this sends prior
	// conversation content to the provider under the NEW account — the
	// UI surfaces that as a consent line.
	CarryContext bool `json:"carry_context,omitempty"`
}

// Errors used by the manager and surfaced as HTTP status codes by the
// handler layer.
var (
	ErrNotFound                 = errors.New("session not found")
	ErrAlreadyEnded             = errors.New("session already ended")
	ErrAlreadyRunning           = errors.New("session already running")
	ErrUnknownProvider          = errors.New("unknown provider")
	ErrProviderUnavailable      = errors.New("provider unavailable")
	ErrAccountSwitchUnsupported = errors.New("account switch only supported for claude and antigravity providers")
)

func newID() string {
	var b [9]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand is not expected to fail; fall back to time-based
		// id to keep the system functional rather than panicking.
		t := time.Now().UnixNano()
		for i := range b {
			b[i] = byte(t >> (i * 8))
		}
	}
	return "ses_" + base64.RawURLEncoding.EncodeToString(b[:])
}
