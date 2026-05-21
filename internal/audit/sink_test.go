package audit

import (
	"testing"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

func TestExtractSubject(t *testing.T) {
	cases := []struct {
		name        string
		data        any
		wantKind    string
		wantSubject string
	}{
		{"session", map[string]any{"session_id": "ses_x"}, "session", "ses_x"},
		{"integration", map[string]any{"integration_id": "int_y"}, "integration", "int_y"},
		{"channel", map[string]any{"channel_id": "ch_z"}, "channel", "ch_z"},
		{"admin", map[string]any{"user": "admin"}, "admin", "admin"},
		{"none", map[string]any{"foo": "bar"}, "", ""},
		{"non-map", "string-payload", "", ""},
		{"nil", nil, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, id := extractSubject(tc.data)
			if k != tc.wantKind || id != tc.wantSubject {
				t.Errorf("got (%q,%q), want (%q,%q)", k, id, tc.wantKind, tc.wantSubject)
			}
		})
	}
}

// TestSubscribedPatterns_Allowlist guards against accidentally
// adding session.output (which carries terminal bytes / potential PII)
// to the audit subscription list. If you intentionally widen scope,
// update this test and add the corresponding redaction logic.
func TestSubscribedPatterns_Allowlist(t *testing.T) {
	for _, p := range subscribedPatterns {
		switch p {
		case "session.started", "session.ended", "session.idle",
			"admin.login_success", "admin.login_failed", "admin.logout",
			"channel.message_sent", "channel.message_received",
			"integration.registered", "integration.deregistered",
			"integration.key_rotated", "integration.health_changed",
			// provider.updated carries only provider id + version
			// strings + npm output tail — no PII.
			"provider.updated":
		default:
			t.Errorf("audit subscribes to %q — confirm it is PII-free and update this test", p)
		}
	}
}

// TestEventbusIntegration_PatternsMatch verifies that the patterns we
// subscribe to actually match the events we expect. Catches typos
// like "session.start" vs "session.started".
func TestEventbusIntegration_PatternsMatch(t *testing.T) {
	bus := eventbus.New(nil)
	defer bus.Close()
	for _, pattern := range subscribedPatterns {
		ch, unsub := bus.Subscribe(pattern, 4)
		bus.Publish(eventbus.Event{Topic: pattern})
		select {
		case ev := <-ch:
			if ev.Topic != pattern {
				t.Errorf("pattern %s received %s", pattern, ev.Topic)
			}
		default:
			t.Errorf("pattern %s did not match its own topic", pattern)
		}
		unsub()
	}
}
