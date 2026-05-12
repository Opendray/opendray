package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// sessionStore is the package-private DB layer for sessions. Manager owns
// it; subsystem callers use Manager's API instead.
type sessionStore struct{ pool *pgxpool.Pool }

func newStore(pool *pgxpool.Pool) *sessionStore { return &sessionStore{pool: pool} }

const sessionSelect = `
    SELECT id, COALESCE(name, ''), provider_id, cwd, args, state,
           COALESCE(pid, 0),
           COALESCE(claude_account_id, ''), COALESCE(claude_session_id, ''),
           COALESCE(parent_session_id, ''),
           started_at, ended_at, exit_code
    FROM sessions`

func (s *sessionStore) Insert(ctx context.Context, sess Session) error {
	argsJSON, err := json.Marshal(sess.Args)
	if err != nil {
		return fmt.Errorf("marshal args: %w", err)
	}
	if argsJSON == nil {
		argsJSON = []byte("[]")
	}
	_, err = s.pool.Exec(ctx, `
        INSERT INTO sessions
            (id, name, provider_id, cwd, args, state, pid,
             claude_account_id, claude_session_id, parent_session_id,
             started_at)
        VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10, $11)`,
		sess.ID, nullableString(sess.Name), sess.ProviderID, sess.Cwd,
		argsJSON, string(sess.State), nullableInt(sess.PID),
		nullableString(sess.ClaudeAccountID), nullableString(sess.ClaudeSessionID),
		nullableString(sess.ParentSessionID),
		sess.StartedAt)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *sessionStore) Get(ctx context.Context, id string) (Session, error) {
	row := s.pool.QueryRow(ctx, sessionSelect+` WHERE id=$1`, id)
	return scanSession(row)
}

func (s *sessionStore) List(ctx context.Context) ([]Session, error) {
	rows, err := s.pool.Query(ctx, sessionSelect+` ORDER BY started_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

// MarkTerminal sets state + ended_at + exit_code, but only if the row
// is not already terminal. Idempotent against repeat exit-detector
// wakeups. Used for both 'stopped' (user-initiated) and 'ended'
// (process exited on its own) transitions.
func (s *sessionStore) MarkTerminal(ctx context.Context, id string, state State, exitCode int) error {
	_, err := s.pool.Exec(ctx, `
        UPDATE sessions
        SET state=$1, ended_at=NOW(), exit_code=$2
        WHERE id=$3 AND state NOT IN ('stopped', 'ended')`,
		string(state), exitCode, id)
	if err != nil {
		return fmt.Errorf("mark terminal: %w", err)
	}
	return nil
}

// Reactivate transitions a terminal row back into running with a fresh
// PID and started_at. Used by Manager.Start.
func (s *sessionStore) Reactivate(ctx context.Context, id string, pid int) error {
	_, err := s.pool.Exec(ctx, `
        UPDATE sessions
        SET state='running', pid=$1, started_at=NOW(),
            ended_at=NULL, exit_code=NULL
        WHERE id=$2`, pid, id)
	if err != nil {
		return fmt.Errorf("reactivate session: %w", err)
	}
	return nil
}

// SetClaudeSessionID updates the agent-side session UUID after spawn.
// The Manager calls this when the provider's PrepareOutput carries a
// freshly-generated UUID (claude / gemini via --session-id), so the
// M18 transcript reader can locate the right *.jsonl file directly
// instead of relying on mtime heuristics.
func (s *sessionStore) SetClaudeSessionID(ctx context.Context, id, claudeSessionID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE sessions SET claude_session_id=$1 WHERE id=$2`,
		claudeSessionID, id)
	if err != nil {
		return fmt.Errorf("set claude_session_id: %w", err)
	}
	return nil
}

// MarkAllRunningAsEnded flips every non-terminal session row to
// 'ended' with exit_code=-1. Called once at gateway startup: a fresh
// Manager has an empty in-memory map, so any row claiming to be
// running/idle/pending is a leftover whose PTY exited with the old
// gateway process. Without this, the web UI keeps trying to attach
// WS streams to dead sessions and hangs on "reconnecting…" forever.
func (s *sessionStore) MarkAllRunningAsEnded(ctx context.Context) (int64, error) {
	res, err := s.pool.Exec(ctx, `
        UPDATE sessions
        SET state='ended', ended_at=NOW(), exit_code=-1
        WHERE state IN ('pending', 'running', 'idle')`)
	if err != nil {
		return 0, fmt.Errorf("reconcile stale sessions: %w", err)
	}
	return res.RowsAffected(), nil
}

// UpdateClaudeAccount rebinds the session's claude_account_id without
// touching state/pid/etc. Used by Manager.SwitchClaudeAccount after a
// successful respawn under a new credential.
func (s *sessionStore) UpdateClaudeAccount(ctx context.Context, id, accountID string) error {
	res, err := s.pool.Exec(ctx, `
        UPDATE sessions SET claude_account_id=$1 WHERE id=$2`,
		nullableString(accountID), id)
	if err != nil {
		return fmt.Errorf("update claude account: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete permanently removes the row. Caller must ensure the session
// is no longer running (Manager.Stop first).
func (s *sessionStore) Delete(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSession(row rowScanner) (Session, error) {
	var (
		s        Session
		argsJSON []byte
		endedAt  sql.NullTime
		exitCode sql.NullInt32
		stateStr string
	)
	err := row.Scan(&s.ID, &s.Name, &s.ProviderID, &s.Cwd, &argsJSON,
		&stateStr, &s.PID, &s.ClaudeAccountID, &s.ClaudeSessionID,
		&s.ParentSessionID,
		&s.StartedAt, &endedAt, &exitCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("scan session: %w", err)
	}
	s.State = State(stateStr)
	_ = json.Unmarshal(argsJSON, &s.Args)
	if endedAt.Valid {
		t := endedAt.Time
		s.EndedAt = &t
	}
	if exitCode.Valid {
		c := int(exitCode.Int32)
		s.ExitCode = &c
	}
	return s, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableInt(i int) any {
	if i == 0 {
		return nil
	}
	return i
}
