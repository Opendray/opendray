package roundtable

import (
	"fmt"
	"sort"
	"strings"
)

// Prompt assembly for the two model beats. The chair (synthesize) uses no
// model, so it needs no prompt.

func proposeSystemPrompt() string {
	return `You are ONE expert seat at a multi-model round table. Several other expert models are proposing an approach to the same topic IN PARALLEL — you cannot see their proposals yet, and they cannot see yours. Bring your own model's distinct perspective; do not hedge toward a bland consensus.

Produce a concrete, opinionated proposal: a clear recommended approach, a task breakdown, and the honest tradeoffs of YOUR approach. Set confidence in [0,1] reflecting how sure you are this is the right approach given what you know.

Return STRICT JSON per the schema. Put the full approach in "plan" (markdown ok); keep "summary" to one line.`
}

func proposeUserPrompt(rt RoundTable, enrich string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Topic\n\n%s\n\n", strings.TrimSpace(rt.Topic))
	if strings.TrimSpace(rt.Cwd) != "" {
		fmt.Fprintf(&b, "Project context: this concerns the codebase at `%s`.\n\n", rt.Cwd)
	}
	if enrich != "" {
		b.WriteString("## Possibly relevant prior context (memories / journal)\n\n")
		b.WriteString(truncate(enrich, 6000))
		b.WriteString("\n\n")
	}
	b.WriteString("Propose your approach now.")
	return b.String()
}

func critiqueSystemPrompt() string {
	return `You are ONE expert seat at a multi-model round table. The PROPOSE beat is done; below are the other seats' proposals plus your own. Critique the OTHER seats' proposals — never your own. Be adversarial but fair: surface real risks, not nitpicks dressed as blockers.

Classify each critique:
- "blocker": the approach is wrong or will fail as proposed.
- "concern": a meaningful risk or gap worth addressing.
- "nit": minor / stylistic.

Reference each critique's target by its provider id (the label on each proposal). Return STRICT JSON per the schema. Return an empty "critiques" array if you genuinely have nothing to raise.`
}

func critiqueUserPrompt(rt RoundTable, ownProvider string, own proposal, proposals map[string]proposal) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Topic\n\n%s\n\n", strings.TrimSpace(rt.Topic))

	// Other seats' proposals, in stable provider order for reproducibility.
	others := make([]string, 0, len(proposals))
	for p := range proposals {
		if p != ownProvider {
			others = append(others, p)
		}
	}
	sort.Strings(others)

	b.WriteString("## Other seats' proposals (critique these)\n\n")
	for _, p := range others {
		prop := proposals[p]
		fmt.Fprintf(&b, "### Proposal from `%s`\n\n", p)
		fmt.Fprintf(&b, "Summary: %s\n\n", prop.Summary)
		fmt.Fprintf(&b, "Plan:\n%s\n\n", truncate(prop.Plan, 4000))
		if len(prop.Tradeoffs) > 0 {
			fmt.Fprintf(&b, "Self-declared tradeoffs: %s\n\n", strings.Join(prop.Tradeoffs, "; "))
		}
	}

	b.WriteString("## Your own proposal (for reference — do NOT critique it)\n\n")
	fmt.Fprintf(&b, "Summary: %s\n\n", own.Summary)

	b.WriteString("Critique the other proposals now.")
	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
