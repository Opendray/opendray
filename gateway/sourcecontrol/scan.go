package sourcecontrol

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/opendray/opendray/gateway/git"
)

// Repo is one entry surfaced in the panel's repo switcher. A repo may
// come from an allowedRoots scan, a user bookmark, or both (which is
// common — bookmarks usually point at discovered repos).
type Repo struct {
	Path         string `json:"path"`         // absolute
	Name         string `json:"name"`         // basename, nicer to display
	IsGit        bool   `json:"isGit"`        // has a .git entry (dir or file for worktrees)
	IsBookmarked bool   `json:"isBookmarked"` // user-pinned
}

// scanLimits bounds the directory walk so a misconfigured allowedRoots
// pointing at $HOME doesn't blow up memory or block the gateway.
type scanLimits struct {
	maxDepth int // folders deep from each root; 0 = unlimited (not recommended)
	maxRepos int // hard stop; once we find this many, scan halts
}

var defaultLimits = scanLimits{maxDepth: 6, maxRepos: 200}

// skipDirs are never descended into — common heavyweight subtrees
// that inflate walk time and never hold real repos worth surfacing.
var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	".venv":        true,
	"venv":         true,
	"__pycache__":  true,
	"dist":         true,
	"build":        true,
	".pub-cache":   true,
	".gradle":      true,
	".cache":       true,
	".npm":         true,
	".pnpm":        true,
	".yarn":        true,
}

// DiscoverRepos walks cfg.AllowedRoots and returns every directory
// containing a .git entry, merged with the explicit bookmark list.
// Bookmarks win on duplicates — the IsBookmarked flag is set — and
// invalid bookmarks (outside allowedRoots, no longer existing) are
// still returned with IsGit=false so the UI can let the user remove
// them.
//
// Result is sorted: bookmarks first (alphabetical), then discovered
// repos (alphabetical). The panel uses that ordering as-is.
func DiscoverRepos(ctx context.Context, cfg Config, bookmarks []string) ([]Repo, error) {
	seen := make(map[string]*Repo)

	// 1. Scan allowed roots for .git directories.
	for _, root := range cfg.AllowedRoots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		if err := scanRoot(ctx, absRoot, defaultLimits, seen); err != nil {
			return nil, fmt.Errorf("sourcecontrol: scan %s: %w", root, err)
		}
	}

	// 2. Overlay bookmarks. A bookmark that resolves inside
	//    allowedRoots is marked on the matching discovered repo (or
	//    added fresh if outside scan depth / depth-limited). Stale
	//    bookmarks whose on-disk target no longer exists are dropped
	//    silently — the UI has a separate "clean up stale bookmarks"
	//    flow if we want it later.
	gitCfg := cfg.gitConfig()
	for _, bm := range bookmarks {
		p := strings.TrimSpace(bm)
		if p == "" {
			continue
		}
		abs, err := git.SecurePath(gitCfg, p)
		if err != nil {
			continue // outside allowedRoots or unresolvable
		}
		r := seen[abs]
		if r == nil {
			if !isGitWorkTree(abs) {
				continue // not a repo (never was, or deleted) — drop
			}
			r = &Repo{Path: abs, Name: filepath.Base(abs), IsGit: true}
			seen[abs] = r
		}
		r.IsBookmarked = true
	}

	out := make([]Repo, 0, len(seen))
	for _, r := range seen {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsBookmarked != out[j].IsBookmarked {
			return out[i].IsBookmarked // bookmarks first
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

func scanRoot(ctx context.Context, root string, lim scanLimits, seen map[string]*Repo) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	// Normalise through EvalSymlinks so discovered paths use the
	// same canonical form SecurePath will produce for bookmarks —
	// otherwise macOS /var/folders/... vs /private/var/folders/...
	// double-counts the same repo.
	if real, err := filepath.EvalSymlinks(rootAbs); err == nil {
		rootAbs = real
	}
	info, err := os.Stat(rootAbs)
	if err != nil || !info.IsDir() {
		return nil // non-existent / non-dir roots are skipped, not fatal
	}
	if isGitWorkTree(rootAbs) && seen[rootAbs] == nil {
		seen[rootAbs] = &Repo{Path: rootAbs, Name: filepath.Base(rootAbs), IsGit: true}
	}
	return filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir // unreadable subtree: skip, don't fail the whole scan
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if lim.maxRepos > 0 && len(seen) >= lim.maxRepos {
			return filepath.SkipAll
		}
		if !d.IsDir() {
			return nil
		}
		// Depth check
		rel, _ := filepath.Rel(rootAbs, path)
		depth := 0
		if rel != "." {
			depth = strings.Count(rel, string(os.PathSeparator)) + 1
		}
		if lim.maxDepth > 0 && depth > lim.maxDepth {
			return filepath.SkipDir
		}
		name := d.Name()
		if skipDirs[name] {
			return filepath.SkipDir
		}
		// Detect .git at this level: if present, this dir is the repo
		// root. Don't descend further — nested .git inside node_modules
		// is already skipped, but stopping here cuts work anyway.
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			if seen[path] == nil {
				seen[path] = &Repo{Path: path, Name: filepath.Base(path), IsGit: true}
			} else {
				seen[path].IsGit = true
			}
			return filepath.SkipDir
		}
		return nil
	})
}

// isGitWorkTree reports whether path has a .git child (directory for
// normal repos, file for linked worktrees).
func isGitWorkTree(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}
