package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ClaudeAccount is one row in claude_accounts — metadata for an account
// managed by the `claude-acc` host tool. The actual OAuth token is NOT
// stored here: it lives at TokenPath on disk (chmod 600) and is read only
// at session-spawn time.
type ClaudeAccount struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	ConfigDir   string    `json:"configDir"`
	TokenPath   string    `json:"tokenPath"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (d *DB) CreateClaudeAccount(ctx context.Context, a ClaudeAccount) (ClaudeAccount, error) {
	row := d.Pool.QueryRow(ctx,
		`INSERT INTO claude_accounts (name, display_name, config_dir, token_path, description, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, display_name, config_dir, token_path, description, enabled, created_at, updated_at`,
		a.Name, a.DisplayName, a.ConfigDir, a.TokenPath, a.Description, a.Enabled,
	)
	return scanClaudeAccount(row)
}

func (d *DB) GetClaudeAccount(ctx context.Context, id string) (ClaudeAccount, error) {
	row := d.Pool.QueryRow(ctx,
		`SELECT id, name, display_name, config_dir, token_path, description, enabled, created_at, updated_at
		 FROM claude_accounts WHERE id = $1`, id)
	return scanClaudeAccount(row)
}

func (d *DB) GetClaudeAccountByName(ctx context.Context, name string) (ClaudeAccount, error) {
	row := d.Pool.QueryRow(ctx,
		`SELECT id, name, display_name, config_dir, token_path, description, enabled, created_at, updated_at
		 FROM claude_accounts WHERE name = $1`, name)
	return scanClaudeAccount(row)
}

func (d *DB) ListClaudeAccounts(ctx context.Context) ([]ClaudeAccount, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, name, display_name, config_dir, token_path, description, enabled, created_at, updated_at
		 FROM claude_accounts ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("store: list claude accounts: %w", err)
	}
	defer rows.Close()

	out := []ClaudeAccount{}
	for rows.Next() {
		a, err := scanClaudeAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (d *DB) UpdateClaudeAccount(ctx context.Context, id string, a ClaudeAccount) (ClaudeAccount, error) {
	row := d.Pool.QueryRow(ctx,
		`UPDATE claude_accounts SET
		    name = $2, display_name = $3, config_dir = $4, token_path = $5,
		    description = $6, enabled = $7, updated_at = now()
		 WHERE id = $1
		 RETURNING id, name, display_name, config_dir, token_path, description, enabled, created_at, updated_at`,
		id, a.Name, a.DisplayName, a.ConfigDir, a.TokenPath, a.Description, a.Enabled,
	)
	return scanClaudeAccount(row)
}

func (d *DB) SetClaudeAccountEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE claude_accounts SET enabled = $2, updated_at = now() WHERE id = $1`,
		id, enabled)
	if err != nil {
		return fmt.Errorf("store: set claude account enabled: %w", err)
	}
	return nil
}

func (d *DB) DeleteClaudeAccount(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM claude_accounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: delete claude account: %w", err)
	}
	return nil
}

// UpdateSessionClaudeAccount sets or clears the claude_account_id binding
// on a session. Pass empty string to clear (NULL).
func (d *DB) UpdateSessionClaudeAccount(ctx context.Context, sessionID, accountID string) error {
	var err error
	if accountID == "" {
		_, err = d.Pool.Exec(ctx,
			`UPDATE sessions SET claude_account_id = NULL, last_active_at = now() WHERE id = $1`,
			sessionID)
	} else {
		_, err = d.Pool.Exec(ctx,
			`UPDATE sessions SET claude_account_id = $2, last_active_at = now() WHERE id = $1`,
			sessionID, accountID)
	}
	if err != nil {
		return fmt.Errorf("store: update session claude account: %w", err)
	}
	return nil
}

func scanClaudeAccount(s scannable) (ClaudeAccount, error) {
	var a ClaudeAccount
	err := s.Scan(
		&a.ID, &a.Name, &a.DisplayName, &a.ConfigDir, &a.TokenPath,
		&a.Description, &a.Enabled, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ClaudeAccount{}, fmt.Errorf("store: claude account not found")
		}
		return ClaudeAccount{}, fmt.Errorf("store: scan claude account: %w", err)
	}
	return a, nil
}
