package sourcecontrol

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// mkRepo creates a dir with a .git subdir so DiscoverRepos treats it
// as a real repo.
func mkRepo(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(path, ".git"), 0o755); err != nil {
		t.Fatalf("mkRepo: %v", err)
	}
}

func TestDiscoverRepos_ScansAllowedRoots(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "alpha"))
	mkRepo(t, filepath.Join(root, "beta"))
	// A deep directory without .git — should not surface.
	if err := os.MkdirAll(filepath.Join(root, "not-a-repo", "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := Config{AllowedRoots: []string{root}}
	got, err := DiscoverRepos(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("DiscoverRepos: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d repos, want 2: %+v", len(got), got)
	}
	for _, r := range got {
		if !r.IsGit {
			t.Errorf("%s: IsGit should be true", r.Path)
		}
		if r.IsBookmarked {
			t.Errorf("%s: IsBookmarked should be false (no bookmarks passed)", r.Path)
		}
	}
}

func TestDiscoverRepos_SkipsHeavyweightDirs(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "app"))
	// node_modules descendant with its own .git — must be ignored.
	mkRepo(t, filepath.Join(root, "app", "node_modules", "some-pkg"))

	cfg := Config{AllowedRoots: []string{root}}
	got, err := DiscoverRepos(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("DiscoverRepos: %v", err)
	}
	if len(got) != 1 || filepath.Base(got[0].Path) != "app" {
		t.Fatalf("expected only 'app', got %+v", got)
	}
}

func TestDiscoverRepos_BookmarksMarkedAndSortedFirst(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "aaa"))
	mkRepo(t, filepath.Join(root, "zzz"))

	cfg := Config{AllowedRoots: []string{root}}
	bookmarks := []string{filepath.Join(root, "zzz")}
	got, err := DiscoverRepos(context.Background(), cfg, bookmarks)
	if err != nil {
		t.Fatalf("DiscoverRepos: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
	if !got[0].IsBookmarked || filepath.Base(got[0].Path) != "zzz" {
		t.Errorf("expected zzz bookmarked and first; got %+v", got[0])
	}
	if got[1].IsBookmarked || filepath.Base(got[1].Path) != "aaa" {
		t.Errorf("expected aaa second, unbookmarked; got %+v", got[1])
	}
}

func TestDiscoverRepos_BookmarkOutsideRootsIgnored(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "in"))

	elsewhere := t.TempDir()
	mkRepo(t, filepath.Join(elsewhere, "out"))

	cfg := Config{AllowedRoots: []string{root}}
	got, err := DiscoverRepos(context.Background(), cfg,
		[]string{filepath.Join(elsewhere, "out")})
	if err != nil {
		t.Fatalf("DiscoverRepos: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("outside-root bookmark must be dropped; got %+v", got)
	}
}

func TestDiscoverRepos_StaleBookmarkDropped(t *testing.T) {
	root := t.TempDir()
	// No repo created — the path doesn't exist at all.
	gone := filepath.Join(root, "deleted-project")

	cfg := Config{AllowedRoots: []string{root}}
	got, err := DiscoverRepos(context.Background(), cfg, []string{gone})
	if err != nil {
		t.Fatalf("DiscoverRepos: %v", err)
	}
	// Stale bookmarks (target gone / not a repo) are silently dropped.
	// Future work: a dedicated "clean up stale bookmarks" API can
	// diff stored bookmarks against this list to surface prunable ones.
	if len(got) != 0 {
		t.Fatalf("stale bookmark should be dropped; got %+v", got)
	}
}

func TestDiscoverRepos_NonExistentRootIsSoft(t *testing.T) {
	cfg := Config{AllowedRoots: []string{"/definitely/does/not/exist/xyz"}}
	got, err := DiscoverRepos(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("expected soft skip, got error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no repos, got %+v", got)
	}
}

func TestDiscoverRepos_ContextCancellation(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		mkRepo(t, filepath.Join(root, "r"+string(rune('a'+i))))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cfg := Config{AllowedRoots: []string{root}}
	_, err := DiscoverRepos(ctx, cfg, nil)
	// Either a context error bubbled up, or the scan returned early —
	// both are acceptable; we're just asserting it doesn't hang.
	_ = err

	// Also verify that a tight deadline doesn't deadlock.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel2()
	_, _ = DiscoverRepos(ctx2, cfg, nil)
}
