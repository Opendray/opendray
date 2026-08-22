// Client for /api/v1/project-docs/* + /project-doc-proposals/* +
// /session-logs/*. Backs the Project page in web (and mirrors
// app/mobile/lib/core/api/project_docs_api.dart shape).
//
// Powers the unified cross-CLI memory L2/L3/L4/L5 surface:
// project_docs holds the goal / plan / tech_stack / recent_activity
// markdown bodies; proposals queue agent-suggested goal/plan
// edits for operator approval; session_logs is the per-cwd journal.

import { api } from './api'

// ── project_docs ──────────────────────────────────────────────

// Since the Cortex blueprint system, a doc kind is a per-project
// section SLUG — the known literals below are the default blueprint
// plus the fixed global KB pages; any custom section slug is valid.
export type KnownDocKind =
  | 'goal'
  | 'plan'
  | 'tech_stack'
  | 'recent_activity'
  // The project's rich, AI-maintained official document (per-project Notes).
  | 'overview'
  // Cross-project Knowledge pages (Experience Flywheel — global only;
  // per-project docs are blueprint sections, no handbook).
  | 'kb_infrastructure'
  | 'kb_conventions'
  | 'kb_lessons'
  | 'kb_reusable'
// `string & {}` keeps literal autocomplete while accepting custom slugs.
export type DocKind = KnownDocKind | (string & {})
/** 'approved' = an AI draft the operator approved — locks like a hand
 * edit but stays distinguishable from one for provenance. */
export type DocAuthor = 'operator' | 'agent' | 'scanner' | 'approved'

// GlobalCwd sentinel: the cwd under which cross-project (global) KB pages live.
// Mirrors projectdoc.GlobalCwd on the backend.
export const GLOBAL_CWD = '__global__'

export interface ProjectDoc {
  id: string
  cwd: string
  kind: DocKind
  content: string
  updated_by: DocAuthor
  updated_at: string
}

export interface ListDocsResponse {
  docs: ProjectDoc[]
}

export async function listProjectDocs(cwd: string): Promise<ProjectDoc[]> {
  const res = await api<ListDocsResponse>(
    `/api/v1/project-docs?cwd=${encodeURIComponent(cwd)}`,
  )
  return res.docs ?? []
}

export async function getProjectDoc(
  cwd: string,
  kind: DocKind,
): Promise<ProjectDoc> {
  return api<ProjectDoc>(
    `/api/v1/project-docs/${kind}?cwd=${encodeURIComponent(cwd)}`,
  )
}

export async function putProjectDoc(input: {
  cwd: string
  kind: DocKind
  content: string
  /** Defaults to 'operator' (a human edit, which locks a KB page from AI
   * overwrite). Pass 'agent' to UNLOCK a KB page so the drafter manages it. */
  updatedBy?: DocAuthor
}): Promise<ProjectDoc> {
  return api<ProjectDoc>(`/api/v1/project-docs/${input.kind}`, {
    method: 'PUT',
    body: {
      cwd: input.cwd,
      content: input.content,
      updated_by: input.updatedBy ?? 'operator',
    },
  })
}

// ── blueprint (Cortex Phase 3) ────────────────────────────────

/** Who keeps a section current: 'ai' (opendray's background curation
 * redrafts it), 'human' (the operator authors it by hand), 'scanner' (a
 * mechanical rebuilder owns it), or 'session' (the agent doing the work
 * writes it live via the kb_page_set MCP tool). 'session' applies to
 * global knowledge pages only — see canBeSessionMaintained. */
export type MaintainerMode = 'ai' | 'human' | 'scanner' | 'session'

/** Who lands an agent-side MCP write: 'proposal' files an operator-approved
 * proposal (goal/plan — long-term, deliberate); 'direct' lets the in-session
 * agent write the live doc when it is unlocked (current_objective —
 * short-term, churns every session). Empty/absent defaults to 'proposal'. */
export type WritePolicy = 'direct' | 'proposal'

/** One section of a project's doc blueprint: its slug IS the doc kind. */
export interface BlueprintSection {
  cwd: string
  slug: string
  title: string
  description?: string
  position: number
  maintainer_mode: MaintainerMode
  /** Agent-side write routing for this section (see WritePolicy). */
  write_policy?: WritePolicy
  prompt_hint?: string
  /** Pinned sections sort first and cannot be deleted (overview; the
   * classic knowledge four). */
  pinned: boolean
  /** Include this section's doc in the spawn banner. Pages with
   * inject=false are reached on demand via cross-layer search. */
  inject: boolean
  /** Knowledge nature ('foundational' | 'emergent') — GLOBAL pages
   * only; empty/absent for per-project sections. */
  nature?: string
  /** Subjects the AI maintainer must never write about on this page.
   * `prompt_hint` steers what the page SHOULD be; everything else the
   * drafter reads is material to fold in, so this is the only way to say
   * "leave this out". Removing a subject from the page body alone does
   * not work — it stays in the feedstock and is re-derived next sweep.
   * GLOBAL knowledge pages only; the backend rejects it elsewhere. */
  exclusions?: string[]
  created_at?: string
  updated_at?: string
}

