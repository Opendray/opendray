package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Session represents a terminal session row.
type Session struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	SessionType     string            `json:"sessionType"`
	ClaudeSessionID string            `json:"claudeSessionId,omitempty"`
	ClaudeAccountID string            `json:"claudeAccountId,omitempty"`
	LLMProviderID   string            `json:"llmProviderId,omitempty"`
	CWD             string            `json:"cwd"`
	Status          string            `json:"status"`
	Model           string            `json:"model"`
	PID             int               `json:"pid,omitempty"`
	ExtraArgs       []string          `json:"extraArgs"`
	EnvOverrides    map[string]string `json:"envOverrides"`
	TotalCostUSD    float64           `json:"totalCostUsd"`
	InputTokens     int64             `json:"inputTokens"`
	OutputTokens    int64             `json:"outputTokens"`
	CreatedAt       time.Time         `json:"createdAt"`
	LastActiveAt    time.Time         `json:"lastActiveAt"`
}

// Plugin represents a registered plugin row.
type Plugin struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Version         string          `json:"version"`
	Enabled         bool            `json:"enabled"`
	Manifest        json.RawMessage `json:"manifest"`
	Config          json.RawMessage `json:"config"`
	HealthStatus    string          `json:"healthStatus"`
	HealthCheckedAt *time.Time      `json:"healthCheckedAt,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

// ── Session queries ─────────────────────────────────────────────

func (d *DB) CreateSession(ctx context.Context, s Session) (Session, error) {
	argsJSON, _ := json.Marshal(s.ExtraArgs)
	envJSON, _ := json.Marshal(s.EnvOverrides)

	var accountArg any
	if s.ClaudeAccountID != "" {
		accountArg = s.ClaudeAccountID
	}
	var providerArg any
	if s.LLMProviderID != "" {
		providerArg = s.LLMProviderID
	}

	row := d.Pool.QueryRow(ctx,
		`INSERT INTO sessions (name, session_type, cwd, model, extra_args, env_overrides,
		                        claude_account_id, llm_provider_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, name, session_type, cwd, status, model, created_at, last_active_at`,
		s.Name, s.SessionType, s.CWD, s.Model, argsJSON, envJSON, accountArg, providerArg,
	)
	var created Session
	err := row.Scan(&created.ID, &created.Name, &created.SessionType, &created.CWD,
		&created.Status, &created.Model, &created.CreatedAt, &created.LastActiveAt)
	if err != nil {
		return Session{}, fmt.Errorf("store: create session: %w", err)
	}
	created.ExtraArgs = s.ExtraArgs
	created.EnvOverrides = s.EnvOverrides
	created.ClaudeAccountID = s.ClaudeAccountID
	created.LLMProviderID = s.LLMProviderID
	return created, nil
}

func (d *DB) GetSession(ctx context.Context, id string) (Session, error) {
	row := d.Pool.QueryRow(ctx,
		`SELECT id, name, session_type, claude_session_id, claude_account_id, llm_provider_id,
		        cwd, status, model, pid,
		        extra_args, env_overrides, total_cost_usd, input_tokens, output_tokens,
		        created_at, last_active_at
		 FROM sessions WHERE id = $1`, id)
	return scanSession(row)
}

func (d *DB) ListSessions(ctx context.Context) ([]Session, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, name, session_type, claude_session_id, claude_account_id, llm_provider_id,
		        cwd, status, model, pid,
		        extra_args, env_overrides, total_cost_usd, input_tokens, output_tokens,
		        created_at, last_active_at
		 FROM sessions ORDER BY last_active_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (d *DB) UpdateSessionStatus(ctx context.Context, id, status string, pid int) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE sessions SET status = $2, pid = $3, last_active_at = now() WHERE id = $1`,
		id, status, pid)
	if err != nil {
		return fmt.Errorf("store: update session status: %w", err)
	}
	return nil
}

func (d *DB) UpdateClaudeSessionID(ctx context.Context, id, claudeSessionID string) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE sessions SET claude_session_id = $2, last_active_at = now() WHERE id = $1`,
		id, claudeSessionID)
	if err != nil {
		return fmt.Errorf("store: update claude session id: %w", err)
	}
	return nil
}

func (d *DB) DeleteSession(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: delete session: %w", err)
	}
	return nil
}

// ── Plugin queries ──────────────────────────────────────────────

func (d *DB) UpsertPlugin(ctx context.Context, p Plugin) (Plugin, error) {
	row := d.Pool.QueryRow(ctx,
		`INSERT INTO plugins (name, version, manifest, enabled)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (name) DO UPDATE SET version = $2, manifest = $3, enabled = $4, updated_at = now()
		 RETURNING id, name, version, enabled, manifest, config, health_status, health_checked_at, created_at, updated_at`,
		p.Name, p.Version, p.Manifest, p.Enabled,
	)
	return scanPlugin(row)
}

// SyncManifest refreshes a plugin's version + manifest from the filesystem
// without touching its enabled flag or user config. Called at startup so
// code-side manifest edits (new fields, updated model lists, etc.) flow
// into the DB on every restart — but existing user preferences survive.
func (d *DB) SyncManifest(ctx context.Context, name, version string, manifest json.RawMessage) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE plugins
		 SET version = $2, manifest = $3, updated_at = now()
		 WHERE name = $1`,
		name, version, manifest,
	)
	if err != nil {
		return fmt.Errorf("store: sync plugin manifest: %w", err)
	}
	return nil
}

