package workspace

import (
	"strings"
	"sync"
)

// Resolver canonicalizes physical worktree paths back to their logical
// project cwd. It is the single enforcement point for design invariant
// #2: no worktree path ever enters the scope-key namespace.
//
// Two providers derive their scope key from the MCP subprocess's own
// working directory (antigravity's HOME-global config, dbtool's
// CWD_FROM_CWD fallback) — inside a worktree that yields the worktree
// path. Every scope-key consumer that might receive such a path calls
// Canonical, which maps managed worktree paths (exact match or any
// subpath) to the owning session's cwd and passes everything else
// through untouched — including non-path keys like integration zones.
type Resolver struct {
	mu sync.RWMutex
	// byRoot maps a worktree checkout root → logical cwd. Subpath
	// matching anchors on the root, so anything inside the worktree
	// canonicalizes to the project.
	byRoot map[string]string
	// byWorkDir maps the exact spawn dir → logical cwd (covers the
	// subdir-cwd case without path walking on the hot path).
	byWorkDir map[string]string
}

func NewResolver() *Resolver {
	return &Resolver{
		byRoot:    make(map[string]string),
		byWorkDir: make(map[string]string),
	}
}

// Register records a managed worktree: root is the worktree checkout
// root, workDir the session's spawn dir inside it, cwd the logical
// project path.
func (r *Resolver) Register(root, workDir, cwd string) {
	if root == "" || cwd == "" {
		return
	}
	if workDir == "" {
		workDir = root
	}
	r.mu.Lock()
	r.byRoot[root] = cwd
	r.byWorkDir[workDir] = cwd
	r.mu.Unlock()
}

// Unregister drops the mapping for a worktree root (and any workDir
// entries under it).
func (r *Resolver) Unregister(root string) {
	if root == "" {
		return
	}
	r.mu.Lock()
	delete(r.byRoot, root)
	for wd := range r.byWorkDir {
		if wd == root || strings.HasPrefix(wd, root+"/") {
			delete(r.byWorkDir, wd)
		}
	}
	r.mu.Unlock()
}

// Canonical maps a managed worktree path to its logical project cwd.
// Unknown paths (and non-path scope keys) come back unchanged.
func (r *Resolver) Canonical(p string) string {
	if p == "" {
		return p
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if cwd, ok := r.byWorkDir[p]; ok {
		return cwd
	}
	if cwd, ok := r.byRoot[p]; ok {
		return cwd
	}
	for root, cwd := range r.byRoot {
		if strings.HasPrefix(p, root+"/") {
			return cwd
		}
	}
	return p
}
