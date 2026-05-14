package memory

import "time"

// Ranking knobs — exported so operators can override at composition
// time if a project's age/use distribution makes the defaults bite.
// The current values balance "recency matters, but a 6-month-old
// fact that's been hit 30 times still wins" — which is the
// observed behaviour the M-PC redesign aims for.
const (
	// AgeDecayDays is the linear decay window. A memory at
	// AgeDecayDays old has multiplier hit AgeFloor; older memories
	// stay there rather than going to zero.
	AgeDecayDays = 180

	// AgeFloor is the minimum age multiplier — guarantees an old
	// memory can still rank above a brand-new mediocre match.
	AgeFloor = 0.5

	// HitsPerBoostUnit controls how fast hit_count lifts the score.
	// 50 hits = +1.0 boost cap.
	HitsPerBoostUnit = 0.02

	// HitBoostCap clamps the hit_count term so a popular-but-stale
	// memory can't dominate fresh signals forever.
	HitBoostCap = 0.5

	// ConfidenceFloor is the minimum confidence multiplier — a
	// memory with zero or unknown confidence still contributes
	// (we'd rather show low-confidence info than nothing) but at
	// reduced weight.
	ConfidenceFloor = 0.3
)

// RankingScore is the M-PC effective-score formula. Splitting it
// out lets memquery / cleaner share the exact same logic when
// they need to reason about ranking outside the Search code path,
// and gives tests a small surface to pin behaviour against.
//
// Formula:
//
//	effective = similarity
//	          × ageDecay(age_days)
//	          × hitBoost(hit_count)
//	          × confidenceFloor(memory.Confidence)
//
// All four factors are in [0, 1+HitBoostCap]; the product stays
// roughly in [0, 1.5] for cosine inputs in [-1, 1], which keeps
// the threshold comparison intuitive.
func RankingScore(similarity float32, m Memory, now time.Time) float32 {
	if similarity <= 0 {
		return 0
	}
	age := ageMultiplier(now.Sub(m.CreatedAt))
	hits := hitMultiplier(m.HitCount)
	conf := confidenceMultiplier(m.Confidence)
	return similarity * age * hits * conf
}

func ageMultiplier(age time.Duration) float32 {
	days := float32(age.Hours()) / 24
	if days < 0 {
		days = 0
	}
	decay := 1 - days/AgeDecayDays
	if decay < AgeFloor {
		decay = AgeFloor
	}
	return decay
}

func hitMultiplier(hitCount int64) float32 {
	if hitCount <= 0 {
		return 1
	}
	boost := float32(hitCount) * HitsPerBoostUnit
	if boost > HitBoostCap {
		boost = HitBoostCap
	}
	return 1 + boost
}

func confidenceMultiplier(c *float32) float32 {
	if c == nil {
		return 1
	}
	v := *c
	if v < ConfidenceFloor {
		return ConfidenceFloor
	}
	if v > 1 {
		return 1
	}
	return v
}
