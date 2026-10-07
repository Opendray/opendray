package opencodeacct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// repo is the persistence surface the Service needs. *store implements
// it against Postgres; tests inject an in-memory fake.
type repo interface {
	Insert(ctx context.Context, a Account) (Account, error)
	Get(ctx context.Context, id string) (Account, error)
	GetByName(ctx context.Context, name string) (Account, error)
	List(ctx context.Context) ([]Account, error)
	Update(ctx context.Context, a Account) (Account, error)
	Delete(ctx context.Context, id string) error
	sessionLoad(ctx context.Context) (map[string]sessionStats, error)
}

type store struct{ pool *pgxpool.Pool }

func newStore(pool *pgxpool.Pool) *store { return &store{pool: pool} }

const accountCols = `id, name, display_name, description, credentials_enc,
           providers, enabled, created_at, updated_at`

const accountSelect = `SELECT ` + accountCols + ` FROM opencode_accounts`

func (s *store) Insert(ctx context.Context, a Account) (Account, error) {
	provs, err := marshalProviders(a.Providers)
	if err != nil {
		return Account{}, err
	}
	row := s.pool.QueryRow(ctx, `
        INSERT INTO opencode_accounts
            (name, display_name, description, credentials_enc, providers, enabled)
        VALUES ($1, $2, $3, $4, $5::jsonb, $6)
        RETURNING `+accountCols,
		a.Name, a.DisplayName, a.Description, a.credentialsEnc, provs, a.Enabled,
	)
	return scan(row)
}

func (s *store) Get(ctx context.Context, id string) (Account, error) {
	return scan(s.pool.QueryRow(ctx, accountSelect+` WHERE id = $1`, id))
}

func (s *store) GetByName(ctx context.Context, name string) (Account, error) {
	return scan(s.pool.QueryRow(ctx, accountSelect+` WHERE name = $1`, name))
}

func (s *store) List(ctx context.Context) ([]Account, error) {
	rows, err := s.pool.Query(ctx, accountSelect+` ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list opencode accounts: %w", err)
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
	provs, err := marshalProviders(a.Providers)
	if err != nil {
		return Account{}, err
	}
	row := s.pool.QueryRow(ctx, `
        UPDATE opencode_accounts SET
            name = $2, display_name = $3, description = $4,
            credentials_enc = $5, providers = $6::jsonb, enabled = $7,
            updated_at = NOW()
        WHERE id = $1
        RETURNING `+accountCols,
		a.ID, a.Name, a.DisplayName, a.Description, a.credentialsEnc, provs, a.Enabled,
	)
	return scan(row)
}

func (s *store) Delete(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx, `DELETE FROM opencode_accounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete opencode account: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// nonTerminalStates are the session states that count toward
// ActiveSessions. Kept in sync with session.IsTerminal().
const nonTerminalStates = `('running', 'starting', 'idle')`

type sessionStats struct {
	ActiveSessions int
	LastUsedAt     *time.Time
}

func (s *store) sessionLoad(ctx context.Context) (map[string]sessionStats, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT oa.id,
               COUNT(s.id) FILTER (WHERE s.state IN `+nonTerminalStates+`) AS active_sessions,
               MAX(s.started_at)                                            AS last_used_at
          FROM opencode_accounts oa
          LEFT JOIN sessions s ON s.opencode_account_id = oa.id
         GROUP BY oa.id`)
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

func marshalProviders(p []string) (string, error) {
	if p == nil {
		p = []string{}
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshal providers: %w", err)
	}
	return string(b), nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scan(s scanner) (Account, error) {
	var (
		a     Account
		provs []byte
	)
	err := s.Scan(
		&a.ID, &a.Name, &a.DisplayName, &a.Description, &a.credentialsEnc,
		&provs, &a.Enabled, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("scan opencode account: %w", err)
	}
	_ = json.Unmarshal(provs, &a.Providers)
	if a.Providers == nil {
		a.Providers = []string{}
	}
	return a, nil
}
