package telegram

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"sync"

	"github.com/opendray/opendray/gateway/telegram/jsonl"
	"github.com/opendray/opendray/kernel/hub"
)

// Forwarder is now stateless + event-driven: no goroutines, no polling.
//
// When the Notifier receives an idle event for a session that has linked
// chats, it calls Forwarder.SendIfChanged(). That method:
//   1. Takes a Buffer().Snapshot()
//   2. Cleans it (StripANSI → stripBoxDrawing → dedup → filterNoise)
//   3. Diffs against the last-sent snapshot for this session
//   4. Sends only genuinely-new lines to every linked chat
//
// No polling = no wasted work when nobody is linked. No goroutine per
// session = no cleanup needed on unlink. Idle fires at most once per
// quiet period (hub.watchIdle gates it), so Telegram gets exactly one
// message per Claude "response cycle" — matching the user's expectation
// of "send once when Claude finishes, not continuously".
type Forwarder struct {
	bot         *Bot
	hub         *hub.Hub
	store       *LinkStore
	logger      *slog.Logger
	multiSelect *MultiSelectStore
	mu          sync.Mutex
	last        map[string]string // sessionID → last cleaned snapshot
}

func NewForwarder(bot *Bot, h *hub.Hub, store *LinkStore, logger *slog.Logger) *Forwarder {
	if logger == nil {
		logger = slog.Default()
	}
	return &Forwarder{
		bot:    bot,
		hub:    h,
		store:  store,
		logger: logger,
		last:   map[string]string{},
	}
}

// SetMultiSelectStore wires the store used to register checkbox state
// right after a PromptMultiSelect message is sent. Nil disables the
// wiring (multi-select will still render, but toggle/submit won't find
// state and will show "expired").
func (f *Forwarder) SetMultiSelectStore(ms *MultiSelectStore) {
	f.multiSelect = ms
}

// EnsureForSession is now a no-op (no goroutines to start). Kept so
// commands.go callers don't need changes.
func (f *Forwarder) EnsureForSession(_ context.Context, _ string) {}

func (f *Forwarder) StopForSession(sessionID string) {
	f.mu.Lock()
	delete(f.last, sessionID)
	f.mu.Unlock()
}

// ResetSnapshot clears the last-sent snapshot for a session so the NEXT
// SendIfChanged pushes the full response (no diff). Called after user
// input arrives from Telegram: a new turn = user wants the full reply,
// not a per-line diff. Without this, short replies whose lines overlap
// with the previous turn (common Markdown headings, numbered lists,
// boilerplate bullets) get filtered below the 5-rune threshold and the
// whole message is dropped.
func (f *Forwarder) ResetSnapshot(sessionID string) {
	f.mu.Lock()
	delete(f.last, sessionID)
	f.mu.Unlock()
}

func (f *Forwarder) HasListeners(sessionID string) bool {
	return len(f.store.ChatsFor(sessionID)) > 0
}

func (f *Forwarder) StopAll() {
	f.mu.Lock()
	f.last = map[string]string{}
	f.mu.Unlock()
}

