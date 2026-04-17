// Package jsonl reads Claude Code's structured JSONL session files.
//
// Claude CLI writes every message to ~/.claude/projects/<encoded-cwd>/<session-id>.jsonl.
// Each line is a JSON object with type "user" | "assistant" | "progress" | "system" | etc.
// Assistant messages contain structured content blocks (text, tool_use, thinking).
//
// By reading this file directly we get clean, structured output without any
// ANSI stripping, VT100 emulation, or TUI chrome filtering. This is the
// same data source yepanywhere uses.
package jsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// claudeBasePath returns the base directory where Claude stores sessions.
// It reads HOME at call time rather than init time, and returns an error
// when HOME is unset so callers can surface a clear diagnostic.
func claudeBasePath() (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return "", fmt.Errorf("HOME environment variable is not set")
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

// Entry is a single JSONL line. We only decode the fields we need.
type Entry struct {
	Type       string    `json:"type"` // user | assistant | progress | system | file-history-snapshot
	UUID       string    `json:"uuid"`
	ParentUUID string    `json:"parentUuid"`
	SessionID  string    `json:"sessionId"`
	Timestamp  time.Time `json:"timestamp"`
	Message    *Message  `json:"message,omitempty"`
}

type Message struct {
	Role    string         `json:"role"` // user | assistant
	Content json.RawMessage `json:"content"`
	Model   string         `json:"model,omitempty"`
}

// ContentBlock is one element of an assistant message's content array.
type ContentBlock struct {
	Type  string `json:"type"` // text | tool_use | tool_result | thinking
	Text  string `json:"text,omitempty"`
	Name  string `json:"name,omitempty"` // for tool_use
	Input *struct {
		Command     string `json:"command,omitempty"`
		Description string `json:"description,omitempty"`
		Content     string `json:"content,omitempty"`
		FilePath    string `json:"file_path,omitempty"`
		Question    string `json:"question,omitempty"` // AskUserQuestion
	} `json:"input,omitempty"`
}

// ── Prompt detection types ────────────────────────────────────

// PromptKind classifies the type of interactive prompt detected.
type PromptKind int

const (
	PromptNone            PromptKind = iota
	PromptYesNo                      // [y/N], (y/n), Allow...?
	PromptYesNoAlways                // [y/N/a]
	PromptNumberedList               // 1. ... 2. ... 3. ...
	PromptLetteredList               // (a) ... (b) ... (c) ...
	PromptBulletList                 // - option1 / - option2 after a question
	PromptPlanApproval               // "Shall I proceed?", "Ready to implement?"
	PromptDefaultValue               // Enter name [default]:
	PromptEnterContinue              // "Press Enter to continue"
	PromptAskUser                    // AskUserQuestion tool_use (free text)
	PromptGenericQuestion            // trailing ? (open-ended, no Yes/No)
	PromptMultiSelect                // numbered list + explicit multi-select phrasing
)

// PromptOption represents a single selectable option.
type PromptOption struct {
	Key   string // "1", "2", "a", "b", or bullet text prefix
	Label string // human-readable label
}

// PromptInfo carries structured information about an interactive prompt.
type PromptInfo struct {
	Kind         PromptKind
	Question     string         // the question text (from AskUserQuestion or detected)
	Options      []PromptOption // extracted options; nil for free-text prompts
	DefaultValue string         // for PromptDefaultValue: the bracketed default
}

// ProjectDir finds Claude's project directory for a given CWD by scanning
// ~/.claude/projects/ for a directory whose name matches all path components
// (with underscore ↔ dash normalization, since Claude's encoding is not
// a simple replacement). Returns "" if no match found.
//
// For multi-account setups (sessions launched with CLAUDE_CONFIG_DIR pointing
// elsewhere), prefer ResolveLatestJSONL which accepts an explicit list of
// Claude base directories.
func ProjectDir(cwd string) string {
	base, err := claudeBasePath()
	if err != nil {
		return ""
	}
	return findProjectDir(base, cwd)
}

// findProjectDir scans a "projects/" root directory for an entry whose
// encoded name matches cwd's path components.
func findProjectDir(projectsRoot, cwd string) string {
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
		if matchesProjectDir(e.Name(), parts) {
			return filepath.Join(projectsRoot, e.Name())
		}
	}
	return ""
}

