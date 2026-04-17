package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// MCPServer represents one configured MCP server row.
type MCPServer struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Transport   string            `json:"transport"` // stdio | sse | http
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	AppliesTo   []string          `json:"appliesTo"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

func (d *DB) CreateMCPServer(ctx context.Context, m MCPServer) (MCPServer, error) {
	argsJSON, _ := json.Marshal(orEmptyList(m.Args))
	envJSON, _ := json.Marshal(orEmptyMap(m.Env))
	headersJSON, _ := json.Marshal(orEmptyMap(m.Headers))
	appliesJSON, _ := json.Marshal(orDefaultApplies(m.AppliesTo))

	row := d.Pool.QueryRow(ctx,
		`INSERT INTO mcp_servers (name, description, transport, command, args, env, url, headers, applies_to, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, name, description, transport, command, args, env, url, headers, applies_to, enabled, created_at, updated_at`,
		m.Name, m.Description, m.Transport, m.Command, argsJSON, envJSON, m.URL, headersJSON, appliesJSON, m.Enabled,
	)
	return scanMCPServer(row)
}

func (d *DB) GetMCPServer(ctx context.Context, id string) (MCPServer, error) {
	row := d.Pool.QueryRow(ctx,
		`SELECT id, name, description, transport, command, args, env, url, headers, applies_to, enabled, created_at, updated_at
		 FROM mcp_servers WHERE id = $1`, id)
	return scanMCPServer(row)
}

func (d *DB) ListMCPServers(ctx context.Context) ([]MCPServer, error) {
	rows, err := d.Pool.Query(ctx,
		`SELECT id, name, description, transport, command, args, env, url, headers, applies_to, enabled, created_at, updated_at
		 FROM mcp_servers ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("store: list mcp servers: %w", err)
	}
	defer rows.Close()

	var out []MCPServer
	for rows.Next() {
		m, err := scanMCPServer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (d *DB) UpdateMCPServer(ctx context.Context, id string, m MCPServer) (MCPServer, error) {
	argsJSON, _ := json.Marshal(orEmptyList(m.Args))
	envJSON, _ := json.Marshal(orEmptyMap(m.Env))
	headersJSON, _ := json.Marshal(orEmptyMap(m.Headers))
	appliesJSON, _ := json.Marshal(orDefaultApplies(m.AppliesTo))

	row := d.Pool.QueryRow(ctx,
		`UPDATE mcp_servers SET
		    name = $2, description = $3, transport = $4, command = $5,
		    args = $6, env = $7, url = $8, headers = $9, applies_to = $10,
		    enabled = $11, updated_at = now()
		 WHERE id = $1
		 RETURNING id, name, description, transport, command, args, env, url, headers, applies_to, enabled, created_at, updated_at`,
		id, m.Name, m.Description, m.Transport, m.Command, argsJSON, envJSON, m.URL, headersJSON, appliesJSON, m.Enabled,
	)
	return scanMCPServer(row)
}

func (d *DB) SetMCPServerEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE mcp_servers SET enabled = $2, updated_at = now() WHERE id = $1`,
		id, enabled)
	if err != nil {
		return fmt.Errorf("store: set mcp server enabled: %w", err)
	}
	return nil
}

func (d *DB) DeleteMCPServer(ctx context.Context, id string) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM mcp_servers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store: delete mcp server: %w", err)
	}
	return nil
}

func scanMCPServer(s scannable) (MCPServer, error) {
	var m MCPServer
	var argsJSON, envJSON, headersJSON, appliesJSON []byte
	err := s.Scan(
		&m.ID, &m.Name, &m.Description, &m.Transport, &m.Command,
		&argsJSON, &envJSON, &m.URL, &headersJSON, &appliesJSON,
		&m.Enabled, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return MCPServer{}, fmt.Errorf("store: mcp server not found")
		}
		return MCPServer{}, fmt.Errorf("store: scan mcp server: %w", err)
	}
	_ = json.Unmarshal(argsJSON, &m.Args)
	_ = json.Unmarshal(envJSON, &m.Env)
	_ = json.Unmarshal(headersJSON, &m.Headers)
	_ = json.Unmarshal(appliesJSON, &m.AppliesTo)
	if m.Args == nil {
		m.Args = []string{}
	}
	if m.Env == nil {
		m.Env = map[string]string{}
	}
	if m.Headers == nil {
		m.Headers = map[string]string{}
	}
	if m.AppliesTo == nil {
		m.AppliesTo = []string{"*"}
	}
	return m, nil
}

func orEmptyList(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func orEmptyMap(v map[string]string) map[string]string {
	if v == nil {
		return map[string]string{}
	}
	return v
}

func orDefaultApplies(v []string) []string {
	if len(v) == 0 {
		return []string{"*"}
	}
	return v
}
