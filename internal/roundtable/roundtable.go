// Package roundtable implements the Round Table (experimental): a
// cross-vendor multi-agent discussion. A DETERMINISTIC chair drives N
// heterogeneous provider seats (claude / codex / antigravity — the three
// providers with a headless worker path) through a fixed three-beat
// schedule — propose → critique → synthesize — and produces a structured
// Verdict the operator approves.
//
// Why this belongs to opendray and not a single CLI: the moat is the
// cross-CLI gateway + shared memory. Same-model panels a user can already
// run with one CLI's subagents; heterogeneous foundation-model families
// around one table is what only the gateway offers.
//
// Phase 1 (this package) stops at the Verdict. Phase 2 (接开发) reuses the
// cortex Escalate pattern to seed the approved plan into a real PTY
// session — see round_tables.resulting_session_id (reserved, unused here).
//
// EXPERIMENTAL: fully self-contained, rollback via ROLLBACK.md.
package roundtable

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Beats — the fixed three-beat schedule the chair drives.
const (
	BeatPropose    = "propose"
	BeatCritique   = "critique"
	BeatSynthesize = "synthesize"
)

// Turn roles.
const (
	RoleSeat   = "seat"
	RoleChair  = "chair"
	RoleSystem = "system"
)

// Round table statuses.
const (
	StatusDraft           = "draft"
	StatusRunning         = "running"
	StatusAwaitingVerdict = "awaiting_verdict"
	StatusFailed          = "failed"
	StatusClosed          = "closed"
)

// Origins.
const (
	OriginOperator    = "operator"
	OriginIntegration = "integration"
)

// ErrNotFound is returned for unknown round-table ids.
var ErrNotFound = errors.New("roundtable: not found")

// Seat is one participant: a provider (+ optional model / account). Seats
// need a headless worker path, so provider is constrained to the same set
// the worker fabric's AgentWorker.buildCommand switch supports.
type Seat struct {
	Provider  string `json:"provider"`             // claude | codex | antigravity
	Model     string `json:"model,omitempty"`      // optional CLI model pin
	AccountID string `json:"account_id,omitempty"` // claude multi-account pin
}

// validSeatProvider mirrors cortex.validConvProvider but requires a
// concrete provider (a seat with no provider is meaningless). MUST stay in
// sync with worker.AgentWorker.buildCommand (claude / codex / antigravity).
// grok / opencode / a standalone gemini seat have no headless path yet.
func validSeatProvider(p string) bool {
	switch p {
	case "claude", "codex", "antigravity":
		return true
	}
	return false
}

// normalizeSeats validates + canonicalises the seat list: each provider
// must be supported, duplicates (same provider) are rejected (one seat per
// vendor for v1), and a non-claude seat's account pin is cleared.
func normalizeSeats(seats []Seat) ([]Seat, error) {
	if len(seats) < 2 {
		return nil, errors.New("roundtable: need at least 2 seats for a discussion")
	}
	seen := make(map[string]bool, len(seats))
	out := make([]Seat, 0, len(seats))
	for _, s := range seats {
		s.Provider = strings.TrimSpace(s.Provider)
		if !validSeatProvider(s.Provider) {
			return nil, fmt.Errorf("roundtable: seat provider %q is not supported (want claude|codex|antigravity)", s.Provider)
		}
		if seen[s.Provider] {
			return nil, fmt.Errorf("roundtable: duplicate seat provider %q (one seat per vendor in v1)", s.Provider)
		}
		seen[s.Provider] = true
		s.Model = strings.TrimSpace(s.Model)
		s.AccountID = strings.TrimSpace(s.AccountID)
		if s.Provider != "claude" {
			s.AccountID = "" // account selection only applies to claude
		}
		out = append(out, s)
	}
	return out, nil
}

// SeatScore is one row of the chair's deterministic ranking.
type SeatScore struct {
	Provider   string  `json:"provider"`
	Blockers   int     `json:"blockers"`   // # blocker-severity critiques against this seat's proposal
	Concerns   int     `json:"concerns"`   // # concern-severity critiques
	Confidence float64 `json:"confidence"` // seat's self-reported confidence
}

