package vaultgit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readIgnore(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	return string(b)
}

func TestEnsureManagedIgnore_CreatesTheBlock(t *testing.T) {
	root := t.TempDir()
	if err := ensureManagedIgnore(root, nil); err != nil {
		t.Fatal(err)
	}
	got := readIgnore(t, root)
	for _, want := range []string{ignoreBegin, ignoreEnd, ".DS_Store", ".trash/"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// A pre-split install has skills/ inside the repo. That is exactly how
// an agent skill ended up committed to someone's private docs repo, so
// the rule has to be derived from the resolved layout, not hardcoded.
func TestEnsureManagedIgnore_IgnoresNestedMachineryDirs(t *testing.T) {
	root := t.TempDir()
	err := ensureManagedIgnore(root, []string{
		filepath.Join(root, "skills"),
		filepath.Join(root, "mcp"),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := readIgnore(t, root)
	if !strings.Contains(got, "skills/") || !strings.Contains(got, "mcp/") {
		t.Fatalf("nested dirs not ignored:\n%s", got)
	}
}

// Once split, those dirs live outside the repo. Passing them must not
// produce `../skills/` rules — app.go passes the resolved set blindly.
func TestEnsureManagedIgnore_SkipsDirsOutsideTheRepo(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := ensureManagedIgnore(root, []string{filepath.Join(outside, "skills")}); err != nil {
		t.Fatal(err)
	}
	if got := readIgnore(t, root); strings.Contains(got, "..") {
		t.Fatalf("escaped the repo:\n%s", got)
	}
}

// The operator's own rules are not ours to rewrite.
func TestEnsureManagedIgnore_PreservesOperatorContent(t *testing.T) {
	root := t.TempDir()
	original := "# mine\n*.pdf\n\n# also mine\ndrafts/\n"
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureManagedIgnore(root, nil); err != nil {
		t.Fatal(err)
	}
	got := readIgnore(t, root)
	for _, want := range []string{"# mine", "*.pdf", "# also mine", "drafts/"} {
		if !strings.Contains(got, want) {
			t.Errorf("clobbered %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, ignoreBegin) {
		t.Errorf("managed block not appended:\n%s", got)
	}
}

// Rewritten on every commit — it must converge, not grow a new block
// each sync, and must not churn the file once settled.
func TestEnsureManagedIgnore_IsIdempotent(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 3; i++ {
		if err := ensureManagedIgnore(root, []string{filepath.Join(root, "skills")}); err != nil {
			t.Fatal(err)
		}
	}
	got := readIgnore(t, root)
	if n := strings.Count(got, ignoreBegin); n != 1 {
		t.Fatalf("block appears %d times:\n%s", n, got)
	}

	before, err := os.Stat(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureManagedIgnore(root, []string{filepath.Join(root, "skills")}); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Error("rewrote an unchanged .gitignore — every sync would dirty the repo")
	}
}

// An install that moves skills/ out must stop carrying the stale rule,
// otherwise a later real skills/ document folder would be invisible.
func TestEnsureManagedIgnore_DropsRulesThatNoLongerApply(t *testing.T) {
	root := t.TempDir()
	if err := ensureManagedIgnore(root, []string{filepath.Join(root, "skills")}); err != nil {
		t.Fatal(err)
	}
	if err := ensureManagedIgnore(root, nil); err != nil {
		t.Fatal(err)
	}
	if got := readIgnore(t, root); strings.Contains(got, "skills/") {
		t.Fatalf("stale rule survived:\n%s", got)
	}
}

func TestReplaceBlock_HandlesAnUnterminatedBlock(t *testing.T) {
	existing := "# mine\n" + ignoreBegin + "\n.DS_Store\n"
	got := replaceBlock(existing, renderBlock([]string{".DS_Store"}))
	if n := strings.Count(got, ignoreBegin); n != 1 {
		t.Fatalf("block appears %d times:\n%s", n, got)
	}
	if !strings.Contains(got, "# mine") {
		t.Fatalf("lost operator content:\n%s", got)
	}
}
