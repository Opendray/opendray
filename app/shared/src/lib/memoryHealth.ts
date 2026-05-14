// Client for /api/v1/memory/health — the M-PA memory health
// dashboard. Returns one aggregate snapshot per cwd combining
// layer-5 facts, capture-engine activity, journal/proposal state.

import { api } from './api'

export interface MemoryHealthSnapshot {
  cwd: string
  generated_at: string
  lookback_days: number

  // Layer 5 — discrete facts.
  new_facts_count: number
  total_facts_count: number
  zero_hit_facts_count: number
  top_hit_fact_text: string
  top_hit_fact_hits: number

  // Capture engine.
  capture_fires: number
  capture_facts_extracted: number
  capture_facts_stored: number
  capture_facts_deduped: number
  capture_failed_fires: number

  // Layer 4 — journal.
  new_journal_count: number
  total_journal_count: number

  // Layer 2-3 — plan / goal.
  plan_last_updated_at?: string
  goal_last_updated_at?: string
  pending_proposals: number
  oldest_pending_days: number
  plan_drift_proposals: number
}

export async function getMemoryHealth(
  cwd: string,
): Promise<MemoryHealthSnapshot> {
  const qs = new URLSearchParams({ cwd })
  return api<MemoryHealthSnapshot>(`/api/v1/memory/health?${qs}`)
}
