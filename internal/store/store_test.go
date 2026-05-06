package store

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// devDSN returns the dev-DB DSN from OPENDRAY_DEV_DB_URL, or t.Skips
// when unset. Never hardcode credentials — committed credentials are
// forever. To run these tests locally, start the bundled Postgres
// (`docker compose -f docker-compose.test.yml up -d`) and export
// OPENDRAY_DEV_DB_URL accordingly. See docs/quickstart.md.
func devDSN(t *testing.T) string {
	t.Helper()
	v := os.Getenv("OPENDRAY_DEV_DB_URL")
	if v == "" {
		t.Skip("OPENDRAY_DEV_DB_URL not set; export a writable Postgres DSN to run this test")
	}
	return v
}

func TestOpen_BadDSN(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Open(ctx, "not a valid dsn", 0)
	if err == nil {
		t.Fatal("expected parse error for malformed DSN")
	}
	if !strings.Contains(err.Error(), "store: parse dsn") {
		t.Errorf("error should be wrapped with parse-dsn context, got: %v", err)
	}
}

func TestOpen_UnreachableDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// 127.0.0.1:1 is RFC2606 reserved + always-refused; ConnectTimeout
	// keeps the test snappy. Returns parse-OK but ping-fail.
	dsn := "postgres://x:y@127.0.0.1:1/db?sslmode=disable&connect_timeout=1"
	_, err := Open(ctx, dsn, 0)
	if err == nil {
		t.Fatal("expected ping failure against unreachable host")
	}
	if !strings.Contains(err.Error(), "store: ping") &&
		!strings.Contains(err.Error(), "store: open pool") {
		t.Errorf("error should be wrapped, got: %v", err)
	}
}

func TestOpen_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st, err := Open(ctx, devDSN(t), 0)
	if err != nil {
		t.Skipf("dev DB unreachable, skipping: %v", err)
	}
	defer st.Close()

	if st.Pool() == nil {
		t.Fatal("Pool() returned nil after successful Open")
	}
	if err := st.Ping(ctx); err != nil {
		t.Errorf("Ping after Open: %v", err)
	}
}

func TestOpen_MaxConnsApplied(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("explicit positive maxConns honoured", func(t *testing.T) {
		st, err := Open(ctx, devDSN(t), 7)
		if err != nil {
			t.Skipf("dev DB unreachable, skipping: %v", err)
		}
		defer st.Close()
		got := st.Pool().Config().MaxConns
		if got != 7 {
			t.Errorf("Pool.MaxConns = %d, want 7", got)
		}
	})

	t.Run("zero falls back to DefaultMaxConns", func(t *testing.T) {
		st, err := Open(ctx, devDSN(t), 0)
		if err != nil {
			t.Skipf("dev DB unreachable, skipping: %v", err)
		}
		defer st.Close()
		got := st.Pool().Config().MaxConns
		if int(got) != DefaultMaxConns {
			t.Errorf("Pool.MaxConns = %d, want DefaultMaxConns(%d)",
				got, DefaultMaxConns)
		}
	})

	t.Run("negative also falls back", func(t *testing.T) {
		st, err := Open(ctx, devDSN(t), -1)
		if err != nil {
			t.Skipf("dev DB unreachable, skipping: %v", err)
		}
		defer st.Close()
		got := st.Pool().Config().MaxConns
		if int(got) != DefaultMaxConns {
			t.Errorf("Pool.MaxConns = %d, want DefaultMaxConns(%d)",
				got, DefaultMaxConns)
		}
	})
}

func TestClose_NilSafe(t *testing.T) {
	// A *Store whose pool was never set must not panic on Close —
	// matters for half-constructed test fixtures.
	(&Store{}).Close()
}
