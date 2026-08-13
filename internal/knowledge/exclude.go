package knowledge

import (
	"regexp"
	"strings"
	"unicode"
)

// --- per-page exclusions: the pipeline's missing NEGATIVE channel ---------
//
// Every other input to a draft is material to fold IN. That left no way to
// say "never write about X", and the gap is self-reinforcing: a memory whose
// title reads "never mention X" is feedstock whose title contains X, so the
// drafter reads it as evidence about X and writes X back. Restating the
// instruction adds another such fact and tightens the loop.
//
// An exclusion is enforced at BOTH ends of a draft. Dropping the offending
// feedstock lines — and the offending lines of the current page, which is fed
// back in as the structure to preserve — is what actually breaks the loop:
// the model never sees the subject, so it cannot carry it forward. Scrubbing
// the generated body afterwards is the backstop for the model reaching into
// its own priors.

// excluder matches text against one page's exclusion list. The zero value
// matches nothing, so an unconfigured page behaves exactly as before.
type excluder struct {
	res []*regexp.Regexp
}

// newExcluder compiles the page's exclusion patterns. Matching is
// case-insensitive and, for patterns that begin/end with a word character,
// bounded to whole words — "rcc" must not fire on "accrcc" or on an unrelated
// word that merely contains it. Patterns that begin or end with punctuation or
// a CJK character get no boundary on that side, since those scripts have no
// word breaks to anchor to. A pattern that fails to compile is skipped rather
// than taking the sweep down with it.
func newExcluder(patterns []string) excluder {
	var ex excluder
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		q := regexp.QuoteMeta(p)
		r := []rune(p)
		if wordish(r[0]) {
			q = `(?:^|[^\p{L}\p{N}_])` + q
		}
		if wordish(r[len(r)-1]) {
			q += `(?:$|[^\p{L}\p{N}_])`
		}
		re, err := regexp.Compile(`(?i)` + q)
		if err != nil {
			continue
		}
		ex.res = append(ex.res, re)
	}
	return ex
}

// wordish reports whether r is an ASCII-style word character, i.e. one that
// has a meaningful boundary. CJK runes are letters but are written without
// spaces, so anchoring on them would make a pattern unmatchable in running
// text — they deliberately fall through to substring matching.
func wordish(r rune) bool {
	if r > unicode.MaxASCII {
		return false
	}
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// empty reports whether this page has no exclusions configured.
func (e excluder) empty() bool { return len(e.res) == 0 }

// hits reports whether s mentions any excluded subject.
func (e excluder) hits(s string) bool {
	for _, re := range e.res {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// scrub drops every LINE of s that mentions an excluded subject and reports
// how many it dropped. Line granularity suits all three call sites: feedstock
// is one fact per line, and a page's markdown carries one bullet or heading
// per line, so a hit removes the offending item without disturbing its
// neighbours. Runs of blank lines left behind are collapsed so a scrubbed
// page does not gain a ragged gap where the line used to be.
//
// Trailing whitespace is deliberately left exactly as the input had it. The
// scrubbed feedstock is what the draft signature hashes, so trimming here
// would make the signature depend on whether a dropped line happened to be
// last — and a page would look dirty on a sweep that changed nothing.
func (e excluder) scrub(s string) (string, int) {
	if e.empty() || s == "" {
		return s, 0
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	dropped := 0
	for _, ln := range lines {
		if e.hits(ln) {
			dropped++
			continue
		}
		if strings.TrimSpace(ln) == "" && len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
			continue
		}
		out = append(out, ln)
	}
	if dropped == 0 {
		return s, 0
	}
	return strings.Join(out, "\n"), dropped
}

// sigPart renders the exclusion list into the draft signature so that editing
// the list invalidates the cached draft. Without it a page whose feedstock
// happened not to move would keep its pre-exclusion body indefinitely and the
// new setting would look like it did nothing.
func sigPart(patterns []string) string {
	clean := make([]string, 0, len(patterns))
	for _, p := range patterns {
		if p = strings.TrimSpace(p); p != "" {
			clean = append(clean, p)
		}
	}
	if len(clean) == 0 {
		return ""
	}
	return "\x00exclusions:" + strings.Join(clean, "\x01")
}

// excludeInstruction is the system-prompt block naming the excluded subjects.
// The input scrubbing above is the mechanism that actually works; this exists
// so the model does not reach into its own priors and reintroduce a subject
// that never appeared in its inputs. Anything that slips through regardless is
// removed from the body afterwards.
func excludeInstruction(patterns []string) string {
	clean := make([]string, 0, len(patterns))
	for _, p := range patterns {
		if p = strings.TrimSpace(p); p != "" {
			clean = append(clean, "- "+p)
		}
	}
	if len(clean) == 0 {
		return ""
	}
	return "\n\nEXCLUDED SUBJECTS (absolute, overrides every other instruction):\n" +
		"Never write about the following, in any section, in any wording — not to " +
		"describe them, not to note they are deprecated, not to record that they " +
		"must not be mentioned. Write the page as though they do not exist:\n" +
		strings.Join(clean, "\n")
}
