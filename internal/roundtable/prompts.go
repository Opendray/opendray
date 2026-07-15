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
	case "grok":
		return "xAI Grok"
	case "opencode":
		// Provider-agnostic — the actual model family depends on the seat's
		// configured provider/model, so name the CLI itself.
		return "OpenCode"
	}
	return provider
}

// chatSystemPrompt is the persona for one member's reply turn. self is the
// seat about to speak (its Persona, if set, becomes an extra role instruction).
func chatSystemPrompt(rt RoundTable, self Seat) string {
	selfProvider := self.Provider
	var others []string
	for _, seat := range rt.Seats {
		if seat.Provider != selfProvider {
			others = append(others, fmt.Sprintf("@%s (%s)", seat.Provider, vendorLabel(seat.Provider)))
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
	if fr := strings.TrimSpace(rt.Framing); fr != "" {
		// Table-level framing: the shared setup everyone works under — the
		// current topic and how the members relate (who leads, who reviews).
		// Comes first so each member reads the room the same way.
		fmt.Fprintf(&b, "DISCUSSION FRAMING (applies to everyone): %s\n\n", fr)
	}
	if self.Persona != "" {
		// The persona is the PRIMARY lens for this member — placed up front and
		// framed as binding so replies actually differ by role (the whole point
		// of assigning one). A weak "stay in role" line gets drowned by the
		// conversational instructions below, so make it directive.
		fmt.Fprintf(&b, "YOUR ROLE IN THIS DISCUSSION: %s\n", self.Persona)
		b.WriteString("This role is your mandate, not a label. Evaluate everything through it: ")
		b.WriteString("champion the concerns this role cares about, challenge points that ignore them, ")
		b.WriteString("and let it shape what you praise, question, and push back on. ")
		b.WriteString("Do not drift into a generic assistant — if this role would object, object.\n\n")
	}
	b.WriteString("You were @mentioned, so it's your turn to speak. Reply as YOURSELF, in the first person, like a message in a group chat:\n")
	b.WriteString("- Be concise and conversational — a chat message, not an essay or a report.\n")
	if self.Persona != "" {
		b.WriteString("- Speak from your role above; take a clear stance rather than hedging to consensus.\n")
	} else {
		b.WriteString("- Bring your own model's genuine perspective; agree, push back, build on what others said, or ask a question.\n")
	}
	b.WriteString("- To bring another member into the discussion, @mention them (e.g. " + otherList + "); they'll get a turn to reply.\n")
	b.WriteString("- Do NOT speak for the other members or the Operator. Do NOT prefix your message with your own name.\n")
	b.WriteString("- Do NOT use tools, browse, or read files — answer from the conversation and any context provided.")
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

// planSystemPrompt asks the drafter to turn the discussion into an ordered,
// role-assigned execution plan — each step handed to the member whose strength
// fits it, so the team can then implement it step by step in a shared project.
func planSystemPrompt(rt RoundTable) string {
	var members []string
	for _, seat := range rt.Seats {
		line := fmt.Sprintf("- %s (%s)", seat.Provider, vendorLabel(seat.Provider))
		if seat.Persona != "" {
			line += ": " + seat.Persona
		}
		members = append(members, line)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "You are turning a multi-model group chat about %q into a concrete, ordered work plan the team will execute.\n\n",
		strings.TrimSpace(rt.Topic))
	b.WriteString("The team members and their strengths:\n")
	b.WriteString(strings.Join(members, "\n"))
	b.WriteString("\n\n")
	if strings.TrimSpace(rt.Framing) != "" {
		fmt.Fprintf(&b, "Team framing: %s\n\n", strings.TrimSpace(rt.Framing))
	}
	b.WriteString("Break the agreed work into a short ordered list of steps. Assign each step to the ONE member whose strength best fits it (e.g. code → the coder, UI/visual design → the design member, review → the reviewer). Order them so each step builds on the previous one (write before review, design assets before wiring them in).\n\n")
	b.WriteString("Reply with ONLY a JSON array, no prose, in this exact shape:\n")
	b.WriteString("[{\"assignee\": \"<member provider id>\", \"task\": \"<what they should do, one or two sentences>\"}]\n")
	b.WriteString("Use only the member provider ids listed above for \"assignee\". Keep it to the steps that are actually needed.")
	return b.String()
}

// truncate caps s to max characters, slicing on a RUNE boundary. Byte
// slicing (s[:max]) can cut a multi-byte UTF-8 character (Chinese, emoji)
// in half, and the resulting invalid UTF-8 makes codex reject its stdin
// prompt outright ("input is not valid UTF-8 (invalid byte at offset N)").
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
