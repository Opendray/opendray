// Package injector renders prior-session memories into a system-
// prompt prefix that the catalog adapter splices into agent CLIs
// at session-spawn time.
//
// Phase A ships two strategies:
//
//   - "none"          → return "" (model still uses memory_search on demand)
//   - "top_k_recent"  → fetch the K most recent project-scoped memories
//     and render a markdown preface
//
// Strategy selection is per-session (session-scoped Profile row)
// with a fall-through to the global default Profile (session_id
// IS NULL).
package injector

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

// Profile mirrors one memory_injection_profiles row.
type Profile struct {
	ID           string         `json:"id"`
	SessionID    string         `json:"session_id,omitempty"`
	StrategyKind string         `json:"strategy_kind"`
	Config       map[string]any `json:"config"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ProfilePatch is the partial-update shape.
type ProfilePatch struct {
	StrategyKind *string
	Config       map[string]any
}

// ProfileStore wraps pgxpool with profile CRUD.
type ProfileStore struct {
	pool *pgxpool.Pool
}

func NewProfileStore(pool *pgxpool.Pool) *ProfileStore {
	return &ProfileStore{pool: pool}
}

var (
	ErrProfileNotFound = errors.New("injector: profile not found")
)

// Insert validates + writes a row.
func (s *ProfileStore) Insert(ctx context.Context, p Profile) (Profile, error) {
	if p.StrategyKind == "" {
		p.StrategyKind = "none"
	}
	if !validStrategy(p.StrategyKind) {
		return Profile{}, fmt.Errorf("injector: unsupported strategy_kind %q", p.StrategyKind)
	}
	if p.Config == nil {
		p.Config = map[string]any{}
	}
	cfgJSON, err := json.Marshal(p.Config)
	if err != nil {
		return Profile{}, fmt.Errorf("injector: marshal config: %w", err)
	}
	if p.ID == "" {
		p.ID = newProfileID()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	_, err = s.pool.Exec(ctx, `
		INSERT INTO memory_injection_profiles
			(id, session_id, strategy_kind, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, $5)`,
		p.ID, nullIfEmpty(p.SessionID), p.StrategyKind, cfgJSON, now,
	)
	if err != nil {
		return Profile{}, fmt.Errorf("injector: insert profile: %w", err)
	}
	return p, nil
}

func (s *ProfileStore) Get(ctx context.Context, id string) (Profile, error) {
	row := s.pool.QueryRow(ctx, profileSelectStmt+` WHERE id = $1`, id)
	p, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	return p, err
}

func (s *ProfileStore) List(ctx context.Context) ([]Profile, error) {
	rows, err := s.pool.Query(ctx, profileSelectStmt+` ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("injector: list profiles: %w", err)
	}
	defer rows.Close()
	var out []Profile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Resolve returns the profile that governs sessionID. Prefers a
// session-scoped row; falls back to the global default; if neither
// exists, returns a synthesised "top_k_relevant" profile (k=5) so
// the cross-agent memory loop works out of the box without the
// operator having to seed an injection profile manually.
//
// Pre PR-M3 this returned a {strategy: "none"} fallback, which
// meant a fresh install never injected any memory into spawn
// system prompts even though the storage layer was working. The
// "none" default was load-bearing on the operator knowing to seed
// a profile via API — nobody did, so the feature was silently
// dormant on every deploy.
//
// With the new default:
//   - Zero memories yet → renderTopKRelevant returns "" anyway, so
//     no spam injection on a fresh project.
//   - One or more project memories → the agent gets a top-5
//     relevant preface in the system prompt on every spawn.
//
// Operators who explicitly want injection off can insert a
// session_id-NULL row with strategy "none" — that wins over this
// synthetic fallback.
func (s *ProfileStore) Resolve(ctx context.Context, sessionID string) Profile {
	if sessionID != "" {
		row := s.pool.QueryRow(ctx, profileSelectStmt+` WHERE session_id = $1 LIMIT 1`, sessionID)
		if p, err := scanProfile(row); err == nil {
			return p
		}
	}
	row := s.pool.QueryRow(ctx, profileSelectStmt+` WHERE session_id IS NULL LIMIT 1`)
	if p, err := scanProfile(row); err == nil {
		return p
	}
	// Default to top_k_recent (not top_k_relevant): recency is less
	// cwd-dependent, and renderTopKRecent falls back to global scope when
	// the cwd has no project memories — so cross-session recall works out
	// of the box without an operator configuring a profile.
	return Profile{
		ID:           "synthetic-default-top-k-recent",
		StrategyKind: "top_k_recent",
		Config:       map[string]any{"k": float64(5)},
	}
}

func (s *ProfileStore) Update(ctx context.Context, id string, p ProfilePatch) (Profile, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return Profile{}, err
	}
	if p.StrategyKind != nil {
		k := *p.StrategyKind
		if !validStrategy(k) {
			return Profile{}, fmt.Errorf("injector: unsupported strategy_kind %q", k)
		}
		if _, err := s.pool.Exec(ctx,
			`UPDATE memory_injection_profiles SET strategy_kind=$1, updated_at=NOW() WHERE id=$2`, k, id); err != nil {
			return Profile{}, fmt.Errorf("injector: update strategy_kind: %w", err)
		}
	}
	if p.Config != nil {
		raw, err := json.Marshal(p.Config)
		if err != nil {
			return Profile{}, fmt.Errorf("injector: marshal config: %w", err)
		}
		if _, err := s.pool.Exec(ctx,
			`UPDATE memory_injection_profiles SET config=$1::jsonb, updated_at=NOW() WHERE id=$2`,
			raw, id); err != nil {
			return Profile{}, fmt.Errorf("injector: update config: %w", err)
		}
	}
	_ = cur
	return s.Get(ctx, id)
}

func (s *ProfileStore) Delete(ctx context.Context, id string) error {
	cmd, err := s.pool.Exec(ctx, `DELETE FROM memory_injection_profiles WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("injector: delete profile: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

const profileSelectStmt = `
	SELECT id,
	       COALESCE(session_id, '') AS session_id,
	       strategy_kind,
	       COALESCE(config, '{}'::jsonb) AS config,
	       created_at, updated_at
	  FROM memory_injection_profiles`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProfile(row rowScanner) (Profile, error) {
	var (
		p          Profile
		configJSON []byte
	)
	err := row.Scan(&p.ID, &p.SessionID, &p.StrategyKind, &configJSON, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return Profile{}, err
	}
	if len(configJSON) > 0 {
		_ = json.Unmarshal(configJSON, &p.Config)
	}
	if p.Config == nil {
		p.Config = map[string]any{}
	}
	return p, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// validStrategy returns whether kind is a recognised injection
// strategy. Phase B widened from {none, top_k_recent} to include
// top_k_relevant, on_keyword, manual_only, hybrid.
func validStrategy(kind string) bool {
	switch kind {
	case "none", "top_k_recent", "top_k_relevant", "on_keyword", "manual_only", "hybrid":
		return true
	}
	return false
}

func newProfileID() string {
	var b [14]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("injector: rand: " + err.Error())
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	if len(enc) > 22 {
		enc = enc[:22]
	}
	return "mip_" + strings.ToLower(enc)
}
