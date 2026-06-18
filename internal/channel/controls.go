package channel

import (
	"fmt"
	"strings"
)

// MetaControlKeyboard, when set truthy on an outbound ChannelMessage's
// Metadata, asks a transport that supports a persistent reply keyboard
// (Telegram) to attach the session-control keyboard to that message and
// suppress any per-message inline button row. Transports without a
// persistent keyboard (Slack, Discord, …) ignore the flag and keep
// rendering the card's inline buttons as before.
const MetaControlKeyboard = "control_keyboard"

// ControlButton is one key on the persistent session-control keyboard:
// the label the user sees and taps, and the slash command that tap maps
// to. A "%s" verb in Command is filled with the chat's current session
// id at tap time; such buttons are inert when no session is active.
type ControlButton struct {
	Label   string
	Command string
}

// NeedsSession reports whether this button's command targets a specific
// session (its template contains a "%s" to fill with the current one).
func (b ControlButton) NeedsSession() bool {
	return strings.Contains(b.Command, "%s")
}

// Resolve fills the command template with sid (if needed) and returns
// the concrete slash-command text to dispatch.
func (b ControlButton) Resolve(sid string) string {
	if b.NeedsSession() {
		return fmt.Sprintf(b.Command, sid)
	}
	return b.Command
}

// ControlKeyboardLayout is the row/column layout of the persistent
// keyboard. It is the single source of truth shared by the transport
// (which renders the labels) and the hub (which maps a tapped label
// back to its command) — so the two can never drift.
//
// Stop/Restart/Remove route through /confirm (a fat-fingered tap must
// never interrupt — let alone delete — a live session); Switch opens
// /list to retarget which session the chat talks to; Peek re-sends the
// current session's latest output; Panel opens the /panel home. Peek
// carries no "%s" so it isn't session-templated — the /peek handler
// resolves the chat's active pin itself and reports cleanly when none is
// selected.
//
// Stop interrupts the agent but keeps the row (restartable); Remove is
// the destructive sibling — it stops AND deletes the session permanently.
func ControlKeyboardLayout() [][]ControlButton {
	return [][]ControlButton{
		{
			{Label: "⏸ Stop", Command: "/confirm stop %s"},
			{Label: "🔄 Restart", Command: "/confirm restart %s"},
			{Label: "🗑 Remove", Command: "/confirm remove %s"},
		},
		{
			{Label: "🔀 Switch", Command: "/list"},
			{Label: "👀 Peek", Command: "/peek"},
			{Label: "🎛 Panel", Command: "/panel"},
		},
	}
}

// MatchControlButton returns the button whose label equals text (after
// trimming surrounding whitespace), reporting whether one matched. Used
// by the inbound path to recognise a keyboard tap, which arrives as the
// label's literal text rather than as a slash command.
func MatchControlButton(text string) (ControlButton, bool) {
	t := strings.TrimSpace(text)
	for _, row := range ControlKeyboardLayout() {
		for _, b := range row {
			if b.Label == t {
				return b, true
			}
		}
	}
	return ControlButton{}, false
}
