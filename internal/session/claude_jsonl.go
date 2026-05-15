package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// claude_jsonl.go: read the Claude Code CLI's per-session structured
// transcript so the idle notification can carry the *real* assistant
// response — not a snapshot of whatever 40 rows of TUI happened to be
// on screen when we sampled.
//
// Claude writes one JSON object per line to:
//
//   <CLAUDE_CONFIG_DIR>/projects/<encoded-cwd>/<session-id>.jsonl
//
// The encoding of `<encoded-cwd>` is non-trivial (Claude does its own
// underscore↔dash normalisation), so we resolve it by scanning the
// projects/ root for a directory whose name contains every component
// of the session's cwd in order.
//
// Multi-account: opendray points each session at one of:
//   - ~/.claude/projects/                           (standalone Claude Code)
//   - ~/.claude-accounts/<account>/projects/        (opendray multi-account;
//                                                    typically a symlink to
//                                                    ~/.claude-accounts/shared/projects/)
//
// We scan all of them and dedupe by canonical (symlink-resolved) path
// so each transcript file contributes at most once.

// claudeJSONLEntry is one line in Claude's transcript file. We decode
// only the fields the snippet needs.
type claudeJSONLEntry struct {
	Type    string          `json:"type"` // user | assistant | progress | system
	Message *claudeJSONLMsg `json:"message,omitempty"`
	UUID    string          `json:"uuid,omitempty"`
	Time    time.Time       `json:"timestamp,omitempty"`

	// M22 — Claude Code stamps each user-typed turn with the cwd it
	// was invoked under. The transcript reader uses this as a
	// canary: if the first cwd we encounter in a file doesn't match
	// the calling session's cwd, the whole file is rejected as
	// "wrong project". Defensive against Claude Code reusing or
	// mis-routing jsonl files across projects.
	Cwd string `json:"cwd,omitempty"`
}

type claudeJSONLMsg struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// claudeContentBlock is one element of a JSONL message's content
// array. Claude returns: text / tool_use / tool_result / thinking.
//
// For tool_use, Input carries the argument map. We only decode the
// fields we need to render a one-line summary; everything else stays
// in `Raw` for callers that want to introspect further.
type claudeContentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	Name      string `json:"name,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`

	// Input fields for assistant.tool_use blocks.
	Input *struct {
		Command     string `json:"command,omitempty"`
		Description string `json:"description,omitempty"`
		Content     string `json:"content,omitempty"`
		FilePath    string `json:"file_path,omitempty"`
		Path        string `json:"path,omitempty"`
		Pattern     string `json:"pattern,omitempty"`
		Query       string `json:"query,omitempty"`
		Question    string `json:"question,omitempty"`
		URL         string `json:"url,omitempty"`
		Prompt      string `json:"prompt,omitempty"`
	} `json:"input,omitempty"`

	// Tool_result Content can be a string OR an array of content
	// blocks (text/image). We capture both shapes via RawContent and
	// expose a string view via ToolResultText().
	RawContent json.RawMessage `json:"content,omitempty"`
	IsError    bool            `json:"is_error,omitempty"`
}

// ProjectInput is one human prompt found in a Claude JSONL
// transcript. Used by the History panel to render every prompt the
// operator has asked in this project (cwd) across all sessions.
type ProjectInput struct {
	Ts        time.Time `json:"ts"`
	Text      string    `json:"text"`
	SessionID string    `json:"session_id"` // Claude's own session id (filename)
}

// HistoryResponse is the result of Manager.History — user prompts
// in this project + an "unsupported" flag for non-Claude providers
// so the UI can render the right empty state without a separate
// provider lookup.
type HistoryResponse struct {
	Entries             []ProjectInput `json:"entries"`
	UnsupportedProvider bool           `json:"unsupported_provider,omitempty"`
}

// ClaudeHistoryConfig drives ProjectInputHistory's path resolution.
// All fields optional — empty values fall back to ~/.claude/projects
// plus every ~/.claude-accounts/*/projects (the same defaults the
// hardcoded path used to expose).
type ClaudeHistoryConfig struct {
	// HistoryRoots is the explicit list of `<dir>/projects` roots to
	// scan. When non-empty, it REPLACES the built-in defaults — the
	// operator opted out of HOME-relative discovery entirely.
	HistoryRoots []string
	// AccountsDir overrides ~/.claude-accounts when looking for
	// per-account project subtrees. Ignored when HistoryRoots is set.
	AccountsDir string
}

