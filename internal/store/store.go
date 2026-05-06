// Package store wraps the postgres connection pool and exposes the
// migration runner.
//
// Subsystem packages (session, integration, channel, ...) accept *Store
// (or just the *pgxpool.Pool from Pool()) and own their own queries —
// store/ is not a query repository.
package store

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

// DefaultMaxConns is the cap applied when caller passes 0 to Open.
// Calibrated for a single-admin home-lab deployment with a few
// integration consumers; bump via config.database.max_conns when the
// fanout grows.
const DefaultMaxConns = 16

// Open creates a pgx pool and pings the database. maxConns ≤ 0 falls
// back to DefaultMaxConns so callers can pass the raw config value
// without conditional branching.
func Open(ctx context.Context, dsn string, maxConns int) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("store: parse dsn: %w", err)
	}
	if maxConns <= 0 {
		maxConns = DefaultMaxConns
	}
	// Cap at int32 max — pgxpool.Config.MaxConns is int32 and a
	// pathological TOML value (e.g. max_conns = 3000000000) would
	// otherwise wrap to a negative int32 and silently fall back to
	// pgx's own default.
	if maxConns > math.MaxInt32 {
		maxConns = math.MaxInt32
	}
	cfg.MaxConns = int32(maxConns)
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: open pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// Ping wraps the pool's ping for /health.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
