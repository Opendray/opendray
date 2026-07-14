package checkpoint

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// store is the package-private DB layer for session_checkpoints.
type store struct{ pool *pgxpool.Pool }

func newStore(pool *pgxpool.Pool) *store { return &store{pool: pool} }

const checkpointSelect = `
    SELECT id, session_id, created_at, trigger, cwd, is_git,
           COALESCE(git_head, ''), git_dirty, diff_bytes,
           untracked_files, untracked_bytes, input_bytes, truncated,
           storage_path, COALESCE(note, '')
    FROM session_checkpoints`

func (s *store) insert(ctx context.Context, cp Checkpoint) error {
	_, err := s.pool.Exec(ctx, `
        INSERT INTO session_checkpoints
            (id, session_id, created_at, trigger, cwd, is_git, git_head,
             git_dirty, diff_bytes, untracked_files, untracked_bytes,
             input_bytes, truncated, storage_path, note)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		cp.ID, cp.SessionID, cp.CreatedAt, string(cp.Trigger), cp.Cwd, cp.IsGit,
		nullableString(cp.GitHead), cp.GitDirty, cp.DiffBytes, cp.UntrackedFiles,
		cp.UntrackedBytes, cp.InputBytes, cp.Truncated, cp.StoragePath,
		nullableString(cp.Note))
	if err != nil {
		return fmt.Errorf("checkpoint: insert: %w", err)
	}
	return nil
}

func (s *store) get(ctx context.Context, id string) (Checkpoint, error) {
	row := s.pool.QueryRow(ctx, checkpointSelect+` WHERE id=$1`, id)
	return scanCheckpoint(row)
}

// listForSession returns a session's checkpoints, newest first.
func (s *store) listForSession(ctx context.Context, sessionID string) ([]Checkpoint, error) {
	rows, err := s.pool.Query(ctx, checkpointSelect+
		` WHERE session_id=$1 ORDER BY created_at DESC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("checkpoint: list: %w", err)
	}
	defer rows.Close()
	var out []Checkpoint
	for rows.Next() {
		cp, err := scanCheckpoint(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cp)
	}
	return out, rows.Err()
}

// delete removes the metadata row and returns its storage_path so the
// caller can reap the on-disk payload.
func (s *store) delete(ctx context.Context, id string) (string, error) {
	var path string
	err := s.pool.QueryRow(ctx,
		`DELETE FROM session_checkpoints WHERE id=$1 RETURNING storage_path`, id).
		Scan(&path)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("checkpoint: delete: %w", err)
	}
	return path, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCheckpoint(row rowScanner) (Checkpoint, error) {
	var (
		cp      Checkpoint
		trigger string
		gitHead string
		storage string
		note    string
	)
	err := row.Scan(&cp.ID, &cp.SessionID, &cp.CreatedAt, &trigger, &cp.Cwd,
		&cp.IsGit, &gitHead, &cp.GitDirty, &cp.DiffBytes, &cp.UntrackedFiles,
		&cp.UntrackedBytes, &cp.InputBytes, &cp.Truncated, &storage, &note)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checkpoint{}, ErrNotFound
	}
	if err != nil {
		return Checkpoint{}, fmt.Errorf("checkpoint: scan: %w", err)
	}
	cp.Trigger = Trigger(trigger)
	cp.GitHead = gitHead
	cp.StoragePath = storage
	cp.Note = note
	return cp, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
