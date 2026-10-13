package session

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// The stored ClaudeSessionID is the id opendray launched claude with
// (`--session-id X` / `--resume X`). Claude names a transcript file after
// the conversation, not the process: after `/clear` (and on some resumes)
// the CLI keeps running under the same launch id but appends to a NEW
// <uuid>.jsonl, tagging every record "session_id":"<launch id>". So the
// stored id can point at an older conversation, or at no file at all —
// and `claude --resume <missing id>` exits at once, leaving a session that
// dies on every start. resolveClaudeResumeID maps the stored id to the
// transcript the session is actually writing.

const (
	// Newest transcripts in the workspace to inspect; a session's live
	// conversation is always among the most recently written.
	claudeResumeMaxCandidates = 20
	// Tail of each candidate searched for the run id. The run's records
	// continue until its process ends, so they sit at the end of the file.
	claudeResumeTailBytes = 2 << 20
)

var claudeWorkspaceUnsafe = regexp.MustCompile(`[^a-zA-Z0-9]`)

// claudeWorkspaceName is the projects/ subdirectory Claude uses for a
// working directory: every non-alphanumeric byte becomes '-'
// (/var/lib/x → -var-lib-x).
func claudeWorkspaceName(workDir string) string {
	return claudeWorkspaceUnsafe.ReplaceAllString(workDir, "-")
}

// resolveClaudeResumeID returns the transcript id `claude --resume`
// should use for storedID under configDir, or "" when no transcript for
// it exists (the caller then starts a fresh conversation instead of a
// doomed resume).
//
// Within the workspace for workDir, the most recently modified transcript
// that either is named storedID or carries storedID as its run id wins.
// Failing that, a storedID transcript anywhere under projects/ is used
// (sessions whose cwd changed). storedID or configDir empty → storedID
// unchanged, since there is nothing to check against.
func resolveClaudeResumeID(configDir, workDir, storedID string) string {
	if storedID == "" || configDir == "" {
		return storedID
	}
	if workDir != "" {
		dir := filepath.Join(configDir, "projects", claudeWorkspaceName(workDir))
		marker := []byte(`"session_id":"` + storedID + `"`)
		for _, path := range newestTranscripts(dir, claudeResumeMaxCandidates) {
			id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
			if id == storedID || tailContains(path, marker, claudeResumeTailBytes) {
				return id
			}
		}
	}
	matches, _ := filepath.Glob(filepath.Join(configDir, "projects", "*", storedID+".jsonl"))
	for _, m := range matches {
		if st, err := os.Lstat(m); err == nil && st.Mode().IsRegular() {
			return storedID
		}
	}
	return ""
}

// newestTranscripts lists dir's regular *.jsonl files, newest first,
// capped at limit.
func newestTranscripts(dir string, limit int) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	type cand struct {
		path string
		mod  int64
	}
	var cands []cand
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".jsonl") || !e.Type().IsRegular() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		cands = append(cands, cand{filepath.Join(dir, e.Name()), info.ModTime().UnixNano()})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].mod > cands[j].mod })
	if len(cands) > limit {
		cands = cands[:limit]
	}
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.path
	}
	return out
}

// tailContains reports whether the last n bytes of path contain needle.
func tailContains(path string, needle []byte, n int64) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return false
	}
	off := st.Size() - n
	if off < 0 {
		off = 0
	}
	buf, err := io.ReadAll(io.NewSectionReader(f, off, st.Size()-off))
	if err != nil {
		return false
	}
	return bytes.Contains(buf, needle)
}

// verifyClaudeResumeID points sess.ClaudeSessionID at the transcript the
// session is actually writing before a `claude --resume`, or clears it
// (so the spawn starts a fresh conversation) when no transcript exists.
// Without this, a stale id makes the CLI exit at once on every start.
// The adapter echoes the id back as the prepared session id, so a
// corrected value is persisted by the spawn.
func (m *Manager) verifyClaudeResumeID(ctx context.Context, sess *Session, workDir string) {
	if sess.ClaudeSessionID == "" || m.claudeAccounts == nil {
		return
	}
	cfg, err := m.claudeAccounts.ResolveClaudeConfigDir(ctx, sess.ClaudeAccountID)
	if err != nil || cfg == "" {
		return
	}
	rid := resolveClaudeResumeID(cfg, workDir, sess.ClaudeSessionID)
	if rid == sess.ClaudeSessionID {
		return
	}
	if rid == "" {
		m.log.Warn("claude transcript for stored session id not found; starting a fresh conversation",
			"session", sess.ID, "claude_session_id", sess.ClaudeSessionID, "config_dir", cfg)
	} else {
		m.log.Info("claude conversation continued in another transcript; resuming that one",
			"session", sess.ID, "stored", sess.ClaudeSessionID, "resume", rid)
	}
	sess.ClaudeSessionID = rid
}