// ProjectInputHistory returns up to `limit` user prompts from every
// JSONL file in any Claude project directory matching `cwd`,
// across the configured roots, chronologically ordered (newest
// first). Empty result when nothing matches.
//
// Errors are swallowed deliberately on the per-file path so one
// malformed transcript doesn't break the whole list.
func ProjectInputHistory(cfg ClaudeHistoryConfig, cwd string, limit int) []ProjectInput {
	if limit <= 0 {
		limit = 200
	}
	roots := resolveClaudeRoots(cfg)
	if len(roots) == 0 {
		return nil
	}
	var out []ProjectInput
	seenRoots := map[string]bool{}
	seenFiles := map[string]bool{}
	for _, root := range roots {
		canon, err := filepath.EvalSymlinks(root)
		if err != nil {
			canon = root // root may not exist; findClaudeProjectDir will return ""
		}
		if seenRoots[canon] {
			continue
		}
		seenRoots[canon] = true
		dir := findClaudeProjectDir(canon, cwd)
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			realPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				realPath = path
			}
			if seenFiles[realPath] {
				continue
			}
			seenFiles[realPath] = true
			sid := strings.TrimSuffix(e.Name(), ".jsonl")
			out = append(out, extractUserInputs(path, sid)...)
		}
	}
	// Newest first.
	sort.Slice(out, func(i, j int) bool { return out[i].Ts.After(out[j].Ts) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// resolveClaudeRoots picks the list of projects-root directories
// to scan. Precedence:
//
//  1. cfg.HistoryRoots — explicit operator override; used as-is.
//  2. ~/.claude/projects + every ~/.claude-accounts/*/projects
//     subtree, where ~/.claude-accounts is overridable via
//     cfg.AccountsDir.
//
// Returns nil only when HOME isn't set AND no explicit roots are
// configured (test environments stub HOME via t.Setenv).
func resolveClaudeRoots(cfg ClaudeHistoryConfig) []string {
	if len(cfg.HistoryRoots) > 0 {
		return cfg.HistoryRoots
	}
	home := os.Getenv("HOME")
	if home == "" {
		return nil
	}
	roots := []string{filepath.Join(home, ".claude", "projects")}
	accountsDir := cfg.AccountsDir
	if accountsDir == "" {
		accountsDir = filepath.Join(home, ".claude-accounts")
	}
	entries, err := os.ReadDir(accountsDir)
	if err != nil {
		return roots
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// `tokens/` holds OAuth files, not project transcripts.
		if e.Name() == "tokens" {
			continue
		}
		roots = append(roots, filepath.Join(accountsDir, e.Name(), "projects"))
	}
	return roots
}

// extractUserInputs reads one JSONL file and returns the user
// prompts inside. Skips entries whose `user.message.content` is
// only tool_result blocks (those are agent feedback, not user
// intent).
func extractUserInputs(path, sessionID string) []ProjectInput {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var out []ProjectInput
	for scanner.Scan() {
		var e claudeJSONLEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if e.Type != "user" || e.Message == nil {
			continue
		}
		text := extractUserText(e.Message.Content)
		if text == "" {
			continue
		}
		out = append(out, ProjectInput{
			Ts:        e.Time,
			Text:      text,
			SessionID: sessionID,
		})
	}
	return out
}

// extractUserText returns the human prompt text from a user
// entry's content. Handles both shapes:
//   - bare JSON string  → treat as the prompt
//   - array of blocks   → concatenate `text` blocks; skip
//     tool_result and any other block types.
//
// Returns "" when the entry carries no text (i.e. tool_result-only
// entries that Claude writes as "feedback for itself").
func extractUserText(raw json.RawMessage) string {
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
	var blocks []claudeContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// ToolResultText returns the human-readable text from a tool_result
// block. Tool results may serialise as a plain string or as an array
// of {type:"text", text:"..."} blocks.
func (b *claudeContentBlock) ToolResultText() string {
	if b == nil || len(b.RawContent) == 0 {
		return ""
	}
	if b.RawContent[0] == '"' {
		var s string
		if err := json.Unmarshal(b.RawContent, &s); err != nil {
			return ""
		}
		return s
	}
	var inner []claudeContentBlock
	if err := json.Unmarshal(b.RawContent, &inner); err != nil {
		return ""
	}
	parts := make([]string, 0, len(inner))
	for _, ib := range inner {
		if ib.Type == "text" && ib.Text != "" {
			parts = append(parts, ib.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// claudeRecentResponse returns the most recent Claude conversation
// turn — assistant text + tool_use blocks + their tool_result
// follow-ups — formatted to mimic Claude's TUI rendering. Empty
// string when no usable transcript exists for `cwd`.
//
// Errors are swallowed deliberately: this is a best-effort
// enrichment for chat notifications, not a critical path.
func claudeRecentResponse(cwd string) string {
	jsonlPath := resolveLatestClaudeJSONL(cwd)
	if jsonlPath == "" {
		return ""
	}
	// 5000 entries is enough headroom for marathon turns with many
	// tool calls — the previous 60-entry window silently dropped
	// the start of any long turn. The channel layer (and any
	// per-channel notify_snippet_max_chars cap) is the right place
	// to constrain user-facing output; the source emits everything.
	out, err := renderClaudeRecentTurn(jsonlPath, 5000)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// resolveLatestClaudeJSONL finds the most recently modified .jsonl
// file inside the projects/<encoded-cwd>/ directory matching cwd.
func resolveLatestClaudeJSONL(cwd string) string {
	home := os.Getenv("HOME")
	if home == "" {
		return ""
	}
	root := filepath.Join(home, ".claude", "projects")
	dir := findClaudeProjectDir(root, cwd)
	if dir == "" {
		return ""
	}
	return findLatestClaudeJSONL(dir)
}

// findClaudeProjectDir scans projectsRoot for a directory whose
// encoded name contains every path component of cwd, in order. Claude
// normalises underscores to dashes inside path parts, so we try the
// raw form then the normalised one.
func findClaudeProjectDir(projectsRoot, cwd string) string {
	entries, err := os.ReadDir(projectsRoot)
	if err != nil {
		return ""
	}
	parts := splitPathParts(cwd)
	if len(parts) == 0 {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if matchesClaudeProjectDir(e.Name(), parts) {
			return filepath.Join(projectsRoot, e.Name())
		}
	}
	return ""
}

// findLatestClaudeJSONL returns the .jsonl file in `dir` with the
// most recent mtime. Empty when none.
func findLatestClaudeJSONL(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var best string
	var bestTime time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(bestTime) {
			best = filepath.Join(dir, e.Name())
			bestTime = info.ModTime()
		}
	}
	return best
}

func splitPathParts(p string) []string {
	out := make([]string, 0, 8)
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func matchesClaudeProjectDir(encoded string, parts []string) bool {
	remaining := encoded
	for _, part := range parts {
		idx := strings.Index(remaining, part)
		if idx >= 0 {
			remaining = remaining[idx+len(part):]
			continue
		}
		// Try with underscores swapped to dashes (Claude's normalisation).
		normalised := strings.ReplaceAll(part, "_", "-")
		idx = strings.Index(remaining, normalised)
		if idx < 0 {
			return false
		}
		remaining = remaining[idx+len(normalised):]
	}
	return true
}

// renderClaudeRecentTurn reads the last `n` JSONL entries and
// renders the most recent conversation turn as Claude would display
// it on screen: assistant text + tool calls in order, with tool
// results attached underneath.
//
// Algorithm:
//  1. Read last n entries.
//  2. Walk backwards to find the first assistant entry. Anchor.
//  3. From that anchor, walk forward and render every entry until
//     the next assistant turn boundary (or end of file).
//
// Result mirrors what the user sees on the TUI: a single "answer"
// turn complete with tool calls and their feedback, no chrome.
func renderClaudeRecentTurn(path string, n int) (string, error) {
	entries, err := readLastClaudeEntries(path, n)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("empty transcript")
	}

	// Find the most recent "turn boundary": the entry RIGHT AFTER
	// the last *real* user message (not a tool_result). Everything
	// from there to end-of-file is one turn — the assistant's
	// answer plus any tool calls and tool results threaded in.
	//
	// If there is no user-message boundary in the window (e.g. we
	// only loaded partway through a long turn), fall back to the
	// first assistant entry we see.
	anchor := -1
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Type == "user" && hasUserText(e) {
			anchor = i + 1
			break
		}
	}
	if anchor < 0 || anchor >= len(entries) {
		// No user-message boundary in the window. Pick the earliest
		// assistant entry that has renderable content.
		for i := 0; i < len(entries); i++ {
			e := entries[i]
			if e.Type != "assistant" || e.Message == nil {
				continue
			}
			blocks, err := parseClaudeContentBlocks(e.Message.Content)
			if err != nil || !hasRenderableBlock(blocks) {
				continue
			}
			anchor = i
			break
		}
	}
	if anchor < 0 || anchor >= len(entries) {
		return "", fmt.Errorf("no assistant turn in last %d entries", n)
	}

	// Index tool_use IDs → their human-readable summary so we can
	// attach a tool_result that arrives later in the conversation.
	toolUseSummary := map[string]string{}

	var out strings.Builder
	for i := anchor; i < len(entries); i++ {
		e := entries[i]
		switch e.Type {
		case "assistant":
			if e.Message == nil {
				continue
			}
			blocks, err := parseClaudeContentBlocks(e.Message.Content)
			if err != nil {
				continue
			}
			renderAssistantBlocks(&out, blocks, toolUseSummary)
		case "user":
			if e.Message == nil {
				continue
			}
			blocks, err := parseClaudeContentBlocks(e.Message.Content)
			if err != nil {
				continue
			}
			renderToolResults(&out, blocks, toolUseSummary)
		}
	}

	if strings.TrimSpace(out.String()) == "" {
		return "", fmt.Errorf("turn rendered empty")
	}
	return out.String(), nil
}

// hasUserText reports whether the entry is a user message with
// genuine human text — not just a tool_result wrapper. Used to find
// the boundary between conversation turns.
func hasUserText(e claudeJSONLEntry) bool {
	if e.Message == nil {
		return false
	}
	// Bare-string content is the legacy "user typed something" shape.
	if len(e.Message.Content) > 0 && e.Message.Content[0] == '"' {
		var s string
		if err := json.Unmarshal(e.Message.Content, &s); err == nil {
			return strings.TrimSpace(s) != ""
		}
	}
	blocks, err := parseClaudeContentBlocks(e.Message.Content)
	if err != nil {
		return false
	}
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if strings.TrimSpace(b.Text) != "" {
				return true
			}
		case "tool_result":
			// Skip — tool results are agent feedback, not user input.
		}
	}
	return false
}

// hasRenderableBlock returns true when the entry has at least one
// text or tool_use block with non-empty content (i.e., not just
// thinking-only).
func hasRenderableBlock(blocks []claudeContentBlock) bool {
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if strings.TrimSpace(b.Text) != "" {
				return true
			}
		case "tool_use":
			if b.Name != "" {
				return true
			}
		}
	}
	return false
}

// readLastClaudeEntries reads the whole file (Claude JSONL files
// rarely exceed a few hundred KB per session) and returns the tail.
// 1 MB scanner buffer accommodates the occasional very long line.
func readLastClaudeEntries(path string, n int) ([]claudeJSONLEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	all := make([]claudeJSONLEntry, 0, 64)
	for scanner.Scan() {
		var e claudeJSONLEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		all = append(all, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	if len(all) > n {
		all = all[len(all)-n:]
	}
	return all, nil
}

func parseClaudeContentBlocks(raw json.RawMessage) ([]claudeContentBlock, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	// User messages encode content as a single string; assistant as an array.
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, err
		}
		return []claudeContentBlock{{Type: "text", Text: s}}, nil
	}
	var blocks []claudeContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// renderAssistantBlocks emits text + tool_use blocks in Claude's TUI
// style. Each text block gets a "●" bullet; each tool_use becomes
// "● ToolName(<key arg>)" so the output reads as a chronological
// flow of "what the assistant did". Thinking blocks are dropped.
//
// toolUseSummary is updated with the rendered title so a later
// tool_result entry (for that ID) can be attached underneath.
func renderAssistantBlocks(out *strings.Builder, blocks []claudeContentBlock, toolUseSummary map[string]string) {
	for _, b := range blocks {
		switch b.Type {
		case "text":
			t := strings.TrimSpace(b.Text)
			if t == "" {
				continue
			}
			out.WriteString("● ")
			out.WriteString(t)
			out.WriteString("\n\n")
		case "tool_use":
			title := formatToolUse(b)
			fmt.Fprintf(out, "● %s\n", title)
			if b.ToolUseID != "" {
				toolUseSummary[b.ToolUseID] = title
			}
		case "thinking":
			// Drop — internal reasoning, not part of the user-visible turn.
		}
	}
}

// renderToolResults emits the textual content of any tool_result
// blocks in this user entry. The first line of each result gets a
// "  └ " prefix; subsequent lines align with 4-space indent so
// multi-line outputs (Read, Bash, web fetches) remain readable
// inside the bullet-list flow.
//
// We deliberately emit the FULL result text — no 3-line / 200-rune
// preview cap. The channel layer (notify_snippet_max_chars or
// Telegram's per-message splitter) is the right place to constrain
// transport-level limits. Capping here silently lost content even
// when the operator explicitly opted into "Unlimited — split into
// multiple messages".
func renderToolResults(out *strings.Builder, blocks []claudeContentBlock, toolUseSummary map[string]string) {
	for _, b := range blocks {
		if b.Type != "tool_result" {
			continue
		}
		text := strings.TrimSpace(b.ToolResultText())
		if text == "" {
			continue
		}
		body := renderToolResultBody(text)
		if body == "" {
			continue
		}
		out.WriteString(body)
		out.WriteString("\n\n")
		// Track which tool calls have been answered.
		delete(toolUseSummary, b.ToolUseID)
	}
}

// formatToolUse turns a tool_use block into the one-line summary
// that goes after the bullet. Example outputs:
//
//	Write(docs/ANDROID_PORT_SPEC.md)
//	Bash(rg "TODO" --type go) — search for outstanding items
//	Read(internal/session/manager.go)
//	Edit(README.md)
//	AskUserQuestion: choose a model
//	WebFetch(https://example.com/api)
//
// Per-arg truncation here would silently lose meaningful content
// from the chat snippet (long shell commands, web URLs, multi-line
// question prompts). The channel layer is responsible for
// platform-level chunking — see internal/channel/telegram. We keep
// a generous safety cap (toolUseArgSoftCap) only to defang
// pathological inputs like a multi-megabyte Bash blob; in normal
// use this never fires.
func formatToolUse(b claudeContentBlock) string {
	name := b.Name
	if name == "" {
		name = "Tool"
	}
	in := b.Input
	if in == nil {
		return name
	}
	switch {
	case in.FilePath != "":
		return fmt.Sprintf("%s(%s)", name, in.FilePath)
	case in.Path != "":
		return fmt.Sprintf("%s(%s)", name, in.Path)
	case in.Command != "":
		cmd := softTruncate(in.Command)
		out := fmt.Sprintf("%s(%s)", name, cmd)
		if in.Description != "" {
			out += " — " + softTruncate(in.Description)
		}
		return out
	case in.Pattern != "":
		return fmt.Sprintf("%s(%s)", name, softTruncate(in.Pattern))
	case in.Query != "":
		return fmt.Sprintf("%s(%s)", name, softTruncate(in.Query))
	case in.URL != "":
		return fmt.Sprintf("%s(%s)", name, softTruncate(in.URL))
	case in.Question != "":
		return fmt.Sprintf("%s: %s", name, softTruncate(in.Question))
	case in.Prompt != "":
		return fmt.Sprintf("%s: %s", name, softTruncate(in.Prompt))
	case in.Description != "":
		return fmt.Sprintf("%s — %s", name, softTruncate(in.Description))
	}
	return name
}

// toolUseArgSoftCap is the only place we still cap a tool_use arg.
// Larger than any reasonable command line / URL / pattern / prompt;
// trips only on adversarial input. The trailing "…" makes the
// truncation obvious if it ever happens.
const toolUseArgSoftCap = 4000

func softTruncate(s string) string {
	return truncateRunes(s, toolUseArgSoftCap)
}

// renderToolResultBody formats a tool_result for embedding under
// its parent tool_use bullet. First non-empty line gets "  └ ";
// subsequent lines align with a 4-space continuation indent so the
// block reads as one logical unit:
//
//	● Read(file.go)
//	  └ first matching line
//	    second matching line
//	    ...
//
// No content is dropped. CR characters are normalised; trailing
// blank lines trimmed; that's it.
func renderToolResultBody(text string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r", ""), "\n")
	// Trim trailing whitespace-only lines so the "\n\n" the caller
	// appends doesn't snowball into 4+ blank lines.
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	for i, l := range lines {
		if i == 0 {
			b.WriteString("  └ ")
		} else {
			b.WriteString("\n    ")
		}
		b.WriteString(l)
	}
	return b.String()
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