/** Whether a knowledge page may be handed to in-session agents
 * (maintainer_mode 'session'). Mirrors the backend's SessionWritable:
 * pinned pages are reserved, and foundational pages carry binding rules
 * an agent must not rewrite mid-task. Offering the mode where the
 * backend would refuse the write is worse than not offering it. */
export function canBeSessionMaintained(
  section: Pick<BlueprintSection, 'pinned' | 'nature'>,
): boolean {
  return !section.pinned && section.nature !== 'foundational'
}

/** Lists the project's blueprint (lazily seeded with defaults). */
export async function listBlueprintSections(
  cwd: string,
): Promise<BlueprintSection[]> {
  const res = await api<{ sections: BlueprintSection[] }>(
    `/api/v1/project-docs/blueprint?cwd=${encodeURIComponent(cwd)}`,
  )
  return res.sections ?? []
}

/** Creates or updates one blueprint section. */
export async function putBlueprintSection(
  section: BlueprintSection,
): Promise<BlueprintSection> {
  return api<BlueprintSection>(
    `/api/v1/project-docs/blueprint/${section.slug}`,
    { method: 'PUT', body: section },
  )
}

/** One line the operator deleted from a knowledge page (deletion-as-
 * signal). status 'active' = deleted once, soft negative context for the
 * drafter; 'banned' = deleted twice — the system reintroduced it and the
 * operator removed it again — hard-scrubbed from every future draft. */
export interface DocLineRemoval {
  id: string
  cwd: string
  kind: string
  line_text: string
  removal_count: number
  status: 'active' | 'banned' | 'dismissed'
  last_removed_at: string
}

/** Lists a page's recorded operator deletions (active + banned). */
export async function listDocRemovals(
  cwd: string,
  kind: string,
): Promise<DocLineRemoval[]> {
  const res = await api<{ removals: DocLineRemoval[] }>(
    `/api/v1/project-docs/removals?cwd=${encodeURIComponent(cwd)}&kind=${encodeURIComponent(kind)}`,
  )
  return res.removals ?? []
}

/** Unbans / forgets one recorded line removal. */
export async function dismissDocRemoval(id: string): Promise<void> {
  await api(`/api/v1/project-docs/removals/${id}/dismiss`, { method: 'POST' })
}

/** Removes a section from the blueprint (its doc content is kept and
 * resurrects if the slug is re-added). The overview is reserved. */
export async function deleteBlueprintSection(
  cwd: string,
  slug: string,
): Promise<void> {
  await api(
    `/api/v1/project-docs/blueprint/${slug}?cwd=${encodeURIComponent(cwd)}`,
    { method: 'DELETE' },
  )
}

// ── lifecycle (P-D) ───────────────────────────────────────────

export type ProjectStatus = 'active' | 'paused' | 'archived'

export interface ProjectSummary {
  cwd: string
  status: ProjectStatus
  updated_by: DocAuthor
  last_activity_at?: string
  idle_days: number
  /** Active project idle past the threshold — suggest archiving. */
  suggest_archive: boolean
}

/** Lists every known project with its lifecycle status + last activity.
 * idleDays overrides the auto-suggest threshold (0 disables). */
export async function listProjects(idleDays?: number): Promise<ProjectSummary[]> {
  const qs = idleDays === undefined ? '' : `?idle_days=${idleDays}`
  const res = await api<{ projects: ProjectSummary[] }>(
    `/api/v1/project-docs/projects${qs}`,
  )
  return res.projects ?? []
}

/** Sets a project's lifecycle status. Frozen (paused/archived) projects are
 * excluded from spawn injection and cross-project Knowledge distillation. */
export async function setProjectLifecycle(
  cwd: string,
  status: ProjectStatus,
): Promise<void> {
  await api('/api/v1/project-docs/lifecycle', {
    method: 'POST',
    body: { cwd, status },
  })
}

// ── proposals ─────────────────────────────────────────────────

/** One line of a review diff. 'context' lines are unchanged and shown
 * only for orientation. */
export interface DiffLine {
  kind: 'context' | 'add' | 'remove'
  text: string
}

