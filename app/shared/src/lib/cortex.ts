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
}): Promise<CortexConversation> {
  return api<CortexConversation>('/api/v1/cortex/conversations', {
    method: 'POST',
    body: input,
  })
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
