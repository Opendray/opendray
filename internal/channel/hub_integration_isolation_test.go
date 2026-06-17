package channel

import (
	"context"
	"testing"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// Third-party integration sessions are self-managed: opendray must never
// mirror their idle/ended notification cards to operator channels (the
// integration owns its own delivery, e.g. its own Telegram bot). The skip
// happens in dispatch before any store access, so a nil-pool test hub is
// sufficient.

func TestDispatch_SkipsIntegrationSessions(t *testing.T) {
	h := newTestHub(t)
	fc := &fakeChannel{id: "ch_test", kind: "telegram"}
	h.channels[fc.id] = fc

	h.dispatch(context.Background(), eventbus.Event{
		Topic: "session.idle",
		Data: map[string]any{
			"session_id":    "ses_x",
			"origin":        "integration",
			"recent_output": "an agent reply meant for the integration's own bot",
		},
	})

	if got := len(fc.sentTexts()); got != 0 {
		t.Fatalf("integration session must not be mirrored to channels, got %d sends", got)
	}
}

func TestOriginFromEvent(t *testing.T) {
	cases := []struct {
		name string
		data any
		want string
	}{
		{"integration", map[string]any{"origin": "integration"}, "integration"},
		{"operator", map[string]any{"origin": "operator"}, "operator"},
		{"missing origin", map[string]any{"session_id": "x"}, ""},
		{"non-map data", "not-a-map", ""},
		{"nil data", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := originFromEvent(eventbus.Event{Data: tc.data}); got != tc.want {
				t.Fatalf("originFromEvent = %q, want %q", got, tc.want)
			}
		})
	}
}
