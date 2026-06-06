package memquery

import (
	"math"
	"testing"
	"time"

	"github.com/opendray/opendray-v2/internal/memory"
)

// The cross-layer ranker is now memory.RankingScoreFields (the same one
// memory_search uses), not a private decayScore. These cases pin the two
// properties that unification gives us: (1) a journal/goal/plan row
// (hitCount=0, conf=nil) still scores as similarity × age-decay, matching
// the old behaviour; (2) a fact with hits is boosted above an
// equal-cosine, equal-age journal row.
func TestEffectiveScoreSharedRanker(t *testing.T) {
	now := time.Now().UTC()
	at := func(age time.Duration) time.Time { return now.Add(-age) }

	t.Run("journal-like row decays on similarity x age", func(t *testing.T) {
		cases := []struct {
			name string
			sim  float32
			age  time.Duration
			want float32
		}{
			{"brand new keeps full score", 0.9, 0, 0.9},
			{"30d old", 1.0, 30 * 24 * time.Hour, 1 - 30.0/180},
			{"180d hits floor", 0.8, 180 * 24 * time.Hour, 0.5 * 0.8},
			{"way past floor", 1.0, 365 * 24 * time.Hour, 0.5},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got := memory.RankingScoreFields(tc.sim, at(tc.age), 0, nil, now)
				if math.Abs(float64(got-tc.want)) > 0.001 {
					t.Errorf("got %f, want %f", got, tc.want)
				}
			})
		}
	})

	t.Run("hit fact outranks equal-cosine journal", func(t *testing.T) {
		fact := memory.RankingScoreFields(0.7, at(0), 30, nil, now)   // popular fact
		journal := memory.RankingScoreFields(0.7, at(0), 0, nil, now) // no hits
		if !(fact > journal) {
			t.Errorf("expected hit fact (%f) to outrank journal (%f)", fact, journal)
		}
	})
}

func TestPgvecLiteral(t *testing.T) {
	tests := []struct {
		name string
		in   []float32
		want string
	}{
		{"empty", nil, "[]"},
		{"single", []float32{0.5}, "[0.5]"},
		{"multiple", []float32{0.1, 0.2, 0.3}, "[0.1,0.2,0.3]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pgvecLiteral(tc.in)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNew_ValidationErrors(t *testing.T) {
	if _, err := New(nil, nil, nil); err == nil {
		t.Error("expected error for all-nil deps")
	}
}
