package session

import (
	"bufio"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// grok_jsonl.go: read the grok CLI's per-session transcript so an
// account switch can carry a recap of the prior conversation into the
// fresh session under the new GROK_HOME (RFC #541, recap-inject option).
//
// grok writes each session under its GROK_HOME:
//
//   <GROK_HOME>/sessions/<percent-encoded-cwd>/<session-uuid>/chat_history.jsonl
//
// The encoded cwd percent-escapes path separators (/ -> %2F). Each line
// of chat_history.jsonl is one JSON object: {type, content, ...}.
//   type "user"       content = [{type:"text", text:"..."}]  (may be a bare string)
//   type "assistant"  content = "prose reply"                (plus tool_calls we ignore)
//   type "reasoning"  content = null                         (dropped: internal thinking)
//   type "tool_result"content = "..."                        (dropped: tool noise)
//   type "system"     content = "..."                        (dropped: system prompt)
//
// Genuine human prompts are wrapped in <user_query>...</user_query>;
// every other user turn is synthetic injected context (<user_info>,
// <rules>, <system-reminder>, continuation summaries) and is dropped.

// GrokHistoryConfig drives grok transcript path resolution. All fields
// optional — empty values fall back to <GROK_HOME or ~/.grok>/sessions
// plus every ~/.grok-accounts/*/sessions subtree.
type GrokHistoryConfig struct {
	// SessionsRoots is the explicit list of `<dir>/sessions` roots to
	// scan. When non-empty it REPLACES the built-in defaults.
	SessionsRoots []string
	// AccountsDir overrides ~/.grok-accounts when looking for per-account
	// session subtrees. Ignored when SessionsRoots is set.
	AccountsDir string
}

// grokChatEntry is one line of grok's chat_history.jsonl. We decode only
// the two fields the recap needs.
type grokChatEntry struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

type grokTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// BuildGrokCarryover reads the transcript for `sessionID` under the grok
// session dir matching `cwd`, and returns a labeled recap block suitable
// for injection (via grok --rules) into a fresh session after an account
// switch. Returns "" (never an error) when there's nothing to carry — a
// missing/empty/unparseable transcript degrades to a fresh session and
// never blocks the switch.
//
// Only genuine <user_query> prompts and assistant prose are kept; the
// recap is tail-truncated to budgetBytes with an elision marker when
// older turns are dropped. Reuses renderCarryover shared with Claude.
func BuildGrokCarryover(cfg GrokHistoryConfig, cwd, sessionID string, budgetBytes int) string {
	if sessionID == "" || cwd == "" {
		return ""
	}
	if budgetBytes <= 0 {
		budgetBytes = defaultCarryoverBudgetBytes
	}

	path := findGrokTranscriptForSession(cfg, cwd, sessionID)
	if path == "" {
		return ""
	}

	turns := readGrokCarryoverTurns(path)
	if len(turns) == 0 {
		return ""
	}
	return renderCarryover(turns, budgetBytes)
}

// LatestGrokSessionID returns the session UUID whose chat_history.jsonl
// under the grok session dir matching `cwd` was modified most recently,
// across every configured root. It resolves Q1 (RFC #541): opendray does
// not track a grok session id on the session row, so at switch time —
// right after the old grok process is stopped — the just-used session's
// transcript is the freshest one for that cwd. Returns "" when nothing
// matches; a dir without a transcript yet is never selected.
func LatestGrokSessionID(cfg GrokHistoryConfig, cwd string) string {
	if cwd == "" {
		return ""
	}
	var (
		best     string
		bestTime time.Time
	)
	for _, root := range cfg.resolveGrokSessionsRoots() {
		cwdDir := findGrokCwdDir(root, cwd)
		if cwdDir == "" {
			continue
		}
		entries, err := os.ReadDir(cwdDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || !safePathElement(e.Name()) {
				continue
			}
			info, err := os.Stat(filepath.Join(cwdDir, e.Name(), "chat_history.jsonl"))
			if err != nil || info.IsDir() {
				continue
			}
			if best == "" || info.ModTime().After(bestTime) {
				best = e.Name()
				bestTime = info.ModTime()
			}
		}
	}
	return best
}

// findGrokTranscriptForSession locates
// <root>/<encoded-cwd>/<sessionID>/chat_history.jsonl across every
// configured root. Fail-closed on sessionID: it must be a single, clean
// path element, so a crafted value can't traverse out of the sessions
// root (R2). The resolved file is also verified to live under its root.
func findGrokTranscriptForSession(cfg GrokHistoryConfig, cwd, sessionID string) string {
	if !safePathElement(sessionID) {
		return ""
	}
	for _, root := range cfg.resolveGrokSessionsRoots() {
		cwdDir := findGrokCwdDir(root, cwd)
		if cwdDir == "" {
			continue
		}
		candidate := filepath.Join(cwdDir, sessionID, "chat_history.jsonl")
		if !withinRoot(root, candidate) {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

// findGrokCwdDir returns the child of `root` whose percent-decoded name
// equals `cwd`. Decoding (rather than re-encoding cwd) is robust to
// whatever exact escaping grok used to name the directory.
func findGrokCwdDir(root, cwd string) string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		decoded, err := url.QueryUnescape(e.Name())
		if err != nil {
			continue
		}
		if decoded == cwd {
			return filepath.Join(root, e.Name())
		}
	}
	return ""
}

// readGrokCarryoverTurns parses a chat_history.jsonl into ordered text
// turns, keeping only genuine <user_query> prompts and assistant prose.
// Oldest-first.
func readGrokCarryoverTurns(path string) []carryoverTurn {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var turns []carryoverTurn
	for scanner.Scan() {
		var e grokChatEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		switch e.Type {
		case "user":
			if q := grokUserQuery(e.Content); q != "" {
				turns = append(turns, carryoverTurn{role: "You", text: q})
			}
		case "assistant":
			if txt := grokAssistantText(e.Content); txt != "" {
				turns = append(turns, carryoverTurn{role: "Assistant", text: txt})
			}
		}
	}
	return turns
}

// grokUserQuery extracts the genuine prompt from a user turn: the text
// inside <user_query>...</user_query>. Returns "" for synthetic turns
// (<user_info>, <rules>, <system-reminder>, continuation summaries),
// which carry no such wrapper.
func grokUserQuery(raw json.RawMessage) string {
	return extractUserQuery(grokBlockText(raw))
}

// grokAssistantText returns the assistant's prose. content is normally a
// bare JSON string; an array-of-blocks shape is handled defensively.
func grokAssistantText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return ""
		}
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(grokBlockText(raw))
}

