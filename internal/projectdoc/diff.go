package projectdoc

import (
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// Reviewing a proposal used to mean reading the whole proposed document
// and spotting by eye what moved — unworkable once a knowledge page runs
// to hundreds of lines. These types carry a line-level diff instead, so
// the UI can show only what changed plus a little surrounding context and
// say how much it collapsed.
//
// The diff is computed here rather than in each client so web and mobile
// render the same review, and adding a client costs no diff algorithm.

// Diff line kinds.
const (
	DiffContext = "context" // unchanged, shown for orientation
	DiffAdd     = "add"
	DiffRemove  = "remove"
)

// DiffLine is one line of a hunk.
type DiffLine struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// DiffHunk is a run of changed lines with its surrounding context.
type DiffHunk struct {
	// SkippedBefore counts the unchanged lines between the previous hunk
	// (or the start of the document) and this one, so the UI can render
	// "N unchanged lines" instead of silently dropping them — silence
	// would leave the reviewer unsure whether anything was hidden.
	SkippedBefore int `json:"skipped_before"`
	// StartLine is this hunk's first line number in the NEW document,
	// 1-based, so the operator can locate the change in the real page.
	StartLine int        `json:"start_line"`
	Lines     []DiffLine `json:"lines"`
}

// DocDiff is the reviewable difference between two document bodies.
type DocDiff struct {
	Hunks     []DiffHunk `json:"hunks"`
	Added     int        `json:"added"`
	Removed   int        `json:"removed"`
	Unchanged bool       `json:"unchanged"`
}

// DefaultDiffContext is how many unchanged lines to keep either side of a
// change — the same default git and GitHub use.
const DefaultDiffContext = 3

// DiffLines computes a line-level diff of two document bodies, keeping
// context unchanged lines around each change.
//
// Line endings and a trailing newline are normalised first: neither is a
// content change the operator needs to review, and leaving them in makes
// a CRLF paste look like every line was rewritten.
func DiffLines(before, after string, context int) DocDiff {
	if context < 0 {
		context = 0
	}
	beforeLines := splitDocLines(before)
	afterLines := splitDocLines(after)

	matcher := difflib.NewMatcher(beforeLines, afterLines)
	groups := matcher.GetGroupedOpCodes(context)

	out := DocDiff{Hunks: []DiffHunk{}}
	prevEnd := 0 // end of the previous group in the BEFORE document
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		hunk := DiffHunk{
			SkippedBefore: group[0].I1 - prevEnd,
			StartLine:     group[0].J1 + 1,
			Lines:         []DiffLine{},
		}
		for _, c := range group {
			switch c.Tag {
			case 'e': // equal
				for _, ln := range beforeLines[c.I1:c.I2] {
					hunk.Lines = append(hunk.Lines, DiffLine{Kind: DiffContext, Text: ln})
				}
			case 'r': // replace — show the old lines then the new ones
				for _, ln := range beforeLines[c.I1:c.I2] {
					hunk.Lines = append(hunk.Lines, DiffLine{Kind: DiffRemove, Text: ln})
					out.Removed++
				}
				for _, ln := range afterLines[c.J1:c.J2] {
					hunk.Lines = append(hunk.Lines, DiffLine{Kind: DiffAdd, Text: ln})
					out.Added++
				}
			case 'd': // delete
				for _, ln := range beforeLines[c.I1:c.I2] {
					hunk.Lines = append(hunk.Lines, DiffLine{Kind: DiffRemove, Text: ln})
					out.Removed++
				}
			case 'i': // insert
				for _, ln := range afterLines[c.J1:c.J2] {
					hunk.Lines = append(hunk.Lines, DiffLine{Kind: DiffAdd, Text: ln})
					out.Added++
				}
			}
			prevEnd = c.I2
		}
		out.Hunks = append(out.Hunks, hunk)
	}
	out.Unchanged = out.Added == 0 && out.Removed == 0
	if out.Unchanged {
		// Identical bodies still produce one all-context group; drop it so
		// callers can test Unchanged alone.
		out.Hunks = []DiffHunk{}
	}
	return out
}

// splitDocLines normalises line endings and drops the trailing empty
// element a final newline produces, so "a\nb" and "a\nb\n" compare equal.
func splitDocLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}
