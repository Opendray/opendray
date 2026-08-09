package notes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newVaultT(t *testing.T, layout Layout) (*Vault, string) {
	t.Helper()
	dir := t.TempDir()
	v, err := New(dir, Options{Layout: layout})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return v, dir
}

func writeFileT(t *testing.T, root, rel, body string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func TestLayoutPathDerivation(t *testing.T) {
	flat, _ := newVaultT(t, LayoutFlat)
	nested, _ := newVaultT(t, LayoutNested)

	if got, want := flat.ProjectDir("opendray-v2"), "opendray-v2"; got != want {
		t.Errorf("flat ProjectDir = %q, want %q", got, want)
	}
	if got, want := flat.PersonalPath("opendray-v2"), "opendray-v2/personal.md"; got != want {
		t.Errorf("flat PersonalPath = %q, want %q", got, want)
	}
	if got, want := nested.ProjectDir("opendray-v2"), "projects/opendray-v2"; got != want {
		t.Errorf("nested ProjectDir = %q, want %q", got, want)
	}
	if got, want := nested.PersonalPath("opendray-v2"), "personal/opendray-v2.md"; got != want {
		t.Errorf("nested PersonalPath = %q, want %q", got, want)
	}
}

func TestUnsetLayoutDoesNotRelocateAnExistingVault(t *testing.T) {
	// A caller that forgets to pass a layout must not silently move a
	// live install's documents to the flat shape.
	v, _ := newVaultT(t, "")
	if got := v.Layout(); got != LayoutNested {
		t.Fatalf("layout = %q, want nested", got)
	}
	if got, want := v.ProjectDir("x"), "projects/x"; got != want {
		t.Errorf("ProjectDir = %q, want %q", got, want)
	}
}

func TestFlatLayoutStepsAsideForReservedNames(t *testing.T) {
	flat, _ := newVaultT(t, LayoutFlat)
	// daily/ is a vault-wide concept; a project must not land on it.
	if got, want := flat.ProjectDir("daily"), "daily-docs"; got != want {
		t.Errorf("ProjectDir(daily) = %q, want %q", got, want)
	}
	// `_` and `.` prefixes belong to the vault (_templates, dotfiles).
	if got, want := flat.ProjectDir("_templates"), "_templates-docs"; got != want {
		t.Errorf("ProjectDir(_templates) = %q, want %q", got, want)
	}
	// A normal name is untouched.
	if got, want := flat.ProjectDir("daily-standup"), "daily-standup"; got != want {
		t.Errorf("ProjectDir(daily-standup) = %q, want %q", got, want)
	}
}

func TestDetectLayout(t *testing.T) {
	empty := t.TempDir()
	if got := DetectLayout(empty); got != LayoutFlat {
		t.Errorf("empty vault = %q, want flat", got)
	}

	nested := t.TempDir()
	writeFileT(t, nested, "projects/foo/README.md", "# foo\n")
	if got := DetectLayout(nested); got != LayoutNested {
		t.Errorf("vault with projects/ = %q, want nested", got)
	}

	// A projects/ directory holding nothing anyone can read is not a
	// nested vault — .DS_Store is exactly the file that once flipped a
	// detection like this one.
	noise := t.TempDir()
	writeFileT(t, noise, "projects/.DS_Store", "x")
	if got := DetectLayout(noise); got != LayoutFlat {
		t.Errorf("vault with only .DS_Store = %q, want flat", got)
	}
}

func TestFlattenMovesDocumentsAndRewritesLinks(t *testing.T) {
	v, root := newVaultT(t, LayoutNested)
	writeFileT(t, root, "projects/app/features/canvas.md", "# canvas\n")
	writeFileT(t, root, "projects/app/README.md",
		"See [[projects/app/features/canvas]] for detail.\n")
	writeFileT(t, root, "personal/app.md", "# my notes\n")
	writeFileT(t, root, "daily/2026-08-09.md", "# today\n")

	res, err := v.Flatten(context.Background(), false)
	if err != nil {
		t.Fatalf("Flatten: %v", err)
	}

	for _, want := range []string{
		"app/features/canvas.md",
		"app/README.md",
		"app/personal.md",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(want))); err != nil {
			t.Errorf("expected %s to exist: %v", want, err)
		}
	}
	// Daily notes are nobody's project and stay put.
	if _, err := os.Stat(filepath.Join(root, "daily", "2026-08-09.md")); err != nil {
		t.Errorf("daily note should not have moved: %v", err)
	}

	// The link that pointed into projects/ must now point at the new
	// location, or the migration has quietly broken the vault.
	body, err := os.ReadFile(filepath.Join(root, "app", "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	if got := string(body); !strings.Contains(got, "[[app/features/canvas]]") {
		t.Errorf("link not repointed, README is: %q", got)
	}

	if len(res.Moves) != 3 {
		t.Errorf("moved %d documents, want 3: %+v", len(res.Moves), res.Moves)
	}
}

