package memory

import (
	"context"
	"errors"
	"testing"
	"time"
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

	// Phase 6 reembed-converge knobs. byEmbedder backs CountByEmbedder;
	// needReembed (must be sorted by ID) backs ListNeedingReembed.
	byEmbedder  map[string]int
	needReembed []Memory
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
	if s.byEmbedder == nil {
		return map[string]int{}, nil
	}
	return s.byEmbedder, nil
}
func (s *fakeStore) ListNeedingReembed(_ context.Context, current string, limit int, afterID string) ([]Memory, error) {
	out := make([]Memory, 0, limit)
	for _, m := range s.needReembed { // assumed sorted by ID
		if m.Embedder == current {
			continue
		}
		if afterID != "" && m.ID <= afterID {
			continue
		}
		out = append(out, m)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
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
func (s *fakeStore) Archive(context.Context, string, string) error               { return nil }
func (s *fakeStore) ArchiveByScope(context.Context, Scope, string, string) (int64, error) {
	return 0, nil
}
func (s *fakeStore) Restore(context.Context, string) error                   { return nil }
func (s *fakeStore) PurgeArchived(context.Context, time.Time) (int64, error) { return 0, nil }
func (s *fakeStore) ArchiveDormantStale(context.Context, Scope, string, time.Time, time.Time, string) (int64, error) {
	return 0, nil
}
func (s *fakeStore) ListArchived(context.Context, Scope, string, int) ([]Memory, error) {
	return nil, nil
}
func (s *fakeStore) Close() error { return nil }

// stubEmbedder is a name-only Embedder for tests that only exercise
// threshold selection (defaultDedupThreshold reads Name()).
type stubEmbedder struct{ name string }

func (e stubEmbedder) Embed(context.Context, []string) ([][]float32, error) {
	return [][]float32{{1}}, nil
}
func (e stubEmbedder) Dimensions() int { return 1 }
func (e stubEmbedder) Name() string    { return e.name }

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

func TestStore_NegativeThresholdDisablesFold(t *testing.T) {
	// A would-be near-duplicate is present, but a negative threshold is
	// the explicit "off" switch, so the write must insert a new row.
	store := &fakeStore{
		hits: []SearchHit{
			{Memory: Memory{ID: "mem_existing", Text: "use pnpm"}, Similarity: 0.99},
		},
	}
	svc := newTestService(t, store, Options{
		DedupThreshold: -1, // explicitly disabled
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
		t.Errorf("expected 1 insert (fold disabled), got %d", len(store.inserted))
	}
	if len(store.updated) != 0 {
		t.Errorf("expected 0 updates, got %d", len(store.updated))
	}
}

func TestDefaultDedupThreshold(t *testing.T) {
	if got := defaultDedupThreshold(NewBM25Embedder(64)); got != 0.2 {
		t.Errorf("BM25 default = %v, want 0.2", got)
	}
	// A non-BM25 (dense) embedder name should get the dense default.
	if got := defaultDedupThreshold(stubEmbedder{name: "http:bge-m3"}); got != 0.85 {
		t.Errorf("dense default = %v, want 0.85", got)
	}
}

// TestStore_UnsetThresholdFoldsByDefault pins the M-U default-on
// behaviour: DedupThreshold 0 resolves to the embedder default (BM25 →
// 0.2), so a sufficiently similar write folds instead of inserting.
func TestStore_UnsetThresholdFoldsByDefault(t *testing.T) {
	store := &fakeStore{
		hits: []SearchHit{
			{Memory: Memory{ID: "mem_existing", Text: "use pnpm"}, Similarity: 0.5},
		},
	}
	svc := newTestService(t, store, Options{DedupThreshold: 0}) // unset → default-on
	id, err := svc.Store(context.Background(), StoreRequest{
		Text: "use pnpm not npm", Scope: ScopeProject, ScopeKey: "/proj/a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "mem_existing" {
		t.Errorf("expected fold into mem_existing (default-on), got %q", id)
	}
	if len(store.inserted) != 0 || len(store.updated) != 1 {
		t.Errorf("expected fold (0 insert, 1 update), got %d insert %d update", len(store.inserted), len(store.updated))
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
	// Fold must be lossless: the superseded text lands in merged_from.
	mf, ok := got.Metadata["merged_from"].([]any)
	if !ok || len(mf) != 1 {
		t.Fatalf("expected merged_from with 1 entry, got %v (%T)", got.Metadata["merged_from"], got.Metadata["merged_from"])
	}
	entry, _ := mf[0].(map[string]any)
	if entry["text"] != "use pnpm" {
		t.Errorf("merged_from should preserve the superseded text %q, got %v", "use pnpm", entry["text"])
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
