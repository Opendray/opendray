package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/opendray/opendray-v2/internal/config"
)

// TestNew_FailsOnUnreachableDB verifies the composition root surfaces
// a clear error when the Postgres URL points nowhere — and that it
// does so quickly rather than hanging on retries.
func TestNew_FailsOnUnreachableDB(t *testing.T) {
	cfg := config.Config{
		Listen: "127.0.0.1:0",
		Database: config.DatabaseConfig{
			URL: "postgres://x:y@127.0.0.1:1/db?sslmode=disable&connect_timeout=1",
		},
		Admin: config.AdminConfig{User: "admin", Password: "x"},
		Log:   config.LogConfig{Level: "error", Format: "text"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := New(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for unreachable DB, got nil")
	}
	// New() should bail out at the store.Open step with a wrapped
	// error mentioning the store layer — not panic, not block.
	if !strings.Contains(err.Error(), "store:") {
		t.Errorf("error should be wrapped with 'store:', got: %v", err)
	}
}

func TestNewLogger_LevelParsing(t *testing.T) {
	tests := []struct {
		level   string
		wantNil bool
	}{
		{"debug", false},
		{"info", false},
		{"warn", false},
		{"error", false},
		{"INFO", false},    // case-insensitive
		{"unknown", false}, // falls back to info, no error
		{"", false},        // default
	}
	for _, tc := range tests {
		t.Run("level="+tc.level, func(t *testing.T) {
			lg, ring, err := newLogger(config.LogConfig{
				Level:  tc.level,
				Format: "text",
			})
			if err != nil {
				t.Fatalf("newLogger(%q): %v", tc.level, err)
			}
			if tc.wantNil {
				if lg != nil {
					t.Error("expected nil logger")
				}
				return
			}
			if lg == nil {
				t.Error("logger nil")
			}
			if ring == nil {
				t.Error("ring buffer nil")
			}
		})
	}
}

func TestNewLogger_JSONFormat(t *testing.T) {
	lg, ring, err := newLogger(config.LogConfig{
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("newLogger json: %v", err)
	}
	if lg == nil || ring == nil {
		t.Error("logger or ring buffer nil")
	}
	// Don't assert on JSON output shape — that's slog's job. We
	// only care that the JSON branch wires up without erroring.
}

func TestNewLogger_BadFilePath(t *testing.T) {
	// Use an existing regular file as the supposed parent directory.
	// os.MkdirAll will fail on all platforms (including Windows) when
	// a path component already exists as a file instead of a directory.
	blocker := filepath.Join(t.TempDir(), "notadir")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := newLogger(config.LogConfig{
		Level: "info",
		File:  filepath.Join(blocker, "opendray.log"),
	})
	if err == nil {
		t.Error("expected error for unwritable log path, got nil")
	}
}
