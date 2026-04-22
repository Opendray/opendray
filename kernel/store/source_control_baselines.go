package store

// source_control_baselines CRUD. Backs the Source Control panel's
// "show me only what changed during this Claude session" feature.
//
// Replaces the earlier in-memory map on gateway/git.Manager, which
// evaporated on gateway restart and leaked entries for abandoned
// sessions. The DB cascade on sessions deletion gives us automatic
// cleanup; explicit delete is still exposed for "Clear baseline" UI.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// SourceControlBaseline records the commit a session started from,
// scoped to one repository. A single session may hold baselines for
// multiple repos (one per repo_path).
type SourceControlBaseline struct {
	SessionID string    `json:"sessionId"`
	RepoPath  string    `json:"repoPath"`
	HeadSHA   string    `json:"headSha"`
	CreatedAt time.Time `json:"createdAt"`
}

// SCBaselineUpsert records / refreshes a baseline. ON CONFLICT updates
// head_sha and created_at so "Re-snapshot" is a single call.
func (d *DB) SCBaselineUpsert(ctx context.Context, sessionID, repoPath, headSHA string) (SourceControlBaseline, error) {
	if sessionID == "" || repoPath == "" || headSHA == "" {
		return SourceControlBaseline{}, errors.New("store: SCBaselineUpsert requires non-empty sessionID, repoPath, headSHA")
	}
	const q = `
		INSERT INTO source_control_baselines (session_id, repo_path, head_sha)
		VALUES ($1, $2, $3)
		ON CONFLICT (session_id, repo_path) DO UPDATE
		   SET head_sha   = EXCLUDED.head_sha,
		       created_at = NOW()
		RETURNING session_id, repo_path, head_sha, created_at
	`
	row := d.Pool.QueryRow(ctx, q, sessionID, repoPath, headSHA)
	var b SourceControlBaseline
	if err := row.Scan(&b.SessionID, &b.RepoPath, &b.HeadSHA, &b.CreatedAt); err != nil {
		return SourceControlBaseline{}, fmt.Errorf("store: SCBaselineUpsert: %w", err)
	}
	return b, nil
}

// SCBaselineGet returns the baseline for (sessionID, repoPath). The
// bool is false when no baseline exists — the caller should treat this
// as "show the normal staged/unstaged view" rather than an error.
func (d *DB) SCBaselineGet(ctx context.Context, sessionID, repoPath string) (SourceControlBaseline, bool, error) {
	const q = `
		SELECT session_id, repo_path, head_sha, created_at
		  FROM source_control_baselines
		 WHERE session_id = $1 AND repo_path = $2
	`
	var b SourceControlBaseline
	err := d.Pool.QueryRow(ctx, q, sessionID, repoPath).
		Scan(&b.SessionID, &b.RepoPath, &b.HeadSHA, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SourceControlBaseline{}, false, nil
	}
	if err != nil {
		return SourceControlBaseline{}, false, fmt.Errorf("store: SCBaselineGet: %w", err)
	}
	return b, true, nil
}

// SCBaselineListSession returns every baseline the session holds across
// repos. Used by the UI to decorate the repo switcher with a "session
// baseline active" indicator per repo.
func (d *DB) SCBaselineListSession(ctx context.Context, sessionID string) ([]SourceControlBaseline, error) {
	const q = `
		SELECT session_id, repo_path, head_sha, created_at
		  FROM source_control_baselines
		 WHERE session_id = $1
		 ORDER BY repo_path
	`
	rows, err := d.Pool.Query(ctx, q, sessionID)
	if err != nil {
		return nil, fmt.Errorf("store: SCBaselineListSession: %w", err)
	}
	defer rows.Close()
	var out []SourceControlBaseline
	for rows.Next() {
		var b SourceControlBaseline
		if err := rows.Scan(&b.SessionID, &b.RepoPath, &b.HeadSHA, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: SCBaselineListSession scan: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// SCBaselineDelete clears one baseline. Returns false when there was
// nothing to delete — callers may ignore. No-op on nonexistent rows.
func (d *DB) SCBaselineDelete(ctx context.Context, sessionID, repoPath string) (bool, error) {
	const q = `
		DELETE FROM source_control_baselines
		 WHERE session_id = $1 AND repo_path = $2
	`
	tag, err := d.Pool.Exec(ctx, q, sessionID, repoPath)
	if err != nil {
		return false, fmt.Errorf("store: SCBaselineDelete: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