// Verdict is the chair's structured synthesis — mechanically assembled
// from seat proposals + critiques, no extra LLM call.
type Verdict struct {
	Recommended   string      `json:"recommended"`    // top-ranked seat's plan
	RecommendedBy string      `json:"recommended_by"` // provider that authored it
	Alternatives  []string    `json:"alternatives"`   // other seats' one-line summaries
	Tradeoffs     []string    `json:"tradeoffs"`      // union of proposal tradeoffs + critique points
	OpenQuestions []string    `json:"open_questions"` // unresolved blocker/concern critiques
	TaskBreakdown []string    `json:"task_breakdown"` // top-ranked seat's tasks
	Ranking       []SeatScore `json:"ranking"`        // full deterministic ranking
}

// RoundTable is one discussion session.
type RoundTable struct {
	ID                 string    `json:"id"`
	Topic              string    `json:"topic"`
	Cwd                string    `json:"cwd,omitempty"`
	Seats              []Seat    `json:"seats"`
	Status             string    `json:"status"`
	Verdict            *Verdict  `json:"verdict,omitempty"`
	ResultingSessionID string    `json:"resulting_session_id,omitempty"`
	Error              string    `json:"error,omitempty"`
	Origin             string    `json:"origin"`
	IntegrationID      string    `json:"integration_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Turn is one entry in the discussion thread. Unlike cortex messages, a
// turn records WHICH seat spoke (seat_provider) — the discussion is
// multi-participant by construction.
type Turn struct {
	ID           string          `json:"id"`
	RoundTableID string          `json:"round_table_id"`
	Beat         string          `json:"beat"`
	SeatProvider string          `json:"seat_provider,omitempty"`
	SeatModel    string          `json:"seat_model,omitempty"`
	Role         string          `json:"role"`
	Content      string          `json:"content"`
	Structured   json.RawMessage `json:"structured,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Store persists round tables + turns on the shared pool.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wires the store.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func newID(prefix string) string {
	var b [9]byte
	_, _ = rand.Read(b[:])
	return prefix + base64.RawURLEncoding.EncodeToString(b[:])
}

