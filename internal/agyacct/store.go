package agyacct

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
    SELECT id, name, display_name, config_dir, description,
           enabled, created_at, updated_at
    FROM antigravity_accounts`

func (s *store) Insert(ctx context.Context, a Account) (Account, error) {
	row := s.pool.QueryRow(ctx, `
        INSERT INTO antigravity_accounts
            (name, display_name, config_dir, description, enabled)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, name, display_name, config_dir, description,
                  enabled, created_at, updated_at`,
		a.Name, a.DisplayName, a.ConfigDir, a.Description, a.Enabled,
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
		return nil, fmt.Errorf("list antigravity accounts: %w", err)
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
        UPDATE antigravity_accounts SET
            name = $2, display_name = $3, config_dir = $4,
            description = $5, enabled = $6, updated_at = NOW()
        WHERE id = $1
        RETURNING id, name, display_name, config_dir, description,
                  enabled, created_at, updated_at`,
		a.ID, a.Name, a.DisplayName, a.ConfigDir, a.Description, a.Enabled,
	)
	return scan(row)
}

func (s *store) Delete(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx, `DELETE FROM antigravity_accounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete antigravity account: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// nonTerminalStates are the session states that count toward
// ActiveSessions. Kept in sync with session.IsTerminal().
const nonTerminalStates = `('running', 'starting', 'idle')`

// sessionStats is the per-account result of sessionLoad.
type sessionStats struct {
	ActiveSessions int
	LastUsedAt     *time.Time // nil = no session ever pinned to this id
}

// sessionLoad returns one row per antigravity_accounts entry with active-
// session count and the most recent started_at for any session ever
// pinned to it. LEFT JOIN so accounts with zero sessions still appear.
func (s *store) sessionLoad(ctx context.Context) (map[string]sessionStats, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT aa.id,
               COUNT(s.id) FILTER (WHERE s.state IN `+nonTerminalStates+`) AS active_sessions,
               MAX(s.started_at)                                            AS last_used_at
          FROM antigravity_accounts aa
          LEFT JOIN sessions s ON s.antigravity_account_id = aa.id
         GROUP BY aa.id`)
	if err != nil {
		return nil, fmt.Errorf("session-load query: %w", err)
	}
	defer rows.Close()
	out := make(map[string]sessionStats)
	for rows.Next() {
		var (
			id     string
			active int
			last   *time.Time
		)
		if err := rows.Scan(&id, &active, &last); err != nil {
			return nil, fmt.Errorf("scan session-load row: %w", err)
		}
		out[id] = sessionStats{ActiveSessions: active, LastUsedAt: last}
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scan(s scanner) (Account, error) {
	var a Account
	err := s.Scan(
		&a.ID, &a.Name, &a.DisplayName, &a.ConfigDir,
		&a.Description, &a.Enabled, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("scan antigravity account: %w", err)
	}
	return a, nil
}
