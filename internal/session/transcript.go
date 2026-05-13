package session

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Turn is one role-tagged transcript entry. The text is normalized
// — no tool-use JSON blobs, no system messages, no thinking. This
// is what the M18 journaler feeds to an LLM summariser.
type Turn struct {
	Role string    // "user" | "assistant"
	Text string    // already trimmed
	Ts   time.Time // when this turn was emitted, when available
}

// Transcript returns the latest CLI conversation as ordered turns.
// Dispatches by provider so each backend (Claude / Codex / Gemini)
// parses its own JSONL format. Returns ([], nil) for providers
// without a transcript reader yet — callers should fall back to
// metadata-only journaling rather than treating it as an error.
//
// maxBytes caps the total turn text returned so a giant
// transcript can't blow up the LLM prompt; 16 KiB is the sane
// default.
func (m *Manager) Transcript(ctx context.Context, sessionID string, maxBytes int) ([]Turn, error) {
	if maxBytes <= 0 {
		maxBytes = 16 * 1024
	}
	sess, err := m.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	var endedAt time.Time
	if sess.EndedAt != nil {
		endedAt = *sess.EndedAt
	}
	switch sess.ProviderID {
	case "claude":
		return claudeTranscript(m.claudeHistoryCfg, sess.Cwd, sess.ClaudeSessionID, sess.StartedAt, endedAt, maxBytes), nil
	case "codex":
		return codexTranscript(m.codexHistoryCfg, sess.Cwd, sess.StartedAt, endedAt, maxBytes), nil
	case "gemini":
		return geminiTranscript(m.geminiHistoryCfg, sess.Cwd, sess.StartedAt, endedAt, maxBytes), nil
	default:
		return nil, nil
	}
}

// TranscriptText is a convenience: dumps the turns as plain
// markdown ("USER:" / "ASSISTANT:" prefix per turn). Used by the
// journaler's LLM prompt so the model gets a familiar shape.
func (m *Manager) TranscriptText(ctx context.Context, sessionID string, maxBytes int) (string, error) {
	turns, err := m.Transcript(ctx, sessionID, maxBytes)
	if err != nil {
		return "", err
	}
	return FormatTranscript(turns), nil
}

// FormatTranscript renders []Turn as a plain-text conversation.
// Public so tests / external callers don't have to reimplement.
func FormatTranscript(turns []Turn) string {
	var b strings.Builder
	for _, t := range turns {
		role := strings.ToUpper(t.Role)
		if role == "" {
			role = "?"
		}
		fmt.Fprintf(&b, "%s: %s\n", role, t.Text)
	}
	return strings.TrimSpace(b.String())
}

// ── Claude ────────────────────────────────────────────────────

