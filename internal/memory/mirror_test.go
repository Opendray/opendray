package memory

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestMatchesEncodedCwd(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
		cwd     string
		want    bool
	}{
		{"exact ordered match", "-Users-alice-projects-foo", "/Users/alice/projects/foo", true},
		{"underscore normalised to dash", "-home-bob-my-app", "/home/bob/my_app", true},
		{"missing trailing segment", "-Users-alice-projects", "/Users/alice/projects/foo", false},
		{"out of order", "-foo-Users-alice", "/Users/alice/foo", false},
		{"superset dir still matches subset cwd", "-Users-alice-projects-foo-bar", "/Users/alice/projects/foo", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchesEncodedCwd(tc.encoded, splitPathParts(tc.cwd))
			if got != tc.want {
				t.Fatalf("matchesEncodedCwd(%q, %q) = %v, want %v",
					tc.encoded, tc.cwd, got, tc.want)
			}
		})
	}
}

func TestFindClaudeMemoryDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// findClaudeMemoryDirs resolves symlinks on the roots; on macOS
	// t.TempDir() lives under /tmp which is itself a symlink to
	// /private/tmp, so resolve the expected paths the same way.
	resolved, err := filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatal(err)
	}

	cwd := "/srv/work/myproj"
	// Standard ~/.claude root + a per-account root: both should be found.
	want := []string{
		filepath.Join(resolved, ".claude", "projects", "-srv-work-myproj", "memory"),
		filepath.Join(resolved, ".claude-accounts", "kev", "projects", "-srv-work-myproj", "memory"),
	}
	// A non-matching project dir that must be ignored.
	decoy := filepath.Join(home, ".claude", "projects", "-srv-work-other", "memory")
	for _, d := range append(append([]string{}, want...), decoy) {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got := findClaudeMemoryDirs(cwd)
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("findClaudeMemoryDirs returned %d dirs, want %d:\n got=%v\nwant=%v",
			len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("dir[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBackfillAllNilGuard(t *testing.T) {
	if _, _, err := (*Mirror)(nil).BackfillAll(context.Background()); err == nil {
		t.Fatal("BackfillAll on nil mirror should error")
	}
	m := &Mirror{} // svc nil
	if _, _, err := m.BackfillAll(context.Background()); err == nil {
		t.Fatal("BackfillAll with nil svc should error")
	}
}
