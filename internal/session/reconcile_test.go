package session

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestClassifyExitState pins the precedence used by waitExit: a user
// stop wins, a gateway shutdown is an interruption (auto-resumed on
// next start), and anything else is a normal end. This is the core of
// the 2026-05-23 fix that made graceful restarts preserve sessions.
func TestClassifyExitState(t *testing.T) {
	cases := []struct {
		name          string
		stop, closing bool
		want          State
	}{
		{"user stop beats shutdown", true, true, StateStopped},
		{"user stop", true, false, StateStopped},
		{"shutdown -> interrupted", false, true, StateInterrupted},
		{"spontaneous exit -> ended", false, false, StateEnded},
	}
	for _, c := range cases {
		if got := classifyExitState(c.stop, c.closing); got != c.want {
			t.Errorf("%s: classifyExitState(%v, %v) = %q, want %q",
				c.name, c.stop, c.closing, got, c.want)
		}
	}
}

func TestStateIsTerminal(t *testing.T) {
	for _, s := range []State{StateStopped, StateEnded, StateInterrupted} {
		if !s.IsTerminal() {
			t.Errorf("%q should be terminal", s)
		}
	}
	for _, s := range []State{StatePending, StateRunning, StateIdle} {
		if s.IsTerminal() {
			t.Errorf("%q should not be terminal", s)
		}
	}
}

func TestAutoResumeMaxFromEnv(t *testing.T) {
	cases := []struct {
		val  string
		want int
	}{
		{"", 0},      // unset -> no cap
		{"0", 0},     // explicit 0 -> no cap
		{"5", 5},     // positive cap
		{"  12 ", 12}, // trimmed
		{"-3", 0},    // negative -> no cap
		{"abc", 0},   // garbage -> no cap
	}
	for _, c := range cases {
		t.Setenv("OPENDRAY_AUTO_RESUME_MAX", c.val)
		if got := autoResumeMaxFromEnv(); got != c.want {
			t.Errorf("OPENDRAY_AUTO_RESUME_MAX=%q -> %d, want %d", c.val, got, c.want)
		}
	}
}

// devDB returns a pool against OPENDRAY_DEV_DB_URL, skipping when unset
// or unreachable (CI default until a Postgres service lands). Mirrors
// internal/githost's helper.
//
// WARNING: TestReconcileStoreInterrupted calls MarkRunningAsInterrupted,
// which flips EVERY non-terminal row in the target database. Point this
// at a DISPOSABLE dev database, never production.
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

// TestReconcileStoreInterrupted verifies the store half of the restart-
// preservation fix: MarkRunningAsInterrupted flips only non-terminal
// rows (and returns them), and ListInterrupted enumerates them — while
// rows the user explicitly stopped/ended stay put.
func TestReconcileStoreInterrupted(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	st := newStore(pool)
	ctx := context.Background()

	mk := func(target State) string {
		id := newID()
		if err := st.Insert(ctx, Session{
			ID: id, Name: "reconcile-test", ProviderID: "claude",
			Cwd: "/tmp", Args: []string{}, State: StateRunning,
			StartedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatalf("insert %s: %v", target, err)
		}
		switch target {
		case StateRunning:
			// already running
		case StateStopped, StateEnded:
			if err := st.MarkTerminal(ctx, id, target, 0); err != nil {
				t.Fatalf("mark %s: %v", target, err)
			}
		default: // idle / pending
			if _, err := pool.Exec(ctx, `UPDATE sessions SET state=$1 WHERE id=$2`, string(target), id); err != nil {
				t.Fatalf("set %s: %v", target, err)
			}
		}
		return id
	}

	running, idle, pending := mk(StateRunning), mk(StateIdle), mk(StatePending)
	stopped, ended := mk(StateStopped), mk(StateEnded)
	all := []string{running, idle, pending, stopped, ended}
	t.Cleanup(func() {
		for _, id := range all {
			_, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE id=$1`, id)
		}
	})

	flipped, err := st.MarkRunningAsInterrupted(ctx)
	if err != nil {
		t.Fatal(err)
	}
	set := func(ids []string) map[string]bool {
		m := make(map[string]bool, len(ids))
		for _, id := range ids {
			m[id] = true
		}
		return m
	}
	got := set(flipped)
	for _, id := range []string{running, idle, pending} {
		if !got[id] {
			t.Errorf("non-terminal session %s should have been flipped to interrupted", id)
		}
	}
	for _, id := range []string{stopped, ended} {
		if got[id] {
			t.Errorf("terminal session %s must not be flipped", id)
		}
	}

	listed := set(mustListInterrupted(t, st, ctx))
	for _, id := range []string{running, idle, pending} {
		if !listed[id] {
			t.Errorf("ListInterrupted missing %s", id)
		}
	}
	for _, id := range []string{stopped, ended} {
		if listed[id] {
			t.Errorf("ListInterrupted must not include terminal %s", id)
		}
	}
}

func mustListInterrupted(t *testing.T, st *sessionStore, ctx context.Context) []string {
	t.Helper()
	ids, err := st.ListInterrupted(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return ids
}
