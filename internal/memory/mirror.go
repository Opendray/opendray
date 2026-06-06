package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Mirror copies an agent CLI's local memory files into the opendray
// pgvector store so cross-CLI search picks them up.
//
// V1 covers Claude Code only — its local memory format is well-known
// and the most common case. Codex / Gemini have less concrete local
// memory conventions; once their habits stabilise we add similar
// ingestors here under per-CLI helpers.
//
// Sync semantics: idempotent. Each memory file is keyed by absolute
// path; metadata.source_mtime tracks freshness. Re-ingesting the
// same path with the same mtime is a no-op. A path's mtime moving
// forward triggers re-ingest (we don't update; we add a new memory
// row tagged with the new mtime so old context isn't lost).
//
// The mirror is a one-way pipe: Claude writes files → opendray
// reads + persists. opendray-memory's own writes never go back to
// the filesystem, so there's no risk of feedback loops.
type Mirror struct {
	svc *Service
	log *slog.Logger
}

// NewMirror returns a Mirror that writes through svc. Pass a non-
// nil logger so per-file decisions get audited; passing nil falls
// back to slog.Default().
func NewMirror(svc *Service, log *slog.Logger) *Mirror {
	if log == nil {
		log = slog.Default()
	}
	if svc == nil {
		return nil
	}
	return &Mirror{svc: svc, log: log.With("component", "memory.mirror")}
}

// SyncCwd ingests every Claude memory file currently sitting in the
// per-project memory dirs that match `cwd`. Returns the number of
// new memories ingested in this call (0 when nothing was new).
// Idempotent: safe to call multiple times in sequence; no-ops when
// no Claude memory dir exists for the cwd.
func (m *Mirror) SyncCwd(ctx context.Context, cwd string) (int, error) {
	if m == nil || m.svc == nil {
		return 0, errors.New("memory: mirror not initialised")
	}
	if strings.TrimSpace(cwd) == "" {
		return 0, errors.New("memory: SyncCwd needs a cwd")
	}

	dirs := findClaudeMemoryDirs(cwd)
	if len(dirs) == 0 {
		return 0, nil
	}

	// Snapshot what we've already ingested for this cwd so we can
	// dedupe against source_path. One List call per cwd is cheaper
	// than one query per file.
	existing, err := m.svc.List(ctx, ScopeProject, cwd, 1000)
	if err != nil {
		return 0, fmt.Errorf("snapshot existing memories: %w", err)
	}
	seen := make(map[string]string, len(existing)) // sourcePath → ingested mtime
	for _, mem := range existing {
		md := mem.Metadata
		if md == nil {
			continue
		}
		path, _ := md["source_path"].(string)
		mtime, _ := md["source_mtime"].(string)
		if path != "" {
			seen[path] = mtime
		}
	}

	ingested := 0
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			m.log.Debug("readdir", "dir", dir, "err", err)
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			// Skip MEMORY.md — it's an index, not a fact.
			if e.Name() == "MEMORY.md" {
				continue
			}
			path := filepath.Join(dir, e.Name())
			info, err := e.Info()
			if err != nil {
				continue
			}
			mtime := info.ModTime().UTC().Format(time.RFC3339)
			if prev, ok := seen[path]; ok && prev == mtime {
				continue
			}

			body, err := os.ReadFile(path)
			if err != nil {
				m.log.Debug("read claude memory", "path", path, "err", err)
				continue
			}
			if len(body) == 0 {
				continue
			}

			meta := map[string]interface{}{
				"source":       "claude_local_memory",
				"source_path":  path,
				"source_mtime": mtime,
				"source_hash":  shortHash(body),
			}
			if _, err := m.svc.Store(ctx, StoreRequest{
				Text:     string(body),
				Scope:    ScopeProject,
				ScopeKey: cwd,
				Metadata: meta,
			}); err != nil {
				m.log.Warn("ingest claude memory", "path", path, "err", err)
				continue
			}
			ingested++
		}
	}
	if ingested > 0 {
		m.log.Info("synced claude memory files",
			"cwd", cwd, "count", ingested)
	}
	return ingested, nil
}

