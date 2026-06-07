// Client for the ambient memory subsystem:
//   /memory-summarizer-providers  — provider CRUD + test + cost
//   /memory-capture-rules         — capture rule CRUD + run-now
//   /memory-injection-profiles    — injection profile CRUD
//
// Mirrors Go shapes in internal/memory/{summarizer,capture,injector}.
// All endpoints sit under the admin-auth group; this file assumes
// the operator bearer token rides every call (the api wrapper
// handles it).

import { api } from './api'

// ── summarizer providers ─────────────────────────────────────────

export type ProviderKind =
  | 'ollama'
  | 'lmstudio'
  | 'anthropic'
  | 'openai'
  | 'integration'

export interface SummarizerProvider {
  id: string
  name: string
  kind: ProviderKind
  model: string
  base_url?: string
  api_key_fingerprint?: string
  api_key_set: boolean
  enabled: boolean
  is_default: boolean
  created_at: string
  updated_at: string
  extra_config?: Record<string, unknown>
}

export interface ProviderCreateInput {
  name: string
  kind: ProviderKind
  model: string
  base_url?: string
  api_key?: string
  is_default?: boolean
  enabled?: boolean
  extra_config?: Record<string, unknown>
}

export interface ProviderPatchInput {
  name?: string
  model?: string
  base_url?: string
  api_key?: string
  enabled?: boolean
  is_default?: boolean
  extra_config?: Record<string, unknown>
}

export interface CostSummary {
  ProviderID: string
  PeriodStart: string
  PeriodEnd: string
  Calls: number
  InputTokens: number
  OutputTokens: number
  EstimatedUSD: number
}

export interface ProviderTestResult {
  ok: boolean
  error?: string
}

export async function listProviders(): Promise<SummarizerProvider[]> {
  const res = await api<{ providers: SummarizerProvider[] }>(
    '/api/v1/memory-summarizer-providers',
  )
  return res.providers ?? []
}

export async function createProvider(
  input: ProviderCreateInput,
): Promise<SummarizerProvider> {
  return api<SummarizerProvider>('/api/v1/memory-summarizer-providers', {
    method: 'POST',
    body: input,
  })
}

export async function updateProvider(
  id: string,
  patch: ProviderPatchInput,
): Promise<SummarizerProvider> {
  return api<SummarizerProvider>(
    `/api/v1/memory-summarizer-providers/${encodeURIComponent(id)}`,
    { method: 'PATCH', body: patch },
  )
}

