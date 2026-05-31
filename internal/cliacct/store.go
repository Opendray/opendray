package cliacct

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type store struct{ pool *pgxpool.Pool }

func newStore(pool *pgxpool.Pool) *store { return &store{pool: pool} }

const accountSelect = `
    SELECT id, name, display_name, config_dir, token_path, description,
           enabled, created_at, updated_at
    FROM claude_accounts`

func (s *store) Insert(ctx context.Context, a Account) (Account, error) {
	row := s.pool.QueryRow(ctx, `
        INSERT INTO claude_accounts
            (name, display_name, config_dir, token_path, description, enabled)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, name, display_name, config_dir, token_path, description,
                  enabled, created_at, updated_at`,
		a.Name, a.DisplayName, a.ConfigDir, a.TokenPath, a.Description, a.Enabled,
	)
	return scan(row)
}

func (s *store) Get(ctx context.Context, id string) (Account, error) {
	row := s.pool.QueryRow(ctx, accountSelect+` WHERE id = $1`, id)
	return scan(row)
}

func (s *store) GetByName(ctx context.Context, name string) (Account, error) {
	row := s.pool.QueryRow(ctx, accountSelect+` WHERE name = $1`, name)
	return scan(row)
}

func (s *store) List(ctx context.Context) ([]Account, error) {
	rows, err := s.pool.Query(ctx, accountSelect+` ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list claude accounts: %w", err)
	}
	defer rows.Close()
	out := []Account{}
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *store) Update(ctx context.Context, a Account) (Account, error) {
	row := s.pool.QueryRow(ctx, `
        UPDATE claude_accounts SET
            name = $2, display_name = $3, config_dir = $4, token_path = $5,
            description = $6, enabled = $7, updated_at = NOW()
        WHERE id = $1
        RETURNING id, name, display_name, config_dir, token_path, description,
                  enabled, created_at, updated_at`,
		a.ID, a.Name, a.DisplayName, a.ConfigDir, a.TokenPath, a.Description, a.Enabled,
	)
	return scan(row)
}

func (s *store) Delete(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx, `DELETE FROM claude_accounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete claude account: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// nonTerminalStates are the session states that count toward
// ActiveSessions and the least-loaded auto-assign heuristic. Stays
// in sync with session.IsTerminal() — anything that returns false
// there should appear here.
const nonTerminalStates = `('running', 'starting', 'idle')`

// sessionStats is the per-account result of sessionLoad. Local to the
// store; Service copies fields into Account before returning.
type sessionStats struct {
	ActiveSessions int
	LastUsedAt     *time.Time // nil = no session ever pinned to this id
}

// sessionLoad returns one row per claude_accounts entry with active-
// session count and the most recent started_at for any session ever
// pinned to it. LEFT JOIN so accounts with zero sessions still appear
// (count = 0, max = NULL). Used by Service.List so we make one DB
// round-trip per List instead of N+1.
func (s *store) sessionLoad(ctx context.Context) (map[string]sessionStats, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT ca.id,
               COUNT(s.id) FILTER (WHERE s.state IN `+nonTerminalStates+`) AS active_sessions,
               MAX(s.started_at)                                            AS last_used_at
          FROM claude_accounts ca
          LEFT JOIN sessions s ON s.claude_account_id = ca.id
         GROUP BY ca.id`)
	if err != nil {
		return nil, fmt.Errorf("session-load query: %w", err)
	}
	defer rows.Close()
	out := make(map[string]sessionStats)
	for rows.Next() {
		var (
			id     string
			active int
			last   *time.Time // pgx scans nullable TIMESTAMPTZ into a *time.Time directly
		)
		if err := rows.Scan(&id, &active, &last); err != nil {
			return nil, fmt.Errorf("scan session-load row: %w", err)
		}
		out[id] = sessionStats{ActiveSessions: active, LastUsedAt: last}
	}
	return out, rows.Err()
}

// pickLeastLoaded returns the enabled account with the smallest count
// of non-terminal sessions; ties broken by name (lexical, ascending)
// so the choice is deterministic and the operator can predict it.
// Variadic 'exclude' lets the caller pass currently-throttled account
// ids (and/or the one we're failing AWAY from) so they don't get
// picked. Returns ErrNotFound when no eligible account exists.
func (s *store) pickLeastLoaded(ctx context.Context, exclude ...string) (string, error) {
	// Use ANY($2::text[]) so an empty exclude list works (the array is
	// just zero-length). Parameterized — never string-interpolated.
	row := s.pool.QueryRow(ctx, `
        SELECT ca.id
          FROM claude_accounts ca
          LEFT JOIN sessions s ON s.claude_account_id = ca.id
                              AND s.state IN `+nonTerminalStates+`
         WHERE ca.enabled = true
           AND NOT (ca.id = ANY($1::text[]))
         GROUP BY ca.id, ca.name
         ORDER BY COUNT(s.id) ASC, ca.name ASC
         LIMIT 1`, exclude)
	var id string
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("pick least-loaded account: %w", err)
	}
	return id, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scan(s scanner) (Account, error) {
	var a Account
	err := s.Scan(
		&a.ID, &a.Name, &a.DisplayName, &a.ConfigDir, &a.TokenPath,
		&a.Description, &a.Enabled, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("scan claude account: %w", err)
	}
	return a, nil
}
