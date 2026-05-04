package applog

import (
	"sync"
	"testing"
	"time"
)

func TestBuffer_PushAndSnapshot_PreservesOrder(t *testing.T) {
	b := NewBuffer(5)
	for i := 0; i < 3; i++ {
		b.Push(Record{Message: rune2str(rune('a' + i)), Time: time.Now()})
	}
	got := b.Snapshot(0)
	if len(got) != 3 {
		t.Fatalf("got %d, want 3", len(got))
	}
	for i, r := range got {
		if r.Message != rune2str(rune('a'+i)) {
			t.Errorf("index %d: got %q, want %q", i, r.Message, rune2str(rune('a'+i)))
		}
	}
}

func TestBuffer_WrapsWhenFull(t *testing.T) {
	b := NewBuffer(3)
	for i := 0; i < 5; i++ {
		b.Push(Record{Message: rune2str(rune('a' + i))})
	}
	got := b.Snapshot(0)
	if len(got) != 3 {
		t.Fatalf("got %d, want 3", len(got))
	}
	want := []string{"c", "d", "e"}
	for i, r := range got {
		if r.Message != want[i] {
			t.Errorf("index %d: got %q, want %q", i, r.Message, want[i])
		}
	}
}

func TestBuffer_SnapshotN_TrimsToMostRecent(t *testing.T) {
	b := NewBuffer(10)
	for i := 0; i < 8; i++ {
		b.Push(Record{Message: rune2str(rune('a' + i))})
	}
	got := b.Snapshot(3)
	want := []string{"f", "g", "h"}
	if len(got) != 3 {
		t.Fatalf("got %d, want 3", len(got))
	}
	for i, r := range got {
		if r.Message != want[i] {
			t.Errorf("index %d: got %q, want %q", i, r.Message, want[i])
		}
	}
}

func TestBuffer_Subscribe_ReceivesNewRecords(t *testing.T) {
	b := NewBuffer(10)
	ch, unsub := b.Subscribe()
	defer unsub()

	go func() {
		for i := 0; i < 3; i++ {
			b.Push(Record{Message: rune2str(rune('a' + i))})
		}
	}()

	got := make([]string, 0, 3)
	timeout := time.After(time.Second)
	for len(got) < 3 {
		select {
		case r := <-ch:
			got = append(got, r.Message)
		case <-timeout:
			t.Fatalf("timed out, got %v", got)
		}
	}
	if got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("unexpected sequence: %v", got)
	}
}

func TestBuffer_Subscribe_DropsWhenSlow(t *testing.T) {
	// Default subscriber buffer is 64; push more than that without
	// reading and verify the buffer doesn't deadlock.
	b := NewBuffer(200)
	ch, unsub := b.Subscribe()
	defer unsub()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 200; i++ {
			b.Push(Record{Message: "x"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Push blocked on slow subscriber")
	}
	// Drain whatever did make it through.
	drain := 0
	for {
		select {
		case <-ch:
			drain++
		default:
			goto out
		}
	}
out:
	if drain == 0 {
		t.Errorf("expected some records to land in subscriber")
	}
}

func TestBuffer_ConcurrentPushers(t *testing.T) {
	b := NewBuffer(1000)
	var wg sync.WaitGroup
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				b.Push(Record{Message: "x"})
			}
		}()
	}
	wg.Wait()
	got := b.Snapshot(0)
	if len(got) != 1000 {
		t.Errorf("got %d, want 1000 (cap)", len(got))
	}
}

func rune2str(r rune) string { return string([]rune{r}) }
