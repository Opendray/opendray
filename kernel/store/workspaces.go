package store

// Workspace queries — multi-tenant foundation (v0.5).
//
// v0.5 ships single-workspace per install: the migration bootstraps
// one row with slug="default" and is_default=TRUE. Every existing
// resource (sessions, claude_accounts, llm_providers, plugin_kv,
// plugin_secret, plugin_consents) carries a workspace_id column,
// backfilled to that default. Existing INSERT queries don't write
// the column yet; the column stays NULLABLE in this migration so
// nothing breaks. v0.5.1 will update writers to set workspace_id;
// v0.5.2 will tighten to NOT NULL.
//
// v0.6 adds:
//   • POST /api/workspaces (create), DELETE /api/workspaces/{slug}
//   • a sidebar workspace switcher backed by the slug stored in the
//     user's session (header X-Workspace-Slug + JWT claim)
//   • per-workspace plugin configs surfaced in the UI
//
// v0.7+ adds workspace_users + RBAC.

import (
	"context"
	"fmt"
	"time"
)

// Workspace is the top-level isolation unit. One row per tenant.
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	IsDefault bool      `json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// GetDefaultWorkspace returns the workspace flagged is_default=TRUE.
// The migration guarantees at least one such row; an error here means
// the schema is corrupted (someone manually nuked the row) — callers
// should treat it as fatal at startup. Cached after first hit because
// the value never changes during a process lifetime in v0.5 (no UI
// to flip the default yet).
func (d *DB) GetDefaultWorkspace(ctx context.Context) (Workspace, error) {
	var ws Workspace
	err := d.Pool.QueryRow(ctx,
		`SELECT id, name, slug, is_default, created_at, updated_at
		 FROM workspaces WHERE is_default = TRUE LIMIT 1`).
		Scan(&ws.ID, &ws.Name, &ws.Slug, &ws.IsDefault, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		return Workspace{}, fmt.Errorf("get default workspace: %w", err)
	}
	return ws, nil
}

// GetWorkspaceBySlug returns the workspace with the given slug, or an
// error if not found. Used by the v0.6 router middleware that resolves
// the workspace from the X-Workspace-Slug header.
func (d *DB) GetWorkspaceBySlug(ctx context.Context, slug string) (Workspace, error) {
	var ws Workspace
	err := d.Pool.QueryRow(ctx,
		`SELECT id, name, slug, is_default, created_at, updated_at
		 FROM workspaces WHERE slug = $1`, slug).
		Scan(&ws.ID, &ws.Name, &ws.Slug, &ws.IsDefault, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		return Workspace{}, fmt.Errorf("get workspace by slug %q: %w", slug, err)
	}
	return ws, nil
}

// ListWorkspaces returns every workspace ordered by created_at. Used
// by the v0.6 switcher UI.
func (d *DB) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, name, slug, is_default, created_at, updated_at
		 FROM workspaces ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()
	var out []Workspace
	for rows.Next() {
		var ws Workspace
		if err := rows.Scan(&ws.ID, &ws.Name, &ws.Slug,
			&ws.IsDefault, &ws.CreatedAt, &ws.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace: %w", err)
		}
		out = append(out, ws)
	}
	return out, rows.Err()
}

// CreateWorkspace adds a new workspace. The slug must be globally
// unique; "default" is reserved (the migration's bootstrap row uses
// it) and CreateWorkspace explicitly rejects it so users can't
// accidentally create a second one with that slug — even though the
// UNIQUE constraint would catch it, a typed error is friendlier.
func (d *DB) CreateWorkspace(ctx context.Context, name, slug string) (Workspace, error) {
	if slug == "default" {
		return Workspace{}, fmt.Errorf("create workspace: slug %q is reserved", slug)
	}
	var ws Workspace
	err := d.Pool.QueryRow(ctx,
		`INSERT INTO workspaces (name, slug, is_default)
		 VALUES ($1, $2, FALSE)
		 RETURNING id, name, slug, is_default, created_at, updated_at`,
		name, slug).
		Scan(&ws.ID, &ws.Name, &ws.Slug, &ws.IsDefault, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		return Workspace{}, fmt.Errorf("insert workspace: %w", err)
	}
	return ws, nil
}

// DeleteWorkspace removes a workspace. The CASCADE foreign keys on
// every workspace-scoped table mean every session, claude_account,
// llm_provider, plugin_kv, plugin_secret, and plugin_consent row
// belonging to it is also deleted. This is permanent — there's no
// soft delete in v0.5. The default workspace cannot be removed.
func (d *DB) DeleteWorkspace(ctx context.Context, id string) error {
	tag, err := d.Pool.Exec(ctx,
		`DELETE FROM workspaces WHERE id = $1 AND is_default = FALSE`, id)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete workspace: %s not found or is_default", id)
	}
	return nil
}
