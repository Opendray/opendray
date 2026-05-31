package cliacct

import (
	"testing"
	"time"
)

func TestThrottleStore_MarkAndQuery(t *testing.T) {
	s := NewThrottleStore()
	if s.IsThrottled("cla_x") {
		t.Error("empty store reports throttled")
	}
	s.MarkThrottled("cla_x", time.Now().Add(time.Hour))
	if !s.IsThrottled("cla_x") {
		t.Error("just-marked account reports not throttled")
	}
}

func TestThrottleStore_EmptyIDIgnored(t *testing.T) {
	s := NewThrottleStore()
	s.MarkThrottled("", time.Now().Add(time.Hour))
	if len(s.ThrottledIDs()) != 0 {
		t.Error("empty id should not be tracked")
	}
}

func TestThrottleStore_ExpiredEntriesGCdOnRead(t *testing.T) {
	s := NewThrottleStore()
	// Mark with an expiry in the past.
	s.MarkThrottled("cla_x", time.Now().Add(-time.Minute))
	if s.IsThrottled("cla_x") {
		t.Error("past-expiry entry should not report throttled")
	}
	// And IsThrottled should have cleaned it up.
	if len(s.ThrottledIDs()) != 0 {
		t.Error("expired entry should be GC'd after IsThrottled read")
	}
}

func TestThrottleStore_NeverShortensThrottle(t *testing.T) {
	// If two events mark the same account with different expirations,
	// the LATER one wins — we want the most pessimistic estimate of
	// when the account will be usable again.
	s := NewThrottleStore()
	later := time.Now().Add(2 * time.Hour)
	s.MarkThrottled("cla_x", later)
	s.MarkThrottled("cla_x", time.Now().Add(30*time.Minute)) // sooner
	until, ok := s.Until("cla_x")
	if !ok {
		t.Fatal("expected throttle entry to still be set")
	}
	if !until.Equal(later) {
		t.Errorf("throttle should not have been shortened; got %v, want %v", until, later)
	}
}

func TestThrottleStore_ExtendsThrottle(t *testing.T) {
	s := NewThrottleStore()
	soon := time.Now().Add(10 * time.Minute)
	s.MarkThrottled("cla_x", soon)
	later := time.Now().Add(2 * time.Hour)
	s.MarkThrottled("cla_x", later)
	until, _ := s.Until("cla_x")
	if !until.Equal(later) {
		t.Errorf("throttle should have been extended; got %v, want %v", until, later)
	}
}

func TestThrottleStore_ThrottledIDsReturnsActiveOnly(t *testing.T) {
	s := NewThrottleStore()
	s.MarkThrottled("active1", time.Now().Add(time.Hour))
	s.MarkThrottled("active2", time.Now().Add(time.Hour))
	s.MarkThrottled("expired", time.Now().Add(-time.Hour))

	ids := s.ThrottledIDs()
	if len(ids) != 2 {
		t.Errorf("expected 2 active throttles, got %d: %v", len(ids), ids)
	}
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	if !set["active1"] || !set["active2"] {
		t.Errorf("expected active1 + active2 in result, got %v", ids)
	}
	if set["expired"] {
		t.Error("expired entry should not be reported")
	}
}

func TestThrottleStore_Clear(t *testing.T) {
	s := NewThrottleStore()
	s.MarkThrottled("cla_x", time.Now().Add(time.Hour))
	s.Clear("cla_x")
	if s.IsThrottled("cla_x") {
		t.Error("cleared entry still reports throttled")
	}
}

func TestThrottleStore_ConcurrentAccess(t *testing.T) {
	// Quick smoke test for race-detector (-race flag).
	s := NewThrottleStore()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			s.MarkThrottled("cla_x", time.Now().Add(time.Hour))
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		_ = s.IsThrottled("cla_x")
		_ = s.ThrottledIDs()
	}
	<-done
}