func TestFlattenDryRunWritesNothing(t *testing.T) {
	v, root := newVaultT(t, LayoutNested)
	writeFileT(t, root, "projects/app/README.md", "# app\n")

	res, err := v.Flatten(context.Background(), true)
	if err != nil {
		t.Fatalf("Flatten: %v", err)
	}
	if !res.DryRun || len(res.Moves) != 1 {
		t.Fatalf("unexpected plan: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(root, "projects", "app", "README.md")); err != nil {
		t.Errorf("source should still be there after a dry run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "app")); !os.IsNotExist(err) {
		t.Errorf("dry run created %s", filepath.Join(root, "app"))
	}
}

func TestFlattenRefusesToOverwriteAndReportsWhy(t *testing.T) {
	v, root := newVaultT(t, LayoutNested)
	writeFileT(t, root, "projects/app/README.md", "# from projects\n")
	writeFileT(t, root, "app/README.md", "# already here\n")

	res, err := v.Flatten(context.Background(), false)
	if err != nil {
		t.Fatalf("Flatten: %v", err)
	}
	if len(res.Moves) != 0 {
		t.Errorf("should have moved nothing, moved %+v", res.Moves)
	}
	if len(res.Skips) != 1 || !strings.Contains(res.Skips[0].Reason, "already exists") {
		t.Fatalf("expected an already-exists skip, got %+v", res.Skips)
	}
	// Both copies survive — reconciling them is the operator's call.
	body, _ := os.ReadFile(filepath.Join(root, "app", "README.md"))
	if string(body) != "# already here\n" {
		t.Errorf("existing document was overwritten: %q", body)
	}
	if _, err := os.Stat(filepath.Join(root, "projects", "app", "README.md")); err != nil {
		t.Errorf("source was removed despite the skip: %v", err)
	}
}

func TestFlattenLeavesLooseAndForeignFilesAlone(t *testing.T) {
	v, root := newVaultT(t, LayoutNested)
	writeFileT(t, root, "projects/stray.md", "# no project\n")
	writeFileT(t, root, "personal/deep/thoughts.md", "# hand-filed\n")

	res, err := v.Flatten(context.Background(), false)
	if err != nil {
		t.Fatalf("Flatten: %v", err)
	}
	if len(res.Moves) != 0 {
		t.Errorf("moved files it should not have: %+v", res.Moves)
	}
	if len(res.Skips) != 2 {
		t.Fatalf("expected 2 explained skips, got %+v", res.Skips)
	}
	for _, s := range res.Skips {
		if s.Reason == "" {
			t.Errorf("skip without a reason: %+v", s)
		}
	}
}

func TestFlattenRepointsProjectOverrides(t *testing.T) {
	v, root := newVaultT(t, LayoutNested)
	writeFileT(t, root, "projects/app/README.md", "# app\n")
	if err := v.SetProjectMapping("/Users/x/code/app", "projects/app"); err != nil {
		t.Fatalf("SetProjectMapping: %v", err)
	}

	res, err := v.Flatten(context.Background(), false)
	if err != nil {
		t.Fatalf("Flatten: %v", err)
	}
	if res.MappingsRewritten != 1 {
		t.Errorf("rewrote %d overrides, want 1", res.MappingsRewritten)
	}
	// An override still naming projects/ would send this project's
	// documents to a directory the migration just emptied.
	got := v.ResolvedProjectDir("/Users/x/code/app")
	if got != "app" {
		t.Errorf("override resolves to %q, want %q", got, "app")
	}
}

func TestFlattenRefusesOnAnAlreadyFlatVault(t *testing.T) {
	v, _ := newVaultT(t, LayoutFlat)
	if _, err := v.Flatten(context.Background(), true); err == nil {
		t.Fatal("expected an error, got none")
	}
}

func TestResolvedPersonalPathFollowsAProjectOverride(t *testing.T) {
	flat, _ := newVaultT(t, LayoutFlat)
	if err := flat.SetProjectMapping("/Users/x/code/app", "clients/acme"); err != nil {
		t.Fatalf("SetProjectMapping: %v", err)
	}
	if got, want := flat.ResolvedPersonalPath("/Users/x/code/app"),
		"clients/acme/personal.md"; got != want {
		t.Errorf("flat ResolvedPersonalPath = %q, want %q", got, want)
	}

	// Nested keeps the two trees separate, so an override moves only
	// the project documents.
	nested, _ := newVaultT(t, LayoutNested)
	if err := nested.SetProjectMapping("/Users/x/code/app", "clients/acme"); err != nil {
		t.Fatalf("SetProjectMapping: %v", err)
	}
	if got, want := nested.ResolvedPersonalPath("/Users/x/code/app"),
		"personal/app.md"; got != want {
		t.Errorf("nested ResolvedPersonalPath = %q, want %q", got, want)
	}
}
