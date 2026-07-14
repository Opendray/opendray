package roundtable

import (
	"fmt"
	"strings"
)

// Prompt assembly for the group chat. Each member speaks in character; the
// transcript (built in chat.go) is the user block.

// vendorLabel names the foundation-model family behind a seat provider — the
// diversity the round table exists to surface.
func vendorLabel(provider string) string {
	switch provider {
	case "claude":
		return "Anthropic"
	case "codex":
		return "OpenAI"
	case "antigravity":
		return "Google Gemini"
	}
	return provider
}

// chatSystemPrompt is the persona for one member's reply turn.
func chatSystemPrompt(rt RoundTable, selfProvider string) string {
	var others []string
	for _, seat := range rt.Seats {
		if seat.Provider != selfProvider {
			others = append(others, fmt.Sprintf("%s (%s)", seat.Provider, vendorLabel(seat.Provider)))
		}
	}
	otherList := "none"
	if len(others) > 0 {
		otherList = strings.Join(others, ", ")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "You are %q, the %s model, one member of a multi-model group chat. ",
		selfProvider, vendorLabel(selfProvider))
	fmt.Fprintf(&b, "The other members are: %s. There is also a human Operator who runs the chat.\n\n", otherList)
	b.WriteString("The Operator @mentioned you, so it's your turn to speak. Reply as YOURSELF, in the first person, like a message in a group chat:\n")
	b.WriteString("- Be concise and conversational — a chat message, not an essay or a report.\n")
	b.WriteString("- Bring your own model's genuine perspective; agree, push back, build on what others said, or ask a question.\n")
	b.WriteString("- Do NOT speak for the other members or the Operator. Do NOT prefix your message with your own name.\n")
	b.WriteString("- Do NOT use tools, browse, or read files — just answer from the conversation.")
	return b.String()
}

// summarySystemPrompt asks the chosen member to condense the discussion.
func summarySystemPrompt(rt RoundTable) string {
	return fmt.Sprintf(`You are summarizing a multi-model group chat about %q for the Operator.

Condense the discussion so far into a concrete plan. Use short markdown sections:
- **Recommendation** — the approach the group is converging on.
- **Key tradeoffs** — the important tensions raised.
- **Open questions** — what's still unresolved.
- **Next steps** — a short task breakdown.

Be faithful to what was actually said; note real disagreement rather than papering over it. Do not invent consensus that isn't there.`, strings.TrimSpace(rt.Topic))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
