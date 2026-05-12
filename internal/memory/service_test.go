package memory

import (
	"context"
	"errors"
	"testing"
)

// fakeStore is a minimal in-memory Store for service unit tests.
// Only the methods Service.Store / Service.Search touch are
// non-trivial; the rest panic so we catch accidental use.
type fakeStore struct {
	inserted []InsertRequest
	updated  []UpdateRequest
	hits     []SearchHit // returned verbatim from Search
	memByID  map[string]Memory

	insertErr error
	updateErr error
	searchErr error
}

func (s *fakeStore) Insert(_ context.Context, req InsertRequest) (string, error) {
	if s.insertErr != nil {
		return "", s.insertErr
	}
	id := "mem_" + req.Text
	s.inserted = append(s.inserted, req)
	if s.memByID == nil {
		s.memByID = map[string]Memory{}
	}
	s.memByID[id] = Memory{
		ID: id, Scope: req.Scope, ScopeKey: req.ScopeKey,
		Text: req.Text, Embedder: req.Embedder, Metadata: req.Metadata,
	}
	return id, nil
}

func (s *fakeStore) Update(_ context.Context, req UpdateRequest) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, req)
	if m, ok := s.memByID[req.ID]; ok {
		m.Text = req.Text
		m.Metadata = req.Metadata
		s.memByID[req.ID] = m
	}
	return nil
}

func (s *fakeStore) Search(_ context.Context, _ SearchQuery) ([]SearchHit, error) {
	if s.searchErr != nil {
		return nil, s.searchErr
	}
	return s.hits, nil
}

// Unimplemented surface — kept as panics so a test using these by
// accident fails loudly instead of silently mis-behaving.
func (s *fakeStore) List(context.Context, Scope, string, int) ([]Memory, error) {
	panic("List not used in these tests")
}
func (s *fakeStore) ListScopeKeys(context.Context, Scope) ([]string, error) {
	panic("ListScopeKeys not used")
}
func (s *fakeStore) CountByEmbedder(context.Context) (map[string]int, error) {
	panic("CountByEmbedder not used")
}
func (s *fakeStore) ListNeedingReembed(context.Context, string, int, string) ([]Memory, error) {
	panic("ListNeedingReembed not used")
}
func (s *fakeStore) RecordHits(context.Context, []string) error { return nil }
func (s *fakeStore) Get(_ context.Context, id string) (Memory, error) {
	if m, ok := s.memByID[id]; ok {
		return m, nil
	}
	return Memory{}, errors.New("not found")
}
func (s *fakeStore) Delete(context.Context, string) error                        { return nil }
func (s *fakeStore) DeleteByScope(context.Context, Scope, string) (int64, error) { return 0, nil }
func (s *fakeStore) Close() error                                                { return nil }

