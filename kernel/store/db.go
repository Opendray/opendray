// Package store provides PostgreSQL database access for the NTC kernel.
package store

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps a PostgreSQL connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// Config holds database connection parameters.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// DSN returns the connection string without the password (password set via config override).
func (c Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable", c.Host, c.Port, c.User, c.DBName)
}

// New creates a new DB with a connection pool.
func New(ctx context.Context, cfg Config) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("store: parse config: %w", err)
	}
	poolCfg.ConnConfig.Password = cfg.Password
	poolCfg.MaxConns = 20
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.HealthCheckPeriod = 1 * time.Minute
	poolCfg.ConnConfig.RuntimeParams["statement_timeout"] = "10000"

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("store: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &DB{Pool: pool}, nil
}

// Migrate runs all embedded SQL migrations in order.
func (d *DB) Migrate(ctx context.Context) error {
	files := []string{
		"migrations/001_init.sql",
		"migrations/002_plugins_version.sql",
		"migrations/003_plugin_config.sql",
		"migrations/004_mcp_servers.sql",
		"migrations/005_claude_accounts.sql",
		"migrations/006_claude_accounts_local_backend.sql",
		"migrations/007_rollback_local_backend.sql",
		"migrations/008_llm_providers.sql",
	}
	for _, path := range files {
		sql, err := migrationsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("store: read migration %s: %w", path, err)
		}
		if _, err := d.Pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("store: exec migration %s: %w", path, err)
		}
	}
	return nil
}

// Close shuts down the connection pool.
func (d *DB) Close() {
	d.Pool.Close()
}
