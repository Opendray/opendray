// Round Table API client (experimental) — a cross-vendor AI GROUP CHAT.
// Members are the seated providers (claude/codex/antigravity/grok/opencode)
// plus the operator. The operator @mentions the members who should reply; each
// mentioned member reads the whole thread and answers in character.
// Wraps the /api/v1/round-tables/* endpoints via api<T>().
import { api } from './api'

// Seat providers with a headless worker path (must match the backend's
// worker.AgentWorker.buildCommand switch). A standalone gemini seat has no
// headless path yet, so it cannot take a seat.
export type SeatProvider =
  | 'claude'
  | 'codex'
  | 'antigravity'
  | 'grok'
  | 'opencode'

export const SEAT_PROVIDERS: readonly SeatProvider[] = [
  'claude',
  'codex',
  'antigravity',
  'grok',
  'opencode',
]

// Vendor family behind each seat — the diversity is the whole point.
// opencode is provider-agnostic, so it names the CLI itself.
export const SEAT_VENDOR: Record<SeatProvider, string> = {
  claude: 'Anthropic',
  codex: 'OpenAI',
  antigravity: 'Google Gemini',
  grok: 'xAI Grok',
  opencode: 'OpenCode',
}

// Per-seat model hint. Blank = the CLI's own default. codex on a plain
// ChatGPT plan rejects most models — set a supported one (e.g. gpt-5.4-mini)
// or the seat fails with "model not supported". opencode takes provider/model
// form (e.g. anthropic/claude-sonnet-4-6).
export const SEAT_MODEL_PLACEHOLDER: Record<SeatProvider, string> = {
  claude: 'default (blank = CLI default)',
  codex: 'e.g. gpt-5.4-mini',
  antigravity: 'default (blank = CLI default)',
  grok: 'default (blank = CLI default)',
  opencode: 'e.g. anthropic/claude-sonnet-4-6',
}

// Sensible default model per seat, pre-filled (editable) so the operator
// doesn't hand-type an exact model string (a one-char typo like
// "gpt-5.4-min" fails the whole seat). codex's own config default (gpt-5.4)
// is rejected on a plain ChatGPT plan, so we default it to the model that
// works there; claude/antigravity default to the CLI's own choice.
export const SEAT_MODEL_DEFAULT: Partial<Record<SeatProvider, string>> = {
  codex: 'gpt-5.4-mini',
}

// Seat providers that support opendray multi-account selection (a per-seat
// account pin honoured by the backend worker). claude binds via config dir +
// OAuth token; antigravity via a dedicated HOME. codex / grok / opencode use a
// single host login, so no per-seat account choice. Mirrors the backend's
// roundtable.providerHasAccounts.
export const SEAT_SUPPORTS_ACCOUNT: ReadonlySet<SeatProvider> = new Set<SeatProvider>(
  ['claude', 'antigravity'],
)

export type RoundTableStatus = 'active' | 'closed'

export type MessageRole = 'operator' | 'seat' | 'system'
export type MessageKind = 'message' | 'summary'

export interface Seat {
  provider: SeatProvider
  model?: string
  account_id?: string
  // Optional per-seat role / system instruction layered on top of the
  // vendor's own voice (e.g. "you are the security reviewer").
  persona?: string
}

export type PlanStepStatus = 'pending' | 'running' | 'done'

// One step of the role-based execution plan: a task assigned to the member
// whose strength fits it. Run one at a time — each spawns a real session in the
// shared project cwd so specialists collaborate through the working tree.
export interface PlanStep {
  assignee: string
  model?: string
  account_id?: string
  task: string
  status: PlanStepStatus
  session_id?: string
}

export interface RoundTable {
  id: string
  topic: string
  cwd?: string
  seats: Seat[]
  // Table-level directive shared by all members (current topic + roles /
  // relationships). Editable live; injected into every reply's prompt.
  framing?: string
  // Role-based execution plan (ordered steps). Empty until drafted.
  plan?: PlanStep[]
  status: RoundTableStatus
  resulting_session_id?: string
  origin: string
  integration_id?: string
  created_at: string
  updated_at: string
}

export interface Message {
  id: string
  round_table_id: string
  role: MessageRole
  seat_provider?: string
  seat_model?: string
  kind: MessageKind
  content: string
  mentions?: string[]
  created_at: string
}

export interface CreateRoundTableInput {
  // Optional — when omitted the chat auto-names itself from the first message.
  topic?: string
  cwd?: string
  seats: Seat[]
  framing?: string
}

// Live edits to a round table: reassign roles (seats) and/or re-frame the
// discussion. Only the fields present are changed.
export interface UpdateRoundTableInput {
  seats?: Seat[]
  framing?: string
  // Bind (or rebind) the shared project working dir after creation, so a plan
  // drafted on a table with no project can be run.
  cwd?: string
}

// One selectable model for a seat provider (value passed to --model,
// "" = CLI default).
export interface SeatModelOption {
  value: string
  label: string
}

