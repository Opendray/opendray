package memory

import (
	"context"
	"testing"
	"time"
)

// batchEmbedder returns one vector per input (unlike stubEmbedder,
// which always returns a single vector) so Reembed's batch indexing
// is exercised honestly.
type batchEmbedder struct{ name string }

func (e batchEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{1, 0}
	}
	return out, nil
}
func (e batchEmbedder) Dimensions() int { return 2 }
func (e batchEmbedder) Name() string    { return e.name }

func newConvergeService(t *testing.T, store Store, emb Embedder) *Service {
	t.Helper()
	svc, err := New(Options{Embedder: emb, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestDriftCount(t *testing.T) {
	store := &fakeStore{byEmbedder: map[string]int{
		"http:bge-m3": 5, // current — not drift
		"bm25":        3, // stale
		"http:old":    2, // stale
	}}
	svc := newConvergeService(t, store, stubEmbedder{name: "http:bge-m3"})

	got, err := svc.driftCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != 5 {
		t.Fatalf("driftCount = %d, want 5 (3 bm25 + 2 http:old)", got)
	}
}

func TestDriftCountConverged(t *testing.T) {
	store := &fakeStore{byEmbedder: map[string]int{"bm25": 7}}
	svc := newConvergeService(t, store, stubEmbedder{name: "bm25"})
	got, err := svc.driftCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("driftCount = %d, want 0 (all rows on current embedder)", got)
	}
}

// Reembed must drain every drifted row, re-stamping it with the
// current embedder, and advance past the cursor so it terminates.
func TestReembedDrainsDriftedRows(t *testing.T) {
	store := &fakeStore{
		needReembed: []Memory{
			{ID: "mem_a", Text: "alpha", Embedder: "bm25"},
			{ID: "mem_b", Text: "beta", Embedder: "bm25"},
			{ID: "mem_c", Text: "gamma", Embedder: "http:old"},
		},
	}
	svc := newConvergeService(t, store, batchEmbedder{name: "http:new"})

	report, err := svc.Reembed(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if report.Reembed != 3 {
		t.Fatalf("reembed = %d, want 3", report.Reembed)
	}
	if len(store.updated) != 3 {
		t.Fatalf("updates = %d, want 3", len(store.updated))
	}
	for _, u := range store.updated {
		if u.Embedder != "http:new" {
			t.Fatalf("row %s updated with embedder %q, want http:new", u.ID, u.Embedder)
		}
	}
}

// RunReembedConverge must return promptly when ctx is cancelled rather
// than block on its sleep timer.
func TestRunReembedConvergeStopsOnCancel(t *testing.T) {
	store := &fakeStore{byEmbedder: map[string]int{"bm25": 4}}
	svc := newConvergeService(t, store, stubEmbedder{name: "bm25"}) // converged

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already done before the first select

	done := make(chan struct{})
	go func() {
		// Huge idle interval: only ctx cancellation can end the loop.
		svc.RunReembedConverge(ctx, ReembedConvergeConfig{IdleInterval: time.Hour})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunReembedConverge did not return on ctx cancel")
	}
}
