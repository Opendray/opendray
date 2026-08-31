package grokacct

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// accountHasCredentials reports whether a GROK_HOME dir holds a logged-in
// grok login token (<GROK_HOME>/auth.json, non-empty regular file).
func accountHasCredentials(home string) bool {
	if home == "" {
		return false
	}
	return fileExists(filepath.Join(home, grokAuthRelPath))
}

// fileExists reports whether path exists and is a non-empty regular file.
// Uses Lstat so symlinks return false — defense in depth against an
// attacker who can write under the accounts dir.
func fileExists(path string) bool {
	st, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return st.Mode().IsRegular() && st.Size() > 0
}

// defaultGrokHome returns the gateway user's own grok home: GROK_HOME when
// set, else ~/.grok. Empty only when neither GROK_HOME nor HOME is set.
func defaultGrokHome() string {
	if v := os.Getenv("GROK_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".grok")
}

// selectSpawnHome is the pure-filesystem half of ResolveSpawnHome: given an
// account's name + GROK_HOME dir it validates the dir is set and holds a
// logged-in grok token, returning the GROK_HOME to inject or a
// guided-login error. No DB access, so it's unit-testable without a store
// (mirrors agyacct's selectSpawnHome / cliacct's selectSpawnCreds).
func selectSpawnHome(name, home string) (string, error) {
	if home == "" {
		return "", fmt.Errorf("grok account %q has no GROK_HOME directory configured", name)
	}
	if !accountHasCredentials(home) {
		return "", fmt.Errorf(
			"grok account %q is not logged in: no %s under %s — run `GROK_HOME=%s grok login` on the host and complete the xAI sign-in",
			name, grokAuthRelPath, home, home)
	}
	return home, nil
}

// discoveredAccount is one local account candidate found on disk.
type discoveredAccount struct {
	name        string
	displayName string
	configDir   string // explicit when non-empty; otherwise Create derives
}

// discoverLocalAccounts returns every grok account that should be surfaced
// in the panel, in discovery order. Two sources:
//
//  1. the gateway user's own grok home (GROK_HOME or ~/.grok) — the
//     primary `grok login`, used when no grok_account_id is pinned to a
//     session. Yielded as a synthetic "default" entry, only when its
//     login token actually exists.
//  2. <accountsDir>/<name>/ — a per-account GROK_HOME created via
//     `GROK_HOME=<dir> grok login`. Only dirs that already contain the
//     login token are emitted (a half-set-up dir isn't a usable account).
//
// Symlinks are rejected at every step; a missing dir is not an error.
func discoverLocalAccounts(accountsDir string) ([]discoveredAccount, error) {
	var out []discoveredAccount
	seen := map[string]bool{}
	emit := func(d discoveredAccount) {
		if d.name == "" || seen[d.name] {
			return
		}
		seen[d.name] = true
		out = append(out, d)
	}

	// 1) Synthetic "default" — the gateway user's own grok home.
	if home := defaultGrokHome(); home != "" && accountHasCredentials(home) {
		emit(discoveredAccount{
			name:        "default",
			displayName: "Default (~/.grok)",
			configDir:   home,
		})
	}

	// 2) Per-account GROK_HOMEs under accountsDir.
	dirEntries, err := os.ReadDir(accountsDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", accountsDir, err)
	}
	for _, e := range dirEntries {
		if !e.IsDir() {
			continue
		}
		// Reject symlinked account dirs so a malicious symlink can't feed
		// an arbitrary GROK_HOME to a spawn.
		if e.Type()&os.ModeSymlink != 0 {
			continue
		}
		dir := filepath.Join(accountsDir, e.Name())
		if !accountHasCredentials(dir) {
			continue // not a logged-in grok home
		}
		emit(discoveredAccount{name: e.Name(), configDir: dir})
	}
	return out, nil
}
