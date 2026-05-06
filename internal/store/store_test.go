package store

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// devDSN matches the convention used elsewhere in the repo
// (memory/summarizer/store_test.go): OPENDRAY_DEV_DB_URL, falling
// back to the home-lab default. Tests t.Skip when the DB is
// unreachable so CI on a fresh laptop still passes.
func devDSN() string {
	if v := os.Getenv("OPENDRAY_DEV_DB_URL"); v != "" {
		return v
	}
	return "postgres://opd2_user:UGuZjQVFtXR3MtKJ6Q@192.168.3.88:5432/opendray_v2?sslmode=disable"
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
	st, err := Open(ctx, devDSN(), 0)
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
		st, err := Open(ctx, devDSN(), 7)
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
		st, err := Open(ctx, devDSN(), 0)
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
		st, err := Open(ctx, devDSN(), -1)
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
