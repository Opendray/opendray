// Package roundtable implements the Round Table (experimental): a
// cross-vendor AI GROUP CHAT. Members are the seated providers
// (claude / codex / antigravity — the three with a headless worker path)
// plus the operator. Everyone posts into one shared thread; the operator
// @mentions the members who should reply, and each mentioned member reads
// the whole conversation and answers in character — like a Telegram group.
//
// Why this belongs to opendray and not a single CLI: the moat is the
// cross-CLI gateway + shared memory. A group chat whose members are
// heterogeneous foundation-model families (Anthropic / OpenAI / Google),
// observable from web + mobile, is something only the gateway offers.
//
// Open-ended by design — no forced verdict. An optional "summarize" asks a
// member to condense the discussion into a plan (Phase 2 can seed that into
// a real PTY session via round_tables.resulting_session_id, reserved here).
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

// Message roles.
const (
	RoleOperator = "operator"
	RoleSeat     = "seat"
	RoleSystem   = "system"
)

// Message kinds.
const (
	KindMessage = "message"
	KindSummary = "summary"
)

// Round table statuses.
const (
	StatusActive = "active"
	StatusClosed = "closed"
)

// Origins.
const (
	OriginOperator    = "operator"
	OriginIntegration = "integration"
)

// MentionAll addresses every seated member at once (@all).
const MentionAll = "all"

// ErrNotFound is returned for unknown round-table ids.
var ErrNotFound = errors.New("roundtable: not found")

// Seat is one AI member: a provider (+ optional model / account). Seats need
// a headless worker path, so provider is constrained to the same set the
// worker fabric's AgentWorker.buildCommand switch supports.
type Seat struct {
	Provider  string `json:"provider"`             // claude | codex | antigravity
	Model     string `json:"model,omitempty"`      // optional CLI model pin
	AccountID string `json:"account_id,omitempty"` // claude multi-account pin
}

// validSeatProvider mirrors the worker's AgentWorker.buildCommand switch
// (claude / codex / antigravity). grok / opencode / a standalone gemini seat
// have no headless path yet.
func validSeatProvider(p string) bool {
	switch p {
	case "claude", "codex", "antigravity":
		return true
	}
	return false
}

