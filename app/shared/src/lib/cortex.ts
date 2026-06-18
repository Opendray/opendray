// Client for /api/v1/cortex/* — the unified module governing the
// Memory → Notes → Knowledge flywheel. Cross-layer endpoints only:
// flywheel status, the quarantine review queue, the AI blueprint
// proposer, and curation conversations. The per-layer CRUD clients
// (projectDocs.ts / memory.ts / knowledge.ts) keep working — their
// routes are dual-mounted under /cortex on the backend.

import { api } from './api'
import type { BlueprintSection } from './projectDocs'
import type { MemoryRecord } from './memory'

// ── flywheel status ───────────────────────────────────────────

export interface CortexStatus {
  notes: {
    projects: number
    active_projects: number
    frozen_projects: number
    pending_proposals: number
  }
  memory: {
    enabled: boolean
    quarantine_count: number
  }
  knowledge: {
    enabled: boolean
    pending_proposals: number
  }
}

export async function getCortexStatus(): Promise<CortexStatus> {
  return api<CortexStatus>('/api/v1/cortex/status')
}

// ── quarantine review queue (Phase 2) ─────────────────────────

export interface QuarantineList {
  memories: MemoryRecord[]
  count: number
}

export async function listQuarantined(limit = 100): Promise<QuarantineList> {
  return api<QuarantineList>(`/api/v1/cortex/memory/quarantine?n=${limit}`)
}

/** Promotes a quarantined memory into the durable tier. */
export async function promoteQuarantined(id: string): Promise<void> {
  await api(`/api/v1/cortex/memory/quarantine/${id}/promote`, {
    method: 'POST',
  })
}

/** Discards (deletes) a quarantined memory. */
export async function discardQuarantined(id: string): Promise<void> {
  await api(`/api/v1/cortex/memory/quarantine/${id}/discard`, {
    method: 'POST',
  })
}

// ── blueprint proposer (Phase 3) ──────────────────────────────

export interface BlueprintProposal {
  project_type: string
  reason: string
  sections: BlueprintSection[]
}

/** Asks the AI to classify the project and propose a tailored section
 * set. Nothing is persisted — apply via applyBlueprint on accept. */
export async function proposeBlueprint(cwd: string): Promise<BlueprintProposal> {
  return api<BlueprintProposal>(
    `/api/v1/cortex/blueprint/propose?cwd=${encodeURIComponent(cwd)}`,
    { method: 'POST' },
  )
}

/** Replaces the project's blueprint with the given section set
 * (sections absent from the list are removed; overview is reserved). */
export async function applyBlueprint(
  cwd: string,
  sections: BlueprintSection[],
): Promise<BlueprintSection[]> {
  const res = await api<{ sections: BlueprintSection[] }>(
    '/api/v1/cortex/blueprint',
    { method: 'PUT', body: { cwd, sections } },
  )
  return res.sections ?? []
}

// ── runtime settings ──────────────────────────────────────────

export type SpawnMode = 'full' | 'lean'

export interface CortexSettings {
  /** full = inject everything inject-flagged; lean = guardrails + a
   * compact index, agents fetch the rest on demand (doc_read /
   * project_search MCP tools). */
  spawn_mode: SpawnMode
}

export async function getCortexSettings(): Promise<CortexSettings> {
  return api<CortexSettings>('/api/v1/cortex/settings')
}

export async function putCortexSettings(
  patch: Partial<CortexSettings>,
): Promise<CortexSettings> {
  return api<CortexSettings>('/api/v1/cortex/settings', {
    method: 'PUT',
    body: patch,
  })
}

// ── curation conversations (Phase 4) ──────────────────────────

export type ConversationTargetKind = 'doc_section' | 'kb_page' | 'blueprint'
export type ConversationStatus = 'open' | 'closed' | 'escalated'

export interface CortexConversation {
  id: string
  target_kind: ConversationTargetKind
  target_cwd: string
  target_slug: string
  status: ConversationStatus
  escalated_session_id?: string
  /** Per-conversation model override (all empty = global `curation`
   * worker). Mutually exclusive: summarizer_id pins a local/HTTP model;
   * provider_id+model pins a cloud-agent CLI. */
  provider_id?: string
  model?: string
  /** Which Claude (cliacct) account a claude turn runs against — Claude
   * is multi-account. Only meaningful when provider_id === 'claude'. */
  claude_account_id?: string
  summarizer_id?: string
  created_at: string
  updated_at: string
}

export interface ConversationMessage {
  id: string
  conversation_id: string
  role: 'operator' | 'ai' | 'system'
  content: string
  /** What the AI's structured revision did: 'applied' | 'proposed' | ''. */
  revision_action?: string
  revision_ref?: string
  created_at: string
}

export async function createConversation(input: {
  target_kind: ConversationTargetKind
  target_cwd: string
  target_slug: string
  /** Optional model override (cloud-agent provider_id+model, OR a local
   * summarizer_id) for the new conversation. */
  provider_id?: string
  model?: string
  claude_account_id?: string
  summarizer_id?: string
}): Promise<CortexConversation> {
  return api<CortexConversation>('/api/v1/cortex/conversations', {
    method: 'POST',
    body: input,
  })
}

/** Pins (or clears, with all empty) a conversation's model override:
 * a cloud-agent provider+model OR a local summarizer_id (mutually
 * exclusive). Returns the updated conversation. */
export async function setConversationProvider(
  id: string,
  override: {
    provider_id?: string
    model?: string
    claude_account_id?: string
    summarizer_id?: string
  },
): Promise<CortexConversation> {
  return api<CortexConversation>(
    `/api/v1/cortex/conversations/${id}/provider`,
    {
      method: 'POST',
      body: {
        provider_id: override.provider_id ?? '',
        model: override.model ?? '',
        claude_account_id: override.claude_account_id ?? '',
        summarizer_id: override.summarizer_id ?? '',
      },
    },
  )
}

export async function listConversations(
  cwd?: string,
  slug?: string,
): Promise<CortexConversation[]> {
  const qs = new URLSearchParams()
  if (cwd) qs.set('cwd', cwd)
  if (slug) qs.set('slug', slug)
  const suffix = qs.toString() ? `?${qs}` : ''
  const res = await api<{ conversations: CortexConversation[] }>(
    `/api/v1/cortex/conversations${suffix}`,
  )
  return res.conversations ?? []
}

export interface ConversationDetail {
  conversation: CortexConversation
  messages: ConversationMessage[]
}

export async function getConversation(id: string): Promise<ConversationDetail> {
  return api<ConversationDetail>(`/api/v1/cortex/conversations/${id}`)
}

/** Sends an operator message. The AI reply lands asynchronously —
 * listen for the `cortex.conversation.reply` event or re-poll. */
export async function sendConversationMessage(
  id: string,
  content: string,
): Promise<ConversationMessage> {
  return api<ConversationMessage>(
    `/api/v1/cortex/conversations/${id}/messages`,
    { method: 'POST', body: { content } },
  )
}

/** Escalates the conversation into a full agent session (grounded in
 * the codebase). Returns the updated conversation with the session id. */
export async function escalateConversation(
  id: string,
): Promise<CortexConversation> {
  return api<CortexConversation>(
    `/api/v1/cortex/conversations/${id}/escalate`,
    { method: 'POST' },
  )
}

export async function closeConversation(id: string): Promise<void> {
  await api(`/api/v1/cortex/conversations/${id}/close`, { method: 'POST' })
}
