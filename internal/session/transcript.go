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
	switch sess.ProviderID {
	case "claude":
		return claudeTranscript(m.claudeHistoryCfg, sess.Cwd, sess.ClaudeSessionID, maxBytes), nil
	case "codex":
		return codexTranscript(m.codexHistoryCfg, sess.Cwd, maxBytes), nil
	case "gemini":
		return geminiTranscript(m.geminiHistoryCfg, sess.Cwd, maxBytes), nil
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

// claudeTranscript walks the latest matching JSONL file and
// returns user + assistant text turns in chronological order.
// Tool-use / tool-result / thinking blocks are dropped — the
// summariser cares about the conversation, not the raw tool call
// payloads.
func claudeTranscript(cfg ClaudeHistoryConfig, cwd, claudeSessID string, maxBytes int) []Turn {
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
		raw := scanner.Bytes()
		var e claudeJSONLEntry
		if err := json.Unmarshal(raw, &e); err != nil {
			continue
		}
		if e.Message == nil {
			continue
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
func codexTranscript(cfg CodexHistoryConfig, cwd string, maxBytes int) []Turn {
	path := resolveLatestCodexJSONL(cfg, cwd)
	if path == "" {
		return nil
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
// session that ran in cwd.
func geminiTranscript(cfg GeminiHistoryConfig, cwd string, maxBytes int) []Turn {
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