// BackfillAll runs a one-time import of every known project's Claude
// memory files into the DB. It is the M-U Phase 5 upgrade step: an
// operator who updates opendray gets their pre-existing file memories
// folded into the single pgvector store immediately, instead of only
// after each project's next session spawn.
//
// It enumerates the project scope_keys already present in the store
// (every cwd opendray has memory for) and SyncCwd's each. A project
// that has file memory but no DB memory yet — i.e. opendray has never
// spawned a session in it — is not enumerated here; it imports on its
// first spawn through the per-spawn mirror (WithMemoryMirror), which
// stays wired as the ongoing capture net. So nothing is lost; this
// only changes *when* already-known projects get imported (now, vs
// their next spawn).
//
// Idempotent: SyncCwd dedupes by source_path + mtime, so running this
// on every startup is safe and becomes a no-op once import is caught
// up. Soft-fails per cwd so one unreadable project dir can't abort the
// whole import. Returns (projects scanned, memories ingested).
func (m *Mirror) BackfillAll(ctx context.Context) (projects, ingested int, err error) {
	if m == nil || m.svc == nil {
		return 0, 0, errors.New("memory: mirror not initialised")
	}
	keys, err := m.svc.ListScopeKeys(ctx, ScopeProject)
	if err != nil {
		return 0, 0, fmt.Errorf("list project scope keys: %w", err)
	}
	for _, cwd := range keys {
		if strings.TrimSpace(cwd) == "" {
			continue
		}
		n, serr := m.SyncCwd(ctx, cwd)
		if serr != nil {
			m.log.Debug("backfill sync", "cwd", cwd, "err", serr)
			continue
		}
		projects++
		ingested += n
	}
	if ingested > 0 {
		m.log.Info("file-memory backfill import complete",
			"projects", projects, "ingested", ingested)
	}
	return projects, ingested, nil
}

// findClaudeMemoryDirs returns every existing
// `<root>/projects/<encoded-cwd>/memory` directory, scanning both
// the standard ~/.claude root and every per-account root under
// ~/.claude-accounts. Empty result when no such dir exists.
//
// Mirrors the discovery logic the History panel uses for jsonl
// transcripts (see internal/session/claude_jsonl.go) — copied
// rather than imported to avoid cross-package coupling.
func findClaudeMemoryDirs(cwd string) []string {
	home := os.Getenv("HOME")
	if home == "" {
		return nil
	}
	roots := []string{filepath.Join(home, ".claude", "projects")}
	accountsDir := filepath.Join(home, ".claude-accounts")
	if entries, err := os.ReadDir(accountsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() || e.Name() == "tokens" {
				continue
			}
			roots = append(roots, filepath.Join(accountsDir, e.Name(), "projects"))
		}
	}

	parts := splitPathParts(cwd)
	if len(parts) == 0 {
		return nil
	}

	out := make([]string, 0, 2)
	seen := make(map[string]bool)
	for _, root := range roots {
		canon, err := filepath.EvalSymlinks(root)
		if err != nil {
			canon = root
		}
		entries, err := os.ReadDir(canon)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if !matchesEncodedCwd(e.Name(), parts) {
				continue
			}
			memDir := filepath.Join(canon, e.Name(), "memory")
			real, err := filepath.EvalSymlinks(memDir)
			if err != nil {
				real = memDir
			}
			if seen[real] {
				continue
			}
			if info, err := os.Stat(memDir); err == nil && info.IsDir() {
				out = append(out, memDir)
				seen[real] = true
			}
		}
	}
	return out
}

// splitPathParts splits cwd on "/" and drops empty segments. Same
// implementation as session.splitPathParts; duplicated here so the
// memory package doesn't depend on session.
func splitPathParts(p string) []string {
	out := make([]string, 0, 8)
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// matchesEncodedCwd checks whether `encoded` (a Claude project dir
// name like "-Users-alice-projects-foo") contains every cwd part
// in order. Same logic as session.matchesClaudeProjectDir.
func matchesEncodedCwd(encoded string, parts []string) bool {
	remaining := encoded
	for _, part := range parts {
		idx := strings.Index(remaining, part)
		if idx >= 0 {
			remaining = remaining[idx+len(part):]
			continue
		}
		// Claude lowercase-or-encodes underscore to dash internally
		// in some cases; try the dash-normalised form.
		normalised := strings.ReplaceAll(part, "_", "-")
		idx = strings.Index(remaining, normalised)
		if idx < 0 {
			return false
		}
		remaining = remaining[idx+len(normalised):]
	}
	return true
}

// shortHash returns the first 16 hex chars of SHA-256(body). Used
// in metadata so dedupe across path/mtime is robust against the
// rare case of an editor that touches the file without changing
// content.
func shortHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:8])
}
