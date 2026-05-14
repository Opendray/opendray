package projectdoc

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// devDB mirrors the helper in other packages — skips when
// OPENDRAY_DEV_DB_URL isn't set so CI doesn't try to talk to a
// nonexistent database.
func devDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("OPENDRAY_DEV_DB_URL")
	if url == "" {
		t.Skip("OPENDRAY_DEV_DB_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Skipf("dev DB unreachable: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("dev DB ping failed: %v", err)
	}
	return pool
}

func TestStaleJournalEntries_EmptyCwd(t *testing.T) {
	svc := NewService(&pgxpool.Pool{}, nil)
	_, err := svc.StaleJournalEntries(context.Background(), "", 0)
	if err != ErrEmptyCwd {
		t.Errorf("expected ErrEmptyCwd, got %v", err)
	}
}

// TestStaleJournalEntries_LiveDB exercises the SQL against a real
// database — verifies the query parses and returns zero rows for
// an unknown cwd. The conflict-aware NOT EXISTS sub-select is the
// piece most likely to break across migrations; this catches a
// missing table or column at boot time.
func TestStaleJournalEntries_LiveDB(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	svc := NewService(pool, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := svc.StaleJournalEntries(ctx, "/__no_such_cwd__", 90*24*time.Hour)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty result for unknown cwd; got %d rows", len(out))
	}
}
