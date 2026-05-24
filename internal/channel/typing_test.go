package channel

import (
	"context"
	"strings"
	"testing"
)

// TestAuthorizeSender pins the single-owner inbound gate: with no
// authorizer everything is allowed (open default); with one configured,
// only a matching sender passes and anyone else — including a message
// with no identity metadata — is rejected (fail-closed). The gate
// covers ALL inbound, so plain text bound for a session's stdin is
// rejected just like a command would be.
func TestAuthorizeSender(t *testing.T) {
	owner := ChannelMessage{Metadata: map[string]any{"tg_user_id": "111"}}
	stranger := ChannelMessage{Metadata: map[string]any{"tg_user_id": "999"}}
	anon := ChannelMessage{} // no metadata at all

	t.Run("nil authorizer allows all", func(t *testing.T) {
		h := NewHub(nil, nil, nil)
		for _, m := range []ChannelMessage{owner, stranger, anon} {
			if !h.authorizeSender(m) {
				t.Error("nil authorizer must allow every sender")
			}
		}
	})

	t.Run("configured authorizer is fail-closed", func(t *testing.T) {
		h := NewHub(nil, nil, nil)
		h.SetSenderAuthorizer(func(m ChannelMessage) bool {
			id, _ := m.Metadata["tg_user_id"].(string)
			return id == "111"
		})
		if !h.authorizeSender(owner) {
			t.Error("owner should be allowed")
		}
		if h.authorizeSender(stranger) {
			t.Error("stranger should be denied")
		}
		if h.authorizeSender(anon) {
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
