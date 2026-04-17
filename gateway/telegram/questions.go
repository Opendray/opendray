package telegram

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/linivek/ntc/gateway/telegram/jsonl"
)

// DetectQuestion scans cleaned terminal output for interactive prompts
// and returns an inline keyboard. This is the backward-compatible entry
// point — it calls DetectPrompt + KeyboardFromPrompt internally.
//
// Returns nil when no question is detected.
func DetectQuestion(cleanedText string, sessionID string) *InlineKeyboardMarkup {
	prompt := DetectPrompt(cleanedText)
	return KeyboardFromPrompt(prompt, sessionID)
}

// DetectPrompt analyzes cleaned terminal output (PTY buffer) and returns
// structured PromptInfo. This is the primary detection function for
// non-JSONL sessions (terminal, codex, etc.) and also serves as fallback
// when JSONL is unavailable for Claude sessions.
//
// Detection priority (highest → lowest specificity):
//  1. [y/N/a]               → PromptYesNoAlways
//  2. [y/N] / [Y/n] / (y/n) → PromptYesNo
//  3. Numbered list 1. 2. 3. → PromptNumberedList
//  4. Lettered (a) (b) (c)   → PromptLetteredList
//  5. Bullet list after ?     → PromptBulletList
//  6. Allow/permit/approve    → PromptYesNo
//  7. Plan approval phrases   → PromptPlanApproval
//  8. Default value [val]:    → PromptDefaultValue
//  9. Press Enter to continue → PromptEnterContinue
//  10. Open-ended question ?  → PromptGenericQuestion (no Yes/No)
//  11. Closed question ?      → PromptYesNo
func DetectPrompt(cleanedText string) jsonl.PromptInfo {
	// Scan the last 15 non-empty lines for patterns.
	lines := strings.Split(cleanedText, "\n")
	var tail []string
	for i := len(lines) - 1; i >= 0 && len(tail) < 15; i-- {
		t := strings.TrimSpace(lines[i])
		if t != "" {
			tail = append([]string{t}, tail...)
		}
	}
	if len(tail) == 0 {
		return jsonl.PromptInfo{}
	}
	joined := strings.Join(tail, " ")

	// 1. [y/N/a]
	if ynaPat.MatchString(joined) {
		return jsonl.PromptInfo{Kind: jsonl.PromptYesNoAlways}
	}

	// 2. [y/N] or [Y/n] or (y/n)
	if ynPat.MatchString(joined) {
		return jsonl.PromptInfo{Kind: jsonl.PromptYesNo}
	}

	// 3. Numbered list
	if opts := detectNumberedFromLines(tail); len(opts) > 0 {
		return jsonl.PromptInfo{Kind: jsonl.PromptNumberedList, Options: opts}
	}

	// 4. Lettered inline options
	if opts := detectLetteredFromJoined(joined); len(opts) > 0 {
		return jsonl.PromptInfo{Kind: jsonl.PromptLetteredList, Options: opts}
	}

	// 5. Bullet list after question
	if opts := detectBulletFromLines(tail); len(opts) > 0 {
		return jsonl.PromptInfo{Kind: jsonl.PromptBulletList, Options: opts}
	}

	// 6. Allow / permit / approve — PromptYesNo with "allow" marker
	//    so KeyboardFromPrompt uses "Allow/Deny" labels.
	if allowPat.MatchString(joined) {
		return jsonl.PromptInfo{Kind: jsonl.PromptYesNo, Question: "allow"}
	}

	// 7. Plan approval
	if planApprovalPat.MatchString(joined) {
		return jsonl.PromptInfo{Kind: jsonl.PromptPlanApproval}
	}

	// 8. Default value prompt
	lastLine := tail[len(tail)-1]
	if m := defaultValuePat.FindStringSubmatch(lastLine); m != nil {
		return jsonl.PromptInfo{Kind: jsonl.PromptDefaultValue, DefaultValue: m[1]}
	}

	// 9. Enter to continue
	if enterContinuePat.MatchString(strings.ToLower(joined)) {
		return jsonl.PromptInfo{Kind: jsonl.PromptEnterContinue}
	}

	// 10/11. Generic trailing question mark
	if questionPat.MatchString(lastLine) {
		if isOpenEndedLine(lastLine) {
			return jsonl.PromptInfo{Kind: jsonl.PromptGenericQuestion}
		}
		return jsonl.PromptInfo{Kind: jsonl.PromptYesNo}
	}

	return jsonl.PromptInfo{}
}

