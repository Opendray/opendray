package memory

import (
	"math"
	"testing"
	"time"
)

func TestRankingScore_BaselineFresh(t *testing.T) {
	// Fresh memory, no hits, no confidence — should equal similarity.
	m := Memory{CreatedAt: time.Now().UTC()}
	got := RankingScore(0.8, m, time.Now().UTC())
	if math.Abs(float64(got-0.8)) > 0.001 {
		t.Errorf("fresh baseline: got %f, want ~0.8", got)
	}
}

func TestRankingScore_AgeDecay(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name    string
		age     time.Duration
		wantMul float32
	}{
		{"today", 0, 1.0},
		{"30d", 30 * 24 * time.Hour, 1 - 30.0/AgeDecayDays},
		{"90d", 90 * 24 * time.Hour, 1 - 90.0/AgeDecayDays},
		{"180d hits floor", AgeDecayDays * 24 * time.Hour, AgeFloor},
		{"way past floor", 730 * 24 * time.Hour, AgeFloor},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := Memory{CreatedAt: now.Add(-tc.age)}
			got := RankingScore(1.0, m, now)
			if math.Abs(float64(got-tc.wantMul)) > 0.001 {
				t.Errorf("got %f, want %f", got, tc.wantMul)
			}
		})
	}
}

func TestRankingScore_HitBoost(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name string
		hits int64
		want float32
	}{
		{"never hit", 0, 1.0},
		{"5 hits → +10%", 5, 1.10},
		{"25 hits → +50%", 25, 1 + HitBoostCap},
		{"100 hits → still capped", 100, 1 + HitBoostCap},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := Memory{CreatedAt: now, HitCount: tc.hits}
			got := RankingScore(1.0, m, now)
			if math.Abs(float64(got-tc.want)) > 0.001 {
				t.Errorf("got %f, want %f", got, tc.want)
			}
		})
	}
}

func TestRankingScore_Confidence(t *testing.T) {
	now := time.Now().UTC()
	mk := func(c float32) *float32 { return &c }
	tests := []struct {
		name string
		conf *float32
		want float32
	}{
		{"nil = 1.0", nil, 1.0},
		{"0.5 = 0.5", mk(0.5), 0.5},
		{"0.1 → floor 0.3", mk(0.1), ConfidenceFloor},
		{"1.2 → cap 1.0", mk(1.2), 1.0},
		{"exact floor", mk(ConfidenceFloor), ConfidenceFloor},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := Memory{CreatedAt: now, Confidence: tc.conf}
			got := RankingScore(1.0, m, now)
			if math.Abs(float64(got-tc.want)) > 0.001 {
				t.Errorf("got %f, want %f", got, tc.want)
			}
		})
	}
}

func TestRankingScore_OldButPopularBeatsNewMediocre(t *testing.T) {
	// The whole point of the formula: a popular 6-month-old memory
	// should outrank a brand-new low-similarity match.
	now := time.Now().UTC()
	popular := Memory{
		CreatedAt: now.Add(-180 * 24 * time.Hour),
		HitCount:  30, // capped boost = +0.5
	}
	mediocreFresh := Memory{CreatedAt: now}

	popularScore := RankingScore(0.8, popular, now)        // 0.8 * 0.5 * 1.5 = 0.60
	mediocreScore := RankingScore(0.5, mediocreFresh, now) // 0.5 * 1 * 1 = 0.50

	if popularScore <= mediocreScore {
		t.Errorf("popular old should beat mediocre fresh: %f vs %f", popularScore, mediocreScore)
	}
}

func TestRankingScore_NegativeSimilarityIsZero(t *testing.T) {
	got := RankingScore(-0.5, Memory{CreatedAt: time.Now().UTC()}, time.Now().UTC())
	if got != 0 {
		t.Errorf("negative similarity should produce 0, got %f", got)
	}
}
