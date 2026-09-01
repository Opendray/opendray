package grokacct

import (
	"fmt"
	"os"
	"path/filepath"
)

// sharedAssetDirs are the account-independent, heavy directories inside a
// grok home that can be shared across accounts via symlink instead of
// duplicated (bin ~317 MB, plus the bundled runtime, vendored deps, and
// caches). Per-account state (auth.json, config.toml, trusted_folders.toml,
// sessions/, projects/, agent_id, worktrees.db) is deliberately NOT here —
// those must stay unique per GROK_HOME.
var sharedAssetDirs = []string{
	"bin",
	"bundled",
	"vendor",
	"downloads",
	"installed-plugins",
	"marketplace-cache",
	"completions",
	"docs",
}

// EnsureSharedAssets symlinks the heavy, account-independent directories of
// the gateway user's own grok home into an account's GROK_HOME, so N
// accounts don't each cost hundreds of MB. Best-effort and idempotent;
// callers log and continue on error. Mirrors catalog.ensureAgySharedCache.
func EnsureSharedAssets(home string) error {
	return ensureSharedAssets(home, defaultGrokHome())
}

// ensureSharedAssets is the pure half of EnsureSharedAssets: symlink each
// shareable dir from src into home when it exists in src and is absent in
// home. No-op when src is empty/missing or equals home (never symlink the
// shared source into itself). Existing entries in home are left untouched.
func ensureSharedAssets(home, src string) error {
	if home == "" || src == "" || home == src {
		return nil
	}
	if st, err := os.Stat(src); err != nil || !st.IsDir() {
		return nil // nothing to share yet
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return fmt.Errorf("mkdir grok home: %w", err)
	}
	for _, d := range sharedAssetDirs {
		srcDir := filepath.Join(src, d)
		if st, err := os.Stat(srcDir); err != nil || !st.IsDir() {
			continue // source doesn't have this dir
		}
		dst := filepath.Join(home, d)
		if _, err := os.Lstat(dst); err == nil {
			continue // already present (real dir or prior symlink) — don't clobber
		}
		if err := os.Symlink(srcDir, dst); err != nil {
			return fmt.Errorf("symlink %s: %w", d, err)
		}
	}
	return nil
}
