package knowledge

import (
	"fmt"
	"regexp"
	"strings"
)

// draftPlaybook is the shared draft shape for procedural candidates —
// produced today by the experience compiler (and formerly by the
// per-project reflector). The structural gate below judges it.
type draftPlaybook struct {
	Title       string   `json:"title"`
	AppliesWhen string   `json:"applies_when"`
	Steps       []string `json:"steps"`
	Pitfalls    []string `json:"pitfalls"`
	// Evidence: verbatim short quotes from the work log proving this
	// procedure actually happened. Required — a draft without evidence
	// is rejected by the structural gate.
	Evidence []string `json:"evidence"`
}

// concreteArtifactRe spots a step that references something REAL: a
// path, a flag, an inline-code span, a file extension, a host:port,
// or a known tool invocation. Steps made only of prose don't count.
var concreteArtifactRe = regexp.MustCompile(
	"(/[\\w.-]+)|(--[\\w-]+)|(`[^`]+`)|([\\w-]+\\.(sh|go|ts|tsx|dart|md|toml|ya?ml|sql|json|service))|(\\b\\d{2,5}/tcp\\b)|(:\\d{2,5}\\b)|(\\b(ssh|curl|git|go|pnpm|npm|docker|pct|psql|systemctl|launchctl|flutter|make|kubectl)\\b)")

// validateDraftPlaybook is the STRUCTURAL quality gate — the floor every
// candidate must clear regardless of which engine drafted it. The prompt
// asks for quality; this enforces it — a one-liner fact, a greeting,
// or an evidence-free claim never becomes a playbook no matter what
// the model returns. Returns ("", true) when the draft qualifies, or
// (reason, false) for the reject log.
func validateDraftPlaybook(d draftPlaybook) (string, bool) {
	title := strings.TrimSpace(d.Title)
	if len(title) < 12 || !strings.Contains(title, " ") {
		return "title too thin", false
	}
	if len(strings.TrimSpace(d.AppliesWhen)) < 10 {
		return "no usable applies_when trigger", false
	}
	steps := 0
	concrete := 0
	for _, st := range d.Steps {
		st = strings.TrimSpace(st)
		if st == "" {
			continue
		}
		steps++
		if concreteArtifactRe.MatchString(st) {
			concrete++
		}
	}
	if steps < 3 {
		return fmt.Sprintf("only %d steps — not a procedure", steps), false
	}
	if concrete < 2 {
		return "steps lack concrete artifacts (commands/paths/files)", false
	}
	evidence := 0
	for _, e := range d.Evidence {
		if len(strings.TrimSpace(e)) >= 15 {
			evidence++
		}
	}
	if evidence == 0 {
		return "no evidence quotes from the work log", false
	}
	return "", true
}

func renderPlaybookBody(d draftPlaybook) string {
	var b strings.Builder
	if strings.TrimSpace(d.AppliesWhen) != "" {
		b.WriteString("**Applies when:** ")
		b.WriteString(strings.TrimSpace(d.AppliesWhen))
		b.WriteString("\n\n")
	}
	if len(d.Steps) > 0 {
		b.WriteString("## Steps\n")
		for i, s := range d.Steps {
			fmt.Fprintf(&b, "%d. %s\n", i+1, strings.TrimSpace(s))
		}
	}
	if len(d.Pitfalls) > 0 {
		b.WriteString("\n## Pitfalls\n")
		for _, p := range d.Pitfalls {
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(p))
			b.WriteByte('\n')
		}
	}
	if len(d.Evidence) > 0 {
		b.WriteString("\n## Evidence (from the work log)\n")
		for _, e := range d.Evidence {
			b.WriteString("> ")
			b.WriteString(strings.TrimSpace(e))
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String())
}
