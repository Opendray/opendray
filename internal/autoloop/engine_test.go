package autoloop

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// ─── fakes ──────────────────────────────────────────────────────────────

type fakeStore struct {
	mu     sync.Mutex
	loops  map[string]*Loop
	runs   map[string][]Run
	nextID int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{loops: map[string]*Loop{}, runs: map[string][]Run{}}
}

func (s *fakeStore) Create(_ context.Context, l Loop) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := l
	s.loops[l.ID] = &cp
	return nil
}
func (s *fakeStore) Get(_ context.Context, id string) (Loop, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.loops[id]
	if !ok {
		return Loop{}, ErrNotFound
	}
	return *l, nil
}
func (s *fakeStore) List(_ context.Context) ([]Loop, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Loop
	for _, l := range s.loops {
		out = append(out, *l)
	}
	return out, nil
}
func (s *fakeStore) ListLive(_ context.Context) ([]Loop, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Loop
	for _, l := range s.loops {
		if l.Status == StatusRunning || l.Status == StatusPaused {
			out = append(out, *l)
		}
	}
	return out, nil
}
func (s *fakeStore) ListRuns(_ context.Context, loopID string) ([]Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Run(nil), s.runs[loopID]...), nil
}
func (s *fakeStore) MarkRunning(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l, ok := s.loops[id]; ok {
		l.Status = StatusRunning
	}
	return nil
}
func (s *fakeStore) SetStatus(_ context.Context, id string, st Status, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l, ok := s.loops[id]; ok {
		l.Status = st
		l.LastReason = reason
	}
	return nil
}
func (s *fakeStore) SaveProgress(_ context.Context, id string, it int, v, r string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l, ok := s.loops[id]; ok {
		l.Iteration = it
		l.LastVerdict = v
		l.LastReason = r
	}
	return nil
}
func (s *fakeStore) AppendRun(_ context.Context, loopID string, it int, prompt string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	id := s.nextID
	s.runs[loopID] = append(s.runs[loopID], Run{ID: id, LoopID: loopID, Iteration: it, Prompt: prompt})
	return id, nil
}
func (s *fakeStore) FinishRun(_ context.Context, runID int64, v, r string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k := range s.runs {
		for i := range s.runs[k] {
			if s.runs[k][i].ID == runID {
				s.runs[k][i].Verdict = v
				s.runs[k][i].Reason = r
			}
		}
	}
	return nil
}
func (s *fakeStore) status(id string) Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l, ok := s.loops[id]; ok {
		return l.Status
	}
	return ""
}
func (s *fakeStore) iteration(id string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l, ok := s.loops[id]; ok {
		return l.Iteration
	}
	return -1
}

// fakeDriver records input and, when armed, fires session.turn_completed on
// the bus as soon as the submit Enter arrives — simulating a CLI settling.
type fakeDriver struct {
	bus       *eventbus.Hub
	output    string
	failInput bool
	silent    bool // armed but never fires turn_completed (timeout test)

	mu    sync.Mutex
	armed map[string]bool
}

func newFakeDriver(bus *eventbus.Hub, output string) *fakeDriver {
	return &fakeDriver{bus: bus, output: output, armed: map[string]bool{}}
}

func (d *fakeDriver) ExpectTurn(id string) {
	d.mu.Lock()
	d.armed[id] = true
	d.mu.Unlock()
}
func (d *fakeDriver) Input(_ context.Context, id string, data []byte) error {
	if d.failInput {
		return errors.New("fake input failure")
	}
	if len(data) == 1 && data[0] == '\r' {
		d.mu.Lock()
		armed := d.armed[id]
		d.armed[id] = false
		out := d.output
		d.mu.Unlock()
		if armed && !d.silent {
			d.bus.Publish(eventbus.Event{
				Topic: "session.turn_completed",
				Data:  map[string]any{"session_id": id, "recent_output": out},
			})
		}
	}
	return nil
}
func (d *fakeDriver) RecentSnippet(string) string { return d.output }

// fakeJudge returns programmed verdicts in order; the last one repeats.
type fakeJudge struct {
	mu       sync.Mutex
	verdicts []Verdict
	calls    int
}

func (j *fakeJudge) Judge(context.Context, Loop, string) (Verdict, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	i := j.calls
	if i >= len(j.verdicts) {
		i = len(j.verdicts) - 1
	}
	j.calls++
	return j.verdicts[i], nil
}

// ─── helpers ────────────────────────────────────────────────────────────

