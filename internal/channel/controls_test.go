package channel

import (
	"strings"
	"testing"
)

func TestMatchControlButton(t *testing.T) {
	// Every label in the layout must match back to its own button.
	for _, row := range ControlKeyboardLayout() {
		for _, want := range row {
			got, ok := MatchControlButton(want.Label)
			if !ok {
				t.Errorf("label %q did not match any button", want.Label)
				continue
			}
			if got.Command != want.Command {
				t.Errorf("label %q → command %q, want %q", want.Label, got.Command, want.Command)
			}
		}
	}

	// Surrounding whitespace (Telegram sometimes pads) still matches.
	if _, ok := MatchControlButton("  ⏸ Stop  "); !ok {
		t.Error("padded label should still match")
	}

	// A normal chat message must NOT be mistaken for a button.
	if _, ok := MatchControlButton("please stop the build"); ok {
		t.Error("ordinary text wrongly matched a control button")
	}
}

func TestControlButtonResolve(t *testing.T) {
	stop, ok := MatchControlButton("⏸ Stop")
	if !ok {
		t.Fatal("Stop button missing")
	}
	if !stop.NeedsSession() {
		t.Error("Stop must target a session")
	}
	if got, want := stop.Resolve("ses_42"), "/confirm stop ses_42"; got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}

	sw, ok := MatchControlButton("🔀 Switch")
	if !ok {
		t.Fatal("Switch button missing")
	}
	if sw.NeedsSession() {
		t.Error("Switch is session-agnostic")
	}
	if got := sw.Resolve("ses_42"); got != "/list" {
		t.Errorf("Switch.Resolve = %q, want /list", got)
	}
}

func TestControlButtonAction(t *testing.T) {
	h := newTestHub(t)
	const ch = "ch_tg"

	// Ordinary text is not a button → fall through.
	if _, _, _, isBtn := h.controlButtonAction(ChannelMessage{ChannelID: ch, Text: "hello there"}); isBtn {
		t.Error("ordinary text must not be treated as a control button")
	}

	// Session-targeting tap with no active session → hint, no dispatch.
	name, _, hint, isBtn := h.controlButtonAction(ChannelMessage{ChannelID: ch, Text: "⏸ Stop"})
	if !isBtn {
		t.Fatal("Stop should be recognised as a button")
	}
	if hint == "" || name != "" {
		t.Errorf("no active session should yield a hint, got name=%q hint=%q", name, hint)
	}

	// With a current session, Stop resolves to /confirm stop <sid>.
	h.lastSess[ch] = "ses_99"
	name, args, hint, isBtn := h.controlButtonAction(ChannelMessage{ChannelID: ch, Text: "⏸ Stop"})
	if !isBtn || hint != "" {
		t.Fatalf("active session: isBtn=%v hint=%q", isBtn, hint)
	}
	if name != "confirm" || len(args) != 2 || args[0] != "stop" || args[1] != "ses_99" {
		t.Errorf("Stop → name=%q args=%v, want confirm [stop ses_99]", name, args)
	}

	// Session-agnostic tap (Switch) dispatches regardless of session.
	name, _, hint, isBtn = h.controlButtonAction(ChannelMessage{ChannelID: "ch_empty", Text: "🔀 Switch"})
	if !isBtn || hint != "" || name != "list" {
		t.Errorf("Switch → isBtn=%v hint=%q name=%q, want list", isBtn, hint, name)
	}
}

func TestControlCommandsAreParseable(t *testing.T) {
	// Each resolved command must parse as a slash command so handleInbound
	// can dispatch it.
	for _, row := range ControlKeyboardLayout() {
		for _, b := range row {
			cmd := b.Resolve("ses_x")
			name, _, ok := ParseCommand(cmd)
			if !ok {
				t.Errorf("resolved command %q does not parse", cmd)
			}
			if strings.TrimSpace(name) == "" {
				t.Errorf("resolved command %q has empty name", cmd)
			}
		}
	}
}
