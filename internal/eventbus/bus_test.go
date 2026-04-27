package eventbus

import (
	"testing"
	"time"
)

func TestPublishSubscribe_Exact(t *testing.T) {
	h := New(nil)
	defer h.Close()

	ch, unsub := h.Subscribe("session.output", 4)
	defer unsub()

	h.Publish(Event{Topic: "session.output", Data: "hello"})
	select {
	case ev := <-ch:
		if ev.Topic != "session.output" {
			t.Errorf("Topic = %q", ev.Topic)
		}
		if ev.Data.(string) != "hello" {
			t.Errorf("Data = %v", ev.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive event")
	}
}

func TestPublishSubscribe_Wildcard(t *testing.T) {
	h := New(nil)
	defer h.Close()

	ch, unsub := h.Subscribe("session.*", 4)
	defer unsub()

	h.Publish(Event{Topic: "session.output"})
	h.Publish(Event{Topic: "session.ended"})
	h.Publish(Event{Topic: "channel.inbound"}) // must not match

	got := drain(ch, 2, 200*time.Millisecond)
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	if got[0].Topic != "session.output" || got[1].Topic != "session.ended" {
		t.Errorf("unexpected order: %v", got)
	}
}

func TestPublish_NonBlockingOnSlowSubscriber(t *testing.T) {
	h := New(nil)
	defer h.Close()

	// Buffer 1; we will publish 5 without reading.
	_, unsub := h.Subscribe("x", 1)
	defer unsub()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 5; i++ {
			h.Publish(Event{Topic: "x"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on slow subscriber")
	}
}

func TestUnsubscribe(t *testing.T) {
	h := New(nil)
	defer h.Close()

	ch, unsub := h.Subscribe("x", 4)
	unsub()

	h.Publish(Event{Topic: "x"})
	// Channel should be closed (unsubscribe closes it).
	if _, ok := <-ch; ok {
		t.Fatal("expected channel to be closed after unsubscribe")
	}
}

func TestClose_ClosesSubscriberChannels(t *testing.T) {
	h := New(nil)
	ch, _ := h.Subscribe("x", 4)
	h.Close()
	if _, ok := <-ch; ok {
		t.Fatal("expected channel to be closed after Hub.Close")
	}
	// Idempotent
	h.Close()
}

func drain(ch <-chan Event, n int, d time.Duration) []Event {
	out := make([]Event, 0, n)
	deadline := time.After(d)
	for len(out) < n {
		select {
		case ev := <-ch:
			out = append(out, ev)
		case <-deadline:
			return out
		}
	}
	return out
}