// claudeTranscript walks the matching JSONL file and returns
// user + assistant text turns in chronological order, scoped to
// the calling session's identity. Tool-use / tool-result /
// thinking blocks are dropped — the summariser cares about the
// conversation, not the raw tool call payloads.
//
// M22 — three layers of isolation defend against transcript
// cross-contamination (one session reading another session's or
// project's jsonl content, then the LLM treating the wrong
// content as "what just happened"):
//
//  1. **Fail-closed on missing UUID file**: when claudeSessID is
//     set but the named *.jsonl doesn't exist, return nil rather
//     than falling back to "latest mtime in dir" — the latest may
//     be an unrelated, accumulating session.
//  2. **Time-window filter**: each parsed turn must fall within
//     [startedAt-30s, endedAt+30s]. Defensive even when we pick
//     the right file: if the file accumulates content across
//     multiple opendray sessions (Claude Code reuse), only the
//     current spawn's turns survive.
//  3. **Cwd canary**: the first jsonl entry that carries a `cwd`
//     field must match the calling session's cwd exactly. One
//     mismatch and the whole file is rejected. Catches the worst
//     case — a jsonl from a different project being mis-routed
//     into this project's dir.
//
// startedAt may be zero (skips lower-bound check). endedAt may be
// zero (session still running; only lower-bound filtering).
func claudeTranscript(cfg ClaudeHistoryConfig, cwd, claudeSessID string, startedAt, endedAt time.Time, maxBytes int) []Turn {
	roots := resolveClaudeRoots(cfg)
	var path string
	for _, r := range roots {
		dir := findClaudeProjectDir(r, cwd)
		if dir == "" {
			continue
		}
		if claudeSessID != "" {
			candidate := filepath.Join(dir, claudeSessID+".jsonl")
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
			// M22.1 — caller asked for a specific UUID. If it
			// isn't here, don't fall back to the latest mtime in
			// this dir; that's how unrelated sessions leak in.
			// Try the next root instead.
			continue
		}
		p := findLatestClaudeJSONL(dir)
		if p != "" {
			path = p
			break
		}
	}
	if path == "" {
		return nil
	}

	// M22.2 — build the time window. ±30s padding absorbs clock
	// skew between Claude Code's jsonl timestamps and opendray's
	// session.started_at / ended_at.
	var windowStart, windowEnd time.Time
	if !startedAt.IsZero() {
		windowStart = startedAt.Add(-30 * time.Second)
	}
	if !endedAt.IsZero() {
		windowEnd = endedAt.Add(30 * time.Second)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)
	var turns []Turn
	bytesUsed := 0
	cwdChecked := false
	for scanner.Scan() {
		raw := scanner.Bytes()
		var e claudeJSONLEntry
		if err := json.Unmarshal(raw, &e); err != nil {
			continue
		}
		// M22.3 — cwd canary. Check the first entry that carries
		// a cwd field. One mismatch and the whole file is
		// abandoned — better an empty transcript than a wrong
		// summary attributed to the wrong project.
		if !cwdChecked && e.Cwd != "" {
			cwdChecked = true
			if e.Cwd != cwd {
				return nil
			}
		}
		if e.Message == nil {
			continue
		}
		// M22.2 — window filter. Entries without a timestamp are
		// kept (defensive); entries outside the window are
		// dropped silently.
		if !e.Time.IsZero() {
			if !windowStart.IsZero() && e.Time.Before(windowStart) {
				continue
			}
			if !windowEnd.IsZero() && e.Time.After(windowEnd) {
				continue
			}
		}
		text := extractClaudeText(e.Message.Content)
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		role := strings.ToLower(e.Message.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		bytesUsed += len(text) + len(role) + 4
		if bytesUsed > maxBytes && len(turns) > 0 {
			// Drop oldest turns until under budget.
			turns = trimTurnsHead(turns, &bytesUsed, maxBytes)
		}
		turns = append(turns, Turn{Role: role, Text: text, Ts: e.Time})
	}
	return turns
}

// extractClaudeText walks the content array, keeping only "text"
// blocks. Tool calls / thinking / results are summarised inline
// when they carry a human-meaningful field, but the bulk of
// payload is dropped.
func extractClaudeText(content json.RawMessage) string {
	if len(content) == 0 {
		return ""
	}
	// Try string first (some Claude entries wrap the message as a
	// plain string rather than a content-block array).
	var asStr string
	if err := json.Unmarshal(content, &asStr); err == nil {
		return asStr
	}
	var blocks []claudeContentBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		return ""
	}
	var parts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if t := strings.TrimSpace(b.Text); t != "" {
				parts = append(parts, t)
			}
		case "tool_use":
			// Tool calls carry interesting metadata for the summariser
			// (which file? what command?) — keep a one-line summary.
			parts = append(parts, summariseClaudeTool(b))
		case "tool_result":
			// Skip raw outputs — these are usually huge build logs
			// the LLM summariser doesn't need.
		}
	}
	return strings.Join(parts, " ")
}

func summariseClaudeTool(b claudeContentBlock) string {
	name := b.Name
	if name == "" {
		return ""
	}
	if b.Input == nil {
		return "(tool " + name + ")"
	}
	switch name {
	case "Edit", "Write", "MultiEdit", "Read":
		if b.Input.FilePath != "" {
			return "(" + name + " " + b.Input.FilePath + ")"
		}
	case "Bash":
		cmd := b.Input.Command
		if i := strings.IndexByte(cmd, '\n'); i >= 0 {
			cmd = cmd[:i]
		}
		if len(cmd) > 80 {
			cmd = cmd[:80] + "…"
		}
		if cmd != "" {
			return "(Bash: " + cmd + ")"
		}
	case "Grep", "Glob":
		if b.Input.Pattern != "" {
			return "(" + name + " " + b.Input.Pattern + ")"
		}
	}
	return "(" + name + ")"
}

// trimTurnsHead drops oldest turns until total bytes fit under
// max. Returns the trimmed slice and updates bytes by reference.
func trimTurnsHead(turns []Turn, bytes *int, max int) []Turn {
	for len(turns) > 0 && *bytes > max {
		head := turns[0]
		*bytes -= len(head.Text) + len(head.Role) + 4
		turns = turns[1:]
	}
	return turns
}

// ── Codex ─────────────────────────────────────────────────────