// KeyboardFromPrompt builds a Telegram InlineKeyboardMarkup from a
// PromptInfo. Returns nil for PromptNone and PromptAskUser (free-text
// prompts where the user should type a reply, not tap a button).
// Also returns nil for PromptGenericQuestion (open-ended questions).
func KeyboardFromPrompt(prompt jsonl.PromptInfo, sessionID string) *InlineKeyboardMarkup {
	switch prompt.Kind {
	case jsonl.PromptYesNoAlways:
		return makeKeyboard(sessionID, []btn{
			{"✅ Yes", "y\r"},
			{"❌ No", "n\r"},
			{"🔓 Always", "a\r"},
		})

	case jsonl.PromptYesNo:
		// "allow" marker → use Allow/Deny labels for permission prompts.
		if prompt.Question == "allow" {
			return makeKeyboard(sessionID, []btn{
				{"✅ Allow", "y\r"},
				{"❌ Deny", "n\r"},
			})
		}
		return makeKeyboard(sessionID, []btn{
			{"✅ Yes", "y\r"},
			{"❌ No", "n\r"},
		})

	case jsonl.PromptNumberedList:
		return makeOptionKeyboard(sessionID, prompt.Options)

	case jsonl.PromptLetteredList:
		return makeOptionKeyboard(sessionID, prompt.Options)

	case jsonl.PromptBulletList:
		return makeOptionKeyboard(sessionID, prompt.Options)

	case jsonl.PromptMultiSelect:
		return makeMultiSelectKeyboard(sessionID, prompt.Options, nil)

	case jsonl.PromptPlanApproval:
		return makeKeyboard(sessionID, []btn{
			{"▶ Proceed", "y\r"},
			{"⏹ Stop", "n\r"},
		})

	case jsonl.PromptDefaultValue:
		label := "Accept"
		if prompt.DefaultValue != "" {
			label = truncateLabel("Accept: "+prompt.DefaultValue, 20)
		}
		return makeKeyboard(sessionID, []btn{
			{label, "\r"},
		})

	case jsonl.PromptEnterContinue:
		return makeKeyboard(sessionID, []btn{
			{"Enter ↵", "\r"},
		})

	case jsonl.PromptAskUser, jsonl.PromptGenericQuestion:
		// Free-text / open-ended: no keyboard buttons.
		// Caller should add a "💬 Reply to answer" hint to the message.
		return nil

	default:
		return nil
	}
}

// ── Keyboard builders ─────────────────────────────────────────

type btn struct {
	text    string
	payload string // raw bytes to send to PTY
}

func makeKeyboard(sessionID string, buttons []btn) *InlineKeyboardMarkup {
	row := make([]InlineKeyboardButton, 0, len(buttons))
	for _, b := range buttons {
		row = append(row, InlineKeyboardButton{
			Text:         b.text,
			CallbackData: "send:" + sessionID + ":" + b.payload,
		})
	}
	// Add a Screen button on a second row for convenience
	row2 := []InlineKeyboardButton{
		{Text: "📺 Screen", CallbackData: "screen:" + sessionID},
	}
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{row, row2},
	}
}

