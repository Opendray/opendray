package projectdoc

import "strings"

// section.go implements read-time markdown heading slicing so an agent can
// pull ONE section of a (potentially large) doc — e.g. a single section of
// the 59K kb_integrations page — instead of swallowing the whole page. Pure
// string work, no DB: callers slice content they already hold.

// headingAt reports the ATX heading level (1-6) and trimmed heading text for
// a line, or (0, "") when the line is not a heading. Requires the hashes to
// start the line and be followed by a space/tab (CommonMark ATX form).
func headingAt(line string) (int, string) {
	n := 0
	for n < len(line) && line[n] == '#' {
		n++
	}
	if n == 0 || n > 6 {
		return 0, ""
	}
	if n >= len(line) || (line[n] != ' ' && line[n] != '\t') {
		return 0, ""
	}
	return n, strings.TrimSpace(line[n:])
}

// normalizeHeading lowercases + collapses whitespace so heading lookups are
// tolerant of casing and spacing differences.
func normalizeHeading(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

type headingSpan struct {
	level int
	text  string
	start int // byte offset of the heading line within content
}

// parseHeadings returns every ATX heading in document order, skipping any
// inside a fenced code block (``` / ~~~) so shell comments / embedded
// markdown in code examples aren't mistaken for headings.
func parseHeadings(content string) []headingSpan {
	var out []headingSpan
	inFence := false
	fenceMarker := "" // the marker that opened the current fence ("```" or "~~~")
	for off := 0; off < len(content); {
		nl := strings.IndexByte(content[off:], '\n')
		var line string
		next := len(content)
		if nl < 0 {
			line = content[off:]
		} else {
			line = content[off : off+nl]
			next = off + nl + 1
		}
		trimmed := strings.TrimSpace(line)
		switch {
		case inFence:
			// A fence closes only on its OWN marker type — a ~~~ line does
			// not close a ``` block (CommonMark).
			if strings.HasPrefix(trimmed, fenceMarker) {
				inFence, fenceMarker = false, ""
			}
		case strings.HasPrefix(trimmed, "```"):
			inFence, fenceMarker = true, "```"
		case strings.HasPrefix(trimmed, "~~~"):
			inFence, fenceMarker = true, "~~~"
		default:
			if lvl, txt := headingAt(line); lvl > 0 {
				out = append(out, headingSpan{level: lvl, text: txt, start: off})
			}
		}
		off = next
	}
	return out
}

// SliceSection returns the markdown section whose heading matches `heading`
// (case/whitespace-insensitive; exact match preferred, else the first heading
// that contains the requested text). The returned slice spans from the
// matched heading line up to — but not including — the next heading of the
// same or higher level, so nested subsections come along. Returns
// (section, true) on a match, ("", false) otherwise. Byte-exact: the result
// is a substring of content, preserving original newlines.
func SliceSection(content, heading string) (string, bool) {
	want := normalizeHeading(heading)
	if want == "" {
		return "", false
	}
	hs := parseHeadings(content)

	match := -1
	for i := range hs {
		if normalizeHeading(hs[i].text) == want {
			match = i
			break
		}
	}
	if match < 0 {
		for i := range hs {
			if strings.Contains(normalizeHeading(hs[i].text), want) {
				match = i
				break
			}
		}
	}
	if match < 0 {
		return "", false
	}

	end := len(content)
	for i := match + 1; i < len(hs); i++ {
		if hs[i].level <= hs[match].level {
			end = hs[i].start
			break
		}
	}
	return content[hs[match].start:end], true
}

// ListSectionHeadings returns the heading text of every section in document
// order (fenced code excluded). Used to build a compact in-prompt index of a
// page's sections so an agent knows what it can pull on demand.
func ListSectionHeadings(content string) []string {
	hs := parseHeadings(content)
	out := make([]string, 0, len(hs))
	for i := range hs {
		out = append(out, hs[i].text)
	}
	return out
}