// ResolveLatestJSONL searches every supplied Claude config directory for
// a project dir matching cwd, then picks the single JSONL file with the
// most recent mtime across all matches. Each basePath should be a Claude
// config root (i.e. the dir that contains "projects/"), such as
// ~/.claude or ~/.claude-accounts/<name>.
//
// An empty basePaths list falls back to $HOME/.claude so single-account
// users don't have to configure anything. Returns "" if nothing matches.
func ResolveLatestJSONL(basePaths []string, cwd string) string {
	if len(basePaths) == 0 {
		if home := os.Getenv("HOME"); home != "" {
			basePaths = []string{filepath.Join(home, ".claude")}
		}
	}
	var best string
	var bestMtime time.Time
	for _, base := range basePaths {
		if base == "" {
			continue
		}
		projects := filepath.Join(base, "projects")
		dir := findProjectDir(projects, cwd)
		if dir == "" {
			continue
		}
		jp := FindLatestJSONL(dir)
		if jp == "" {
			continue
		}
		info, err := os.Stat(jp)
		if err != nil {
			continue
		}
		if info.ModTime().After(bestMtime) {
			best = jp
			bestMtime = info.ModTime()
		}
	}
	return best
}

func splitPathParts(p string) []string {
	var out []string
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// matchesProjectDir checks if all path parts appear in the encoded dir
// name, in order, allowing underscore ↔ dash substitution.
func matchesProjectDir(encoded string, parts []string) bool {
	remaining := encoded
	for _, part := range parts {
		// Try exact match first, then with underscores replaced by dashes
		idx := strings.Index(remaining, part)
		if idx < 0 {
			// Try normalized variant
			normalized := strings.ReplaceAll(part, "_", "-")
			idx = strings.Index(remaining, normalized)
			if idx < 0 {
				return false
			}
			remaining = remaining[idx+len(normalized):]
		} else {
			remaining = remaining[idx+len(part):]
		}
	}
	return true
}

// FindLatestJSONL finds the most recently modified .jsonl file in a
// Claude project directory. Returns "" if none found.
func FindLatestJSONL(projectDir string) string {
	entries, err := os.ReadDir(projectDir)
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
			best = filepath.Join(projectDir, e.Name())
			bestTime = info.ModTime()
		}
	}
	return best
}

// LastResponse reads the JSONL file and extracts the most recent assistant
// response as clean, formatted text. Returns the formatted text and whether
// the response looks like it's waiting for user input (ends with a question
// or tool approval prompt).
//
// Deprecated: prefer LastResponseDetail for richer prompt metadata.
func LastResponse(jsonlPath string) (text string, isQuestion bool, err error) {
	text, prompt, err := LastResponseDetail(jsonlPath)
	return text, prompt.Kind != PromptNone, err
}

// LastResponseDetail reads the JSONL file and extracts the most recent
// assistant response along with structured prompt metadata. The returned
// PromptInfo enables callers to build the correct Telegram keyboard for
// each prompt type (numbered options, free-text, y/N, etc.).
func LastResponseDetail(jsonlPath string) (text string, prompt PromptInfo, err error) {
	if jsonlPath == "" {
		return "", PromptInfo{}, fmt.Errorf("no jsonl path")
	}

	entries, err := readLastEntries(jsonlPath, 20)
	if err != nil {
		return "", PromptInfo{}, err
	}

	// Walk backwards to find the last assistant entry with content.
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Type != "assistant" || e.Message == nil {
			continue
		}

		blocks, err := parseContentBlocks(e.Message.Content)
		if err != nil || len(blocks) == 0 {
			continue
		}

		text := formatBlocks(blocks)
		if strings.TrimSpace(text) == "" {
			continue
		}

		prompt := analyzePrompt(blocks, text)
		return text, prompt, nil
	}

	return "", PromptInfo{}, fmt.Errorf("no assistant response found in last 20 entries")
}

// readLastEntries reads the last N JSONL entries from the file.
func readLastEntries(path string, n int) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var all []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line limit
	for scanner.Scan() {
		var e Entry
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

func parseContentBlocks(raw json.RawMessage) ([]ContentBlock, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	// Content can be a string (user messages) or array (assistant messages)
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, err
		}
		return []ContentBlock{{Type: "text", Text: s}}, nil
	}
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// formatBlocks converts structured content blocks into clean, readable text.
func formatBlocks(blocks []ContentBlock) string {
	var out strings.Builder
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if strings.TrimSpace(b.Text) != "" {
				out.WriteString(b.Text)
				out.WriteString("\n")
			}
		case "tool_use":
			desc := ""
			if b.Input != nil {
				if b.Input.Description != "" {
					desc = " — " + b.Input.Description
				}
				if b.Input.Command != "" {
					desc += "\n  " + b.Input.Command
				}
				if b.Input.FilePath != "" {
					desc += "\n  " + b.Input.FilePath
				}
			}
			out.WriteString(fmt.Sprintf("⚡ %s%s\n", b.Name, desc))
		case "thinking":
			// Skip thinking blocks — internal reasoning, not useful for Telegram
		}
	}
	return out.String()
}

