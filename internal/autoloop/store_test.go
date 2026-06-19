package autoloop

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// loopTestSchema is an isolated schema this test owns end-to-end: it is
// dropped + recreated each run, so the test never touches the real public
// tables even when OPENDRAY_DEV_DB_URL points at a populated database.
const loopTestSchema = "autoloop_test"

// loopTestDDL mirrors the table shape of migration 0069 (the memory_workers
// ALTER in that migration is validated separately by the full ephemeral
// migration run; PgStore itself never touches memory_workers).
const loopTestDDL = `
CREATE TABLE sessions (id TEXT PRIMARY KEY);
CREATE TABLE integrations (id TEXT PRIMARY KEY);
CREATE TABLE session_loops (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    origin TEXT NOT NULL DEFAULT 'operator',
    integration_id TEXT REFERENCES integrations(id) ON DELETE SET NULL,
    kind TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    goal TEXT NOT NULL DEFAULT '',
    prompt TEXT NOT NULL,
    interval_seconds INT,
    max_iterations INT NOT NULL DEFAULT 20,
    deadline_at TIMESTAMPTZ,
    failure_cap INT NOT NULL DEFAULT 3,
    judge_task TEXT,
    iteration INT NOT NULL DEFAULT 0,
    last_verdict TEXT,
    last_reason TEXT,
    config JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ
);
CREATE TABLE session_loop_runs (
    id BIGSERIAL PRIMARY KEY,
    loop_id TEXT NOT NULL REFERENCES session_loops(id) ON DELETE CASCADE,
    iteration INT NOT NULL,
    prompt TEXT NOT NULL,
    verdict TEXT,
    reason TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ
);`

func loopTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	url := os.Getenv("OPENDRAY_DEV_DB_URL")
	if url == "" {
		t.Skip("OPENDRAY_DEV_DB_URL not set; export a writable Postgres DSN to run PgStore tests")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Skipf("bad OPENDRAY_DEV_DB_URL: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = loopTestSchema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Skipf("dev DB unreachable, skipping: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("dev DB ping failed, skipping: %v", err)
	}
	for _, stmt := range []string{
		"DROP SCHEMA IF EXISTS " + loopTestSchema + " CASCADE",
		"CREATE SCHEMA " + loopTestSchema,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			pool.Close()
			t.Fatalf("schema setup (%s): %v", stmt, err)
		}
	}
	if _, err := pool.Exec(ctx, loopTestDDL); err != nil {
		pool.Close()
		t.Fatalf("ddl: %v", err)
	}
	// seed stub parents.
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(id) VALUES ('s1'); INSERT INTO integrations(id) VALUES ('i1');`); err != nil {
		pool.Close()
		t.Fatalf("seed parents: %v", err)
	}
	cleanup := func() {
		_, _ = pool.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+loopTestSchema+" CASCADE")
		pool.Close()
	}
	return pool, cleanup
}

func TestPgStoreRoundTrip(t *testing.T) {
	pool, cleanup := loopTestDB(t)
	defer cleanup()
	ctx := context.Background()
	s := NewPgStore(pool)
	dl := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	goal := Loop{
		ID: "lp_goal", SessionID: "s1", Origin: OriginIntegration, IntegrationID: "i1",
		Kind: KindGoal, Status: StatusPending, Goal: "do it", Prompt: "seed",
		MaxIterations: 5, DeadlineAt: &dl, FailureCap: 2, JudgeTask: "loop_judge",
		CreatedAt: time.Now().UTC(),
	}
	if err := s.Create(ctx, goal); err != nil {
		t.Fatalf("create goal: %v", err)
	}
	interval := Loop{
		ID: "lp_int", SessionID: "s1", Origin: OriginOperator,
		Kind: KindInterval, Status: StatusPending, Prompt: "tick",
		IntervalSeconds: 30, MaxIterations: 3, DeadlineAt: &dl, FailureCap: 3,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.Create(ctx, interval); err != nil {
		t.Fatalf("create interval: %v", err)
	}

	got, err := s.Get(ctx, "lp_goal")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Kind != KindGoal || got.IntegrationID != "i1" || got.JudgeTask != "loop_judge" {
		t.Errorf("goal round-trip mismatch: %+v", got)
	}
	if got.DeadlineAt == nil || !got.DeadlineAt.Equal(dl) {
		t.Errorf("deadline round-trip: got %v want %v", got.DeadlineAt, dl)
	}
	if gi, _ := s.Get(ctx, "lp_int"); gi.IntervalSeconds != 30 || gi.IntegrationID != "" {
		t.Errorf("interval round-trip mismatch: %+v", gi)
	}

	if _, err := s.Get(ctx, "missing"); err != ErrNotFound {
		t.Errorf("Get(missing) = %v, want ErrNotFound", err)
	}

	all, _ := s.List(ctx)
	if len(all) != 2 {
		t.Errorf("list = %d, want 2", len(all))
	}

	// lifecycle: run → progress → terminal.
	if err := s.MarkRunning(ctx, "lp_goal"); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	live, _ := s.ListLive(ctx)
	if len(live) != 1 || live[0].ID != "lp_goal" {
		t.Errorf("listlive = %+v, want only lp_goal", live)
	}
	runID, err := s.AppendRun(ctx, "lp_goal", 1, "seed")
	if err != nil {
		t.Fatalf("append run: %v", err)
	}
	if err := s.SaveProgress(ctx, "lp_goal", 1, DecisionContinue, "progress"); err != nil {
		t.Fatalf("save progress: %v", err)
	}
	if err := s.FinishRun(ctx, runID, DecisionContinue, "progress"); err != nil {
		t.Fatalf("finish run: %v", err)
	}
	if err := s.SetStatus(ctx, "lp_goal", StatusDone, "goal met"); err != nil {
		t.Fatalf("set status: %v", err)
	}

	done, _ := s.Get(ctx, "lp_goal")
	if done.Status != StatusDone || done.Iteration != 1 || done.LastVerdict != DecisionContinue {
		t.Errorf("post-lifecycle loop = %+v", done)
	}
	if done.EndedAt == nil {
		t.Error("terminal status should stamp ended_at")
	}

	runs, _ := s.ListRuns(ctx, "lp_goal")
	if len(runs) != 1 || runs[0].Verdict != DecisionContinue || runs[0].EndedAt == nil {
		t.Errorf("runs = %+v", runs)
	}

	// FK cascade: deleting the session removes its loops + runs.
	if _, err := pool.Exec(ctx, "DELETE FROM sessions WHERE id='s1'"); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if remaining, _ := s.List(ctx); len(remaining) != 0 {
		t.Errorf("loops after session delete = %d, want 0 (cascade)", len(remaining))
	}
}