func testEngine(t *testing.T, store Store, driver SessionDriver, judge Judger, bus *eventbus.Hub, opts ...Option) *Engine {
	t.Helper()
	base := []Option{
		WithIntervalUnit(time.Millisecond),
		WithInputDelays(0, 0),
		WithTurnTimeout(2 * time.Second),
	}
	return New(store, driver, judge, bus, nil, append(base, opts...)...)
}

// awaitTerminal blocks until a terminal autoloop.* event for loopID arrives.
func awaitTerminal(t *testing.T, bus *eventbus.Hub, loopID string, timeout time.Duration) Status {
	t.Helper()
	ch, unsub := bus.Subscribe("autoloop.*", 64)
	defer unsub()
	deadline := time.After(timeout)
	for {
		select {
		case ev := <-ch:
			d, _ := ev.Data.(map[string]any)
			if d == nil || d["loop_id"] != loopID {
				continue
			}
			if st := Status(d["status"].(string)); st.IsTerminal() {
				return st
			}
		case <-deadline:
			t.Fatalf("timed out waiting for terminal event for %s", loopID)
			return ""
		}
	}
}

func futureDeadline() *time.Time { d := time.Now().Add(time.Hour); return &d }

// ─── tests ──────────────────────────────────────────────────────────────

func TestIntervalLoopRunsToBudget(t *testing.T) {
	bus := eventbus.New(nil)
	defer bus.Close()
	store := newFakeStore()
	driver := newFakeDriver(bus, "")
	eng := testEngine(t, store, driver, &fakeJudge{}, bus)
	defer eng.Shutdown(context.Background())

	// subscribe before create so we never miss the terminal event.
	done := make(chan Status, 1)
	id := make(chan string, 1)
	go func() { done <- awaitTerminal(t, bus, <-id, 5*time.Second) }()

	l, err := eng.Create(context.Background(), CreateRequest{
		SessionID: "s1", Kind: KindInterval, Prompt: "tick",
		IntervalSeconds: MinIntervalSeconds, MaxIterations: 3, DeadlineAt: futureDeadline(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id <- l.ID

	if st := <-done; st != StatusDone {
		t.Fatalf("status = %q, want done", st)
	}
	if got := store.iteration(l.ID); got != 3 {
		t.Errorf("iteration = %d, want 3", got)
	}
	runs, _ := store.ListRuns(context.Background(), l.ID)
	if len(runs) != 3 {
		t.Errorf("runs = %d, want 3", len(runs))
	}
}

func TestGoalLoopReachesDone(t *testing.T) {
	bus := eventbus.New(nil)
	defer bus.Close()
	store := newFakeStore()
	driver := newFakeDriver(bus, "agent output")
	judge := &fakeJudge{verdicts: []Verdict{
		{Decision: DecisionContinue, NextPrompt: "keep going"},
		{Decision: DecisionContinue, NextPrompt: "almost"},
		{Decision: DecisionDone, Reason: "goal met"},
	}}
	eng := testEngine(t, store, driver, judge, bus)
	defer eng.Shutdown(context.Background())

	done := make(chan Status, 1)
	id := make(chan string, 1)
	go func() { done <- awaitTerminal(t, bus, <-id, 5*time.Second) }()

	l, err := eng.Create(context.Background(), CreateRequest{
		SessionID: "s1", Kind: KindGoal, Prompt: "start", Goal: "do the thing",
		MaxIterations: 10, DeadlineAt: futureDeadline(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id <- l.ID

	if st := <-done; st != StatusDone {
		t.Fatalf("status = %q, want done", st)
	}
	if got := store.iteration(l.ID); got != 3 {
		t.Errorf("iteration = %d, want 3", got)
	}
}

func TestGoalLoopEscalatesOnFailureCap(t *testing.T) {
	bus := eventbus.New(nil)
	defer bus.Close()
	store := newFakeStore()
	driver := newFakeDriver(bus, "err")
	judge := &fakeJudge{verdicts: []Verdict{{Decision: DecisionFail, Reason: "broken"}}}
	eng := testEngine(t, store, driver, judge, bus)
	defer eng.Shutdown(context.Background())

	done := make(chan Status, 1)
	id := make(chan string, 1)
	go func() { done <- awaitTerminal(t, bus, <-id, 5*time.Second) }()

	l, err := eng.Create(context.Background(), CreateRequest{
		SessionID: "s1", Kind: KindGoal, Prompt: "start", Goal: "g",
		MaxIterations: 10, FailureCap: 2, DeadlineAt: futureDeadline(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id <- l.ID

	if st := <-done; st != StatusEscalated {
		t.Fatalf("status = %q, want escalated", st)
	}
	if got := store.iteration(l.ID); got != 2 {
		t.Errorf("iteration = %d, want 2 (failure cap)", got)
	}
}

func TestGoalLoopEscalatesOnTurnTimeout(t *testing.T) {
	bus := eventbus.New(nil)
	defer bus.Close()
	store := newFakeStore()
	driver := newFakeDriver(bus, "")
	driver.silent = true // never fires turn_completed
	judge := &fakeJudge{verdicts: []Verdict{{Decision: DecisionContinue}}}
	eng := testEngine(t, store, driver, judge, bus, WithTurnTimeout(50*time.Millisecond))
	defer eng.Shutdown(context.Background())

	done := make(chan Status, 1)
	id := make(chan string, 1)
	go func() { done <- awaitTerminal(t, bus, <-id, 5*time.Second) }()

	l, err := eng.Create(context.Background(), CreateRequest{
		SessionID: "s1", Kind: KindGoal, Prompt: "start", Goal: "g",
		MaxIterations: 10, FailureCap: 2, DeadlineAt: futureDeadline(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id <- l.ID

	if st := <-done; st != StatusEscalated {
		t.Fatalf("status = %q, want escalated", st)
	}
}

func TestStopHaltsLoop(t *testing.T) {
	bus := eventbus.New(nil)
	defer bus.Close()
	store := newFakeStore()
	driver := newFakeDriver(bus, "")
	// big interval so it won't naturally terminate during the test.
	eng := New(store, driver, &fakeJudge{}, bus, nil,
		WithIntervalUnit(time.Second), WithInputDelays(0, 0))
	defer eng.Shutdown(context.Background())

	l, err := eng.Create(context.Background(), CreateRequest{
		SessionID: "s1", Kind: KindInterval, Prompt: "p",
		IntervalSeconds: 3600, MaxIterations: 100, DeadlineAt: futureDeadline(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := eng.Stop(context.Background(), l.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if st := store.status(l.ID); st != StatusStopped {
		t.Fatalf("status = %q, want stopped", st)
	}
	// stopping a terminal loop is rejected.
	if err := eng.Stop(context.Background(), l.ID); !errors.Is(err, ErrNotRunnable) {
		t.Fatalf("second stop err = %v, want ErrNotRunnable", err)
	}
}

func TestPauseResume(t *testing.T) {
	bus := eventbus.New(nil)
	defer bus.Close()
	store := newFakeStore()
	driver := newFakeDriver(bus, "")
	eng := New(store, driver, &fakeJudge{}, bus, nil,
		WithIntervalUnit(time.Second), WithInputDelays(0, 0))
	defer eng.Shutdown(context.Background())

	l, err := eng.Create(context.Background(), CreateRequest{
		SessionID: "s1", Kind: KindInterval, Prompt: "p",
		IntervalSeconds: 3600, MaxIterations: 100, DeadlineAt: futureDeadline(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := eng.Pause(context.Background(), l.ID); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if st := store.status(l.ID); st != StatusPaused {
		t.Fatalf("status = %q, want paused", st)
	}
	if err := eng.Resume(context.Background(), l.ID); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if st := store.status(l.ID); st != StatusRunning {
		t.Fatalf("status after resume = %q, want running", st)
	}
}

func TestReconcileReArmsRunning(t *testing.T) {
	bus := eventbus.New(nil)
	defer bus.Close()
	store := newFakeStore()
	// a loop persisted as running (as if the gateway restarted mid-flight),
	// at iteration 3 of 3 so it terminates on the first reconciled tick.
	dl := time.Now().Add(time.Hour)
	store.loops["lp_x"] = &Loop{
		ID: "lp_x", SessionID: "s1", Kind: KindInterval, Status: StatusRunning,
		Prompt: "p", IntervalSeconds: 1, MaxIterations: 3, Iteration: 3, DeadlineAt: &dl,
	}
	driver := newFakeDriver(bus, "")
	eng := testEngine(t, store, driver, &fakeJudge{}, bus)
	defer eng.Shutdown(context.Background())

	done := make(chan Status, 1)
	go func() { done <- awaitTerminal(t, bus, "lp_x", 5*time.Second) }()

	if err := eng.ReconcileStartup(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if st := <-done; st != StatusDone {
		t.Fatalf("reconciled loop status = %q, want done", st)
	}
}
