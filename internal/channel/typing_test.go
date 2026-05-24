package channel

import (
	"context"
	"strings"
	"testing"
)

// TestAuthorizeSender pins the inbound gate's precedence: a per-channel
// owner allowlist (from the dashboard) wins and is fail-closed; with no
// per-channel owners the global env predicate applies; with neither,
// everything is allowed (open default). The gate covers ALL inbound, so
// plain text bound for a session's stdin is gated like a command.
func TestAuthorizeSender(t *testing.T) {
	owner := ChannelMessage{Metadata: map[string]any{"tg_user_id": "111"}}
	stranger := ChannelMessage{Metadata: map[string]any{"tg_user_id": "999"}}
	anon := ChannelMessage{} // no metadata at all
	none := chatConfig{}     // no per-channel owners

	t.Run("no owners + no env predicate allows all", func(t *testing.T) {
		h := NewHub(nil, nil, nil)
		for _, m := range []ChannelMessage{owner, stranger, anon} {
			if !h.authorizeSender(none, m) {
				t.Error("open default must allow every sender")
			}
		}
	})

	t.Run("env predicate applies when no per-channel owners", func(t *testing.T) {
		h := NewHub(nil, nil, nil)
		h.SetSenderAuthorizer(func(m ChannelMessage) bool {
			id, _ := m.Metadata["tg_user_id"].(string)
			return id == "111"
		})
		if !h.authorizeSender(none, owner) {
			t.Error("owner should be allowed by env predicate")
		}
		if h.authorizeSender(none, stranger) {
			t.Error("stranger should be denied by env predicate")
		}
	})

	t.Run("per-channel owners win and are fail-closed", func(t *testing.T) {
		h := NewHub(nil, nil, nil)
		// Env predicate would allow 111, but the per-channel allowlist
		// (only 222) takes precedence — so 111 is now denied, 222 allowed.
		h.SetSenderAuthorizer(func(m ChannelMessage) bool {
			id, _ := m.Metadata["tg_user_id"].(string)
			return id == "111"
		})
		cc := chatConfig{OwnerUserIDs: "222, 333"}
		if !h.authorizeSender(cc, ChannelMessage{Metadata: map[string]any{"tg_user_id": "222"}}) {
			t.Error("listed owner 222 should be allowed")
		}
		if h.authorizeSender(cc, owner) {
			t.Error("111 not in the per-channel allowlist should be denied (precedence over env)")
		}
		if h.authorizeSender(cc, anon) {
			t.Error("identity-less message should be denied (fail-closed)")
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