// SendIfChanged is called by the Notifier on each idle event. For Claude
// sessions, it reads the structured JSONL file (clean, no ANSI); for other
// session types, it falls back to the buffer snapshot pipeline.
// Returns the number of chats notified.
func (f *Forwarder) SendIfChanged(ctx context.Context, sessionID string) int {
	chats := f.store.ChatsFor(sessionID)
	if len(chats) == 0 {
		f.logger.Debug("forwarder: no linked chats", "session", sessionID)
		return 0
	}

	// Try JSONL first (gives clean structured output for Claude sessions).
	current, keyboard, prompt := f.tryJSONL(ctx, sessionID)

	// Fallback: buffer snapshot for non-Claude or when JSONL unavailable.
	if current == "" {
		ts, ok := f.hub.GetTerminalSession(sessionID)
		if !ok {
			f.logger.Debug("forwarder: session not running", "session", sessionID)
			return 0
		}
		raw := string(ts.Buffer().Snapshot())
		lines := strings.Split(raw, "\n")
		const tailN = 80
		if len(lines) > tailN {
			lines = lines[len(lines)-tailN:]
		}
		current = CleanForTelegram(strings.Join(lines, "\n"))
	}

	// Compute diff and update snapshot under the lock to avoid TOCTOU:
	// two concurrent calls for the same session would both read the same
	// prev, compute identical diffs, and send duplicates.
	f.mu.Lock()
	prev := f.last[sessionID]
	newContent := diffLines(prev, current)
	newContent = strings.TrimSpace(newContent)
	if len([]rune(newContent)) < 5 {
		f.mu.Unlock()
		f.logger.Debug("forwarder: no meaningful diff",
			"session", sessionID,
			"diff_runes", len([]rune(newContent)),
		)
		return 0
	}
	// Optimistically record the snapshot now; roll back if all sends fail.
	f.last[sessionID] = current
	f.mu.Unlock()

	const maxChunk = 3500
	sent := 0
	chunks := splitForTelegram(newContent, maxChunk)
	for i, chunk := range chunks {
		body := formatForTelegram(chunk)
		opts := &SendOpts{ParseMode: "HTML", DisablePreview: true}
		// Attach keyboard to the last chunk only
		if keyboard != nil && i == len(chunks)-1 {
			opts.ReplyMarkup = keyboard
		}
		for _, chatID := range chats {
			msgID, err := f.bot.Send(ctx, chatID, body, opts)
			if err != nil {
				f.logger.Warn("forwarder: send failed",
					"session", sessionID, "chat", chatID, "error", err,
				)
				continue
			}
			sent++
			// Register checkbox state only on the final chunk (where the
			// keyboard was attached) and only for multi-select prompts.
			if i == len(chunks)-1 &&
				prompt.Kind == jsonl.PromptMultiSelect &&
				f.multiSelect != nil && msgID > 0 {
				f.multiSelect.Create(chatID, msgID, sessionID, prompt.Options)
			}
		}
	}

	// If ALL sends failed, roll back the snapshot so the next idle
	// cycle retries with the same diff.
	if sent == 0 {
		f.mu.Lock()
		f.last[sessionID] = prev
		f.mu.Unlock()
	}

	f.logger.Info("forwarder: pushed output",
		"session", sessionID, "chats", sent, "runes", len([]rune(newContent)),
	)
	return sent
}

// tryJSONL attempts to read the latest assistant response from Claude's
// structured JSONL file. Returns (cleanText, keyboard) or ("", nil) if
// the JSONL is unavailable (non-Claude session, file not found, etc.).
//
// Uses LastResponseDetail for structured prompt detection — numbered
// options, lettered choices, AskUserQuestion, default-value prompts,
// etc. are all handled and produce the correct inline keyboard.
func (f *Forwarder) tryJSONL(ctx context.Context, sessionID string) (string, *InlineKeyboardMarkup, jsonl.PromptInfo) {
	// We need the session's CWD to find the project directory.
	sess, ok, err := f.hub.Get(ctx, sessionID)
	if err != nil || !ok {
		return "", nil, jsonl.PromptInfo{}
	}
	if sess.SessionType != "claude" {
		return "", nil, jsonl.PromptInfo{}
	}

	bases := claudeBasePaths(ctx, f.hub.DB(), sess, f.bot.Cfg().ExtraClaudeDirs)
	jsonlPath := jsonl.ResolveLatestJSONL(bases, sess.CWD)
	if jsonlPath == "" {
		f.logger.Debug("forwarder: no JSONL found", "session", sessionID, "bases", bases)
		return "", nil, jsonl.PromptInfo{}
	}

	text, prompt, err := jsonl.LastResponseDetail(jsonlPath)
	if err != nil || strings.TrimSpace(text) == "" {
		f.logger.Debug("forwarder: JSONL read failed or empty", "session", sessionID, "error", err)
		return "", nil, jsonl.PromptInfo{}
	}

	// For free-text prompts (AskUser / GenericQuestion), append a hint
	// so the user knows to type a reply instead of tapping a button.
	if prompt.Kind == jsonl.PromptAskUser || prompt.Kind == jsonl.PromptGenericQuestion {
		text = strings.TrimSpace(text) + "\n\n💬 Reply to this message to answer."
	}
	// Multi-select: append a hint that free-text typing also works.
	if prompt.Kind == jsonl.PromptMultiSelect {
		text = strings.TrimSpace(text) + "\n\n☑ Check the options you want, then tap <b>Submit</b>. You can also type a reply directly in the chat (it will be sent as a separate message)."
	}

	kb := KeyboardFromPrompt(prompt, sessionID)

	f.logger.Info("forwarder: using JSONL output",
		"session", sessionID, "chars", len(text), "prompt", prompt.Kind)
	return text, kb, prompt
}

