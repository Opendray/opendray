package channel

import (
	"context"
	"strings"
	"testing"
)

// TestAuthorizeControl pins the single-owner gate: control commands
// are gated by the authorizer when one is set, read-only commands
// never are, and an unset authorizer allows everything (preserving the
// open default).
func TestAuthorizeControl(t *testing.T) {
	owner := ChannelMessage{Metadata: map[string]any{"tg_user_id": "111"}}
	stranger := ChannelMessage{Metadata: map[string]any{"tg_user_id": "999"}}

	t.Run("nil authorizer allows all", func(t *testing.T) {
		h := NewHub(nil, nil, nil)
		if !h.authorizeControl("end", stranger) {
			t.Error("nil authorizer must allow control commands")
		}
	})

	t.Run("read-only commands are never gated", func(t *testing.T) {
		h := NewHub(nil, nil, nil)
		h.SetControlAuthorizer(func(ChannelMessage) bool { return false })
		for _, name := range []string{"list", "help", "notify", "start"} {
			if !h.authorizeControl(name, stranger) {
				t.Errorf("%s should not be gated", name)
			}
		}
	})

	t.Run("control commands honor the authorizer", func(t *testing.T) {
		h := NewHub(nil, nil, nil)
		h.SetControlAuthorizer(func(m ChannelMessage) bool {
			id, _ := m.Metadata["tg_user_id"].(string)
			return id == "111"
		})
		for _, name := range []string{"end", "resume", "select", "confirm"} {
			if !h.authorizeControl(name, owner) {
				t.Errorf("%s should be allowed for owner", name)
			}
			if h.authorizeControl(name, stranger) {
				t.Errorf("%s should be denied for stranger", name)
			}
		}
	})
}

// TestConfirmCardHandler verifies the destructive-action confirmation:
// the Yes button must carry the real command (/end for stop, /resume
// for restart), so a single tap can't act without a second confirm.
func TestConfirmCardHandler(t *testing.T) {
	mk := func(args ...string) CommandContext {
		return CommandContext{Command: "confirm", Args: args}
	}
	cases := []struct {
		verb, sid, wantCmd string
	}{
		{"stop", "ses_abc", "cmd:/end ses_abc"},
		{"restart", "ses_abc", "cmd:/resume ses_abc"},
	}
	for _, c := range cases {
		card, err := confirmCardHandler(context.Background(), mk(c.verb, c.sid))
		if err != nil {
			t.Fatalf("%s: %v", c.verb, err)
		}
		if !strings.Contains(buttonValues(card), c.wantCmd) {
			t.Errorf("%s: confirm card missing %q; got %s", c.verb, c.wantCmd, buttonValues(card))
		}
		// Cancel must be present and non-destructive.
		if !strings.Contains(buttonValues(card), "cmd:/list") {
			t.Errorf("%s: confirm card missing a Cancel→/list button", c.verb)
		}
	}

	// Unknown verb / missing args degrade to a harmless card, no panic.
	if _, err := confirmCardHandler(context.Background(), mk()); err != nil {
		t.Errorf("empty args should not error: %v", err)
	}
	if _, err := confirmCardHandler(context.Background(), mk("frobnicate", "ses_x")); err != nil {
		t.Errorf("unknown verb should not error: %v", err)
	}
}

func buttonValues(card *Card) string {
	var vals []string
	for _, el := range card.Elements {
		if a, ok := el.(CardActions); ok {
			for _, row := range a.Buttons {
				for _, b := range row {
					vals = append(vals, b.Value)
				}
			}
		}
	}
	return strings.Join(vals, " ")
}
