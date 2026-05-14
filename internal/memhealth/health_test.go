package memhealth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNew_NilPool(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("expected error for nil pool")
	}
}

func TestComputeForCwd_EmptyCwd(t *testing.T) {
	svc, err := New(&pgxpool.Pool{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ComputeForCwd(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty cwd")
	}
}

// devDB is the same opt-in helper the summarizer store tests use.
// We don't bring in a shared helper module yet — copy-paste keeps
// the test deps minimal.
func devDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("OPENDRAY_DEV_DB_URL")
	if url == "" {
		t.Skip("OPENDRAY_DEV_DB_URL not set; export a writable Postgres DSN to run this test")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Skipf("dev DB unreachable, skipping: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("dev DB ping failed, skipping: %v", err)
	}
	return pool
}

// TestComputeForCwd_LiveDB sanity-checks that the SQL parses and
// returns zero-valued counts for an unknown cwd. We don't seed
// data here — that would require touching every table; the goal
// is just "queries run, no schema regressions".
func TestComputeForCwd_LiveDB(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()

	svc, err := New(pool)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cwd := "/__memhealth_test_nonexistent__"
	snap, err := svc.ComputeForCwd(ctx, cwd)
	if err != nil {
		t.Fatalf("ComputeForCwd: %v", err)
	}
	if snap.Cwd != cwd {
		t.Errorf("Cwd: got %q, want %q", snap.Cwd, cwd)
	}
	if snap.LookbackDays != LookbackDays {
		t.Errorf("LookbackDays: got %d, want %d", snap.LookbackDays, LookbackDays)
	}
	if snap.GeneratedAt.IsZero() {
		t.Errorf("GeneratedAt should be set")
	}
	// Unknown cwd → every count must be zero.
	for _, tc := range []struct {
		name string
		got  int
	}{
		{"NewFactsCount", snap.NewFactsCount},
		{"TotalFactsCount", snap.TotalFactsCount},
		{"ZeroHitFactsCount", snap.ZeroHitFactsCount},
		{"CaptureFires", snap.CaptureFires},
		{"NewJournalCount", snap.NewJournalCount},
		{"TotalJournalCount", snap.TotalJournalCount},
		{"PendingProposals", snap.PendingProposals},
		{"PlanDriftProposals", snap.PlanDriftProposals},
	} {
		if tc.got != 0 {
			t.Errorf("%s for unknown cwd: got %d, want 0", tc.name, tc.got)
		}
	}
	if snap.PlanLastUpdatedAt != nil {
		t.Errorf("PlanLastUpdatedAt: expected nil for unknown cwd, got %v", snap.PlanLastUpdatedAt)
	}
	if snap.GoalLastUpdatedAt != nil {
		t.Errorf("GoalLastUpdatedAt: expected nil for unknown cwd, got %v", snap.GoalLastUpdatedAt)
	}
}

func TestComputeForCwd_CancelledContext(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	svc, err := New(pool)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = svc.ComputeForCwd(ctx, "/x")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		// pgx wraps, so just ensure we got *some* error
		t.Logf("got error (acceptable): %v", err)
	}
}