// ── HTML formatting for Telegram ────────────────────────────────

// FormatForTelegramExport is the exported alias for tests.
func FormatForTelegramExport(s string) string { return formatForTelegram(s) }

// formatForTelegram converts Markdown (from Claude's JSONL) or plain text
// into Telegram HTML. Handles: ## headers → <b>, **bold** → <b>,
// *italic* → <i>, `code` → <code>, ```blocks``` → <pre>, - items → •.
func formatForTelegram(s string) string {
	lines := strings.Split(s, "\n")
	var out strings.Builder
	inCodeBlock := false

	// Markdown tables have no native Telegram rendering and look terrible
	// on mobile when wrapped in <pre> (column wrap, visible |---|---|
	// separators, unrendered markdown). Instead, convert each data row
	// into a vertical "Header: value" block.
	var tableBuf []string
	flushTable := func() {
		if len(tableBuf) == 0 {
			return
		}
		out.WriteString(renderTable(tableBuf))
		tableBuf = nil
	}

	for _, line := range lines {
		// Code block toggle: ```
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			flushTable()
			if inCodeBlock {
				out.WriteString("</pre>\n")
				inCodeBlock = false
			} else {
				out.WriteString("<pre>")
				// If there's content after ``` (language tag), skip it
				inCodeBlock = true
			}
			continue
		}
		if inCodeBlock {
			out.WriteString(escapeHTML(line) + "\n")
			continue
		}

		// Markdown table row: trimmed line starts with '|' and has ≥3 pipes
		// (covers both header, separator, and data rows). Accumulate until
		// the run ends.
		if isTableRow(trimmed) {
			tableBuf = append(tableBuf, trimmed)
			continue
		}
		flushTable()

		if trimmed == "" {
			out.WriteString("\n")
			continue
		}

		// ## Headers → <b> (any level: #, ##, ###, etc.)
		if len(trimmed) > 1 && trimmed[0] == '#' {
			header := strings.TrimLeft(trimmed, "#")
			header = strings.TrimSpace(header)
			if header != "" {
				out.WriteString("\n<b>" + escapeHTML(header) + "</b>\n")
				continue
			}
		}

		// Bullet items: - or * or •
		if strings.HasPrefix(trimmed, "- ") {
			out.WriteString("  • " + inlineMarkdown(trimmed[2:]) + "\n")
			continue
		}
		if strings.HasPrefix(trimmed, "* ") && !strings.HasPrefix(trimmed, "**") {
			out.WriteString("  • " + inlineMarkdown(trimmed[2:]) + "\n")
			continue
		}
		if strings.HasPrefix(trimmed, "• ") {
			out.WriteString("  • " + inlineMarkdown(trimmed[2:]) + "\n")
			continue
		}

		// Tool calls (from JSONL formatBlocks)
		if len(trimmed) > 0 {
			first, _ := firstRune(trimmed)
			if first == '⚡' || first == '✅' || first == '✗' || first == '🔧' {
				out.WriteString("<b>" + escapeHTML(trimmed) + "</b>\n")
				continue
			}
		}

		// Normal line — convert inline markdown
		out.WriteString(inlineMarkdown(trimmed) + "\n")
	}

	flushTable()
	if inCodeBlock {
		out.WriteString("</pre>")
	}
	return strings.TrimSpace(out.String())
}