func (d *DB) ListPlugins(ctx context.Context, enabledOnly bool) ([]Plugin, error) {
	q := `SELECT id, name, version, enabled, manifest, config, health_status, health_checked_at, created_at, updated_at
	      FROM plugins`
	if enabledOnly {
		q += ` WHERE enabled = true`
	}
	q += ` ORDER BY name`
	rows, err := d.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store: list plugins: %w", err)
	}
	defer rows.Close()

	var plugins []Plugin
	for rows.Next() {
		p, err := scanPlugin(rows)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, p)
	}
	return plugins, nil
}

func (d *DB) DeletePlugin(ctx context.Context, name string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM plugins WHERE name = $1`, name)
	if err != nil {
		return fmt.Errorf("store: delete plugin: %w", err)
	}
	return nil
}

func (d *DB) GetPluginConfig(ctx context.Context, name string) (json.RawMessage, error) {
	var cfg json.RawMessage
	err := d.Pool.QueryRow(ctx, `SELECT config FROM plugins WHERE name = $1`, name).Scan(&cfg)
	if err != nil {
		return nil, fmt.Errorf("store: get plugin config: %w", err)
	}
	return cfg, nil
}

func (d *DB) UpdatePluginEnabled(ctx context.Context, name string, enabled bool) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE plugins SET enabled = $2, updated_at = now() WHERE name = $1`,
		name, enabled)
	if err != nil {
		return fmt.Errorf("store: update plugin enabled: %w", err)
	}
	return nil
}

func (d *DB) UpdatePluginConfig(ctx context.Context, name string, config json.RawMessage) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE plugins SET config = $2, updated_at = now() WHERE name = $1`,
		name, config)
	if err != nil {
		return fmt.Errorf("store: update plugin config: %w", err)
	}
	return nil
}

func (d *DB) UpdatePluginHealth(ctx context.Context, name, status string) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE plugins SET health_status = $2, health_checked_at = now(), updated_at = now() WHERE name = $1`,
		name, status)
	if err != nil {
		return fmt.Errorf("store: update plugin health: %w", err)
	}
	return nil
}

// ── Scanners ────────────────────────────────────────────────────

type scannable interface {
	Scan(dest ...any) error
}

func scanSession(s scannable) (Session, error) {
	var sess Session
	var claudeID, accountID, providerID *string
	var pid *int
	var argsJSON, envJSON []byte

	err := s.Scan(&sess.ID, &sess.Name, &sess.SessionType, &claudeID, &accountID, &providerID,
		&sess.CWD, &sess.Status, &sess.Model, &pid, &argsJSON, &envJSON,
		&sess.TotalCostUSD, &sess.InputTokens, &sess.OutputTokens,
		&sess.CreatedAt, &sess.LastActiveAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Session{}, fmt.Errorf("store: session not found")
		}
		return Session{}, fmt.Errorf("store: scan session: %w", err)
	}
	if claudeID != nil {
		sess.ClaudeSessionID = *claudeID
	}
	if accountID != nil {
		sess.ClaudeAccountID = *accountID
	}
	if providerID != nil {
		sess.LLMProviderID = *providerID
	}
	if pid != nil {
		sess.PID = *pid
	}
	_ = json.Unmarshal(argsJSON, &sess.ExtraArgs)
	_ = json.Unmarshal(envJSON, &sess.EnvOverrides)
	if sess.ExtraArgs == nil {
		sess.ExtraArgs = []string{}
	}
	if sess.EnvOverrides == nil {
		sess.EnvOverrides = map[string]string{}
	}
	return sess, nil
}

func scanPlugin(s scannable) (Plugin, error) {
	var p Plugin
	err := s.Scan(&p.ID, &p.Name, &p.Version, &p.Enabled, &p.Manifest, &p.Config,
		&p.HealthStatus, &p.HealthCheckedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Plugin{}, fmt.Errorf("store: plugin not found")
		}
		return Plugin{}, fmt.Errorf("store: scan plugin: %w", err)
	}
	return p, nil
}
