package projectdoc

import (
	"fmt"
	"strings"
)

// Documents are read and written through the same tools: doc_read hands an
// agent a page, and kb_page_set takes the full body back. That only works
// if what the agent reads is exactly what it may write.
//
// It wasn't. doc_read framed every document with a markdown H1 of its own
// slug plus a "_last updated by_" line, and an agent following the tool's
// own instruction — read the page, fold your change in, write it back —
// wrote the frame into the page. The next read framed the result again.
// Two live pages arrived this way: kb_conventions opens with
// "# kb_conventions", and kb_infrastructure carries that above its real
// title, so every spawn banner showed two H1s.
//
// Two changes close it. The frame is no longer a markdown heading, so it
// neither looks like content nor renders as a title if it does slip
// through; and writes strip any frame they find, which also cleans up the
// pages already carrying one.

// FrameDocForRead renders a document for an agent to read. The header is
// an HTML comment: invisible when rendered, obviously not prose, and it
// says outright that it is not part of the body.
func FrameDocForRead(slug, updatedBy, content string) string {
	if updatedBy == "" {
		updatedBy = "unknown"
	}
	return fmt.Sprintf(
		"<!-- doc_read: %s · last updated by %s · the body below is exactly what a write expects — do not include this line -->\n\n%s",
		slug, updatedBy, content)
}

// StripDocFrame removes any doc_read framing from the top of a body
// an agent is writing back.
//
// Deliberately conservative — it only removes what doc_read itself could
// have produced, at the very top of the document:
//
//   - the "<!-- doc_read: … -->" comment
//   - an H1 that is EXACTLY the slug ("# kb_projects"), or the project-doc
//     variant ("# Project goal") — never any other H1, because a page's own
//     title is real content ("# Home Lab Infrastructure Reference")
//   - a "_last updated by …_" line following either of those
//
// Repeats are handled: a page that round-tripped several times carries
// several frames. Anything further down the document is left alone.
func StripDocFrame(slug, content string) string {
	rest := content
	for {
		trimmed := stripOneWrapperLine(slug, rest)
		if trimmed == rest {
			break
		}
		rest = trimmed
	}
	return rest
}

// stripOneWrapperLine removes a single leading wrapper line (plus the blank
// lines after it) and reports the remainder. Returns its input unchanged
// when the document does not start with one.
func stripOneWrapperLine(slug, content string) string {
	line, remainder := splitFirstLine(content)
	trimmed := strings.TrimSpace(line)

	switch {
	case strings.HasPrefix(trimmed, "<!-- doc_read:") && strings.HasSuffix(trimmed, "-->"):
	case trimmed == "# "+slug:
	case trimmed == "# Project "+slug:
	// Only meaningful directly under a frame line; a body that genuinely
	// opens with italics is not this, because doc_read always emitted the
	// title first — and by the time we see it here, that title is gone.
	case strings.HasPrefix(trimmed, "_last updated by ") && strings.HasSuffix(trimmed, "_"):
	default:
		return content
	}
	return strings.TrimLeft(remainder, "\n")
}

// splitFirstLine returns the first line and everything after it.
func splitFirstLine(s string) (first, rest string) {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}