// ── Prompt analysis ───────────────────────────────────────────

// analyzePrompt inspects content blocks and formatted text to determine
// what kind of interactive prompt (if any) the assistant is presenting.
// Detection priority: AskUserQuestion tool_use > [y/N/a] > [y/N] >
// numbered list > lettered list > bullet list > Allow/permit > plan
// approval > default value > enter-continue > open/closed question.
func analyzePrompt(blocks []ContentBlock, text string) PromptInfo {
	// 1. Structural: AskUserQuestion tool_use block
	if q, ok := detectAskUserQuestion(blocks); ok {
		return PromptInfo{Kind: PromptAskUser, Question: q}
	}

	lower := strings.ToLower(text)

	// 2. Explicit bracket patterns (highest textual specificity)
	if strings.Contains(lower, "[y/n/a]") {
		return PromptInfo{Kind: PromptYesNoAlways}
	}
	if reYN.MatchString(lower) {
		return PromptInfo{Kind: PromptYesNo}
	}

	// 3. Numbered option list: "1. ...\n2. ...\n3. ..." preceded by a question-like line
	if opts := detectNumberedOptions(text); len(opts) > 0 {
		// Upgrade to multi-select if the question wording makes it explicit.
		if isMultiSelectPhrasing(lower) {
			return PromptInfo{Kind: PromptMultiSelect, Options: opts}
		}
		return PromptInfo{Kind: PromptNumberedList, Options: opts}
	}

	// 4. Lettered inline options: "(a) ..., (b) ..., (c) ..."
	if opts := detectLetteredOptions(text); len(opts) > 0 {
		return PromptInfo{Kind: PromptLetteredList, Options: opts}
	}

	// 5. Bullet-point option list preceded by a question
	if opts := detectBulletOptions(text); len(opts) > 0 {
		return PromptInfo{Kind: PromptBulletList, Options: opts}
	}

	// 6. Allow / permit / approve — marker in Question for distinct labels.
	if reAllow.MatchString(lower) {
		return PromptInfo{Kind: PromptYesNo, Question: "allow"}
	}

	// 7. Plan approval: "shall I proceed?", "ready to implement?", etc.
	if detectPlanApproval(text) {
		return PromptInfo{Kind: PromptPlanApproval}
	}

	// 8. Default value prompt: "Enter name [default]:" or "[default]:"
	if def, ok := detectDefaultValue(text); ok {
		return PromptInfo{Kind: PromptDefaultValue, DefaultValue: def}
	}

	// 9. Enter-to-continue
	if detectEnterContinue(text) {
		return PromptInfo{Kind: PromptEnterContinue}
	}

	// 10. Generic trailing question mark — distinguish open-ended vs yes/no
	if detectTrailingQuestion(text) {
		if isOpenEndedQuestion(text) {
			return PromptInfo{Kind: PromptGenericQuestion}
		}
		return PromptInfo{Kind: PromptYesNo}
	}

	return PromptInfo{}
}

// detectAskUserQuestion scans content blocks for a tool_use with name
// "AskUserQuestion" and returns the question text.
func detectAskUserQuestion(blocks []ContentBlock) (string, bool) {
	for _, b := range blocks {
		if b.Type == "tool_use" && b.Name == "AskUserQuestion" && b.Input != nil && b.Input.Question != "" {
			return b.Input.Question, true
		}
	}
	return "", false
}

