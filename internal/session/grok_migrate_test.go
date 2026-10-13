package session

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// stageGrokSession writes <home>/sessions/<enc(cwd)>/<id>/ with a
// chat_history.jsonl holding body, and returns the session dir.
func stageGrokSession(t *testing.T, home, cwd, id, body string) string {
	t.Helper()
	dir := filepath.Join(home, "sessions", url.QueryEscape(cwd), id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "chat_history.jsonl"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func readGrokHistory(t *testing.T, home, cwd, id string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(home, "sessions", url.QueryEscape(cwd), id, "chat_history.jsonl"))
	if err != nil {
		t.Fatalf("read migrated history: %v", err)
	}
	return string(b)
}

const grokTestID = "c715a14b-3dd5-4b67-a18e-dcd0d79251b5"

func TestMigrateGrokSession_CopiesDirIntoNewHome(t *testing.T) {
	tmp := t.TempDir()
	oldHome, newHome, cwd := filepath.Join(tmp, "a"), filepath.Join(tmp, "b"), "/var/lib/proj"
	src := stageGrokSession(t, oldHome, cwd, grokTestID, "HISTORY\n")
	if err := os.WriteFile(filepath.Join(src, "summary.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "summary.json.lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	ok, err := migrateGrokSession(oldHome, newHome, cwd, grokTestID)
	if err != nil || !ok {
		t.Fatalf("migrate = (%v, %v), want (true, nil)", ok, err)
	}
	if got := readGrokHistory(t, newHome, cwd, grokTestID); got != "HISTORY\n" {
		t.Errorf("history = %q", got)
	}
	dst := filepath.Join(newHome, "sessions", url.QueryEscape(cwd), grokTestID)
	if _, err := os.Stat(filepath.Join(dst, "summary.json")); err != nil {
		t.Errorf("sibling files should be copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "summary.json.lock")); !os.IsNotExist(err) {
		t.Errorf("lock files must not be copied; stat err=%v", err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("source must be left in place (rollback resumes it): %v", err)
	}
}

func TestMigrateGrokSession_NoSourceIsNotResumable(t *testing.T) {
	tmp := t.TempDir()
	ok, err := migrateGrokSession(filepath.Join(tmp, "a"), filepath.Join(tmp, "b"), "/x", grokTestID)
	if err != nil || ok {
		t.Errorf("missing source = (%v, %v), want (false, nil)", ok, err)
	}
}

func TestMigrateGrokSession_SameHome(t *testing.T) {
	tmp := t.TempDir()
	stageGrokSession(t, tmp, "/x", grokTestID, "H\n")
	if ok, err := migrateGrokSession(tmp, tmp, "/x", grokTestID); err != nil || !ok {
		t.Errorf("same home = (%v, %v), want (true, nil)", ok, err)
	}
}

func TestMigrateGrokSession_RejectsUnsafeID(t *testing.T) {
	tmp := t.TempDir()
	stageGrokSession(t, filepath.Join(tmp, "a"), "/x", grokTestID, "H\n")
	if ok, _ := migrateGrokSession(filepath.Join(tmp, "a"), filepath.Join(tmp, "b"), "/x", "../"+grokTestID); ok {
		t.Error("a traversing session id must not migrate")
	}
}

func TestMigrateGrokSession_ReplacesStaleDest(t *testing.T) {
	tmp := t.TempDir()
	oldHome, newHome, cwd := filepath.Join(tmp, "a"), filepath.Join(tmp, "b"), "/x"
	stageGrokSession(t, oldHome, cwd, grokTestID, "LATEST\n")
	stale := stageGrokSession(t, newHome, cwd, grokTestID, "STALE\n")
	if err := os.WriteFile(filepath.Join(stale, "leftover"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if ok, err := migrateGrokSession(oldHome, newHome, cwd, grokTestID); err != nil || !ok {
		t.Fatalf("migrate = (%v, %v)", ok, err)
	}
	if got := readGrokHistory(t, newHome, cwd, grokTestID); got != "LATEST\n" {
		t.Errorf("stale destination should be replaced, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(stale, "leftover")); !os.IsNotExist(err) {
		t.Error("stale destination contents should be gone")
	}
}

func TestMigrateGrokSession_RejectsSymlink(t *testing.T) {
	tmp := t.TempDir()
	oldHome, newHome, cwd := filepath.Join(tmp, "a"), filepath.Join(tmp, "b"), "/x"
	src := stageGrokSession(t, oldHome, cwd, grokTestID, "H\n")
	secret := filepath.Join(tmp, "secret")
	if err := os.WriteFile(secret, []byte("SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(src, "events.jsonl")); err != nil {
		t.Skip("symlinks unsupported: " + err.Error())
	}
	ok, err := migrateGrokSession(oldHome, newHome, cwd, grokTestID)
	if ok || err == nil {
		t.Fatalf("symlinked source = (%v, %v), want refusal", ok, err)
	}
	dst := filepath.Join(newHome, "sessions", url.QueryEscape(cwd), grokTestID)
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Errorf("destination must not exist after refusal; err=%v", err)
	}
	if _, err := os.Lstat(dst + ".migrate.tmp"); !os.IsNotExist(err) {
		t.Errorf("temp dir must be cleaned up; err=%v", err)
	}
}
