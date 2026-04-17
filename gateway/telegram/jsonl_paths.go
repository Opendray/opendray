package telegram

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/linivek/ntc/kernel/store"
)

// claudeBasePaths builds the ordered list of Claude config directories to
// search for structured JSONL session files. Each returned path is a Claude
// config root (the directory that contains a "projects/" subdir).
//
// Priority (de-duplicated, first-win on ordering):
//   1. Session's bound account's ConfigDir — when a session is launched with
//      CLAUDE_CONFIG_DIR set (multi-account), this is where its JSONL lives.
//   2. Every enabled claude_accounts row's ConfigDir — covers the shared
//      session case where the same session may be driven by different
//      accounts over time; ResolveLatestJSONL picks the freshest file.
//   3. $HOME/.claude — default location used by single-account setups.
//   4. User-supplied extras from the plugin's extraClaudeDirs config field.
//
// Passing a nil db or a session with no ClaudeAccountID is safe: the helper
// simply skips those lookups and still returns a usable default list.
func claudeBasePaths(ctx context.Context, db *store.DB, sess store.Session, extra []string) []string {
	seen := map[string]bool{}
	var paths []string
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		paths = append(paths, p)
	}

	if db != nil {
		if sess.ClaudeAccountID != "" {
			if acc, err := db.GetClaudeAccount(ctx, sess.ClaudeAccountID); err == nil {
				add(acc.ConfigDir)
			}
		}
		if accs, err := db.ListClaudeAccounts(ctx); err == nil {
			for _, a := range accs {
				if a.Enabled {
					add(a.ConfigDir)
				}
			}
		}
	}

	if home := os.Getenv("HOME"); home != "" {
		add(filepath.Join(home, ".claude"))
	}
	for _, p := range extra {
		add(strings.TrimSpace(p))
	}
	return paths
}

// parseExtraClaudeDirs splits a comma / newline separated plugin-config
// string into trimmed directory entries. Empty input returns nil.
func parseExtraClaudeDirs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// Split on comma or newline — both feel natural in a textarea-style field.
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})
	var out []string
	for _, f := range fields {
		if t := strings.TrimSpace(f); t != "" {
			out = append(out, t)
		}
	}
	return out
}
