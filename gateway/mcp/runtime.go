// Package mcp manages MCP server definitions and injects them into
// agent CLI sessions by writing per-session temporary config files.
// No global configs (~/.claude.json, ~/.codex/config.toml) are touched.
package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/opendray/opendray/kernel/store"
)

// Runtime is the session-scoped MCP injector. It also holds the
// database-backed CRUD for MCP server definitions.
type Runtime struct {
	db      *store.DB
	logger  *slog.Logger
	baseDir string

	mu   sync.Mutex
	dirs map[string]string // sessionID → per-session tmp dir
}

// Config configures the MCP runtime.
type Config struct {
	DB      *store.DB
	Logger  *slog.Logger
	BaseDir string // parent dir for per-session scratch; defaults to /tmp/opendray-mcp
}

// New constructs a Runtime.
func New(cfg Config) *Runtime {
	base := cfg.BaseDir
	if base == "" {
		base = filepath.Join(os.TempDir(), "opendray-mcp")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	_ = os.MkdirAll(base, 0o700)
	return &Runtime{
		db:      cfg.DB,
		logger:  logger,
		baseDir: base,
		dirs:    make(map[string]string),
	}
}

// DB exposes the underlying store for handlers.
func (r *Runtime) DB() *store.DB { return r.db }

// SupportsAgent reports whether an agent has a renderer registered.
func (r *Runtime) SupportsAgent(agent string) bool { return supportsAgent(agent) }

// RenderFor builds a per-session config file for the given agent and
// returns the args / env that must be appended to the CLI spawn.
//
// It is a no-op (zero Injection, nil error) when:
//   - the agent has no renderer
//   - no enabled MCP server applies to the agent
//
// The caller MUST invoke Cleanup(sessionID) when the session ends.
func (r *Runtime) RenderFor(ctx context.Context, sessionID, agent string) (Injection, error) {
	rdr, ok := renderers[agent]
	if !ok {
		return Injection{}, nil
	}

	servers, err := r.db.ListMCPServers(ctx)
	if err != nil {
		return Injection{}, fmt.Errorf("mcp: list servers: %w", err)
	}
	filtered := filterForAgent(servers, agent)
	if len(filtered) == 0 {
		return Injection{}, nil
	}

	dir := filepath.Join(r.baseDir, sessionID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Injection{}, fmt.Errorf("mcp: mkdir session dir: %w", err)
	}

	inj, err := rdr.render(dir, filtered)
	if err != nil {
		// best-effort cleanup so we don't leak on error
		_ = os.RemoveAll(dir)
		return Injection{}, err
	}

	if len(inj.Args) == 0 && len(inj.Env) == 0 {
		_ = os.RemoveAll(dir)
		return Injection{}, nil
	}

	r.mu.Lock()
	r.dirs[sessionID] = dir
	r.mu.Unlock()

	r.logger.Info("mcp injected",
		"session", sessionID, "agent", agent,
		"servers", len(filtered), "dir", dir)
	return inj, nil
}

// Cleanup removes the per-session scratch directory. Safe to call
// multiple times; safe to call for sessions that had no injection.
func (r *Runtime) Cleanup(sessionID string) {
	r.mu.Lock()
	dir, ok := r.dirs[sessionID]
	if ok {
		delete(r.dirs, sessionID)
	}
	r.mu.Unlock()
	if !ok {
		return
	}
	if err := os.RemoveAll(dir); err != nil {
		r.logger.Warn("mcp cleanup failed", "session", sessionID, "error", err)
	}
}

// filterForAgent keeps only enabled servers that apply to agent.
// applies_to of ["*"] (or empty) means every agent.
func filterForAgent(servers []store.MCPServer, agent string) []store.MCPServer {
	out := make([]store.MCPServer, 0, len(servers))
	for _, s := range servers {
		if !s.Enabled {
			continue
		}
		if appliesToAgent(s.AppliesTo, agent) {
			out = append(out, s)
		}
	}
	return out
}

func appliesToAgent(applies []string, agent string) bool {
	if len(applies) == 0 {
		return true
	}
	for _, a := range applies {
		if a == "*" || a == agent {
			return true
		}
	}
	return false
}