// GET the selectable models per provider — antigravity/opencode are enumerated
// live from their CLIs; claude/codex/grok are curated. Drives the seat model
// dropdown so nobody hand-types a model string.
export async function listSeatModels(): Promise<
  Record<string, SeatModelOption[]>
> {
  const res = await api<{ models: Record<string, SeatModelOption[]> }>(
    '/api/v1/round-tables/models',
  )
  return res.models ?? {}
}

// PATCH — reassign roles / re-frame a live table.
export async function updateRoundTable(
  id: string,
  input: UpdateRoundTableInput,
): Promise<RoundTable> {
  return api<RoundTable>(`/api/v1/round-tables/${id}`, {
    method: 'PATCH',
    body: input,
  })
}

// GET list — unwraps the { round_tables } envelope, defaults to [].
export async function listRoundTables(cwd?: string): Promise<RoundTable[]> {
  const q = cwd ? `?cwd=${encodeURIComponent(cwd)}` : ''
  const res = await api<{ round_tables: RoundTable[] }>(
    `/api/v1/round-tables${q}`,
  )
  return res.round_tables ?? []
}

// GET one — the table plus its full chat thread.
export async function getRoundTable(
  id: string,
): Promise<{ round_table: RoundTable; messages: Message[] }> {
  return api<{ round_table: RoundTable; messages: Message[] }>(
    `/api/v1/round-tables/${id}`,
  )
}

// POST create — opens an active group chat.
export async function createRoundTable(
  input: CreateRoundTableInput,
): Promise<RoundTable> {
  return api<RoundTable>('/api/v1/round-tables', {
    method: 'POST',
    body: input,
  })
}

// POST a message. @mentioned members (@claude/@codex/@antigravity/@grok/
// @opencode/@all) reply asynchronously — poll GET while replies land.
export async function postMessage(id: string, content: string): Promise<Message> {
  return api<Message>(`/api/v1/round-tables/${id}/messages`, {
    method: 'POST',
    body: { content },
  })
}

// POST plan/draft — a member drafts a role-assigned execution plan (async).
export async function draftPlan(id: string, provider?: string): Promise<void> {
  await api<void>(`/api/v1/round-tables/${id}/plan/draft`, {
    method: 'POST',
    body: { provider: provider ?? '' },
  })
}

// PUT plan — replace the plan (operator edits the drafted steps).
export async function setPlan(
  id: string,
  steps: PlanStep[],
): Promise<RoundTable> {
  return api<RoundTable>(`/api/v1/round-tables/${id}/plan`, {
    method: 'PUT',
    body: { steps },
  })
}

// POST plan/run — launch a real session to carry out one step; returns its id.
export async function runPlanStep(
  id: string,
  index: number,
  opts?: { cwd?: string; account_id?: string; args?: string[] },
): Promise<{ session_id: string }> {
  return api<{ session_id: string }>(`/api/v1/round-tables/${id}/plan/run`, {
    method: 'POST',
    body: {
      index,
      cwd: opts?.cwd,
      account_id: opts?.account_id,
      args: opts?.args,
    },
  })
}

// POST continue — resume a paused auto-discussion for another burst (async).
export async function continueRoundTable(id: string): Promise<void> {
  await api<void>(`/api/v1/round-tables/${id}/continue`, { method: 'POST' })
}

// POST summarize — a member condenses the chat into a plan (async).
export async function summarizeRoundTable(
  id: string,
  provider?: string,
): Promise<void> {
  await api<void>(`/api/v1/round-tables/${id}/summarize`, {
    method: 'POST',
    body: { provider: provider ?? '' },
  })
}

// POST handoff — spawn a real agent session (full tool access) to implement
// the discussion. Returns the new session id. The round-table members only
// chat (read-only); this is the bridge to actual code changes.
export interface HandoffInput {
  provider: string
  cwd?: string
  model?: string
  account_id?: string
  // Force a brand-new session even if a prior handoff session is still alive
  // (default: continue in the existing one when possible).
  force_new?: boolean
  // Extra CLI flags for the spawned session — e.g. the provider's bypass /
  // skip-permissions flag when the operator opts into YOLO mode (mirrors
  // normal session creation).
  args?: string[]
}
export async function handoffRoundTable(
  id: string,
  input: HandoffInput,
): Promise<{ session_id: string }> {
  return api<{ session_id: string }>(`/api/v1/round-tables/${id}/handoff`, {
    method: 'POST',
    body: input,
  })
}

// POST close — keeps the thread but stops new messages.
export async function closeRoundTable(id: string): Promise<void> {
  await api<void>(`/api/v1/round-tables/${id}/close`, { method: 'POST' })
}

// POST reopen — flips a closed chat back to active so it can resume.
export async function reopenRoundTable(id: string): Promise<void> {
  await api<void>(`/api/v1/round-tables/${id}/reopen`, { method: 'POST' })
}

// DELETE — permanently removes the chat and its messages.
export async function deleteRoundTable(id: string): Promise<void> {
  await api<void>(`/api/v1/round-tables/${id}`, { method: 'DELETE' })
}
