package pg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestStartStop spins up a real embedded PG, runs a trivial query,
// and tears it down. Skipped under `go test -short` because the library
// downloads a ~50 MB binary on first run.
func TestStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("embedded PG integration test — skip under -short")
	}

	dir := t.TempDir()
	// Keep the cache outside t.TempDir so repeated runs in the same dev
	// session reuse the downloaded binary. Cleaned up manually if needed.
	cache := filepath.Join(os.TempDir(), "opendray-pg-test-cache")

	cfg := Config{
		DataDir:  filepath.Join(dir, "pg"),
		CacheDir: cache,
		Port:     pickFreePort(t),
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pg, err := Start(ctx, cfg)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pg.Stop()

	if pg.Password() == "" {
		t.Error("Password() empty after generate-on-start")
	}

	// Round-trip a real query to confirm the DB was created and is reachable.
	dsn := fmt.Sprintf("host=127.0.0.1 port=%d user=%s password=%s dbname=%s sslmode=disable",
		pg.Port(), pg.UserName(), pg.Password(), pg.DBName())
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	var one int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatalf("query: %v", err)
	}
	if one != 1 {
		t.Errorf("SELECT 1 returned %d", one)
	}

	// Stop twice — second call is a no-op.
	if err := pg.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}
	if err := pg.Stop(); err != nil {
		t.Errorf("Stop (2nd): %v", err)
	}
}

func pickFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}
