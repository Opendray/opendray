package session

import (
	"testing"
	"time"
)

// TestCheckTurnComplete pins the turn-detection state machine that
// drives session.turn_completed (and a chat channel's "typing…"
// indicator): a turn fires exactly once, only after output is seen at
// or past the arming instant, and only once the session has been quiet
// for >= threshold.
func TestCheckTurnComplete(t *testing.T) {
	const threshold = 3 * time.Second
	base := time.Now()

	t.Run("not armed never fires", func(t *testing.T) {
		rs := &runningSession{lastActivity: base}
		if rs.checkTurnComplete(base.Add(time.Hour), threshold) {
			t.Fatal("unarmed session should never report a completed turn")
		}
	})

	t.Run("armed but no output yet keeps waiting", func(t *testing.T) {
		rs := &runningSession{lastActivity: base.Add(-time.Minute)}
		rs.arm(base) // armed now; lastActivity is before expectAt
		if rs.checkTurnComplete(base.Add(threshold+time.Second), threshold) {
			t.Fatal("must not fire before any output arrives after arming")
		}
	})

	t.Run("output seen but still chatty does not fire", func(t *testing.T) {
		rs := &runningSession{}
		rs.arm(base)
		rs.markActive(base.Add(time.Second)) // output after arming
		// Only 1s of quiet < 3s threshold.
		if rs.checkTurnComplete(base.Add(2*time.Second), threshold) {
			t.Fatal("must not fire while still within the quiescence window")
		}
	})

	t.Run("output then quiet fires exactly once", func(t *testing.T) {
		rs := &runningSession{}
		rs.arm(base)
		out := base.Add(time.Second)
		rs.markActive(out)
		now := out.Add(threshold)
		if !rs.checkTurnComplete(now, threshold) {
			t.Fatal("should fire once quiet >= threshold after output")
		}
		// Disarmed after firing — a second poll must stay silent until
		// the next ExpectTurn/arm.
		if rs.checkTurnComplete(now.Add(time.Minute), threshold) {
			t.Fatal("must fire only once per arming")
		}
	})

	t.Run("re-arming after a fire allows a second turn", func(t *testing.T) {
		rs := &runningSession{}
		rs.arm(base)
		rs.markActive(base.Add(time.Second))
		if !rs.checkTurnComplete(base.Add(time.Second+threshold), threshold) {
			t.Fatal("first turn should fire")
		}
		later := base.Add(time.Minute)
		rs.arm(later)
		rs.markActive(later.Add(time.Second))
		if !rs.checkTurnComplete(later.Add(time.Second+threshold), threshold) {
			t.Fatal("second turn should fire after re-arming")
		}
	})
}
