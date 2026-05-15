// Mirror of internal/memory/ranking.go — kept in sync manually
// (the formula is small, the cost of a TypeScript port is zero,
// and the value of letting UI explain "why this ranks where it
// does" without a backend round-trip is high).

import type { MemoryRecord } from './memory'

export const AGE_DECAY_DAYS = 180
export const AGE_FLOOR = 0.5
export const HITS_PER_BOOST_UNIT = 0.02
export const HIT_BOOST_CAP = 0.5
export const CONFIDENCE_FLOOR = 0.3

export interface RankingBreakdown {
  similarity: number
  ageMultiplier: number
  hitMultiplier: number
  confidenceMultiplier: number
  effectiveScore: number
  ageDays: number
}

/**
 * Compute the same effective score the Go backend uses, plus the
 * intermediate multipliers so the UI can show a tooltip
 * explaining the math. `similarity` is optional — for the
 * inspector (which lists memories without a query) callers pass
 * 1.0 to see the "if we matched perfectly, how would this rank"
 * baseline.
 */
export function rankingBreakdown(
  mem: MemoryRecord,
  similarity = 1,
  now: Date = new Date(),
): RankingBreakdown {
  if (similarity <= 0) {
    return {
      similarity: 0,
      ageMultiplier: 0,
      hitMultiplier: 0,
      confidenceMultiplier: 0,
      effectiveScore: 0,
      ageDays: 0,
    }
  }
  const ageMs = now.getTime() - new Date(mem.created_at).getTime()
  const ageDays = Math.max(0, ageMs / 86_400_000)
  const ageMultiplier = Math.max(AGE_FLOOR, 1 - ageDays / AGE_DECAY_DAYS)
  const hitBoost = Math.min(mem.hit_count * HITS_PER_BOOST_UNIT, HIT_BOOST_CAP)
  const hitMultiplier = 1 + hitBoost
  const conf =
    mem.confidence == null
      ? 1
      : Math.min(1, Math.max(CONFIDENCE_FLOOR, mem.confidence))
  return {
    similarity,
    ageMultiplier,
    hitMultiplier,
    confidenceMultiplier: conf,
    effectiveScore: similarity * ageMultiplier * hitMultiplier * conf,
    ageDays,
  }
}
