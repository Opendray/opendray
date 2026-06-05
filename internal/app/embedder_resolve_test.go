package app

import "testing"

// TestDecideAutoEmbedder exhaustively covers the backend="auto" tiering
// table (M-U Phase 7 §11.2). The owner-confirmed posture is upgrade-only:
// never silently downgrade dense→BM25, never churn.
func TestDecideAutoEmbedder(t *testing.T) {
	tests := []struct {
		name           string
		httpConfigured bool
		httpReachable  bool
		intentDense    bool
		want           embedderDecision
	}{
		{"fresh install, no model", false, false, false, decideBM25Floor},
		{"no http config but dense rows exist -> fail-closed", false, false, true, decideFailClosed},
		{"configured + reachable -> dense", true, true, false, decideDenseHTTP},
		{"configured + reachable + dense rows -> dense", true, true, true, decideDenseHTTP},
		{"configured but unreachable, dense rows -> keep dense (degrade)", true, false, true, decideDenseHTTP},
		{"configured but unreachable, no dense rows -> bm25 floor", true, false, false, decideBM25Floor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decideAutoEmbedder(tt.httpConfigured, tt.httpReachable, tt.intentDense)
			if got != tt.want {
				t.Fatalf("decideAutoEmbedder(cfg=%v,reach=%v,dense=%v) = %d, want %d",
					tt.httpConfigured, tt.httpReachable, tt.intentDense, got, tt.want)
			}
		})
	}
}

// TestDenseIntent verifies dense-row intent derivation: BM25 rows never
// count as dense intent, zero-count embedders are ignored, and the
// non-BM25 embedder with the most rows wins.
func TestDenseIntent(t *testing.T) {
	tests := []struct {
		name      string
		counts    map[string]int
		wantName  string
		wantDense bool
	}{
		{"empty store", map[string]int{}, "", false},
		{"bm25 only", map[string]int{"bm25": 12}, "", false},
		{"bm25 zero", map[string]int{"bm25": 0}, "", false},
		{"single dense", map[string]int{"http:qwen3": 5}, "http:qwen3", true},
		{"mixed picks max non-bm25", map[string]int{"bm25": 100, "http:qwen3": 7, "http:bge-m3": 3}, "http:qwen3", true},
		{"dense zero-count ignored", map[string]int{"http:qwen3": 0, "bm25": 4}, "", false},
		{"bm25 prefix variants ignored", map[string]int{"bm25-384": 9}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, dense := denseIntent(tt.counts)
			if dense != tt.wantDense || name != tt.wantName {
				t.Fatalf("denseIntent(%v) = (%q,%v), want (%q,%v)",
					tt.counts, name, dense, tt.wantName, tt.wantDense)
			}
		})
	}
}
