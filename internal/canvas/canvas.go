// Package canvas implements the Canvas (beta): a live HTML preview surface
// that any cloud agent can push to via the `canvas_render` MCP tool, so the
// operator sees the actual rendered UI while it is being built — not just a
// screenshot or a prose description. The operator then annotates the preview
// (pins on elements, region selections) and those annotations are seeded back
// into the agent's session as a prompt. A two-way visual channel that removes
// the ambiguity of "describe the change in words".
//
// Why this belongs to opendray and not a single CLI: the gateway already owns
// the session PTYs, the shared per-CLI memory MCP every agent is attached to,
// and the web workbench. Rendering an agent's HTML next to its terminal and
// routing pixel-level feedback back into the conversation is something only the
// gateway can offer, uniformly across Claude / Codex / Gemini / …
//
// Scoped by cwd (project working dir), exactly like the session Inspector
// tabs. BETA: fully self-contained, rollback by dropping canvas_artifacts +
// deleting this package + the MCP tool + the web panel.
package canvas

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned for unknown artifact ids.
var ErrNotFound = errors.New("canvas: not found")

// DefaultSlug names the artifact an agent renders when it does not pick a slug
// and no canvas is focused.
const DefaultSlug = "default"

// Canvas kinds. The Canvas is a general visual surface: a screen mock, or a
// diagram the agent draws for the operator (flow / mind map / relationships).
const (
	KindUI      = "ui"      // a screen / component mock
	KindFlow    = "flow"    // a flowchart / process diagram
	KindMindmap = "mindmap" // a mind map
	KindGraph   = "graph"   // a relationship / entity diagram
	KindDoc     = "doc"     // a formatted document / spec page
)

// ValidKind reports whether k is a known canvas kind.
func ValidKind(k string) bool {
	switch k {
	case KindUI, KindFlow, KindMindmap, KindGraph, KindDoc:
		return true
	}
	return false
}

// NormKind lowercases + validates a kind. Unknown/empty returns "" so callers
// can distinguish "not specified" (keep the stored kind) from an explicit one.
func NormKind(k string) string {
	k = strings.ToLower(strings.TrimSpace(k))
	if ValidKind(k) {
		return k
	}
	return ""
}

// Artifact is one named HTML canvas within a project.
type Artifact struct {
	ID        string    `json:"id"`
	Cwd       string    `json:"cwd"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Kind      string    `json:"kind"`
	HTML      string    `json:"html"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Summary is an Artifact without its (potentially large) html body — used for
// the list endpoint that populates the Canvas tab selector.
type Summary struct {
	ID        string    `json:"id"`
	Cwd       string    `json:"cwd"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Kind      string    `json:"kind"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store is the canvas_artifacts persistence layer.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wires the store over a pgx pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func newID(prefix string) string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return prefix + base64.RawURLEncoding.EncodeToString(b[:])
}

func normSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return DefaultSlug
	}
	return slug
}

// Upsert renders (creates or replaces) the artifact for (cwd, slug), bumping
// its version. Returns the stored row. cwd must be non-empty. An empty kind
// keeps the stored kind on update (and defaults to "ui" on insert), so a
// re-render that omits it doesn't silently turn a diagram back into a mock.
func (s *Store) Upsert(ctx context.Context, cwd, slug, title, kind, html string) (Artifact, error) {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return Artifact{}, errors.New("canvas: cwd is required")
	}
	slug = normSlug(slug)
	kind = NormKind(kind)
	row := s.pool.QueryRow(ctx, `
		INSERT INTO canvas_artifacts (id, cwd, slug, title, kind, html)
		VALUES ($1, $2, $3, $4, COALESCE(NULLIF($5, ''), 'ui'), $6)
		ON CONFLICT (cwd, slug) DO UPDATE SET
			title      = EXCLUDED.title,
			kind       = COALESCE(NULLIF($5, ''), canvas_artifacts.kind),
			html       = EXCLUDED.html,
			version    = canvas_artifacts.version + 1,
			updated_at = NOW()
		RETURNING id, cwd, slug, title, kind, html, version, created_at, updated_at`,
		newID("cv_"), cwd, slug, strings.TrimSpace(title), kind, html)
	return scanArtifact(row)
}

// Get returns one artifact (with html) by id.
func (s *Store) Get(ctx context.Context, id string) (Artifact, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, cwd, slug, title, kind, html, version, created_at, updated_at
		FROM canvas_artifacts WHERE id = $1`, id)
	a, err := scanArtifact(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Artifact{}, ErrNotFound
	}
	return a, err
}

// List returns artifact summaries (no html) for a cwd, newest first.
func (s *Store) List(ctx context.Context, cwd string) ([]Summary, error) {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return nil, errors.New("canvas: cwd is required")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, cwd, slug, title, kind, version, created_at, updated_at
		FROM canvas_artifacts WHERE cwd = $1
		ORDER BY updated_at DESC`, cwd)
	if err != nil {
		return nil, fmt.Errorf("list canvas: %w", err)
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.ID, &s.Cwd, &s.Slug, &s.Title, &s.Kind, &s.Version, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan canvas: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetBySlug returns one artifact by its project-scoped slug.
func (s *Store) GetBySlug(ctx context.Context, cwd, slug string) (Artifact, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, cwd, slug, title, kind, html, version, created_at, updated_at
		FROM canvas_artifacts WHERE cwd = $1 AND slug = $2`,
		strings.TrimSpace(cwd), normSlug(slug))
	a, err := scanArtifact(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Artifact{}, ErrNotFound
	}
	return a, err
}

// Delete removes one artifact. Missing id is a no-op (returns ErrNotFound).
func (s *Store) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM canvas_artifacts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete canvas: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanArtifact(row pgx.Row) (Artifact, error) {
	var a Artifact
	if err := row.Scan(&a.ID, &a.Cwd, &a.Slug, &a.Title, &a.Kind, &a.HTML, &a.Version, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return Artifact{}, err
	}
	return a, nil
}
