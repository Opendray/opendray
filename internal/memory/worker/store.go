package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// store wraps the pgxpool with memory_workers + memory_worker_calls
// operations. Unexported; access goes through Resolver / Registry.
type store struct {
	pool *pgxpool.Pool
}

func newStore(pool *pgxpool.Pool) *store {
	return &store{pool: pool}
}

// Get returns the config for one task. Returns pgx.ErrNoRows if
// the row doesn't exist (the seed migration inserts all four, so
// this should only happen on tampered DBs).
func (s *store) Get(ctx context.Context, task TaskKind) (Config, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT task, kind, COALESCE(summarizer_id, ''),
		       COALESCE(provider_id, ''), COALESCE(account_id, ''),
		       enabled, updated_at
		  FROM memory_workers
		 WHERE task = $1`, string(task))
	var c Config
	var taskStr, kindStr string
	err := row.Scan(&taskStr, &kindStr, &c.SummarizerID, &c.ProviderID,
		&c.AccountID, &c.Enabled, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Config{}, ErrNoWorkerConfigured
		}
		return Config{}, fmt.Errorf("memory worker store: get: %w", err)
	}
	c.Task = TaskKind(taskStr)
	c.Kind = WorkerKind(kindStr)
	return c, nil
}

// List returns every task's config (always 4 rows after seed).
func (s *store) List(ctx context.Context) ([]Config, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT task, kind, COALESCE(summarizer_id, ''),
		       COALESCE(provider_id, ''), COALESCE(account_id, ''),
		       enabled, updated_at
		  FROM memory_workers
		 ORDER BY task`)
	if err != nil {
		return nil, fmt.Errorf("memory worker store: list: %w", err)
	}
	defer rows.Close()
	var out []Config
	for rows.Next() {
		var c Config
		var taskStr, kindStr string
		if err := rows.Scan(&taskStr, &kindStr, &c.SummarizerID, &c.ProviderID,
			&c.AccountID, &c.Enabled, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.Task = TaskKind(taskStr)
		c.Kind = WorkerKind(kindStr)
		out = append(out, c)
	}
	return out, rows.Err()
}

// Upsert writes the task's config. Operator-facing UI calls this
// when the user picks a different worker kind / provider.
func (s *store) Upsert(ctx context.Context, c Config) error {
	if err := c.Valid(); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO memory_workers
		    (task, kind, summarizer_id, provider_id, account_id, enabled, updated_at)
		VALUES
		    ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), $6, NOW())
		ON CONFLICT (task) DO UPDATE SET
		    kind          = EXCLUDED.kind,
		    summarizer_id = EXCLUDED.summarizer_id,
		    provider_id   = EXCLUDED.provider_id,
		    account_id    = EXCLUDED.account_id,
		    enabled       = EXCLUDED.enabled,
		    updated_at    = NOW()`,
		string(c.Task), string(c.Kind), c.SummarizerID,
		c.ProviderID, c.AccountID, c.Enabled)
	if err != nil {
		return fmt.Errorf("memory worker store: upsert: %w", err)
	}
	return nil
}

// CallSummary is one row from memory_worker_calls for the UI's
// metrics list.
type CallSummary struct {
	ID           int64      `json:"id"`
	Task         TaskKind   `json:"task"`
	WorkerKind   WorkerKind `json:"worker_kind"`
	ProviderID   string     `json:"provider_id"`
	AccountID    string     `json:"account_id"`
	StartedAt    time.Time  `json:"started_at"`
	DurationMS   int64      `json:"duration_ms"`
	Success      bool       `json:"success"`
	ErrorMessage string     `json:"error_message"`
	InputBytes   int        `json:"input_bytes"`
	OutputBytes  int        `json:"output_bytes"`
	TokensIn     int        `json:"tokens_in"`
	TokensOut    int        `json:"tokens_out"`
}

// RecordCall persists a metrics row. Best-effort: failures here
// don't propagate to the caller (we don't want telemetry errors
// to mask the underlying success / failure of the LLM call).
func (s *store) RecordCall(ctx context.Context, c CallSummary) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO memory_worker_calls
		    (task, worker_kind, provider_id, account_id,
		     duration_ms, success, error_message,
		     input_bytes, output_bytes, tokens_in, tokens_out)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		string(c.Task), string(c.WorkerKind), c.ProviderID, c.AccountID,
		c.DurationMS, c.Success, c.ErrorMessage,
		c.InputBytes, c.OutputBytes, c.TokensIn, c.TokensOut)
	if err != nil {
		// Caller's logger isn't reachable here; silently drop —
		// the metrics layer is best-effort by design.
		_ = err
	}
}

// ListCalls returns the most recent N call rows for the UI.
// Optionally filtered by task.
func (s *store) ListCalls(ctx context.Context, task TaskKind, limit int) ([]CallSummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := `
		SELECT id, task, worker_kind, provider_id, account_id,
		       started_at, duration_ms, success, error_message,
		       input_bytes, output_bytes, tokens_in, tokens_out
		  FROM memory_worker_calls
		 WHERE ($1 = '' OR task = $1)
		 ORDER BY started_at DESC
		 LIMIT $2`
	rows, err := s.pool.Query(ctx, q, string(task), limit)
	if err != nil {
		return nil, fmt.Errorf("memory worker store: list calls: %w", err)
	}
	defer rows.Close()
	var out []CallSummary
	for rows.Next() {
		var c CallSummary
		var taskStr, kindStr string
		if err := rows.Scan(&c.ID, &taskStr, &kindStr, &c.ProviderID, &c.AccountID,
			&c.StartedAt, &c.DurationMS, &c.Success, &c.ErrorMessage,
			&c.InputBytes, &c.OutputBytes, &c.TokensIn, &c.TokensOut); err != nil {
			return nil, err
		}
		c.Task = TaskKind(taskStr)
		c.WorkerKind = WorkerKind(kindStr)
		out = append(out, c)
	}
	return out, rows.Err()
}
