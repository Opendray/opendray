package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Flags used to be accepted only BEFORE the subcommand, because Go's
// flag package stops parsing at the first non-flag argument. The
// failure was silent and, for `notes flatten --apply`, actively
// misleading: apply stayed false, the migration performed a dry run,
// and it told the operator to "re-run with --apply" — which is what
// they had just done.
//
// These run `list` and `read`, which write nothing, so the assertion is
// about argument order rather than about touching a vault.
func TestNotesFlagsAreAcceptedAfterTheSubcommand(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "note.md"), []byte("# hi\n"), 0o644); err != nil {
		t.Fatalf("seed note: %v", err)
	}

	after := runNotesCapturingStdout(t, []string{"--root", dir, "list", "--json"})
	if !json.Valid([]byte(strings.TrimSpace(after))) {
		t.Errorf("--json after the subcommand was ignored; got %q", after)
	}

	before := runNotesCapturingStdout(t, []string{"--root", dir, "--json", "list"})
	if !json.Valid([]byte(strings.TrimSpace(before))) {
		t.Errorf("--json before the subcommand broke; got %q", before)
	}
}

func TestNotesPositionalArgsSurviveFlagsAfterTheSubcommand(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "note.md"), []byte("# hi\nbody\n"), 0o644); err != nil {
		t.Fatalf("seed note: %v", err)
	}
	// The path is a positional arg AFTER the subcommand and must not be
	// swallowed by the second parse pass.
	out := runNotesCapturingStdout(t, []string{"--root", dir, "read", "note.md"})
	if !strings.Contains(out, "body") {
		t.Errorf("read lost its path argument; got %q", out)
	}
}

func runNotesCapturingStdout(t *testing.T, args []string) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	code := runNotes(args)
	os.Stdout = orig
	_ = w.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if code != 0 {
		t.Fatalf("runNotes(%v) = %d, output: %s", args, code, out)
	}
	return string(out)
}
