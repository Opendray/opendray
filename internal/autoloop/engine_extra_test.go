package autoloop

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

func TestCreateRejectsInvalid(t *testing.T) {
	bus := eventbus.New(nil)
	defer bus.Close()
	eng := testEngine(t, newFakeStore(), newFakeDriver(bus, ""), &fakeJudge{}, bus)
	_, err := eng.Create(context.Background(), CreateRequest{
		SessionID: "", Kind: KindGoal, Prompt: "p", DeadlineAt: futureDeadline(),
	})
	if !errors.Is(err, ErrEmptySession) {
		t.Fatalf("err = %v, want ErrEmptySession", err)
	}
}

func TestGetListRunsPassthrough(t *testing.T) {
	bus := eventbus.New(nil)
	defer bus.Close()
	store := newFakeStore()
	eng := New(store, newFakeDriver(bus, ""), &fakeJudge{}, bus, nil,
		WithIntervalUnit(time.Second), WithInputDelays(0, 0)) // long interval: won't terminate
	defer eng.Shutdown(context.Background())

	l, err := eng.Create(context.Background(), CreateRequest{
		SessionID: "s1", Kind: KindInterval, Prompt: "p",
		IntervalSeconds: 3600, MaxIterations: 100, DeadlineAt: futureDeadline(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := eng.Get(context.Background(), l.ID)
	if err != nil || got.ID != l.ID {
		t.Fatalf("Get = (%+v, %v)", got, err)
	}
	list, err := eng.List(context.Background())
	if err != nil || len(list) != 1 {
		t.Fatalf("List = (%d loops, %v)", len(list), err)
	}
	if _, err := eng.Runs(context.Background(), l.ID); err != nil {
		t.Fatalf("Runs: %v", err)
	}

	// control ops on a missing loop surface ErrNotFound.
	if err := eng.Pause(context.Background(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Pause(missing) = %v, want ErrNotFound", err)
	}
	if err := eng.Resume(context.Background(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Resume(missing) = %v, want ErrNotFound", err)
	}
}

func TestWithClockUsedForDeadline(t *testing.T) {
	bus := eventbus.New(nil)
	defer bus.Close()
	// Clock pinned to 2030; a deadline in 2026 is "past" relative to it.
	fixed := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	eng := New(newFakeStore(), newFakeDriver(bus, ""), &fakeJudge{}, bus, nil,
		WithClock(func() time.Time { return fixed }), WithInputDelays(0, 0))
	dl := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := eng.Create(context.Background(), CreateRequest{
		SessionID: "s1", Kind: KindGoal, Prompt: "p", DeadlineAt: &dl,
	})
	if !errors.Is(err, ErrPastDeadline) {
		t.Fatalf("err = %v, want ErrPastDeadline (clock not applied?)", err)
	}
}

func TestResumeRejectsNonPaused(t *testing.T) {
	bus := eventbus.New(nil)
	defer bus.Close()
	store := newFakeStore()
	eng := New(store, newFakeDriver(bus, ""), &fakeJudge{}, bus, nil,
		WithIntervalUnit(time.Second), WithInputDelays(0, 0))
	defer eng.Shutdown(context.Background())

	l, err := eng.Create(context.Background(), CreateRequest{
		SessionID: "s1", Kind: KindInterval, Prompt: "p",
		IntervalSeconds: 3600, MaxIterations: 100, DeadlineAt: futureDeadline(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// running (not paused) → resume rejected.
	if err := eng.Resume(context.Background(), l.ID); !errors.Is(err, ErrNotRunnable) {
		t.Fatalf("Resume(running) = %v, want ErrNotRunnable", err)
	}
}