/** A run of changed lines with its surrounding context. */
export interface DiffHunk {
  /** Unchanged lines between the previous hunk and this one. Rendered as
   * "N unchanged lines" so the reviewer knows something was collapsed
   * rather than wondering whether the diff is complete. */
  skipped_before: number
  /** 1-based first line of this hunk in the NEW document. */
  start_line: number
  lines: DiffLine[]
}

/** The reviewable difference between the live doc and a proposal. Computed
 * server-side so every client shows the same review. */
export interface DocDiff {
  hunks: DiffHunk[]
  added: number
  removed: number
  unchanged: boolean
}

export interface DocProposal {
  id: string
  cwd: string
  kind: DocKind
  proposed_content: string
  proposed_by_session?: string
  reason: string
  /** When the proposal has been decided, the verdict. */
  decision?: 'approved' | 'rejected'
  decided_at?: string
  /** The prior live content at the time of proposal. */
  prior_content?: string
  /** Line-level change from prior_content to proposed_content. Present on
   * the review endpoints; absent on paths that don't serve a review. */
  diff?: DocDiff
  created_at: string
}

export async function listPendingProposals(cwd?: string): Promise<DocProposal[]> {
  const qs = cwd ? `?cwd=${encodeURIComponent(cwd)}` : ''
  const res = await api<{ proposals: DocProposal[] }>(
    `/api/v1/project-doc-proposals/pending${qs}`,
  )
  return res.proposals ?? []
}

export async function approveProposal(id: string): Promise<ProjectDoc> {
  return api<ProjectDoc>(`/api/v1/project-doc-proposals/${id}/approve`, {
    method: 'POST',
  })
}

export async function rejectProposal(id: string): Promise<void> {
  await api(`/api/v1/project-doc-proposals/${id}/reject`, {
    method: 'POST',
  })
}

// ── session_logs (journal) ────────────────────────────────────

export type LogKind = 'session_summary' | 'manual' | 'decision'

export interface SessionLogEntry {
  id: string
  cwd: string
  session_id?: string
  kind: LogKind
  title: string
  content: string
  updated_by: DocAuthor | 'summarizer'
  created_at: string
}

export async function listSessionLogs(
  cwd: string,
  limit = 50,
): Promise<SessionLogEntry[]> {
  const res = await api<{ logs: SessionLogEntry[] }>(
    `/api/v1/session-logs?cwd=${encodeURIComponent(cwd)}&n=${limit}`,
  )
  return res.logs ?? []
}

export async function appendSessionLog(input: {
  cwd: string
  kind?: LogKind
  session_id?: string
  title?: string
  content: string
}): Promise<SessionLogEntry> {
  return api<SessionLogEntry>('/api/v1/session-logs', {
    method: 'POST',
    body: { ...input, updated_by: 'operator' },
  })
}

export async function deleteSessionLog(id: string): Promise<void> {
  await api(`/api/v1/session-logs/${id}`, { method: 'DELETE' })
}

// M-PD — list stale journal entries that the daily conflict
// detector hasn't tied to any pending finding. Used by the
// Journal tab's Stale subview to bulk-prune accumulated noise.
export async function listStaleSessionLogs(
  cwd: string,
  days = 90,
): Promise<SessionLogEntry[]> {
  const qs = new URLSearchParams({ cwd, days: String(days) })
  const res = await api<{ stale: SessionLogEntry[] }>(
    `/api/v1/session-logs/stale?${qs}`,
  )
  return res.stale ?? []
}

// ── reset ─────────────────────────────────────────────────────

export interface ResetProjectMemoryOptions {
  cwd: string
  /** Also wipe tech_stack + recent_activity (default false; they auto-rebuild on next spawn). */
  include_scanner_docs?: boolean
  /** Also wipe memory_cleanup_decisions for this cwd (default true). */
  include_cleanup_decisions?: boolean
}

export interface ResetProjectMemoryCounts {
  project_docs: number
  project_doc_proposals: number
  session_logs: number
  memory_cleanup_decisions: number
}

/**
 * Wipes per-cwd project memory state in a transaction:
 * project_docs (goal/plan, optionally scanner-managed docs too),
 * project_doc_proposals, session_logs, and memory_cleanup_decisions
 * for this cwd.
 *
 * Does NOT touch the pgvector `memories` table — call
 * deleteMemoriesByScope('project', cwd) separately when the
 * operator opts in.
 */
export async function resetProjectMemory(
  opts: ResetProjectMemoryOptions,
): Promise<ResetProjectMemoryCounts> {
  return api<ResetProjectMemoryCounts>('/api/v1/project-docs/reset', {
    method: 'POST',
    body: opts,
  })
}
