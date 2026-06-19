package autoloop

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStore is the pgx-backed persistence for loops + their iteration audit.
// It satisfies the Store interface the engine depends on. Loops are persisted
// so they survive a gateway restart (ReconcileStartup re-arms running loops).
type PgStore struct {
	pool *pgxpool.Pool
}

// NewPgStore wraps a pool.
func NewPgStore(pool *pgxpool.Pool) *PgStore { return &PgStore{pool: pool} }

const loopColumns = `id, session_id, origin, COALESCE(integration_id, ''), kind, status,
	goal, prompt, COALESCE(interval_seconds, 0), max_iterations, deadline_at,
	failure_cap, COALESCE(judge_task, ''), iteration,
	COALESCE(last_verdict, ''), COALESCE(last_reason, ''),
	created_at, started_at, ended_at`

func scanLoop(row pgx.Row) (Loop, error) {
	var l Loop
	err := row.Scan(
		&l.ID, &l.SessionID, &l.Origin, &l.IntegrationID, &l.Kind, &l.Status,
		&l.Goal, &l.Prompt, &l.IntervalSeconds, &l.MaxIterations, &l.DeadlineAt,
		&l.FailureCap, &l.JudgeTask, &l.Iteration,
		&l.LastVerdict, &l.LastReason,
		&l.CreatedAt, &l.StartedAt, &l.EndedAt,
	)
	return l, err
}

// Create inserts a new loop row.
func (s *PgStore) Create(ctx context.Context, l Loop) error {
	var intervalSeconds *int
	if l.Kind == KindInterval && l.IntervalSeconds > 0 {
		v := l.IntervalSeconds
		intervalSeconds = &v
	}
	var integrationID, judgeTask *string
	if l.IntegrationID != "" {
		integrationID = &l.IntegrationID
	}
	if l.JudgeTask != "" {
		judgeTask = &l.JudgeTask
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO session_loops
			(id, session_id, origin, integration_id, kind, status, goal, prompt,
			 interval_seconds, max_iterations, deadline_at, failure_cap, judge_task,
			 iteration, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		l.ID, l.SessionID, l.Origin, integrationID, l.Kind, l.Status, l.Goal, l.Prompt,
		intervalSeconds, l.MaxIterations, l.DeadlineAt, l.FailureCap, judgeTask,
		l.Iteration, l.CreatedAt)
	if err != nil {
		return fmt.Errorf("autoloop: create loop: %w", err)
	}
	return nil
}

// Get returns one loop by id, or ErrNotFound.
func (s *PgStore) Get(ctx context.Context, id string) (Loop, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+loopColumns+` FROM session_loops WHERE id = $1`, id)
	l, err := scanLoop(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Loop{}, ErrNotFound
	}
	if err != nil {
		return Loop{}, fmt.Errorf("autoloop: get loop: %w", err)
	}
	return l, nil
}

// List returns all loops, newest first.
func (s *PgStore) List(ctx context.Context) ([]Loop, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+loopColumns+` FROM session_loops ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("autoloop: list loops: %w", err)
	}
	defer rows.Close()
	return collectLoops(rows)
}

// ListLive returns loops in a non-terminal state (running / paused). Used by
// ReconcileStartup to re-arm loops after a gateway restart.
func (s *PgStore) ListLive(ctx context.Context) ([]Loop, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+loopColumns+`
		FROM session_loops WHERE status IN ('running','paused') ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("autoloop: list live loops: %w", err)
	}
	defer rows.Close()
	return collectLoops(rows)
}

func collectLoops(rows pgx.Rows) ([]Loop, error) {
	var out []Loop
	for rows.Next() {
		l, err := scanLoop(rows)
		if err != nil {
			return nil, fmt.Errorf("autoloop: scan loop: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// MarkRunning flips a loop to running and stamps started_at (idempotent on
// started_at: only set the first time).
func (s *PgStore) MarkRunning(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE session_loops
		   SET status = 'running', started_at = COALESCE(started_at, NOW())
		 WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("autoloop: mark running: %w", err)
	}
	return nil
}

// SetStatus updates the loop status + last reason. Terminal states stamp
// ended_at so the audit shows when the loop finished.
func (s *PgStore) SetStatus(ctx context.Context, id string, st Status, reason string) error {
	var endedAt *time.Time
	if st.IsTerminal() {
		now := time.Now()
		endedAt = &now
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE session_loops
		   SET status = $2,
		       last_reason = NULLIF($3, ''),
		       ended_at = COALESCE($4, ended_at)
		 WHERE id = $1`, id, st, reason, endedAt)
	if err != nil {
		return fmt.Errorf("autoloop: set status: %w", err)
	}
	return nil
}

// SaveProgress records the loop's iteration count + latest verdict/reason.
func (s *PgStore) SaveProgress(ctx context.Context, id string, iteration int, verdict, reason string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE session_loops
		   SET iteration = $2, last_verdict = NULLIF($3, ''), last_reason = NULLIF($4, '')
		 WHERE id = $1`, id, iteration, verdict, reason)
	if err != nil {
		return fmt.Errorf("autoloop: save progress: %w", err)
	}
	return nil
}

// AppendRun opens an audit row for one iteration and returns its id.
func (s *PgStore) AppendRun(ctx context.Context, loopID string, iteration int, prompt string) (int64, error) {
	var runID int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO session_loop_runs (loop_id, iteration, prompt)
		VALUES ($1, $2, $3) RETURNING id`, loopID, iteration, prompt).Scan(&runID)
	if err != nil {
		return 0, fmt.Errorf("autoloop: append run: %w", err)
	}
	return runID, nil
}

// FinishRun closes an audit row with the iteration's verdict + reason.
func (s *PgStore) FinishRun(ctx context.Context, runID int64, verdict, reason string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE session_loop_runs
		   SET verdict = NULLIF($2, ''), reason = NULLIF($3, ''), ended_at = NOW()
		 WHERE id = $1`, runID, verdict, reason)
	if err != nil {
		return fmt.Errorf("autoloop: finish run: %w", err)
	}
	return nil
}

// Run is one persisted iteration audit row (read side for the API).
type Run struct {
	ID        int64      `json:"id"`
	LoopID    string     `json:"loop_id"`
	Iteration int        `json:"iteration"`
	Prompt    string     `json:"prompt"`
	Verdict   string     `json:"verdict,omitempty"`
	Reason    string     `json:"reason,omitempty"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

// ListRuns returns the iteration audit for a loop, oldest first.
func (s *PgStore) ListRuns(ctx context.Context, loopID string) ([]Run, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, loop_id, iteration, prompt,
		       COALESCE(verdict, ''), COALESCE(reason, ''), started_at, ended_at
		  FROM session_loop_runs WHERE loop_id = $1 ORDER BY iteration ASC, id ASC`, loopID)
	if err != nil {
		return nil, fmt.Errorf("autoloop: list runs: %w", err)
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		var r Run
		if err := rows.Scan(&r.ID, &r.LoopID, &r.Iteration, &r.Prompt,
			&r.Verdict, &r.Reason, &r.StartedAt, &r.EndedAt); err != nil {
			return nil, fmt.Errorf("autoloop: scan run: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
