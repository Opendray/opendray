// Package sourcecontrol orchestrates the "source-control" panel
// plugin: repo discovery, multi-file diff, commit history, and DB-
// backed per-session baselines. Low-level git invocations are
// delegated to gateway/git; this package provides the higher-level
// shape the panel needs.
package sourcecontrol

import (
	"time"

	"github.com/opendray/opendray/gateway/git"
)

// Config is the plugin's effective configuration after manifest-
// default fallback and $HOME expansion. Mirrors the subset of fields
// the source-control plugin surfaces via configSchema; forge +
// bookmarks live in plugin_kv (managed via dedicated API, not the
// Configure form).
type Config struct {
	AllowedRoots     []string
	GitBinary        string
	LogLimit         int
	DiffContext      int
	CommandTimeout   time.Duration
	MarkdownPreview  bool
}

// gitConfig adapts our Config into the one gateway/git understands so
// SecurePath / Status / Diff / Log / Branches all see the same roots
// and timeouts. Kept in one place so upstream signature drift is a
// single-line fix.
func (c Config) gitConfig() git.Config {
	return git.Config{
		AllowedRoots: append([]string{}, c.AllowedRoots...),
		GitBinary:    c.GitBinary,
		LogLimit:     c.LogLimit,
		DiffContext:  c.DiffContext,
		Timeout:      c.CommandTimeout,
	}
}