// normalizeSeats validates + canonicalises the seat list: each provider must
// be supported, duplicates (same provider) are rejected (one seat per
// vendor), and a non-claude seat's account pin is cleared.
func normalizeSeats(seats []Seat) ([]Seat, error) {
	if len(seats) < 1 {
		return nil, errors.New("roundtable: need at least one seat")
	}
	seen := make(map[string]bool, len(seats))
	out := make([]Seat, 0, len(seats))
	for _, s := range seats {
		s.Provider = strings.TrimSpace(s.Provider)
		if !validSeatProvider(s.Provider) {
			return nil, fmt.Errorf("roundtable: seat provider %q is not supported (want claude|codex|antigravity)", s.Provider)
		}
		if seen[s.Provider] {
			return nil, fmt.Errorf("roundtable: duplicate seat provider %q", s.Provider)
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

// parseMentions returns the seated providers a message addresses via
// @provider tokens (case-insensitive). @all expands to every seat. Only
// providers actually seated at this table are returned; unknown @tokens are
// ignored. Order follows seat order for deterministic reply sequencing.
func parseMentions(content string, seats []Seat) []string {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "@"+MentionAll) {
		out := make([]string, len(seats))
		for i, s := range seats {
			out[i] = s.Provider
		}
		return out
	}
	var out []string
	for _, s := range seats {
		if strings.Contains(lower, "@"+s.Provider) {
			out = append(out, s.Provider)
		}
	}
	return out
}

// RoundTable is one group chat.
type RoundTable struct {
	ID                 string    `json:"id"`
	Topic              string    `json:"topic"`
	Cwd                string    `json:"cwd,omitempty"`
	Seats              []Seat    `json:"seats"`
	Status             string    `json:"status"`
	ResultingSessionID string    `json:"resulting_session_id,omitempty"`
	Origin             string    `json:"origin"`
	IntegrationID      string    `json:"integration_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Message is one entry in the group-chat thread. seat_provider records which
// member spoke (” for the operator / system); mentions lists who the
// message addressed.
type Message struct {
	ID           string    `json:"id"`
	RoundTableID string    `json:"round_table_id"`
	Role         string    `json:"role"`
	SeatProvider string    `json:"seat_provider,omitempty"`
	SeatModel    string    `json:"seat_model,omitempty"`
	Kind         string    `json:"kind"`
	Content      string    `json:"content"`
	Mentions     []string  `json:"mentions,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Store persists round tables + messages on the shared pool.
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

// Create opens an active group chat.
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
		RETURNING id, topic, cwd, seats, status, resulting_session_id, origin, integration_id, created_at, updated_at`,
		newID("rt_"), strings.TrimSpace(topic), strings.TrimSpace(cwd), seatsJSON, origin, strings.TrimSpace(integrationID))
	return scanRoundTable(row)
}

// Get returns one round table by id.
func (s *Store) Get(ctx context.Context, id string) (RoundTable, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, topic, cwd, seats, status, resulting_session_id, origin, integration_id, created_at, updated_at
		  FROM round_tables WHERE id = $1`, id)
	rt, err := scanRoundTable(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoundTable{}, ErrNotFound
	}
	return rt, err
}

// List returns round tables newest first. Empty cwd lists all.
func (s *Store) List(ctx context.Context, cwd string, limit int) ([]RoundTable, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows pgx.Rows
	var err error
	if strings.TrimSpace(cwd) == "" {
		rows, err = s.pool.Query(ctx, `
			SELECT id, topic, cwd, seats, status, resulting_session_id, origin, integration_id, created_at, updated_at
			  FROM round_tables ORDER BY updated_at DESC LIMIT $1`, limit)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id, topic, cwd, seats, status, resulting_session_id, origin, integration_id, created_at, updated_at
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

// SetStatus updates the chat status (active | closed).
func (s *Store) SetStatus(ctx context.Context, id, status string) error {
	if status != StatusActive && status != StatusClosed {
		return fmt.Errorf("roundtable: bad status %q", status)
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE round_tables SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("roundtable: set status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AppendMessage adds one message to the thread and bumps the table's
// updated_at.
func (s *Store) AppendMessage(ctx context.Context, m Message) (Message, error) {
	switch m.Role {
	case RoleOperator, RoleSeat, RoleSystem:
	default:
		return Message{}, fmt.Errorf("roundtable: bad message role %q", m.Role)
	}
	if m.Kind == "" {
		m.Kind = KindMessage
	}
	mentions := m.Mentions
	if mentions == nil {
		mentions = []string{}
	}
	mentionsJSON, err := json.Marshal(mentions)
	if err != nil {
		return Message{}, fmt.Errorf("roundtable: marshal mentions: %w", err)
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO round_table_messages
			(id, round_table_id, role, seat_provider, seat_model, kind, content, mentions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, round_table_id, role, seat_provider, seat_model, kind, content, mentions, created_at`,
		newID("rtm_"), m.RoundTableID, m.Role, m.SeatProvider, m.SeatModel, m.Kind, m.Content, mentionsJSON)
	out, err := scanMessage(row)
	if err != nil {
		return Message{}, fmt.Errorf("roundtable: append message: %w", err)
	}
	_, _ = s.pool.Exec(ctx, `UPDATE round_tables SET updated_at = NOW() WHERE id = $1`, m.RoundTableID)
	return out, nil
}

// Messages returns a table's thread, oldest first.
func (s *Store) Messages(ctx context.Context, roundTableID string, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, round_table_id, role, seat_provider, seat_model, kind, content, mentions, created_at
		  FROM round_table_messages
		 WHERE round_table_id = $1
		 ORDER BY created_at ASC LIMIT $2`, roundTableID, limit)
	if err != nil {
		return nil, fmt.Errorf("roundtable: list messages: %w", err)
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(dest ...any) error }

func scanRoundTable(row rowScanner) (RoundTable, error) {
	var rt RoundTable
	var seatsJSON []byte
	err := row.Scan(&rt.ID, &rt.Topic, &rt.Cwd, &seatsJSON, &rt.Status,
		&rt.ResultingSessionID, &rt.Origin, &rt.IntegrationID, &rt.CreatedAt, &rt.UpdatedAt)
	if err != nil {
		return RoundTable{}, err
	}
	if len(seatsJSON) > 0 {
		if err := json.Unmarshal(seatsJSON, &rt.Seats); err != nil {
			return RoundTable{}, fmt.Errorf("roundtable: unmarshal seats: %w", err)
		}
	}
	return rt, nil
}

func scanMessage(row rowScanner) (Message, error) {
	var m Message
	var mentionsJSON []byte
	if err := row.Scan(&m.ID, &m.RoundTableID, &m.Role, &m.SeatProvider, &m.SeatModel,
		&m.Kind, &m.Content, &mentionsJSON, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if len(mentionsJSON) > 0 {
		if err := json.Unmarshal(mentionsJSON, &m.Mentions); err != nil {
			return Message{}, fmt.Errorf("roundtable: unmarshal mentions: %w", err)
		}
	}
	return m, nil
}