export async function deleteProvider(id: string): Promise<void> {
  await api(`/api/v1/memory-summarizer-providers/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export async function testProvider(id: string): Promise<ProviderTestResult> {
  return api<ProviderTestResult>(
    `/api/v1/memory-summarizer-providers/${encodeURIComponent(id)}/test`,
    { method: 'POST' },
  )
}

export async function providerCost(
  id: string,
  since?: string,
): Promise<CostSummary> {
  const q = since ? `?since=${encodeURIComponent(since)}` : ''
  return api<CostSummary>(
    `/api/v1/memory-summarizer-providers/${encodeURIComponent(id)}/cost${q}`,
  )
}

// ── capture rules ────────────────────────────────────────────────

export type TriggerKind = 'after_messages' | 'on_idle' | 'k_chars' | 'manual'
// 'session' was retired in the M-U unification (session ≡ project).
export type TargetScope = 'project' | 'global'

export interface CaptureRule {
  id: string
  session_id?: string
  name: string
  enabled: boolean
  trigger_kind: TriggerKind
  trigger_config: Record<string, unknown>
  summarizer_provider_id?: string
  dedup_threshold: number
  target_scope: TargetScope
  created_at: string
  updated_at: string
}

export interface CaptureRuleCreateInput {
  name: string
  trigger_kind: TriggerKind
  trigger_config?: Record<string, unknown>
  enabled?: boolean
  session_id?: string
  summarizer_provider_id?: string
  dedup_threshold?: number
  target_scope?: TargetScope
}

export interface CaptureRulePatchInput {
  name?: string
  enabled?: boolean
  trigger_kind?: TriggerKind
  trigger_config?: Record<string, unknown>
  summarizer_provider_id?: string
  dedup_threshold?: number
  target_scope?: TargetScope
}

export async function listCaptureRules(): Promise<CaptureRule[]> {
  const res = await api<{ rules: CaptureRule[] }>('/api/v1/memory-capture-rules')
  return res.rules ?? []
}

export async function createCaptureRule(
  input: CaptureRuleCreateInput,
): Promise<CaptureRule> {
  return api<CaptureRule>('/api/v1/memory-capture-rules', {
    method: 'POST',
    body: input,
  })
}

export async function updateCaptureRule(
  id: string,
  patch: CaptureRulePatchInput,
): Promise<CaptureRule> {
  return api<CaptureRule>(
    `/api/v1/memory-capture-rules/${encodeURIComponent(id)}`,
    { method: 'PATCH', body: patch },
  )
}

export async function deleteCaptureRule(id: string): Promise<void> {
  await api(`/api/v1/memory-capture-rules/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export interface RunNowResult {
  rule_id: string
  sessions_invoked: number
}

export async function runCaptureRuleNow(id: string): Promise<RunNowResult> {
  return api<RunNowResult>(
    `/api/v1/memory-capture-rules/${encodeURIComponent(id)}/run-now`,
    { method: 'POST' },
  )
}

// ── injection profiles ───────────────────────────────────────────

export type InjectionStrategy =
  | 'none'
  | 'top_k_recent'
  | 'top_k_relevant'
  | 'on_keyword'
  | 'manual_only'
  | 'hybrid'

export interface InjectionProfile {
  id: string
  session_id?: string
  strategy_kind: InjectionStrategy
  config: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface InjectionProfileCreateInput {
  strategy_kind: InjectionStrategy
  config?: Record<string, unknown>
  session_id?: string
}

export interface InjectionProfilePatchInput {
  strategy_kind?: InjectionStrategy
  config?: Record<string, unknown>
}

export async function listInjectionProfiles(): Promise<InjectionProfile[]> {
  const res = await api<{ profiles: InjectionProfile[] }>(
    '/api/v1/memory-injection-profiles',
  )
  return res.profiles ?? []
}

export async function createInjectionProfile(
  input: InjectionProfileCreateInput,
): Promise<InjectionProfile> {
  return api<InjectionProfile>('/api/v1/memory-injection-profiles', {
    method: 'POST',
    body: input,
  })
}

export async function updateInjectionProfile(
  id: string,
  patch: InjectionProfilePatchInput,
): Promise<InjectionProfile> {
  return api<InjectionProfile>(
    `/api/v1/memory-injection-profiles/${encodeURIComponent(id)}`,
    { method: 'PATCH', body: patch },
  )
}

export async function deleteInjectionProfile(id: string): Promise<void> {
  await api(`/api/v1/memory-injection-profiles/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

// ── helper: human descriptors for the UI ─────────────────────────

export const PROVIDER_KIND_LABELS: Record<ProviderKind, string> = {
  ollama: 'Ollama (local)',
  lmstudio: 'LM Studio (local)',
  anthropic: 'Anthropic API',
  openai: 'OpenAI API',
  integration: 'Integration (your local service)',
}

export const TRIGGER_KIND_LABELS: Record<TriggerKind, string> = {
  after_messages: 'After N user messages',
  on_idle: 'After session idle for N seconds',
  k_chars: 'After K cumulative characters',
  manual: 'Manual only (run-now button)',
}

export const STRATEGY_LABELS: Record<InjectionStrategy, string> = {
  none: 'None — model uses memory_search on demand',
  top_k_recent: 'Top K recent (most recently stored)',
  top_k_relevant: 'Top K relevant (semantic search)',
  on_keyword: 'On keyword (Phase v1.1 — placeholder)',
  manual_only: 'Manual only — UI/API trigger',
  hybrid: 'Hybrid — single short summary line',
}
