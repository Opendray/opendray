package vaultgit

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

// The incident this guards against, reproduced exactly: a remote that
// is empty because every push failed, a local repo holding all the real
// work, and someone reaching for "reset to remote" to escape a rebase
// conflict.

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func write(t *testing.T, dir, rel, body string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// vaultWithEmptyRemote builds the exact shape of the incident: origin
// holds one initial commit, the local vault holds real documents that
// were never pushed.
func vaultWithEmptyRemote(t *testing.T) (root string, srv *httptest.Server) {
	t.Helper()
	base := t.TempDir()
	remote := filepath.Join(base, "remote")
	root = filepath.Join(base, "vault")

	// Remote: an "empty" repo — just a README, like a freshly created
	// GitHub repo whose pushes never landed.
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, remote, "init", "-b", "main")
	write(t, remote, "README.md", "# docs\n")
	git(t, remote, "add", ".")
	git(t, remote, "commit", "-m", "Initial commit")

	git(t, base, "clone", remote, root)
	// Local work that never reached the remote.
	write(t, root, "projects/app.md", "# app\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "Notes")
	write(t, root, "untracked-draft.md", "# draft\n")

	h, err := NewHandlers(root, nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	h.Mount(r)
	srv = httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return root, srv
}

func postReset(t *testing.T, srv *httptest.Server, body string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Post(srv.URL+"/vault/git/reset-to-remote",
		"application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

// Without confirmation the reset must refuse and say what it would
// cost. Silence here is what destroyed 354 files.
func TestResetToRemote_RefusesUnconfirmedLoss(t *testing.T) {
	root, srv := vaultWithEmptyRemote(t)

	code, body := postReset(t, srv, `{}`)
	if code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 — an unconfirmed destructive reset must not run", code)
	}
	loss, ok := body["loss"].(map[string]any)
	if !ok {
		t.Fatalf("no loss report in %v", body)
	}
	if loss["unpushed_commits"].(float64) != 1 {
		t.Errorf("unpushed_commits = %v, want 1", loss["unpushed_commits"])
	}
	if loss["untracked_files"].(float64) != 1 {
		t.Errorf("untracked_files = %v, want 1", loss["untracked_files"])
	}

	// And nothing may have been touched.
	if _, err := os.Stat(filepath.Join(root, "projects/app.md")); err != nil {
		t.Errorf("committed work was destroyed by a refused reset: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "untracked-draft.md")); err != nil {
		t.Errorf("untracked file was destroyed by a refused reset: %v", err)
	}
}

// Confirmed, the reset proceeds — but everything it removes is parked
// first. A confirmation dialog is not a backup.
func TestResetToRemote_ConfirmedResetLeavesARecoveryPoint(t *testing.T) {
	root, srv := vaultWithEmptyRemote(t)

	code, body := postReset(t, srv, `{"confirm":true}`)
	if code != http.StatusOK {
		t.Fatalf("status = %d (%v), want 200", code, body)
	}

	// The reset really happened.
	if _, err := os.Stat(filepath.Join(root, "projects/app.md")); !os.IsNotExist(err) {
		t.Error("confirmed reset did not reset")
	}
	if _, err := os.Stat(filepath.Join(root, "untracked-draft.md")); !os.IsNotExist(err) {
		t.Error("clean -fd did not run")
	}

	ref, _ := body["rescue_ref"].(string)
	if ref == "" {
		t.Fatal("no rescue ref reported")
	}
	// The unpushed commit is reachable again.
	files := git(t, root, "ls-tree", "-r", "--name-only", ref)
	if !bytes.Contains([]byte(files), []byte("projects/app.md")) {
		t.Errorf("rescue ref %q does not hold the unpushed work:\n%s", ref, files)
	}
	// And so is the untracked file, which no ref could have saved.
	stashed := git(t, root, "stash", "list")
	if !bytes.Contains([]byte(stashed), []byte("before reset-to-remote")) {
		t.Errorf("working tree was not stashed:\n%s", stashed)
	}
	restored := git(t, root, "show", "--name-only", "--format=", "stash@{0}^3")
	if !bytes.Contains([]byte(restored), []byte("untracked-draft.md")) {
		t.Errorf("untracked file not captured by the stash:\n%s", restored)
	}
}

// A clean tree that is level with the remote loses nothing, so the
// button must stay one click. Demanding confirmation for a no-op is
// how confirmations get trained away.
func TestResetToRemote_CleanTreeNeedsNoConfirmation(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote")
	root := filepath.Join(base, "vault")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, remote, "init", "-b", "main")
	write(t, remote, "README.md", "# docs\n")
	git(t, remote, "add", ".")
	git(t, remote, "commit", "-m", "Initial commit")
	git(t, base, "clone", remote, root)

	h, err := NewHandlers(root, nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	h.Mount(r)
	srv := httptest.NewServer(r)
	defer srv.Close()

	code, body := postReset(t, srv, `{}`)
	if code != http.StatusOK {
		t.Fatalf("status = %d (%v), want 200 — nothing would be lost", code, body)
	}
	if ref, _ := body["rescue_ref"].(string); ref != "" {
		t.Errorf("rescue ref %q created for a lossless reset", ref)
	}
}