// isTableRow reports whether a line is a markdown table row. A table row
// starts with '|' and has at least 3 pipes total (i.e. at least 2 columns).
func isTableRow(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "|") {
		return false
	}
	return strings.Count(trimmed, "|") >= 3
}

// splitTableRow strips leading/trailing '|' and splits into trimmed cells.
func splitTableRow(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.TrimPrefix(row, "|")
	row = strings.TrimSuffix(row, "|")
	parts := strings.Split(row, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}
	return cells
}

// isSeparatorRow reports whether a split row is the markdown alignment
// separator (e.g. "|---|:---:|---:|"): every cell contains only '-', ':',
// or whitespace, and at least one '-'.
func isSeparatorRow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	sawDash := false
	for _, c := range cells {
		if c == "" {
			continue
		}
		for _, r := range c {
			switch r {
			case '-':
				sawDash = true
			case ':', ' ', '\t':
				// allowed
			default:
				return false
			}
		}
	}
	return sawDash
}

// renderTable converts markdown table rows into a mobile-friendly
// vertical "Header: value" layout. Headers and cell values go through
// inlineMarkdown so **bold** / `code` / *italic* render as HTML.
//
// Layout:
//
//	<b>HeaderA:</b> valueA
//	<b>HeaderB:</b> valueB
//
//	<b>HeaderA:</b> valueA2
//	<b>HeaderB:</b> valueB2
//
// If the table is malformed (no separator, single row, etc.), it falls
// back to emitting each row as a bullet line so nothing is lost.
func renderTable(rows []string) string {
	if len(rows) == 0 {
		return ""
	}

	parsed := make([][]string, 0, len(rows))
	sepIdx := -1
	for i, r := range rows {
		cells := splitTableRow(r)
		parsed = append(parsed, cells)
		if sepIdx == -1 && i > 0 && isSeparatorRow(cells) {
			sepIdx = i
		}
	}

	// Malformed (no separator or no data rows): emit each row as a bullet.
	if sepIdx < 1 || sepIdx >= len(parsed)-1 {
		var b strings.Builder
		for _, cells := range parsed {
			if isSeparatorRow(cells) {
				continue
			}
			joined := strings.Join(cells, " │ ")
			b.WriteString("  • " + inlineMarkdown(joined) + "\n")
		}
		return b.String()
	}

	headers := parsed[sepIdx-1]
	dataRows := parsed[sepIdx+1:]

	var b strings.Builder
	for i, cells := range dataRows {
		if i > 0 {
			b.WriteString("\n")
		}
		for j, header := range headers {
			var value string
			if j < len(cells) {
				value = cells[j]
			}
			if header == "" && value == "" {
				continue
			}
			b.WriteString("<b>")
			b.WriteString(escapeHTML(header))
			b.WriteString(":</b> ")
			b.WriteString(inlineMarkdown(value))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// inlineMarkdown converts **bold**, *italic*, and `code` to HTML tags.
func inlineMarkdown(s string) string {
	s = escapeHTML(s)
	// **bold** → <b>bold</b>
	s = boldRe.ReplaceAllString(s, "<b>$1</b>")
	// *italic* → <i>italic</i> (but not inside bold which is already converted)
	// Named groups preserve the surrounding characters that the regex must
	// consume (Go regex has no lookahead).
	s = italicRe.ReplaceAllString(s, "${pre}<i>${body}</i>${post}")
	// `code` → <code>code</code>
	s = codeRe.ReplaceAllString(s, "<code>$1</code>")
	return s
}

var (
	boldRe   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe = regexp.MustCompile(`(?P<pre>^|[^*])\*(?P<body>[^*]+?)\*(?P<post>[^*]|$)`)
	codeRe   = regexp.MustCompile("`([^`]+)`")
)

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func firstRune(s string) (rune, bool) {
	for _, r := range s {
		return r, true
	}
	return 0, false
}

// ── Diff + noise filter ────────────────────────────────────────

// diffLines returns lines present in curr but NOT in prev.
func diffLines(prev, curr string) string {
	prevSet := make(map[string]bool)
	for _, line := range strings.Split(prev, "\n") {
		t := strings.TrimSpace(line)
		if t != "" {
			prevSet[t] = true
		}
	}
	var out []string
	for _, line := range strings.Split(curr, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if prevSet[t] {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func filterNoise(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if isSeparatorLine(t) {
			continue
		}
		if len([]rune(t)) <= 2 && !isAlphanumeric(t) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func stripBoxDrawing(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	for _, r := range s {
		if (r >= 0x2500 && r <= 0x25FF) ||
			r == '╭' || r == '╮' || r == '╯' || r == '╰' {
			out.WriteRune(' ')
			continue
		}
		out.WriteRune(r)
	}
	result := multiSpace.ReplaceAllString(out.String(), " ")
	return result
}

var multiSpace = regexp.MustCompile(`[ ]{2,}`)

func isSeparatorLine(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '-' && r != '=' && r != '_' && r != '~' && r != '*' && r != '.' {
			return false
		}
	}
	return len(strings.TrimSpace(s)) > 2
}

func isAlphanumeric(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r >= 0x4E00 {
			return true
		}
	}
	return false
}

func dedup(s string) string {
	lines := strings.Split(s, "\n")
	seen := make(map[string]bool, len(lines))
	out := make([]string, 0, len(lines))
	blankCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			if blankCount <= 1 {
				out = append(out, "")
			}
			continue
		}
		blankCount = 0
		if seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// CleanForTelegram is the full pipeline: ANSI strip → box-drawing strip →
// Claude TUI chrome filter → dedup → noise filter.
// Used by Forwarder for diff-based notifications where dedup helps.
func CleanForTelegram(raw string) string {
	text := StripANSI(raw)
	text = stripBoxDrawing(text)
	text = filterClaudeChrome(text)
	text = dedup(text)
	text = filterNoise(text)
	return text
}

// CleanForScreen is a lighter pipeline for /screen display. It preserves
// the actual terminal layout: no dedup (code often has repeated patterns),
// no noise filter (separators like "---" are meaningful content).
// Only collapses runs of 3+ blank lines to keep it readable.
func CleanForScreen(raw string) string {
	text := StripANSI(raw)
	text = stripBoxDrawing(text)
	text = filterClaudeChrome(text)
	text = collapseBlankLines(text)
	return text
}

// collapseBlankLines reduces runs of 3+ blank lines to 1, keeping the
// layout readable without destroying intentional spacing.
func collapseBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	blanks := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blanks++
			if blanks <= 2 {
				out = append(out, "")
			}
			continue
		}
		blanks = 0
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// filterClaudeChrome surgically removes Claude Code CLI's TUI elements
// (status bar, prompt, permissions, keyboard hints, MCP status, git
// branch) that survive ANSI/box-drawing stripping. Unlike the previous
// line-dropping approach, this REMOVES only the matched chrome portions
// and keeps surrounding content — important because cursor-positioning
// removal glues chrome and content onto the same line.
func filterClaudeChrome(s string) string {
	// Pass 1: regex-remove chrome fragments from within lines.
	for _, re := range chromePatterns {
		s = re.ReplaceAllString(s, " ")
	}

	// Pass 2: drop lines that are now empty or pure-symbol debris.
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			out = append(out, "")
			continue
		}
		// Pure symbols / UI debris (no letters, digits, or CJK)
		if !hasReadableContent(t) {
			continue
		}
		// Very short fragments (≤3 chars) that are UI debris after
		// chrome patterns carved out the middle of a line.
		if len([]rune(t)) <= 3 {
			continue
		}
		// "copy" button label
		if t == "copy" {
			continue
		}
		out = append(out, strings.TrimSpace(line))
	}
	return strings.Join(out, "\n")
}

// chromePatterns match Claude Code CLI's TUI elements. Each is replaced
// with a space (not removed entirely) so surrounding content doesn't
// smash together. Applied in order.
var chromePatterns = []*regexp.Regexp{
	// Model bar: "pus 4.6 | Max]", "Opus 4.6 ...", "Sonnet 4.6"
	regexp.MustCompile(`(?i)(?:pus|opus|sonnet|haiku)\s*4\.\d\S*(?:\s*[\|·]\s*\w+\]?)?`),
	// Permission mode + hint
	regexp.MustCompile(`(?i)(?:bypass\s*permissions?\s*(?:on|off)?\s*)?(?:\(shift\+tab\s*(?:to\s*)?cycle\)|\bshift\+tabtocycle\b)`),
	regexp.MustCompile(`(?i)\bbypass\s*permissions?\s*(?:on|off)?\b`),
	// Expand hints
	regexp.MustCompile(`(?i)(?:Listed\s+\d+\s+(?:directories|files)\s*)?\(ctrl\+[or]\s*to\s*expand\)`),
	// Git branch in prompt — closing paren may be stripped; repo name has underscores
	regexp.MustCompile(`[\w_]+\s+git:\([^)\n]*\)?[…\.\s]*`),
	// Bare "remote_claude_code" or similar repo names in prompt debris
	regexp.MustCompile(`\bremote_claude_code\b`),
	// MCP status
	regexp.MustCompile(`(?i)\d*MCP\s*server\s*(?:failed|s)?\s*(?:·\s*/mcp)?`),
	// Prompt symbols
	regexp.MustCompile(`❯\s*`),
	regexp.MustCompile(`⏵+\s*`),
	regexp.MustCompile(`⏺\s*`),
	// Baked/Churned timer
	regexp.MustCompile(`(?i)(?:Baked|Churned)\s+for\s+[\dmsh\s]+`),
	// Cerebrating
	regexp.MustCompile(`\+Cerebrating\.\.[^)]*\)`),
	// Collapse/expand markers
	regexp.MustCompile(`\d+\s*lines?\s*\(ctrl\+r\s*to\s*expand\)`),
	// Claude thinking/generating/cultivating status
	regexp.MustCompile(`[✢⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]\s*(?:Cultivating|Thinking|Generating|Reasoning|Connecting|Compacting|Reading|Searching|Executing|Streaming|Writing|Updating|Prestidigitatting|Cerebrating)[^)\n]*(?:\([^)]*\))?`),
	// Token/cost counters
	regexp.MustCompile(`(?:↓|↑)\s*[\d,.]+\s*(?:tokens?|k)\b`),
}

func hasReadableContent(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || (r >= 0x4E00 && r <= 0x9FFF) ||
			(r >= 0x3400 && r <= 0x4DBF) {
			return true
		}
	}
	return false
}

// ── ANSI stripping ─────────────────────────────────────────────

func StripANSI(s string) string {
	out := make([]byte, 0, len(s))
	i := 0
	n := len(s)
	for i < n {
		b := s[i]
		if b == 0x1b {
			i++
			if i >= n {
				break
			}
			switch s[i] {
			case '[':
				i++
				for i < n && s[i] >= 0x30 && s[i] <= 0x3F {
					i++
				}
				for i < n && s[i] >= 0x20 && s[i] <= 0x2F {
					i++
				}
				if i < n && s[i] >= 0x40 && s[i] <= 0x7E {
					i++
				}
			case ']':
				i++
				for i < n && s[i] != 0x07 && s[i] != 0x1b {
					i++
				}
				if i < n && s[i] == 0x07 {
					i++
				} else if i+1 < n && s[i] == 0x1b && s[i+1] == '\\' {
					i += 2
				}
			case 'P', 'X', '_', '^':
				i++
				for i+1 < n {
					if s[i] == 0x1b && s[i+1] == '\\' {
						i += 2
						break
					}
					i++
				}
			case '(', ')', '*', '+':
				i++
				if i < n {
					i++
				}
			default:
				if s[i] >= 0x20 && s[i] <= 0x7E {
					i++
				}
			}
			continue
		}
		if b < 0x20 && b != '\n' && b != '\t' {
			i++
			continue
		}
		out = append(out, b)
		i++
	}
	result := string(out)
	result = orphanCSI.ReplaceAllString(result, "")
	result = blankRun.ReplaceAllString(result, "\n\n")
	return result
}

var (
	orphanCSI = regexp.MustCompile(`\[[\d;:?]*[a-zA-Z~]`)
	blankRun  = regexp.MustCompile(`\n{3,}`)
)
