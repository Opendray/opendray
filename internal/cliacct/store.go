package cliacct

import (
	"context"
	"errors"
	"fmt"

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
