package session

import "strings"

// CoalesceRulesArgs merges every "--rules <value>" pair in args into a
// single --rules holding all values joined by a separator. Grok rejects
// a repeated --rules ("cannot be used multiple times"), but several
// independent spawn injections (memory guidance, ambient/project/
// knowledge banners, skill index, integration prompt, carryover) each
// emit their own fragment, and a caller may pass one more in the
// request args. The merged flag takes the position of the first
// --rules.
//
// A --rules whose next token starts with "--" (or with none at all) is
// treated as having no value and is dropped — injector-emitted values
// are always prose, never flag-shaped, and consuming a following flag
// as a value would corrupt the command line. Args with zero or one
// well-formed --rules (and no valueless flag) are returned unchanged,
// so this is a safe no-op for every non-grok provider.
func CoalesceRulesArgs(args []string) []string {
	wellFormed := func(i int) bool {
		return i+1 < len(args) && !strings.HasPrefix(args[i+1], "--")
	}
	var values []string
	valueless := false
	for i := 0; i < len(args); i++ {
		if args[i] != "--rules" {
			continue
		}
		if wellFormed(i) {
			values = append(values, args[i+1])
			i++
		} else {
			valueless = true
		}
	}
	if len(values) <= 1 && !valueless {
		return args
	}
	merged := strings.Join(values, "\n\n---\n\n")
	out := make([]string, 0, len(args))
	placed := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--rules" {
			if wellFormed(i) {
				i++ // consume the value token
			}
			if !placed && len(values) > 0 {
				out = append(out, "--rules", merged)
				placed = true
			}
			continue
		}
		out = append(out, args[i])
	}
	return out
}
