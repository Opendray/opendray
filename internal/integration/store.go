package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type store struct{ pool *pgxpool.Pool }

func newStore(pool *pgxpool.Pool) *store { return &store{pool: pool} }

func (s *store) Insert(ctx context.Context, i Integration) error {
	scopesJSON, err := json.Marshal(i.Scopes)
	if err != nil {
		return fmt.Errorf("marshal scopes: %w", err)
	}
	if scopesJSON == nil {
		scopesJSON = []byte("[]")
	}
	policy := i.MemoryPolicy
	if policy == "" {
		policy = MemoryPolicyQuarantine
	}
	mcpServers := []byte(i.MCPServers)
	if len(mcpServers) == 0 {
		mcpServers = []byte("[]")
	}
	_, err = s.pool.Exec(ctx, `
        INSERT INTO integrations
            (id, name, base_url, route_prefix, api_key_hash, scopes, version,
             enabled, health_status, created_at, is_system, memory_policy,
             default_provider_id, default_model, default_claude_account_id,
             mcp_servers, system_prompt, bypass_permissions)
        VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, $12,
                $13, $14, $15, $16::jsonb, $17, $18)`,
		i.ID, i.Name, i.BaseURL, i.RoutePrefix, i.apiKeyHash, scopesJSON,
		nullIfEmpty(i.Version), i.Enabled, string(i.HealthStatus), i.CreatedAt,
		i.IsSystem, string(policy),
		i.DefaultProviderID, i.DefaultModel, i.DefaultClaudeAccountID,
		mcpServers, i.SystemPrompt, i.BypassPermissions)
	if err != nil {
		return fmt.Errorf("insert integration: %w", err)
	}
	return nil
}

func (s *store) Get(ctx context.Context, id string) (Integration, error) {
	return s.scan(s.pool.QueryRow(ctx, selectStmt+` WHERE id=$1`, id))
}

func (s *store) GetByPrefix(ctx context.Context, prefix string) (Integration, error) {
	return s.scan(s.pool.QueryRow(ctx, selectStmt+` WHERE route_prefix=$1`, prefix))
}

func (s *store) List(ctx context.Context) ([]Integration, error) {
	rows, err := s.pool.Query(ctx, selectStmt+` ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list integrations: %w", err)
	}
	defer rows.Close()
	var out []Integration
	for rows.Next() {
		i, err := s.scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *store) ListEnabled(ctx context.Context) ([]Integration, error) {
	rows, err := s.pool.Query(ctx,
		selectStmt+` WHERE enabled = TRUE ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list enabled integrations: %w", err)
	}
	defer rows.Close()
	var out []Integration
	for rows.Next() {
		i, err := s.scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// Update applies partial changes; pass nil to leave a field alone.
func (s *store) Update(ctx context.Context, id string, patch UpdatePatch) error {
	if patch.BaseURL != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE integrations SET base_url=$1 WHERE id=$2`, *patch.BaseURL, id); err != nil {
			return fmt.Errorf("update base_url: %w", err)
		}
	}
	if patch.Scopes != nil {
		raw, err := json.Marshal(*patch.Scopes)
		if err != nil {
			return fmt.Errorf("marshal scopes: %w", err)
		}
		if _, err := s.pool.Exec(ctx,
			`UPDATE integrations SET scopes=$1::jsonb WHERE id=$2`, raw, id); err != nil {
			return fmt.Errorf("update scopes: %w", err)
		}
	}
	if patch.Version != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE integrations SET version=$1 WHERE id=$2`,
			nullIfEmpty(*patch.Version), id); err != nil {
			return fmt.Errorf("update version: %w", err)
		}
	}
	if patch.Enabled != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE integrations SET enabled=$1 WHERE id=$2`,
			*patch.Enabled, id); err != nil {
			return fmt.Errorf("update enabled: %w", err)
		}
	}
	if patch.MemoryPolicy != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE integrations SET memory_policy=$1 WHERE id=$2`,
			string(*patch.MemoryPolicy), id); err != nil {
			return fmt.Errorf("update memory_policy: %w", err)
		}
	}
	if patch.DefaultProviderID != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE integrations SET default_provider_id=$1 WHERE id=$2`,
			*patch.DefaultProviderID, id); err != nil {
			return fmt.Errorf("update default_provider_id: %w", err)
		}
	}
	if patch.DefaultModel != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE integrations SET default_model=$1 WHERE id=$2`,
			*patch.DefaultModel, id); err != nil {
			return fmt.Errorf("update default_model: %w", err)
		}
	}
	if patch.DefaultClaudeAccountID != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE integrations SET default_claude_account_id=$1 WHERE id=$2`,
			*patch.DefaultClaudeAccountID, id); err != nil {
			return fmt.Errorf("update default_claude_account_id: %w", err)
		}
	}
	if patch.MCPServers != nil {
		raw := []byte(*patch.MCPServers)
		if len(raw) == 0 {
			raw = []byte("[]")
		}
		if _, err := s.pool.Exec(ctx,
			`UPDATE integrations SET mcp_servers=$1::jsonb WHERE id=$2`,
			raw, id); err != nil {
			return fmt.Errorf("update mcp_servers: %w", err)
		}
	}
	if patch.SystemPrompt != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE integrations SET system_prompt=$1 WHERE id=$2`,
			*patch.SystemPrompt, id); err != nil {
			return fmt.Errorf("update system_prompt: %w", err)
		}
	}
	if patch.BypassPermissions != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE integrations SET bypass_permissions=$1 WHERE id=$2`,
			*patch.BypassPermissions, id); err != nil {
			return fmt.Errorf("update bypass_permissions: %w", err)
		}
	}
	return nil
}

// UpdateAPIKey rotates the bcrypt hash and stamps rotated_at.
func (s *store) UpdateAPIKey(ctx context.Context, id, hash string) error {
	res, err := s.pool.Exec(ctx,
		`UPDATE integrations SET api_key_hash=$1, rotated_at=NOW() WHERE id=$2`,
		hash, id)
	if err != nil {
		return fmt.Errorf("update api key: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateHealth records the latest probe result.
func (s *store) UpdateHealth(ctx context.Context, id string, status HealthStatus, payload map[string]any, when time.Time) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal health payload: %w", err)
	}
	if payload == nil {
		raw = []byte("null")
	}
	_, err = s.pool.Exec(ctx, `
        UPDATE integrations
        SET health_status = $1,
            health_payload = $2::jsonb,
            health_last_seen = $3
        WHERE id = $4`,
		string(status), raw, when, id)
	if err != nil {
		return fmt.Errorf("update health: %w", err)
	}
	return nil
}

func (s *store) Delete(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx, `DELETE FROM integrations WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete integration: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const selectStmt = `
    SELECT id, name, base_url, route_prefix, api_key_hash,
           COALESCE(scopes, '[]'::jsonb), COALESCE(version, ''),
           enabled, health_status, health_payload,
           health_last_seen, created_at, rotated_at,
           COALESCE(is_system, FALSE),
           COALESCE(memory_policy, 'quarantine'),
           COALESCE(default_provider_id, ''),
           COALESCE(default_model, ''),
           COALESCE(default_claude_account_id, ''),
           COALESCE(mcp_servers, '[]'::jsonb),
           COALESCE(system_prompt, ''),
           COALESCE(bypass_permissions, FALSE)
    FROM integrations`

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *store) scan(row rowScanner) (Integration, error) {
	i, err := s.scanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Integration{}, ErrNotFound
	}
	return i, err
}

