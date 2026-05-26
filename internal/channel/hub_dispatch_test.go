package channel

import "testing"

// shouldDispatchTopic is the pure topic-filter predicate used inside
// (*Hub).dispatch. The bug fixed alongside issues #222/#223 added the
// NotifyTopicNone sentinel so the UI can express "explicitly muted by
// topic filter" without colliding with the historical "notify_on=[] =
// match all" default. These cases pin every branch.
func TestShouldDispatchTopic(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		topics []string
		topic  string
		want   bool
	}{
		{"nil filter matches all", nil, "session.idle", true},
		{"empty filter matches all", []string{}, "session.idle", true},
		{"empty filter matches ended too", []string{}, "session.ended", true},
		{"sentinel none drops idle", []string{NotifyTopicNone}, "session.idle", false},
		{"sentinel none drops ended", []string{NotifyTopicNone}, "session.ended", false},
		{"sentinel none drops started", []string{NotifyTopicNone}, "session.started", false},
		{"single topic match", []string{"session.idle"}, "session.idle", true},
		{"single topic miss", []string{"session.idle"}, "session.ended", false},
		{"multi topic match first", []string{"session.idle", "session.ended"}, "session.idle", true},
		{"multi topic match last", []string{"session.idle", "session.ended"}, "session.ended", true},
		{"multi topic miss", []string{"session.idle", "session.ended"}, "session.started", false},
		{"sentinel mixed with real topic stays a match", []string{NotifyTopicNone, "session.idle"}, "session.idle", true},
		{"sentinel mixed treats sentinel as ordinary string when not sole entry", []string{NotifyTopicNone, "session.idle"}, "session.ended", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldDispatchTopic(tc.topics, tc.topic); got != tc.want {
				t.Errorf("shouldDispatchTopic(%v, %q) = %v, want %v", tc.topics, tc.topic, got, tc.want)
			}
		})
	}
}