// makeOptionKeyboard lays out option buttons in rows of 3 (or 1 per row
// if labels are long), capped at 9 options. Each button shows
// "key. label" and sends "key\r" to the PTY.
func makeOptionKeyboard(sessionID string, opts []jsonl.PromptOption) *InlineKeyboardMarkup {
	if len(opts) == 0 {
		return nil
	}
	if len(opts) > 9 {
		opts = opts[:9]
	}

	// Decide layout: 1 per row if any label > 25 runes, else 3 per row.
	perRow := 3
	for _, o := range opts {
		if len([]rune(o.Label)) > 25 {
			perRow = 1
			break
		}
	}

	var rows [][]InlineKeyboardButton
	var current []InlineKeyboardButton
	for _, o := range opts {
		label := truncateLabel(o.Key+". "+o.Label, 30)
		current = append(current, InlineKeyboardButton{
			Text:         label,
			CallbackData: "send:" + sessionID + ":" + o.Key + "\r",
		})
		if len(current) >= perRow {
			rows = append(rows, current)
			current = nil
		}
	}
	if len(current) > 0 {
		rows = append(rows, current)
	}

	// Screen row
	rows = append(rows, []InlineKeyboardButton{
		{Text: "📺 Screen", CallbackData: "screen:" + sessionID},
	})

	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

// makeMultiSelectKeyboard builds a checkbox-style keyboard for a
// multi-select prompt. Each option becomes a toggle button whose label
// is prefixed with ☑ when checked and ☐ otherwise; a "✅ Submit" and
// "📺 Screen" row is appended.
//
// Callback data:
//
//	multi_toggle:<sessionID>:<key>
//	multi_submit:<sessionID>
//
// The messageID is NOT encoded in the callback — Telegram automatically
// includes it via CallbackQuery.Message.MessageID, and decoupling the
// callback data from the state key keeps it well under the 64-byte
// limit when sessionIDs are UUIDs. The caller is expected to call
// MultiSelectStore.Create(chatID, messageID, sessionID, options) right
// after sending the message.
func makeMultiSelectKeyboard(sessionID string, opts []jsonl.PromptOption, checked map[string]bool) *InlineKeyboardMarkup {
	if len(opts) == 0 {
		return nil
	}
	if len(opts) > 9 {
		opts = opts[:9]
	}

	perRow := 1 // checkbox rows are clearer one-per-row
	for _, o := range opts {
		// If every label is short (<14 runes) switch to 2-per-row to save screen.
		if len([]rune(o.Key+". "+o.Label)) > 14 {
			perRow = 1
			break
		}
		perRow = 2
	}

	var rows [][]InlineKeyboardButton
	var current []InlineKeyboardButton
	for _, o := range opts {
		mark := "☐"
		if checked[o.Key] {
			mark = "☑"
		}
		label := truncateLabel(mark+" "+o.Key+". "+o.Label, 30)
		current = append(current, InlineKeyboardButton{
			Text:         label,
			CallbackData: "multi_toggle:" + sessionID + ":" + o.Key,
		})
		if len(current) >= perRow {
			rows = append(rows, current)
			current = nil
		}
	}
	if len(current) > 0 {
		rows = append(rows, current)
	}
	// Submit + Screen row
	rows = append(rows, []InlineKeyboardButton{
		{Text: "✅ Submit", CallbackData: "multi_submit:" + sessionID},
		{Text: "📺 Screen", CallbackData: "screen:" + sessionID},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

// truncateLabel caps a label at maxRunes, appending "..." if truncated.
func truncateLabel(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-3]) + "..."
}

// ── PTY-based detection helpers ───────────────────────────────

// detectNumberedFromLines scans lines for a numbered list preceded by
// a question-context line. Requires consecutive numbering from 1.
func detectNumberedFromLines(lines []string) []jsonl.PromptOption {
	if len(lines) < 2 {
		return nil
	}

	var opts []jsonl.PromptOption
	questionFound := false
	for _, line := range lines {
		t := strings.TrimSpace(line)

		// Check numbered pattern first to avoid lines like
		// "1) Option Alpha" being misclassified as question context.
		m := numberedOptPat.FindStringSubmatch(t)
		if m != nil {
			num := m[1]
			label := strings.TrimSpace(m[2])
			expected := fmt.Sprintf("%d", len(opts)+1)
			if num != expected {
				opts = nil
				continue
			}
			opts = append(opts, jsonl.PromptOption{Key: num, Label: label})
			continue
		}

		if hasQuestionContext(t) {
			questionFound = true
			continue
		}

		if len(opts) > 0 {
			break
		}
	}

	if len(opts) < 2 || len(opts) > 9 || !questionFound {
		return nil
	}
	return opts
}

// detectLetteredFromJoined scans joined text for inline lettered options.
// Uses a split-based approach: find all (letter) markers, extract
// the text between them as labels.
func detectLetteredFromJoined(joined string) []jsonl.PromptOption {
	markers := letteredMarkerPat.FindAllStringSubmatchIndex(joined, -1)
	if len(markers) < 2 {
		return nil
	}

	var opts []jsonl.PromptOption
	for i, loc := range markers {
		key := strings.ToLower(joined[loc[2]:loc[3]])
		expected := string(rune('a' + i))
		if key != expected {
			return nil
		}
		labelStart := loc[1]
		var labelEnd int
		if i+1 < len(markers) {
			labelEnd = markers[i+1][0]
		} else {
			labelEnd = len(joined)
		}
		label := strings.TrimSpace(joined[labelStart:labelEnd])
		label = strings.TrimRight(label, " ,;.?？")
		label = strings.TrimSuffix(label, " or")
		label = strings.TrimSuffix(label, " and")
		label = strings.TrimSpace(label)
		if label == "" {
			return nil
		}
		opts = append(opts, jsonl.PromptOption{Key: key, Label: label})
	}
	if len(opts) > 9 {
		opts = opts[:9]
	}
	return opts
}

// detectBulletFromLines scans lines for a bullet list after a question.
func detectBulletFromLines(lines []string) []jsonl.PromptOption {
	if len(lines) < 2 {
		return nil
	}

	questionFound := false
	var opts []jsonl.PromptOption
	for _, line := range lines {
		t := strings.TrimSpace(line)

		// Check bullet pattern first.
		m := bulletOptPat.FindStringSubmatch(t)
		if m != nil && questionFound {
			label := strings.TrimSpace(m[1])
			if label != "" {
				key := fmt.Sprintf("%d", len(opts)+1)
				opts = append(opts, jsonl.PromptOption{Key: key, Label: label})
			}
			continue
		}

		if hasQuestionContext(t) {
			questionFound = true
			opts = nil
			continue
		}

		if len(opts) > 0 {
			break
		}
	}

	if len(opts) < 2 || len(opts) > 9 {
		return nil
	}
	return opts
}

// hasQuestionContext checks if a line is a question or introduces options.
func hasQuestionContext(line string) bool {
	t := strings.TrimSpace(line)
	if strings.HasSuffix(t, "?") || strings.HasSuffix(t, "？") {
		return true
	}
	lower := strings.ToLower(t)
	hasKeywords := questionKeywordPat.MatchString(lower)
	// Colon-ending only counts if combined with question keywords.
	if strings.HasSuffix(t, ":") && hasKeywords {
		return true
	}
	return hasKeywords
}

// isOpenEndedLine returns true if the question line is open-ended
// (what/which/how/where/who) rather than a yes/no question.
func isOpenEndedLine(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))

	// "X or Y?" → open-ended (choosing between alternatives)
	if orChoicePat.MatchString(lower) {
		return true
	}

	// Open-ended starters
	if openEndedPat.MatchString(lower) {
		return true
	}

	return false
}