func newTestService(t *testing.T, store Store, opts Options) *Service {
	t.Helper()
	opts.Embedder = NewBM25Embedder(64)
	opts.Store = store
	svc, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestStore_NoDedupWhenDisabled(t *testing.T) {
	store := &fakeStore{}
	svc := newTestService(t, store, Options{
		DedupThreshold: 0, // off
	})
	id, err := svc.Store(context.Background(), StoreRequest{
		Text: "use pnpm not npm", Scope: ScopeProject, ScopeKey: "/proj/a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Errorf("empty id")
	}
	if len(store.inserted) != 1 {
		t.Errorf("expected 1 insert, got %d", len(store.inserted))
	}
	if len(store.updated) != 0 {
		t.Errorf("expected 0 updates, got %d", len(store.updated))
	}
}

func TestStore_DedupMergesWhenSimilarEnough(t *testing.T) {
	store := &fakeStore{
		hits: []SearchHit{
			{
				Memory:     Memory{ID: "mem_existing", Text: "use pnpm", Metadata: map[string]any{"type": "user_preference"}},
				Similarity: 0.92,
			},
		},
	}
	svc := newTestService(t, store, Options{
		DedupThreshold: 0.85,
	})
	id, err := svc.Store(context.Background(), StoreRequest{
		Text: "use pnpm not npm", Scope: ScopeProject, ScopeKey: "/proj/a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "mem_existing" {
		t.Errorf("expected merge into mem_existing, got %q", id)
	}
	if len(store.inserted) != 0 {
		t.Errorf("expected 0 inserts (dedup hit), got %d", len(store.inserted))
	}
	if len(store.updated) != 1 {
		t.Fatalf("expected 1 update, got %d", len(store.updated))
	}
	got := store.updated[0]
	if got.ID != "mem_existing" {
		t.Errorf("update targeting wrong id: %q", got.ID)
	}
	if got.Text != "use pnpm not npm" {
		t.Errorf("update should carry new text, got %q", got.Text)
	}
	if got.Metadata["type"] != "user_preference" {
		t.Errorf("update should preserve existing type, got %v", got.Metadata["type"])
	}
	if dc, ok := got.Metadata["deduped_count"].(int); !ok || dc != 1 {
		t.Errorf("expected deduped_count=1, got %v (%T)", got.Metadata["deduped_count"], got.Metadata["deduped_count"])
	}
}

func TestStore_DedupSkipsWhenBelowThreshold(t *testing.T) {
	store := &fakeStore{
		hits: []SearchHit{
			{Memory: Memory{ID: "mem_existing", Text: "node 20 is fine"}, Similarity: 0.55},
		},
	}
	svc := newTestService(t, store, Options{DedupThreshold: 0.85})
	_, err := svc.Store(context.Background(), StoreRequest{
		Text: "use pnpm not npm", Scope: ScopeProject, ScopeKey: "/proj/a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.inserted) != 1 {
		t.Errorf("expected 1 insert (below threshold = new row), got %d", len(store.inserted))
	}
	if len(store.updated) != 0 {
		t.Errorf("expected 0 updates, got %d", len(store.updated))
	}
}

func TestStore_DedupSearchErrorFallsThroughToInsert(t *testing.T) {
	store := &fakeStore{searchErr: errors.New("pgvector exploded")}
	svc := newTestService(t, store, Options{DedupThreshold: 0.85})
	_, err := svc.Store(context.Background(), StoreRequest{
		Text: "use pnpm not npm", Scope: ScopeProject, ScopeKey: "/proj/a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.inserted) != 1 {
		t.Errorf("search error should degrade to insert, got %d inserts", len(store.inserted))
	}
}

// Gatekeeper tests (M12).

type fakeGatekeeper struct {
	durable  bool
	category string
	reason   string
	err      error
	calls    []string
}

func (g *fakeGatekeeper) Judge(_ context.Context, text string) (bool, string, string, error) {
	g.calls = append(g.calls, text)
	return g.durable, g.category, g.reason, g.err
}

func TestStore_GatekeeperRejects(t *testing.T) {
	store := &fakeStore{}
	gk := &fakeGatekeeper{durable: false, reason: "looks like ephemeral state"}
	svc := newTestService(t, store, Options{Gatekeeper: gk})
	_, err := svc.Store(context.Background(), StoreRequest{
		Text: "currently editing app.go line 412", Scope: ScopeProject, ScopeKey: "/proj/a",
	})
	if !errors.Is(err, ErrNotDurable) {
		t.Errorf("expected ErrNotDurable, got %v", err)
	}
	if len(store.inserted) != 0 {
		t.Errorf("expected 0 inserts on rejection, got %d", len(store.inserted))
	}
	if len(gk.calls) != 1 {
		t.Errorf("gatekeeper should have been called once, got %d", len(gk.calls))
	}
}

func TestStore_GatekeeperAllowsAndTagsCategory(t *testing.T) {
	store := &fakeStore{}
	gk := &fakeGatekeeper{durable: true, category: "user_preference"}
	svc := newTestService(t, store, Options{Gatekeeper: gk})
	id, err := svc.Store(context.Background(), StoreRequest{
		Text: "use pnpm not npm", Scope: ScopeProject, ScopeKey: "/proj/a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Errorf("empty id")
	}
	if len(store.inserted) != 1 {
		t.Fatalf("expected 1 insert, got %d", len(store.inserted))
	}
	meta := store.inserted[0].Metadata
	if meta["type"] != "user_preference" {
		t.Errorf("expected auto-tagged type=user_preference, got %v", meta)
	}
}

func TestStore_GatekeeperRespectsCallerCategory(t *testing.T) {
	store := &fakeStore{}
	gk := &fakeGatekeeper{durable: true, category: "user_preference"}
	svc := newTestService(t, store, Options{Gatekeeper: gk})
	_, err := svc.Store(context.Background(), StoreRequest{
		Text:     "use pnpm not npm",
		Scope:    ScopeProject,
		ScopeKey: "/proj/a",
		Metadata: map[string]any{"type": "feedback"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := store.inserted[0].Metadata["type"]; got != "feedback" {
		t.Errorf("caller's type should win, got %v", got)
	}
}

func TestStore_GatekeeperErrorDegradesToAllow(t *testing.T) {
	store := &fakeStore{}
	gk := &fakeGatekeeper{err: errors.New("LM Studio timed out")}
	svc := newTestService(t, store, Options{Gatekeeper: gk})
	_, err := svc.Store(context.Background(), StoreRequest{
		Text: "x", Scope: ScopeProject, ScopeKey: "/proj/a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.inserted) != 1 {
		t.Errorf("gatekeeper error should degrade to allow, got %d inserts", len(store.inserted))
	}
}

func TestDedupedCount_HandlesJSONFloatRoundtrip(t *testing.T) {
	cases := []struct {
		in   map[string]any
		want int
	}{
		{nil, 0},
		{map[string]any{}, 0},
		{map[string]any{"deduped_count": 3}, 3},
		{map[string]any{"deduped_count": int64(5)}, 5},
		{map[string]any{"deduped_count": 7.0}, 7},
		{map[string]any{"deduped_count": "hello"}, 0},
	}
	for _, tc := range cases {
		if got := dedupedCount(tc.in); got != tc.want {
			t.Errorf("dedupedCount(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
