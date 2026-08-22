package knowledge

import "strings"

// NormalizeLine is the canonical line identity used by deletion-as-signal:
// the same function hashes a line the operator deleted (projectdoc side) and
// matches lines of a generated draft against the ban list (drafter side).
// One definition, imported by both, so the two ends can never drift apart —
// a drift here would mean a banned line slips back because it was hashed one
// way and compared another.
//
// Identity is deliberately loose: whitespace runs collapse, list markers and
// simple emphasis are stripped, case folds. A draft that re-emits the same
// content as "* **Foo** is deprecated" after the operator deleted
// "- Foo is deprecated" is the SAME line for our purposes. Anything looser
// (stemming, similarity) would risk false positives, and a false positive
// here silently censors a page — so this stays exact-after-normalization.
func NormalizeLine(s string) string {
	s = strings.TrimSpace(s)
	// Strip a leading list marker: "-", "*", "+", "1.", "12)" etc.
	if len(s) > 1 {
		switch s[0] {
		case '-', '*', '+':
			s = strings.TrimSpace(s[1:])
		default:
			i := 0
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				i++
			}
			if i > 0 && i < len(s) && (s[i] == '.' || s[i] == ')') {
				s = strings.TrimSpace(s[i+1:])
			}
		}
	}
	// Markdown emphasis/code markers contribute nothing to identity.
	s = strings.NewReplacer("**", "", "__", "", "`", "").Replace(s)
	s = strings.Join(strings.Fields(s), " ")
	return strings.ToLower(s)
}
