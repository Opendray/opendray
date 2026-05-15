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
)

// IsTerminal reports whether the session is no longer running. Both
// stopped and ended are terminal.
func (s State) IsTerminal() bool {
	return s == StateStopped || s == StateEnded
}

// Session is the public view of a PTY-backed CLI session. Runtime
// resources (PTY fd, ring buffer, subscribers) live on the Manager's
// internal struct, not here.
type Session struct {
	ID              string   `json:"id"`
	Name            string   `json:"name,omitempty"`
	ProviderID      string   `json:"provider_id"`
	Cwd             string   `json:"cwd"`
	Args            []string `json:"args"`
	State           State    `json:"state"`
	PID             int      `json:"pid,omitempty"`
	ClaudeAccountID string   `json:"claude_account_id,omitempty"`
	ClaudeSessionID string   `json:"claude_session_id,omitempty"`
	// ParentSessionID links a session spawned on behalf of another
	// (e.g. the Inspector's Tasks tab spawns shell children of an
	// AI session). Empty for top-level sessions. Used purely for UI
	// grouping — children are independent processes.
	ParentSessionID string     `json:"parent_session_id,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	ExitCode        *int       `json:"exit_code,omitempty"`
}

// CreateRequest is the JSON body for POST /api/v1/sessions.
type CreateRequest struct {
	Name            string   `json:"name"`
	ProviderID      string   `json:"provider_id"`
	ClaudeAccountID string   `json:"claude_account_id,omitempty"`
	ParentSessionID string   `json:"parent_session_id,omitempty"`
	Cwd             string   `json:"cwd"`
	Args            []string `json:"args"`
}

func (r CreateRequest) Validate() error {
	if r.ProviderID == "" {
		return errors.New("provider_id is required")
	}
	if r.Cwd == "" {
		return errors.New("cwd is required")
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
}

// Errors used by the manager and surfaced as HTTP status codes by the
// handler layer.
var (
	ErrNotFound                 = errors.New("session not found")
	ErrAlreadyEnded             = errors.New("session already ended")
	ErrAlreadyRunning           = errors.New("session already running")
	ErrUnknownProvider          = errors.New("unknown provider")
	ErrProviderUnavailable      = errors.New("provider unavailable")
	ErrAccountSwitchUnsupported = errors.New("account switch only supported for claude provider")
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