// detectNumberedOptions looks for a numbered list (1. ... 2. ... 3. ...)
// in the last 15 lines of text. Requires a question-context line
// preceding the list (ending with ? or containing question keywords)
// and consecutive numbering starting from 1 with 2–9 items.
func detectNumberedOptions(text string) []PromptOption {
	lines := tailNonEmpty(text, 15)
	if len(lines) < 2 {
		return nil
	}

	// Scan for consecutive numbered lines. Check the numbered pattern
	// BEFORE question-context to avoid false matches on lines like
	// "1) Option Alpha" where "option" is a question keyword.
	var opts []PromptOption
	questionFound := false
	for _, line := range lines {
		t := strings.TrimSpace(line)

		// Try numbered pattern first.
		m := reNumberedOpt.FindStringSubmatch(t)
		if m != nil {
			num := m[1]
			label := strings.TrimSpace(m[2])
			expected := fmt.Sprintf("%d", len(opts)+1)
			if num != expected {
				opts = nil // reset — not consecutive
				continue
			}
			opts = append(opts, PromptOption{Key: num, Label: label})
			continue
		}

		// Not a numbered line — check question context.
		if hasQuestionContext(t) {
			questionFound = true
			continue
		}

		// Non-numbered, non-question line after we started collecting — stop.
		if len(opts) > 0 {
			break
		}
	}

	if len(opts) < 2 || len(opts) > 9 || !questionFound {
		return nil
	}
	return opts
}

// detectLetteredOptions looks for inline lettered options like
// "(a) continue, (b) modify, or (c) stop" in the last 5 lines.
// Uses a split-based approach: find all (letter) markers, extract
// the text between them as labels.
func detectLetteredOptions(text string) []PromptOption {
	lines := tailNonEmpty(text, 5)
	joined := strings.Join(lines, " ")

	// Find all (letter) marker positions.
	markers := reLetteredMarker.FindAllStringSubmatchIndex(joined, -1)
	if len(markers) < 2 {
		return nil
	}

	var opts []PromptOption
	for i, loc := range markers {
		// loc: [fullStart, fullEnd, groupStart, groupEnd]
		key := strings.ToLower(joined[loc[2]:loc[3]])
		expected := string(rune('a' + i))
		if key != expected {
			return nil
		}

		// Label: text from after this marker to start of next marker (or end).
		labelStart := loc[1] // end of full match "(x) "
		var labelEnd int
		if i+1 < len(markers) {
			labelEnd = markers[i+1][0]
		} else {
			labelEnd = len(joined)
		}
		label := strings.TrimSpace(joined[labelStart:labelEnd])
		// Strip trailing punctuation and connectors.
		label = strings.TrimRight(label, " ,;.?？")
		label = strings.TrimSuffix(label, " or")
		label = strings.TrimSuffix(label, " and")
		label = strings.TrimSpace(label)
		if label == "" {
			return nil
		}
		opts = append(opts, PromptOption{Key: key, Label: label})
	}

	if len(opts) > 9 {
		opts = opts[:9]
	}
	return opts
}

// detectBulletOptions looks for a bullet-point list (- item / • item)
// preceded by a question-like line in the last 15 lines of text.
func detectBulletOptions(text string) []PromptOption {
	lines := tailNonEmpty(text, 15)
	if len(lines) < 2 {
		return nil
	}

	questionFound := false
	var opts []PromptOption
	for _, line := range lines {
		t := strings.TrimSpace(line)

		// Try bullet pattern first to avoid "* Approach A" being
		// misclassified as question context (keyword "approach").
		m := reBulletOpt.FindStringSubmatch(t)
		if m != nil && questionFound {
			label := strings.TrimSpace(m[1])
			if label != "" {
				key := fmt.Sprintf("%d", len(opts)+1)
				opts = append(opts, PromptOption{Key: key, Label: label})
			}
			continue
		}

		if hasQuestionContext(t) {
			questionFound = true
			opts = nil // reset: options follow the question
			continue
		}

		// Non-bullet, non-question line after collecting → stop
		if len(opts) > 0 {
			break
		}
	}

	if len(opts) < 2 || len(opts) > 9 {
		return nil
	}
	return opts
}

// detectPlanApproval checks for plan-approval patterns in the tail text.
func detectPlanApproval(text string) bool {
	lines := tailNonEmpty(text, 5)
	joined := strings.ToLower(strings.Join(lines, " "))
	return rePlanApproval.MatchString(joined)
}

