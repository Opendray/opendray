package notes

import (
	"os"
	"path/filepath"
	"testing"
)

// On a pre-split install opendray's skills/ and mcp/ roots sit inside
// the documents root, and the doc library listed them as folders — the
// gateway's own configuration presented to the operator as their
// writing. Hiding them is what makes the old layout tolerable for
// anyone who never migrates.
func TestList_HidesNestedMachineryDirs(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("# x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("projects/app/overview.md")
	write("skills/secretary/SKILL.md")
	write("mcp/registry.md")

	v, err := New(root, Options{HiddenDirs: []string{
		filepath.Join(root, "skills"),
		filepath.Join(root, "mcp"),
	}})
	if err != nil {
		t.Fatal(err)
	}

	got, err := v.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != "projects/app/overview.md" {
		t.Fatalf("listing = %v, want only the operator's document", got)
	}
}

// Once an install is split those roots live outside the vault entirely.
// Passing them anyway must be harmless — app.go does exactly that,
// unconditionally, rather than branching on layout.
func TestList_HiddenDirsOutsideTheVaultAreIgnored(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# x\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	v, err := New(root, Options{HiddenDirs: []string{
		filepath.Join(elsewhere, "skills"),
		filepath.Join(elsewhere, "mcp"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(v.hidden) != 0 {
		t.Fatalf("hidden = %v, want nothing (those dirs are not in the vault)", v.hidden)
	}

	got, err := v.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("listing = %v, want the one note", got)
	}
}

// Hiding the root itself would empty the entire doc library. Guard it:
// a misconfiguration that silently blanks someone's documents is worse
// than the folders it was trying to hide.
func TestList_VaultRootIsNeverHidden(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# x\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	v, err := New(root, Options{HiddenDirs: []string{root, root + "/"}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := v.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("listing = %v, want the note — the root must not be hideable", got)
	}
}

// A hidden dir must not shadow a same-named directory deeper in the
// tree: `skills/` at the root is opendray's, `projects/skills/` is the
// operator writing about skills.
func TestList_HiddenMatchIsRootedNotByName(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"skills/SKILL.md", "projects/skills/notes.md"} {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("# x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	v, err := New(root, Options{HiddenDirs: []string{filepath.Join(root, "skills")}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := v.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != "projects/skills/notes.md" {
		t.Fatalf("listing = %v, want only projects/skills/notes.md", got)
	}
}