// grokBlockText concatenates the text of an array-of-blocks content, and
// also accepts a bare string. Non-text blocks are ignored.
func grokBlockText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return ""
		}
		return s
	}
	var blocks []grokTextBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// extractUserQuery returns the trimmed text inside the first
// <user_query>...</user_query> pair, or "" when absent/malformed.
func extractUserQuery(s string) string {
	const open, close = "<user_query>", "</user_query>"
	i := strings.Index(s, open)
	if i < 0 {
		return ""
	}
	rest := s[i+len(open):]
	j := strings.Index(rest, close)
	if j < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:j])
}

// resolveGrokSessionsRoots picks the `<dir>/sessions` roots to scan.
// Precedence: explicit SessionsRoots override, else <GROK_HOME or
// ~/.grok>/sessions plus every ~/.grok-accounts/*/sessions subtree
// (~/.grok-accounts overridable via AccountsDir).
func (cfg GrokHistoryConfig) resolveGrokSessionsRoots() []string {
	if len(cfg.SessionsRoots) > 0 {
		return cfg.SessionsRoots
	}
	home := os.Getenv("HOME")
	grokHome := os.Getenv("GROK_HOME")
	if grokHome == "" {
		if home == "" {
			return nil
		}
		grokHome = filepath.Join(home, ".grok")
	}
	roots := []string{filepath.Join(grokHome, "sessions")}

	accountsDir := cfg.AccountsDir
	if accountsDir == "" {
		if home == "" {
			return roots
		}
		accountsDir = filepath.Join(home, ".grok-accounts")
	}
	entries, err := os.ReadDir(accountsDir)
	if err != nil {
		return roots
	}
	for _, e := range entries {
		if e.IsDir() {
			roots = append(roots, filepath.Join(accountsDir, e.Name(), "sessions"))
		}
	}
	return roots
}

// safePathElement reports whether s is a single, clean path element with
// no separators or parent references — safe to Join without escaping.
func safePathElement(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	if strings.ContainsRune(s, '/') || strings.ContainsRune(s, filepath.Separator) {
		return false
	}
	if strings.Contains(s, "..") {
		return false
	}
	return filepath.Clean(s) == s
}

// withinRoot reports whether `path` resolves inside `root` (both cleaned).
// Defense-in-depth alongside safePathElement.
func withinRoot(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