// codexTranscript reads the latest Codex rollout JSONL for cwd.
// Format:
//
//	{"timestamp":"...","payload":{"type":"message","role":"user|assistant","content":[{"text":"..."}]}}
//
// Tool calls live in payload.type=function_call and are summarised
// inline like Claude tools.
// codexTranscript matches by session_meta.cwd already; the M22
// time-window filter is added on top so accumulated rollouts can't
// leak across sessions in the same cwd.
func codexTranscript(cfg CodexHistoryConfig, cwd string, startedAt, endedAt time.Time, maxBytes int) []Turn {
	path := resolveLatestCodexJSONL(cfg, cwd)
	if path == "" {
		return nil
	}
	var windowStart, windowEnd time.Time
	if !startedAt.IsZero() {
		windowStart = startedAt.Add(-30 * time.Second)
	}
	if !endedAt.IsZero() {
		windowEnd = endedAt.Add(30 * time.Second)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)
	var turns []Turn
	bytesUsed := 0
	for scanner.Scan() {
		var entry struct {
			Timestamp time.Time `json:"timestamp"`
			Payload   struct {
				Type    string `json:"type"`
				Role    string `json:"role"`
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if !entry.Timestamp.IsZero() {
			if !windowStart.IsZero() && entry.Timestamp.Before(windowStart) {
				continue
			}
			if !windowEnd.IsZero() && entry.Timestamp.After(windowEnd) {
				continue
			}
		}
		var role, text string
		switch entry.Payload.Type {
		case "message":
			role = strings.ToLower(entry.Payload.Role)
			var parts []string
			for _, c := range entry.Payload.Content {
				if t := strings.TrimSpace(c.Text); t != "" {
					parts = append(parts, t)
				}
			}
			text = strings.Join(parts, " ")
		case "function_call":
			role = "assistant"
			text = "(" + entry.Payload.Name + ")"
		default:
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" || (role != "user" && role != "assistant") {
			continue
		}
		bytesUsed += len(text) + len(role) + 4
		if bytesUsed > maxBytes && len(turns) > 0 {
			turns = trimTurnsHead(turns, &bytesUsed, maxBytes)
		}
		turns = append(turns, Turn{Role: role, Text: text, Ts: entry.Timestamp})
	}
	return turns
}

// resolveLatestCodexJSONL walks the configured sessions root and
// returns the newest rollout file whose recorded cwd matches.
// Re-uses codexRolloutMatchesCwd already defined in codex_jsonl.go
// for the cwd predicate.
func resolveLatestCodexJSONL(cfg CodexHistoryConfig, cwd string) string {
	root := cfg.SessionsRoot
	if root == "" {
		if home := os.Getenv("HOME"); home != "" {
			root = filepath.Join(home, ".codex", "sessions")
		}
	}
	if root == "" {
		return ""
	}
	type cand struct {
		path string
		mt   time.Time
	}
	var best cand
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}
		if !codexRolloutMatchesCwd(p, cwd) {
			return nil
		}
		if best.path == "" || info.ModTime().After(best.mt) {
			best = cand{p, info.ModTime()}
		}
		return nil
	})
	return best.path
}

// ── Gemini ────────────────────────────────────────────────────

// geminiTranscript reads Gemini CLI's chats.json (NOT JSONL — it's
// a single JSON document with a "messages" array). Picks the
// session that ran in cwd. M22 time-window filter applies to
// individual messages even after the right session is selected,
// to defend against chats.json accumulating across spawns.
func geminiTranscript(cfg GeminiHistoryConfig, cwd string, startedAt, endedAt time.Time, maxBytes int) []Turn {
	path := resolveGeminiChatsFile(cfg)
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc struct {
		Sessions []struct {
			Cwd      string `json:"cwd"`
			Messages []struct {
				Role    string    `json:"role"`
				Content string    `json:"content"`
				Time    time.Time `json:"timestamp"`
			} `json:"messages"`
			Updated time.Time `json:"updated"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	// Pick the most recent session matching cwd. Gemini uses
	// "user" / "model" roles — normalise "model" → "assistant".
	var best *struct {
		Cwd      string `json:"cwd"`
		Messages []struct {
			Role    string    `json:"role"`
			Content string    `json:"content"`
			Time    time.Time `json:"timestamp"`
		} `json:"messages"`
		Updated time.Time `json:"updated"`
	}
	for i := range doc.Sessions {
		s := &doc.Sessions[i]
		if s.Cwd != cwd {
			continue
		}
		if best == nil || s.Updated.After(best.Updated) {
			best = s
		}
	}
	if best == nil {
		return nil
	}
	// Walk messages, normalise role + cap bytes.
	sort.SliceStable(best.Messages, func(i, j int) bool {
		return best.Messages[i].Time.Before(best.Messages[j].Time)
	})
	var windowStart, windowEnd time.Time
	if !startedAt.IsZero() {
		windowStart = startedAt.Add(-30 * time.Second)
	}
	if !endedAt.IsZero() {
		windowEnd = endedAt.Add(30 * time.Second)
	}
	var turns []Turn
	bytesUsed := 0
	for _, m := range best.Messages {
		role := strings.ToLower(m.Role)
		if role == "model" {
			role = "assistant"
		}
		if role != "user" && role != "assistant" {
			continue
		}
		text := strings.TrimSpace(m.Content)
		if text == "" {
			continue
		}
		if !m.Time.IsZero() {
			if !windowStart.IsZero() && m.Time.Before(windowStart) {
				continue
			}
			if !windowEnd.IsZero() && m.Time.After(windowEnd) {
				continue
			}
		}
		bytesUsed += len(text) + len(role) + 4
		if bytesUsed > maxBytes && len(turns) > 0 {
			turns = trimTurnsHead(turns, &bytesUsed, maxBytes)
		}
		turns = append(turns, Turn{Role: role, Text: text, Ts: m.Time})
	}
	return turns
}

// resolveGeminiChatsFile picks the chats.json path from cfg or
// from ~/.gemini/chats.json (the Gemini CLI default).
func resolveGeminiChatsFile(cfg GeminiHistoryConfig) string {
	if cfg.ProjectsFile != "" {
		return cfg.ProjectsFile
	}
	home := os.Getenv("HOME")
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".gemini", "chats.json")
}