// ── Compiled patterns ─────────────────────────────────────────

var (
	ynaPat          = regexp.MustCompile(`(?i)\[y/n/a\]`)
	ynPat           = regexp.MustCompile(`(?i)[\[(]y/n[\])]`)
	allowPat        = regexp.MustCompile(`(?i)(?:allow|permit|approve)\s.*\?`)
	planApprovalPat = regexp.MustCompile(`(?i)(?:shall i proceed|ready to implement|want me to (?:go ahead|start|proceed)|should i (?:start|begin|proceed|go ahead|implement)|do you want me to (?:proceed|continue|implement|start))\s*\??`)
	defaultValuePat = regexp.MustCompile(`\[([^\]]+)\]\s*:\s*$`)
	enterContinuePat = regexp.MustCompile(`(?i)(?:press|hit)\s+enter\b|enter to continue`)
	questionPat      = regexp.MustCompile(`[?？]\s*$`)

	numberedOptPat    = regexp.MustCompile(`^(\d+)[.)]\s+(.+)$`)
	letteredMarkerPat = regexp.MustCompile(`\(([a-zA-Z])\)\s*`)
	bulletOptPat      = regexp.MustCompile(`^[-•*]\s+(.+)$`)
	questionKeywordPat = regexp.MustCompile(`\b(?:which|choose|select|prefer|pick|would you like|options?|approach(?:es)?)\b`)
	openEndedPat      = regexp.MustCompile(`^\s*(?:which|what|how|where|who|describe|name|enter|type|specify|provide)\b`)
	orChoicePat       = regexp.MustCompile(`\b\w+\s+or\s+\w+.*[?？]`)
)