// Create opens a round table in status=draft.
func (s *Store) Create(ctx context.Context, topic, cwd string, seats []Seat, origin, integrationID string) (RoundTable, error) {
	if strings.TrimSpace(topic) == "" {
		return RoundTable{}, errors.New("roundtable: topic is required")
	}
	norm, err := normalizeSeats(seats)
	if err != nil {
		return RoundTable{}, err
	}
	if origin == "" {
		origin = OriginOperator
	}
	if origin != OriginOperator && origin != OriginIntegration {
		return RoundTable{}, fmt.Errorf("roundtable: bad origin %q", origin)
	}
	seatsJSON, err := json.Marshal(norm)
	if err != nil {
		return RoundTable{}, fmt.Errorf("roundtable: marshal seats: %w", err)
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO round_tables (id, topic, cwd, seats, origin, integration_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, topic, cwd, seats, status, verdict,
		          resulting_session_id, error, origin, integration_id, created_at, updated_at`,
		newID("rt_"), strings.TrimSpace(topic), strings.TrimSpace(cwd), seatsJSON, origin, strings.TrimSpace(integrationID))
	return scanRoundTable(row)
}

// Get returns one round table by id.
func (s *Store) Get(ctx context.Context, id string) (RoundTable, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, topic, cwd, seats, status, verdict,
		       resulting_session_id, error, origin, integration_id, created_at, updated_at
		  FROM round_tables WHERE id = $1`, id)
	rt, err := scanRoundTable(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoundTable{}, ErrNotFound
	}
	return rt, err
}

// List returns round tables newest first. Empty cwd lists all (admin
// overview); a cwd filters to that project.
func (s *Store) List(ctx context.Context, cwd string, limit int) ([]RoundTable, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows pgx.Rows
	var err error
	if strings.TrimSpace(cwd) == "" {
		rows, err = s.pool.Query(ctx, `
			SELECT id, topic, cwd, seats, status, verdict,
			       resulting_session_id, error, origin, integration_id, created_at, updated_at
			  FROM round_tables ORDER BY updated_at DESC LIMIT $1`, limit)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id, topic, cwd, seats, status, verdict,
			       resulting_session_id, error, origin, integration_id, created_at, updated_at
			  FROM round_tables WHERE cwd = $1 ORDER BY updated_at DESC LIMIT $2`, cwd, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("roundtable: list: %w", err)
	}
	defer rows.Close()
	var out []RoundTable
	for rows.Next() {
		rt, err := scanRoundTable(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rt)
	}
	return out, rows.Err()
}

// SetStatus updates the status and (optionally) the error message.
func (s *Store) SetStatus(ctx context.Context, id, status, errMsg string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE round_tables SET status = $1, error = $2, updated_at = NOW() WHERE id = $3`,
		status, errMsg, id)
	if err != nil {
		return fmt.Errorf("roundtable: set status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetVerdict stores the chair's synthesis and moves the table to
// awaiting_verdict.
func (s *Store) SetVerdict(ctx context.Context, id string, v Verdict) error {
	vJSON, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("roundtable: marshal verdict: %w", err)
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE round_tables
		   SET verdict = $1, status = $2, error = '', updated_at = NOW()
		 WHERE id = $3`, vJSON, StatusAwaitingVerdict, id)
	if err != nil {
		return fmt.Errorf("roundtable: set verdict: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AppendTurn records one discussion turn.
func (s *Store) AppendTurn(ctx context.Context, t Turn) (Turn, error) {
	switch t.Role {
	case RoleSeat, RoleChair, RoleSystem:
	default:
		return Turn{}, fmt.Errorf("roundtable: bad turn role %q", t.Role)
	}
	var structured any
	if len(t.Structured) > 0 {
		structured = []byte(t.Structured)
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO round_table_turns
			(id, round_table_id, beat, seat_provider, seat_model, role, content, structured)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, round_table_id, beat, seat_provider, seat_model, role, content, structured, created_at`,
		newID("rtt_"), t.RoundTableID, t.Beat, t.SeatProvider, t.SeatModel, t.Role, t.Content, structured)
	out, err := scanTurn(row)
	if err != nil {
		return Turn{}, fmt.Errorf("roundtable: append turn: %w", err)
	}
	_, _ = s.pool.Exec(ctx, `UPDATE round_tables SET updated_at = NOW() WHERE id = $1`, t.RoundTableID)
	return out, nil
}

// Turns returns a table's discussion thread, oldest first.
func (s *Store) Turns(ctx context.Context, roundTableID string, limit int) ([]Turn, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, round_table_id, beat, seat_provider, seat_model, role, content, structured, created_at
		  FROM round_table_turns
		 WHERE round_table_id = $1
		 ORDER BY created_at ASC LIMIT $2`, roundTableID, limit)
	if err != nil {
		return nil, fmt.Errorf("roundtable: list turns: %w", err)
	}
	defer rows.Close()
	var out []Turn
	for rows.Next() {
		t, err := scanTurn(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(dest ...any) error }

func scanRoundTable(row rowScanner) (RoundTable, error) {
	var rt RoundTable
	var seatsJSON, verdictJSON []byte
	err := row.Scan(&rt.ID, &rt.Topic, &rt.Cwd, &seatsJSON, &rt.Status, &verdictJSON,
		&rt.ResultingSessionID, &rt.Error, &rt.Origin, &rt.IntegrationID, &rt.CreatedAt, &rt.UpdatedAt)
	if err != nil {
		return RoundTable{}, err
	}
	if len(seatsJSON) > 0 {
		if err := json.Unmarshal(seatsJSON, &rt.Seats); err != nil {
			return RoundTable{}, fmt.Errorf("roundtable: unmarshal seats: %w", err)
		}
	}
	if len(verdictJSON) > 0 {
		var v Verdict
		if err := json.Unmarshal(verdictJSON, &v); err != nil {
			return RoundTable{}, fmt.Errorf("roundtable: unmarshal verdict: %w", err)
		}
		rt.Verdict = &v
	}
	return rt, nil
}

func scanTurn(row rowScanner) (Turn, error) {
	var t Turn
	var structured []byte
	if err := row.Scan(&t.ID, &t.RoundTableID, &t.Beat, &t.SeatProvider, &t.SeatModel,
		&t.Role, &t.Content, &structured, &t.CreatedAt); err != nil {
		return Turn{}, err
	}
	if len(structured) > 0 {
		t.Structured = json.RawMessage(structured)
	}
	return t, nil
}
