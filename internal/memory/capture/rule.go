// Package capture turns raw transcript-message events into durable
// memory entries by running a configurable summarizer LLM after
// each rule-defined trigger fires.
//
// Phase A scope (see ADR / planner output):
//   - Single trigger kind: after_messages (every N new user messages
//     since the last capture, fire a summarizer call).
//   - Polling-based detection: the engine ticks every 10 seconds,
//     reads each live session's user-prompt history via
//     session.Manager.History, and compares against the last cursor
//     state to decide whether the threshold is met.
//   - Dedup: every fact extracted goes through memory.Search; hits
//     above the rule's dedup_threshold are skipped, the rest are
//     written via memory.Service.Store with source_kind='summarizer'.
package capture

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Rule mirrors one memory_capture_rules row.
//
// SessionID nil/"" means "global default applying to every session
// not pinned to a more specific rule". TriggerConfig is JSONB —
// shape depends on TriggerKind:
//
//	after_messages: {"n": 6}   (default 6)
//
// SummarizerProviderID nil = use the registry's is_default row.
type Rule struct {
	ID                   string         `json:"id"`
	SessionID            string         `json:"session_id,omitempty"`
	Name                 string         `json:"name"`
	Enabled              bool           `json:"enabled"`
	TriggerKind          string         `json:"trigger_kind"`
	TriggerConfig        map[string]any `json:"trigger_config"`
	SummarizerProviderID string         `json:"summarizer_provider_id,omitempty"`
	DedupThreshold       float32        `json:"dedup_threshold"`
	TargetScope          string         `json:"target_scope"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

// RulePatch is the partial update shape for store.UpdateRule.
type RulePatch struct {
	Name                 *string
	Enabled              *bool
	TriggerKind          *string
	TriggerConfig        map[string]any
	SummarizerProviderID *string // empty string → clear (NULL)
	DedupThreshold       *float32
	TargetScope          *string
}

// RuleStore wraps pgxpool with capture-rules CRUD.
type RuleStore struct {
	pool *pgxpool.Pool
}

func NewRuleStore(pool *pgxpool.Pool) *RuleStore {
	return &RuleStore{pool: pool}
}

// Sentinel errors.
var (
	ErrRuleNotFound = errors.New("capture: rule not found")
)

// Insert validates + inserts a row, returning the persisted Rule.
//
// trigger_kind is constrained to 'after_messages' in Phase A.
// dedup_threshold defaults to 0.85, target_scope to 'project'.
func (s *RuleStore) Insert(ctx context.Context, r Rule) (Rule, error) {
	if r.Name == "" {
		return Rule{}, errors.New("capture: rule name required")
	}
	if r.TriggerKind == "" {
		r.TriggerKind = "after_messages"
	}
	switch r.TriggerKind {
	case "after_messages", "on_idle", "k_chars", "manual":
	default:
		return Rule{}, fmt.Errorf("capture: unsupported trigger_kind %q", r.TriggerKind)
	}
	if r.DedupThreshold == 0 {
		r.DedupThreshold = 0.85
	}
	if r.DedupThreshold < 0 || r.DedupThreshold > 1 {
		return Rule{}, fmt.Errorf("capture: dedup_threshold must be in [0,1], got %g", r.DedupThreshold)
	}
	if r.TargetScope == "" {
		r.TargetScope = "project"
	}
	switch r.TargetScope {
	case "session", "project", "global":
	default:
		return Rule{}, fmt.Errorf("capture: invalid target_scope %q", r.TargetScope)
	}
	if r.ID == "" {
		r.ID = newRuleID()
	}
	if r.TriggerConfig == nil {
		r.TriggerConfig = map[string]any{}
	}
	cfgJSON, err := json.Marshal(r.TriggerConfig)
	if err != nil {
		return Rule{}, fmt.Errorf("capture: marshal trigger_config: %w", err)
	}
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now
	_, err = s.pool.Exec(ctx, `
		INSERT INTO memory_capture_rules
			(id, session_id, name, enabled, trigger_kind, trigger_config,
			 summarizer_provider_id, dedup_threshold, target_scope,
			 created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $10)`,
		r.ID, nullIfEmpty(r.SessionID), r.Name, r.Enabled, r.TriggerKind, cfgJSON,
		nullIfEmpty(r.SummarizerProviderID), r.DedupThreshold, r.TargetScope,
		now,
	)
	if err != nil {
		return Rule{}, fmt.Errorf("capture: insert rule: %w", err)
	}
	return r, nil
}

// Get returns a single rule by ID.
func (s *RuleStore) Get(ctx context.Context, id string) (Rule, error) {
	row := s.pool.QueryRow(ctx, ruleSelectStmt+` WHERE id = $1`, id)
	r, err := scanRule(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Rule{}, ErrRuleNotFound
	}
	return r, err
}

// List returns all rules; UI can filter client-side.
func (s *RuleStore) List(ctx context.Context) ([]Rule, error) {
	rows, err := s.pool.Query(ctx, ruleSelectStmt+` ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("capture: list rules: %w", err)
	}
	defer rows.Close()
	var out []Rule
	for rows.Next() {
		r, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Resolve picks the rule that should govern a session: prefers the
// session-scoped row, falls back to the global default. Returns
// ErrRuleNotFound when neither exists.
func (s *RuleStore) Resolve(ctx context.Context, sessionID string) (Rule, error) {
	if sessionID != "" {
		row := s.pool.QueryRow(ctx, ruleSelectStmt+` WHERE session_id = $1 AND enabled = TRUE LIMIT 1`, sessionID)
		r, err := scanRule(row)
		if err == nil {
			return r, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return Rule{}, err
		}
	}
	row := s.pool.QueryRow(ctx, ruleSelectStmt+` WHERE session_id IS NULL AND enabled = TRUE LIMIT 1`)
	r, err := scanRule(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Rule{}, ErrRuleNotFound
	}
	return r, err
}

// Update applies partial changes.
func (s *RuleStore) Update(ctx context.Context, id string, p RulePatch) (Rule, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Rule{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Rule{}, fmt.Errorf("capture: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if p.Name != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE memory_capture_rules SET name=$1, updated_at=NOW() WHERE id=$2`, *p.Name, id); err != nil {
			return Rule{}, fmt.Errorf("capture: update name: %w", err)
		}
	}
	if p.Enabled != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE memory_capture_rules SET enabled=$1, updated_at=NOW() WHERE id=$2`, *p.Enabled, id); err != nil {
			return Rule{}, fmt.Errorf("capture: update enabled: %w", err)
		}
	}
	if p.TriggerKind != nil {
		switch *p.TriggerKind {
		case "after_messages", "on_idle", "k_chars", "manual":
		default:
			return Rule{}, fmt.Errorf("capture: unsupported trigger_kind %q", *p.TriggerKind)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE memory_capture_rules SET trigger_kind=$1, updated_at=NOW() WHERE id=$2`, *p.TriggerKind, id); err != nil {
			return Rule{}, fmt.Errorf("capture: update trigger_kind: %w", err)
		}
	}
	if p.TriggerConfig != nil {
		raw, err := json.Marshal(p.TriggerConfig)
		if err != nil {
			return Rule{}, fmt.Errorf("capture: marshal trigger_config: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE memory_capture_rules SET trigger_config=$1::jsonb, updated_at=NOW() WHERE id=$2`,
			raw, id); err != nil {
			return Rule{}, fmt.Errorf("capture: update trigger_config: %w", err)
		}
	}
	if p.SummarizerProviderID != nil {
		val := nullIfEmpty(*p.SummarizerProviderID)
		if _, err := tx.Exec(ctx,
			`UPDATE memory_capture_rules SET summarizer_provider_id=$1, updated_at=NOW() WHERE id=$2`,
			val, id); err != nil {
			return Rule{}, fmt.Errorf("capture: update summarizer_provider_id: %w", err)
		}
	}
	if p.DedupThreshold != nil {
		if *p.DedupThreshold < 0 || *p.DedupThreshold > 1 {
			return Rule{}, fmt.Errorf("capture: dedup_threshold must be in [0,1]")
		}
		if _, err := tx.Exec(ctx,
			`UPDATE memory_capture_rules SET dedup_threshold=$1, updated_at=NOW() WHERE id=$2`,
			*p.DedupThreshold, id); err != nil {
			return Rule{}, fmt.Errorf("capture: update dedup_threshold: %w", err)
		}
	}
	if p.TargetScope != nil {
		switch *p.TargetScope {
		case "session", "project", "global":
		default:
			return Rule{}, fmt.Errorf("capture: invalid target_scope %q", *p.TargetScope)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE memory_capture_rules SET target_scope=$1, updated_at=NOW() WHERE id=$2`,
			*p.TargetScope, id); err != nil {
			return Rule{}, fmt.Errorf("capture: update target_scope: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Rule{}, fmt.Errorf("capture: commit: %w", err)
	}
	_ = cur // (we re-Get below; cur kept only for any future "validate against existing" guards)
	return s.Get(ctx, id)
}

// Delete removes a rule.
func (s *RuleStore) Delete(ctx context.Context, id string) error {
	cmd, err := s.pool.Exec(ctx, `DELETE FROM memory_capture_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("capture: delete rule: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrRuleNotFound
	}
	return nil
}

const ruleSelectStmt = `
	SELECT id,
	       COALESCE(session_id, '')              AS session_id,
	       name, enabled,
	       trigger_kind,
	       COALESCE(trigger_config, '{}'::jsonb) AS trigger_config,
	       COALESCE(summarizer_provider_id, '')  AS summarizer_provider_id,
	       dedup_threshold, target_scope,
	       created_at, updated_at
	  FROM memory_capture_rules`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRule(row rowScanner) (Rule, error) {
	var (
		r          Rule
		threshold  float64
		configJSON []byte
	)
	err := row.Scan(
		&r.ID, &r.SessionID, &r.Name, &r.Enabled,
		&r.TriggerKind, &configJSON,
		&r.SummarizerProviderID, &threshold, &r.TargetScope,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return Rule{}, err
	}
	r.DedupThreshold = float32(threshold)
	if len(configJSON) > 0 {
		_ = json.Unmarshal(configJSON, &r.TriggerConfig)
	}
	if r.TriggerConfig == nil {
		r.TriggerConfig = map[string]any{}
	}
	return r, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func newRuleID() string {
	var b [14]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("capture: rand: " + err.Error())
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	if len(enc) > 22 {
		enc = enc[:22]
	}
	return "mcr_" + strings.ToLower(enc)
}