func (s *store) scanRow(row rowScanner) (Integration, error) {
	var (
		i              Integration
		scopesRaw      []byte
		healthStatus   string
		healthRaw      []byte
		healthLastSeen sql.NullTime
		rotatedAt      sql.NullTime
		memoryPolicy   string
		mcpServersRaw  []byte
	)
	err := row.Scan(
		&i.ID, &i.Name, &i.BaseURL, &i.RoutePrefix, &i.apiKeyHash,
		&scopesRaw, &i.Version, &i.Enabled, &healthStatus, &healthRaw,
		&healthLastSeen, &i.CreatedAt, &rotatedAt, &i.IsSystem,
		&memoryPolicy,
		&i.DefaultProviderID, &i.DefaultModel, &i.DefaultClaudeAccountID,
		&mcpServersRaw, &i.SystemPrompt, &i.BypassPermissions,
	)
	if err != nil {
		return Integration{}, err
	}
	i.MemoryPolicy = MemoryPolicy(memoryPolicy)
	if len(mcpServersRaw) == 0 {
		mcpServersRaw = []byte("[]")
	}
	i.MCPServers = json.RawMessage(mcpServersRaw)
	_ = json.Unmarshal(scopesRaw, &i.Scopes)
	if i.Scopes == nil {
		i.Scopes = []string{}
	}
	// Consumer-only integrations were stored with a synthetic
	// "_consumer_<id>" route prefix to satisfy the UNIQUE NOT NULL
	// constraint. Blank it on read so callers (UI, demo client)
	// see "no proxy" cleanly.
	if strings.HasPrefix(i.RoutePrefix, "_consumer_") {
		i.RoutePrefix = ""
	}
	i.HealthStatus = HealthStatus(healthStatus)
	if len(healthRaw) > 0 && string(healthRaw) != "null" {
		_ = json.Unmarshal(healthRaw, &i.HealthPayload)
	}
	if healthLastSeen.Valid {
		t := healthLastSeen.Time
		i.HealthLastSeen = &t
	}
	if rotatedAt.Valid {
		t := rotatedAt.Time
		i.RotatedAt = &t
	}
	return i, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// UpdatePatch carries the optional fields for store.Update.
type UpdatePatch struct {
	BaseURL      *string
	Scopes       *[]string
	Version      *string
	Enabled      *bool
	MemoryPolicy *MemoryPolicy

	// Spawn defaults for sessions this integration creates. A non-nil
	// pointer sets the column (empty string clears the default).
	DefaultProviderID      *string
	DefaultModel           *string
	DefaultClaudeAccountID *string

	// Provider-agnostic spawn profile. A non-nil pointer sets the column
	// (empty mcp_servers / system_prompt clears it).
	MCPServers        *json.RawMessage
	SystemPrompt      *string
	BypassPermissions *bool
}
