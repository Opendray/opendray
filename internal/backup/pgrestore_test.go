package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestPgRestore_List_RejectsGarbage confirms `pg_restore --list`
// surfaces an error on a file that isn't a valid archive — the core of
// post-backup verification. Skipped when pg_restore isn't installed.
func TestPgRestore_List_RejectsGarbage(t *testing.T) {
	pr, err := NewPgRestore("")
	if err != nil {
		t.Skip("pg_restore not on PATH; skipping verification test")
	}
	garbage := filepath.Join(t.TempDir(), "not-a-dump.bin")
	if err := os.WriteFile(garbage, []byte("this is not a pg_dump archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := pr.List(context.Background(), garbage); err == nil {
		t.Fatal("expected pg_restore --list to reject a non-archive file")
	}
}

func TestPgRestore_List_RequiresPath(t *testing.T) {
	pr, err := NewPgRestore("")
	if err != nil {
		t.Skip("pg_restore not on PATH; skipping")
	}
	if _, err := pr.List(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty dump path")
	}
}