// detectDefaultValue looks for a default-value prompt like "[default]:"
// in the last non-empty line.
func detectDefaultValue(text string) (string, bool) {
	lines := tailNonEmpty(text, 3)
	if len(lines) == 0 {
		return "", false
	}
	last := lines[len(lines)-1]
	m := reDefaultValue.FindStringSubmatch(last)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// detectEnterContinue checks for "press enter to continue" patterns.
func detectEnterContinue(text string) bool {
	lines := tailNonEmpty(text, 3)
	joined := strings.ToLower(strings.Join(lines, " "))
	return reEnterContinue.MatchString(joined)
}

// isMultiSelectPhrasing returns true when the prompt text (already
// lowercased) explicitly invites more than one answer. Upgrades a
// numbered list from single-pick to checkbox UI.
func isMultiSelectPhrasing(lower string) bool {
	needles := []string{
		"select all that apply",
		"choose all that apply",
		"choose one or more",
		"select one or more",
		"choose multiple",
		"select multiple",
		"(多选)", "（多选）",
		"(可多选)", "（可多选）",
		"可以多选", "可多选",
		"多项选择",
	}
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}

// detectTrailingQuestion checks if the text ends with a question mark.
func detectTrailingQuestion(text string) bool {
	lines := tailNonEmpty(text, 3)
	if len(lines) == 0 {
		return false
	}
	last := strings.TrimSpace(lines[len(lines)-1])
	return strings.HasSuffix(last, "?") || strings.HasSuffix(last, "？")
}

// isOpenEndedQuestion returns true if the question is open-ended
// (what/which/how/where/who/describe/name/enter) rather than
// closed/yes-no (do/does/is/are/can/should/would/shall/will/have).
// Also returns true if the question contains "or" connecting alternatives.
func isOpenEndedQuestion(text string) bool {
	lines := tailNonEmpty(text, 3)
	if len(lines) == 0 {
		return false
	}
	// Find the last line ending with ?
	var questionLine string
	for i := len(lines) - 1; i >= 0; i-- {
		t := strings.TrimSpace(lines[i])
		if strings.HasSuffix(t, "?") || strings.HasSuffix(t, "？") {
			questionLine = t
			break
		}
	}
	if questionLine == "" {
		return false
	}
	lower := strings.ToLower(questionLine)

	// "X or Y?" pattern → open-ended (choosing between alternatives)
	if reOrChoice.MatchString(lower) {
		return true
	}

	// Open-ended starters: which, what, how, where, who, describe, name, enter
	if reOpenEnded.MatchString(lower) {
		return true
	}

	// Closed-form starters: do, does, is, are, can, should, would, shall, will, have
	// → NOT open-ended → returns false → caller gives Yes/No
	return false
}

// hasQuestionContext returns true if a line looks like a question
// or question-introducing context (ends with ?, or ends with : AND
// contains question keywords, or just contains question keywords).
func hasQuestionContext(line string) bool {
	t := strings.TrimSpace(line)
	if strings.HasSuffix(t, "?") || strings.HasSuffix(t, "？") {
		return true
	}
	lower := strings.ToLower(t)
	hasKeywords := reQuestionKeywords.MatchString(lower)
	// Colon-ending only counts as question context if it also has
	// question keywords (avoids "git log:" false positives).
	if strings.HasSuffix(t, ":") && hasKeywords {
		return true
	}
	return hasKeywords
}

// tailNonEmpty returns the last n non-empty lines from text.
func tailNonEmpty(text string, n int) []string {
	lines := strings.Split(text, "\n")
	var result []string
	for i := len(lines) - 1; i >= 0 && len(result) < n; i-- {
		t := strings.TrimSpace(lines[i])
		if t != "" {
			result = append([]string{t}, result...)
		}
	}
	return result
}

// ── Compiled patterns ─────────────────────────────────────────

var (
	reYN              = regexp.MustCompile(`[\[(]y/n[\])]`)
	reAllow           = regexp.MustCompile(`(?:allow|permit|approve)\s.*\?`)
	reNumberedOpt     = regexp.MustCompile(`^(\d+)[.)]\s+(.+)$`)
	reLetteredMarker  = regexp.MustCompile(`\(([a-zA-Z])\)\s*`)
	reBulletOpt       = regexp.MustCompile(`^[-•*]\s+(.+)$`)
	rePlanApproval    = regexp.MustCompile(`(?:shall i proceed|ready to implement|want me to (?:go ahead|start|proceed)|should i (?:start|begin|proceed|go ahead|implement)|do you want me to (?:proceed|continue|implement|start))\s*\??`)
	reDefaultValue    = regexp.MustCompile(`\[([^\]]+)\]\s*:\s*$`)
	reEnterContinue   = regexp.MustCompile(`(?:press|hit)\s+enter\b|enter to continue`)
	reQuestionKeywords = regexp.MustCompile(`\b(?:which|choose|select|prefer|pick|would you like|options?|approach(?:es)?)\b`)
	reOpenEnded       = regexp.MustCompile(`^\s*(?:which|what|how|where|who|describe|name|enter|type|specify|provide)\b`)
	reOrChoice        = regexp.MustCompile(`\b\w+\s+or\s+\w+.*\?`)
)
