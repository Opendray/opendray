// Package telegram — markdown → Telegram-flavoured HTML converter.
//
// Telegram supports a small HTML subset (<b>, <i>, <u>, <s>, <code>,
// <pre>, <a href>, <blockquote>) but no tables, no nested lists, no
// CSS. To mirror what Claude prints in its TUI we:
//
//   - Parse fenced code blocks (```), wrap them in <pre> with HTML
//     escaping inside.
//   - Convert Markdown tables to a vertical "Header: value" layout —
//     mobile-readable, no broken pipe characters.
//   - Convert # / ## / ### headings to <b>.
//   - Convert -/* bullets to "  • " lines.
//   - inline: **bold** → <b>, `code` → <code>, *italic* → <i>.
//   - Escape any stray <, >, & in plain text.
//
// Ported from opendray v1's gateway/telegram/forwarder.go.
package telegram

import (
	"regexp"
	"strings"
)

// formatForTelegram converts Markdown text to the small HTML subset
// Telegram supports. Output is safe to send with parse_mode=HTML.
func formatForTelegram(s string) string {
	lines := strings.Split(s, "\n")
	var out strings.Builder
	inCodeBlock := false

	var tableBuf []string
	flushTable := func() {
		if len(tableBuf) == 0 {
			return
		}
		out.WriteString(renderTable(tableBuf))
		tableBuf = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Fenced code block toggle.
		if strings.HasPrefix(trimmed, "```") {
			flushTable()
			if inCodeBlock {
				out.WriteString("</pre>\n")
				inCodeBlock = false
			} else {
				out.WriteString("<pre>")
				inCodeBlock = true
			}
			continue
		}
		if inCodeBlock {
			out.WriteString(escapeHTML(line))
			out.WriteString("\n")
			continue
		}

		// Markdown table run.
		if isTableRow(trimmed) {
			tableBuf = append(tableBuf, trimmed)
			continue
		}
		flushTable()

		if trimmed == "" {
			out.WriteString("\n")
			continue
		}

		// Standalone separator runs (---, ___, ===) — drop. Telegram
		// shows them as literal punctuation rather than a horizontal
		// rule, which is uglier than just removing them.
		if isSeparatorRunLine(trimmed) {
			continue
		}

		// Markdown headers (any level).
		if len(trimmed) > 1 && trimmed[0] == '#' {
			header := strings.TrimLeft(trimmed, "#")
			header = strings.TrimSpace(header)
			if header != "" {
				out.WriteString("\n<b>")
				out.WriteString(escapeHTML(header))
				out.WriteString("</b>\n")
				continue
			}
		}

		// Block quote.
		if strings.HasPrefix(trimmed, "> ") {
			out.WriteString("<blockquote>")
			out.WriteString(inlineMarkdown(trimmed[2:]))
			out.WriteString("</blockquote>\n")
			continue
		}

		// Bullets.
		if strings.HasPrefix(trimmed, "- ") {
			out.WriteString("  • ")
			out.WriteString(inlineMarkdown(trimmed[2:]))
			out.WriteString("\n")
			continue
		}
		if strings.HasPrefix(trimmed, "* ") && !strings.HasPrefix(trimmed, "**") {
			out.WriteString("  • ")
			out.WriteString(inlineMarkdown(trimmed[2:]))
			out.WriteString("\n")
			continue
		}
		if strings.HasPrefix(trimmed, "• ") {
			out.WriteString("  • ")
			out.WriteString(inlineMarkdown(trimmed[2:]))
			out.WriteString("\n")
			continue
		}

		// Lines starting with our own bullet/check sigils — make
		// them stand out so the assistant's "● " bullets render bold.
		if r, ok := firstRune(trimmed); ok {
			switch r {
			case '●', '⚡', '✅', '✗', '🔧', '└':
				out.WriteString("<b>")
				out.WriteString(escapeHTML(string(r)))
				out.WriteString("</b>")
				rest := strings.TrimLeft(trimmed[len(string(r)):], " ")
				if rest != "" {
					out.WriteString(" ")
					out.WriteString(inlineMarkdown(rest))
				}
				out.WriteString("\n")
				continue
			}
		}

		out.WriteString(inlineMarkdown(trimmed))
		out.WriteString("\n")
	}

	flushTable()
	if inCodeBlock {
		out.WriteString("</pre>")
	}
	// Trim only leading/trailing newlines — preserve indentation
	// inside lines (e.g. the leading two spaces on "  • bullet").
	return strings.Trim(out.String(), "\n")
}

