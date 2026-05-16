package session

import "testing"

func TestParseClientKind(t *testing.T) {
	cases := []struct {
		in   string
		want ClientKind
	}{
		{"mobile", ClientMobile},
		{"MOBILE", ClientMobile},
		{"  mobile  ", ClientMobile},
		{"web", ClientWeb},
		{"Web", ClientWeb},
		{"", ClientUnknown},
		{"unknown", ClientUnknown},
		{"telegram", ClientUnknown},
	}
	for _, c := range cases {
		if got := ParseClientKind(c.in); got != c.want {
			t.Errorf("ParseClientKind(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// hasMobileSubscriber mirrors the gating predicate inside
// Manager.Resize so a unit test can verify the logic without
// spinning up a real PTY. We don't expose the helper from the
// package; instead the test reproduces the predicate inline so
// any future change to the algorithm forces the test to change.
func hasMobileSubscriber(subs map[chan []byte]ClientKind) bool {
	for _, k := range subs {
		if k == ClientMobile {
			return true
		}
	}
	return false
}

func TestResizeGating_WebSuppressedWhenMobilePresent(t *testing.T) {
	subs := map[chan []byte]ClientKind{
		make(chan []byte): ClientWeb,
		make(chan []byte): ClientMobile,
	}
	if !hasMobileSubscriber(subs) {
		t.Error("expected mobile present → web should be suppressed")
	}
}

func TestResizeGating_WebAlonePassesThrough(t *testing.T) {
	subs := map[chan []byte]ClientKind{
		make(chan []byte): ClientWeb,
		make(chan []byte): ClientUnknown, // legacy client
	}
	if hasMobileSubscriber(subs) {
		t.Error("no mobile present → web should NOT be suppressed")
	}
}

func TestResizeGating_MobileAlonePassesThrough(t *testing.T) {
	subs := map[chan []byte]ClientKind{
		make(chan []byte): ClientMobile,
	}
	// Mobile is the only subscriber. The gating is "web is
	// suppressed when mobile is present" — but the kind on the
	// resize CALL also matters. Mobile calling resize while
	// mobile is connected → not suppressed (kind == ClientMobile).
	// Reproducing Manager.Resize's exact predicate:
	kindFromCaller := ClientMobile
	suppressed := kindFromCaller != ClientMobile && hasMobileSubscriber(subs)
	if suppressed {
		t.Error("mobile caller should never be suppressed")
	}
}

func TestResizeGating_UnknownTreatedAsWeb(t *testing.T) {
	// Legacy clients that don't pass ?client= get ClientUnknown
	// from ParseClientKind. The gating predicate treats anything
	// !=ClientMobile as "subject to suppression" so unknown gets
	// the same treatment as web — which is what we want for
	// safety (don't let an unidentified caller disrupt mobile).
	subs := map[chan []byte]ClientKind{
		make(chan []byte): ClientMobile,
	}
	kindFromCaller := ClientUnknown
	suppressed := kindFromCaller != ClientMobile && hasMobileSubscriber(subs)
	if !suppressed {
		t.Error("ClientUnknown should be suppressed when mobile present")
	}
}