// isTableRow reports whether `trimmed` is a markdown table row
// (starts with '|' and has ≥3 pipes total = ≥2 columns).
func isTableRow(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "|") {
		return false
	}
	return strings.Count(trimmed, "|") >= 3
}

// splitTableRow strips outer pipes, splits on '|', trims each cell.
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

// isSeparatorRow returns true when every cell is composed of only
// '-', ':' and whitespace (the markdown table alignment separator).
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
				// ok
			default:
				return false
			}
		}
	}
	return sawDash
}

// renderTable converts buffered table rows into a "Header: value"
// vertical layout, separated by blank lines between data rows.
// Falls back to bullet rows when the table is malformed.
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
	if sepIdx < 1 || sepIdx >= len(parsed)-1 {
		var b strings.Builder
		for _, cells := range parsed {
			if isSeparatorRow(cells) {
				continue
			}
			b.WriteString("  • ")
			b.WriteString(inlineMarkdown(strings.Join(cells, " │ ")))
			b.WriteString("\n")
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

// inlineMarkdown converts the inline subset `**bold**`, `*italic*`,
// “ `code` “ to HTML tags. Other characters are HTML-escaped.
func inlineMarkdown(s string) string {
	s = escapeHTML(s)
	s = boldRe.ReplaceAllString(s, "<b>$1</b>")
	s = italicRe.ReplaceAllString(s, "${pre}<i>${body}</i>${post}")
	s = codeRe.ReplaceAllString(s, "<code>$1</code>")
	return s
}

var (
	boldRe   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe = regexp.MustCompile(`(?P<pre>^|[^*])\*(?P<body>[^*]+?)\*(?P<post>[^*]|$)`)
	codeRe   = regexp.MustCompile("`([^`]+)`")
)

// escapeHTML escapes the characters Telegram's HTML parser cares
// about (&, <, >). Unicode is left alone — Telegram is UTF-8 native.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// isSeparatorRunLine reports whether a line is just dashes /
// underscores / equals signs / tildes — Markdown horizontal rule
// candidates that Telegram cannot render. Length ≥ 3.
func isSeparatorRunLine(s string) bool {
	if len([]rune(s)) < 3 {
		return false
	}
	for _, r := range s {
		switch r {
		case '-', '_', '=', '~', ' ', '\t':
			continue
		default:
			return false
		}
	}
	return true
}

func firstRune(s string) (rune, bool) {
	for _, r := range s {
		return r, true
	}
	return 0, false
}

// rebalanceHTMLChunks ensures each chunk in `chunks` is independently
// valid HTML by tracking whether `<pre>` is open at chunk-end. When
// open, we append `</pre>` to the current chunk and prepend `<pre>`
// to the next so neither chunk has a dangling opener.
//
// Other inline tags (`<b>`, `<i>`, `<code>`) come from inlineMarkdown
// which always emits balanced pairs on a single line, so a line-
// boundary chunk split won't break them.
func rebalanceHTMLChunks(chunks []string) []string {
	if len(chunks) == 0 {
		return chunks
	}
	out := make([]string, len(chunks))
	preOpen := false
	for i, c := range chunks {
		opens := strings.Count(c, "<pre>")
		closes := strings.Count(c, "</pre>")
		buf := strings.Builder{}
		if preOpen {
			buf.WriteString("<pre>")
		}
		buf.WriteString(c)
		// New balance after this chunk's own tags.
		newBalance := preOpen
		if opens > closes {
			newBalance = true
		} else if closes > opens {
			newBalance = false
		}
		if newBalance {
			buf.WriteString("</pre>")
			preOpen = true
		} else {
			preOpen = false
		}
		out[i] = buf.String()
	}
	return out
}
