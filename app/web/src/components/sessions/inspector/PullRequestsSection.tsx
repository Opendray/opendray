import { type ComponentPropsWithoutRef, useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  GitPullRequest,
  GitPullRequestDraft,
  Loader2,
  ExternalLink,
  KeyRound,
  CheckCircle2,
  XCircle,
  CircleDashed,
  Plus,
  GitMerge,
  X,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { toast } from 'sonner'
import ReactMarkdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'

import {
  type CheckRun,
  type GitPullRequest as PR,
  createGitPR,
  getGitPR,
  getPRChecks,
  listGitPRs,
  mergeGitPR,
} from '@/lib/githost'
import { cn } from '@/lib/utils'

interface PullRequestsSectionProps {
  cwd: string
}

// PullRequestsSection sits in GitPanel under the working tree. Tries
// `GET /git/prs?path=<cwd>` — backend detects the remote and either
// returns PRs from the matching git_hosts row or `need_token: true`,
// which we render as a deep link to /plugins.
//
// Each row opens a right-side detail drawer (description, status, CI
// checks for any state, and a Merge action when the PR is open). The
// "Create PR" button at the top of the section opens a small inline
// form (no modal) that defaults the title from a placeholder hint
// since we don't have easy access to the latest commit message from
// here — operator types or pastes from terminal.
export function PullRequestsSection({ cwd }: PullRequestsSectionProps) {
  const [state, setState] = useState<'open' | 'closed' | 'all'>('open')
  const [creating, setCreating] = useState(false)
  // The PR the detail drawer is showing. We hold the row object itself
  // (not just its number) so the drawer survives the 60s list refetch
  // — otherwise a PR that gets merged/closed elsewhere, or drops off
  // the 20-item window, would yank the drawer shut mid-read.
  const [detailPR, setDetailPR] = useState<PR | null>(null)
  const qc = useQueryClient()

  const { data, isLoading, error } = useQuery({
    queryKey: ['git-prs', cwd, state],
    queryFn: () => listGitPRs(cwd, state),
    refetchInterval: 60_000,
  })

  return (
    <section className="flex flex-col gap-1">
      <div className="flex items-center justify-between px-1">
        <div className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium flex items-center gap-1.5">
          <GitPullRequest className="size-3" />
          Pull requests
        </div>
        <div className="flex items-center gap-1.5">
          {!data?.need_token && (
            <button
              type="button"
              onClick={() => setCreating((v) => !v)}
              className="text-[10px] text-state-running hover:text-foreground flex items-center gap-0.5"
              title="Create a new PR for the current branch"
            >
              <Plus className="size-3" />
              {creating ? 'Cancel' : 'Create'}
            </button>
          )}
          <div className="flex items-center gap-0.5 text-[10px]">
            {(['open', 'closed', 'all'] as const).map((s) => (
              <button
                key={s}
                type="button"
                onClick={() => setState(s)}
                className={cn(
                  'px-1.5 py-0.5 rounded-sm transition-colors',
                  state === s
                    ? 'text-foreground bg-card'
                    : 'text-muted-foreground/60 hover:text-foreground',
                )}
              >
                {s}
              </button>
            ))}
          </div>
        </div>
      </div>

      {creating && !data?.need_token && (
        <CreatePRForm
          cwd={cwd}
          onDone={() => {
            setCreating(false)
            qc.invalidateQueries({ queryKey: ['git-prs', cwd] })
          }}
        />
      )}

      {isLoading && (
        <div className="flex items-center gap-2 text-[11px] text-muted-foreground px-1 py-1">
          <Loader2 className="size-3 animate-spin" />
          Fetching…
        </div>
      )}
      {error && (
        <div className="text-[11px] text-state-failed px-1 py-1">
          {(error as Error).message}
        </div>
      )}
      {data && data.error && !data.need_token && (
        <div className="text-[11px] text-state-failed px-1 py-1 break-words">
          {data.error}
        </div>
      )}
      {data && data.need_token && <NeedTokenHint host={data.remote.host} />}
      {data && !data.error && !data.need_token && data.prs.length === 0 && (
        <div className="text-[11px] text-muted-foreground/60 px-1 py-1">
          No {state === 'all' ? '' : `${state} `}PRs.
        </div>
      )}
      {data && !data.need_token && (
        <div className="flex flex-col">
          {data.prs.map((p) => (
            <PRRow key={p.number} pr={p} onOpen={() => setDetailPR(p)} />
          ))}
        </div>
      )}

      {detailPR && (
        <PRDetailDrawer
          cwd={cwd}
          pr={detailPR}
          onClose={() => setDetailPR(null)}
          onMerged={() => {
            setDetailPR(null)
            qc.invalidateQueries({ queryKey: ['git-prs', cwd] })
          }}
        />
      )}
    </section>
  )
}

// CreatePRForm is the small in-place form that takes a title + head
// branch (we don't auto-detect the current branch — the operator
// knows what they want to push). The body field is intentionally
// minimal; opendray's typical PRs get richer descriptions from the
// terminal-side `gh pr create` flow. This is for the quick case.
function CreatePRForm({
  cwd,
  onDone,
}: {
  cwd: string
  onDone: () => void
}) {
  const [title, setTitle] = useState('')
  const [head, setHead] = useState('')
  const [body, setBody] = useState('')
  const [draft, setDraft] = useState(false)

  const create = useMutation({
    mutationFn: () =>
      createGitPR({
        dir: cwd,
        title: title.trim(),
        head: head.trim(),
        body: body.trim() || undefined,
        draft,
      }),
    onSuccess: (pr) => {
      toast.success(`PR #${pr.number} opened`, {
        description: pr.title,
        action: pr.url
          ? { label: 'Open', onClick: () => window.open(pr.url, '_blank') }
          : undefined,
      })
      onDone()
    },
    onError: (err) => {
      toast.error('Create PR failed', {
        description: (err as Error).message,
      })
    },
  })

  const canSubmit = title.trim() !== '' && head.trim() !== '' && !create.isPending

  return (
    <div className="border border-border rounded-md p-2 flex flex-col gap-1.5 bg-card/40">
      <input
        type="text"
        placeholder="PR title"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        className="text-[12px] bg-transparent border border-border rounded px-2 py-1 outline-none focus:border-state-running"
        autoFocus
      />
      <input
        type="text"
        placeholder="Source branch (head)"
        value={head}
        onChange={(e) => setHead(e.target.value)}
        className="text-[12px] bg-transparent border border-border rounded px-2 py-1 outline-none focus:border-state-running font-mono"
      />
      <textarea
        placeholder="Description (optional)"
        value={body}
        onChange={(e) => setBody(e.target.value)}
        rows={3}
        className="text-[11px] bg-transparent border border-border rounded px-2 py-1 outline-none focus:border-state-running font-mono resize-y"
      />
      <div className="flex items-center justify-between">
        <label className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
          <input
            type="checkbox"
            checked={draft}
            onChange={(e) => setDraft(e.target.checked)}
          />
          Draft
        </label>
        <button
          type="button"
          disabled={!canSubmit}
          onClick={() => create.mutate()}
          className={cn(
            'text-[11px] px-2 py-0.5 rounded transition-colors flex items-center gap-1',
            canSubmit
              ? 'bg-state-running text-background hover:opacity-90'
              : 'bg-muted/30 text-muted-foreground/50 cursor-not-allowed',
          )}
        >
          {create.isPending && <Loader2 className="size-3 animate-spin" />}
          Create PR
        </button>
      </div>
    </div>
  )
}

// PRRow is the per-PR card. Click opens the detail drawer; the
// external-link icon still jumps to the host's web view.
function PRRow({ pr, onOpen }: { pr: PR; onOpen: () => void }) {
  return (
    <div className="border-b border-border/30 last:border-b-0">
      <button
        type="button"
        onClick={onOpen}
        className="w-full px-1 py-1.5 flex items-start gap-2 hover:bg-card rounded-sm group text-left"
        title={`#${pr.number} · ${pr.author} · ${pr.head} → ${pr.base}`}
      >
        <PRStateIcon pr={pr} className="mt-0.5" />
        <div className="flex flex-col min-w-0 flex-1">
          <span className="text-[12px] truncate group-hover:text-foreground">
            {pr.title}
          </span>
          <span className="text-[10px] text-muted-foreground/70 font-mono truncate">
            #{pr.number} · {pr.author} · {pr.head} → {pr.base} ·{' '}
            {relTime(pr.updated_at)}
          </span>
        </div>
        <a
          href={pr.url}
          target="_blank"
          rel="noopener noreferrer"
          onClick={(e) => e.stopPropagation()}
          className="text-muted-foreground/50 hover:text-foreground"
          title="Open on host"
        >
          <ExternalLink className="size-3 mt-0.5" />
        </a>
      </button>
    </div>
  )
}

// PRStateIcon renders the PR glyph coloured by state (draft / merged /
// closed / open). Shared between the row and the drawer header.
function PRStateIcon({ pr, className }: { pr: PR; className?: string }) {
  if (pr.draft) {
    return (
      <GitPullRequestDraft
        className={cn('size-3 text-muted-foreground/60 shrink-0', className)}
      />
    )
  }
  return (
    <GitPullRequest
      className={cn(
        'size-3 shrink-0',
        pr.state === 'merged'
          ? 'text-purple-400'
          : pr.state === 'closed'
            ? 'text-state-failed'
            : 'text-state-running',
        className,
      )}
    />
  )
}

// StateBadge is the textual status pill in the drawer header.
function StateBadge({ pr }: { pr: PR }) {
  const { label, cls } = pr.draft
    ? { label: 'Draft', cls: 'text-muted-foreground/80 border-border' }
    : pr.state === 'merged'
      ? { label: 'Merged', cls: 'text-purple-400 border-purple-400/40' }
      : pr.state === 'closed'
        ? { label: 'Closed', cls: 'text-state-failed border-state-failed/40' }
        : { label: 'Open', cls: 'text-state-running border-state-running/40' }
  return (
    <span
      className={cn(
        'text-[10px] uppercase tracking-wide px-1.5 py-0.5 rounded border shrink-0',
        cls,
      )}
    >
      {label}
    </span>
  )
}

// mdComponents keeps the rendered PR body inside the panel: links open
// in a new tab, code blocks scroll instead of overflowing the narrow
// drawer, and images are width-constrained. The `any` props mirror the
// codebase's other react-markdown overrides (see NoteEditor).
const mdComponents: Components = {
  a: (props: ComponentPropsWithoutRef<'a'>) => (
    <a
      {...props}
      target="_blank"
      rel="noopener noreferrer"
      className="text-state-running hover:underline break-words"
    />
  ),
  pre: (props: ComponentPropsWithoutRef<'pre'>) => (
    <pre
      {...props}
      className="overflow-x-auto rounded bg-card/60 p-2 text-[11px]"
    />
  ),
  img: (props: ComponentPropsWithoutRef<'img'>) => (
    <img {...props} alt={props.alt ?? ''} className="max-w-full rounded" />
  ),
}

// PRDetailDrawer is the right-side panel shown when a PR row is
// clicked. It works for every PR state: the description (markdown), a
// status badge, and CI checks render for open / closed / merged
// alike; the Merge action is only offered while the PR is open.
//
// Mounted into a portal on document.body so it overlays the whole
// viewport rather than being clipped by the inspector sidebar.
function PRDetailDrawer({
  cwd,
  pr,
  onClose,
  onMerged,
}: {
  cwd: string
  pr: PR
  onClose: () => void
  onMerged: () => void
}) {
  const [method, setMethod] = useState<'squash' | 'merge' | 'rebase'>('squash')
  const [deleteBranch, setDeleteBranch] = useState(true)

  // Close on Escape.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  // Full PR incl. body. Seeded from the list row so the header paints
  // instantly; the body fills in when the single-PR fetch resolves
  // (the list endpoint omits body to stay lean).
  const detail = useQuery({
    queryKey: ['git-pr', cwd, pr.number],
    queryFn: () => getGitPR(cwd, pr.number),
    // placeholderData (not initialData) seeds the header instantly from
    // the list row while still flagging the body fetch as in-flight via
    // isPlaceholderData. initialData would mark the query "fresh" and
    // suppress the loading state, flashing "no description" first.
    placeholderData: pr,
  })
  const full = detail.data ?? pr

  const checks = useQuery({
    queryKey: ['git-pr-checks', cwd, pr.number],
    queryFn: () => getPRChecks(cwd, pr.number),
    refetchInterval: 30_000,
  })

  const merge = useMutation({
    mutationFn: () =>
      mergeGitPR({
        dir: cwd,
        number: pr.number,
        method,
        delete_branch: deleteBranch,
      }),
    onSuccess: (mergedPR) => {
      toast.success(`PR #${pr.number} merged`, {
        description: mergedPR.title,
      })
      onMerged()
    },
    onError: (err) => {
      toast.error('Merge failed', {
        description: (err as Error).message,
      })
    },
  })

  // Aggregate check status for the headline summary. Pending while any
  // check is still queued / in_progress; failed if any concluded with
  // anything other than success / neutral / skipped.
  const checkSummary = aggregateChecks(checks.data ?? [])
  // body is undefined until the detail fetch resolves; '' means the PR
  // genuinely has no description.
  const body = full.body?.trim() ?? ''
  // While the query is still serving the seeded list row, the body
  // hasn't been fetched yet — drives the description loading state.
  const bodyPending = detail.isPlaceholderData

  return createPortal(
    <div className="fixed inset-0 z-[60] flex justify-end">
      <div
        aria-hidden="true"
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px]"
        onClick={onClose}
      />
      <div
        role="dialog"
        aria-modal="true"
        className="relative h-full w-full max-w-md bg-background border-l border-border shadow-2xl flex flex-col"
      >
        {/* Header */}
        <div className="flex items-start gap-2 px-3 py-2.5 border-b border-border">
          <PRStateIcon pr={full} className="mt-1" />
          <div className="min-w-0 flex-1">
            <div className="text-[13px] font-medium leading-snug break-words">
              {full.title}
            </div>
            <div className="text-[10px] text-muted-foreground/70 font-mono mt-0.5">
              #{full.number} · {full.author}
            </div>
          </div>
          <a
            href={full.url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-muted-foreground/50 hover:text-foreground p-0.5"
            title="Open on host"
          >
            <ExternalLink className="size-3.5" />
          </a>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="text-muted-foreground/50 hover:text-foreground p-0.5"
          >
            <X className="size-4" />
          </button>
        </div>

        {/* Meta row */}
        <div className="px-3 py-2 border-b border-border/50 flex items-center gap-2 flex-wrap text-[11px]">
          <StateBadge pr={full} />
          <span className="font-mono text-muted-foreground/80 truncate">
            {full.head} → {full.base}
          </span>
          <span className="text-muted-foreground/50">
            · {relTime(full.updated_at)}
          </span>
        </div>

        {/* Scrollable content */}
        <div className="flex-1 overflow-y-auto px-3 py-3 flex flex-col gap-4">
          {/* Description */}
          <section className="flex flex-col gap-1.5">
            <h3 className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
              Description
            </h3>
            {detail.isError ? (
              <div className="text-[11px] text-state-failed">
                Couldn't load details: {(detail.error as Error).message}
              </div>
            ) : bodyPending ? (
              <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
                <Loader2 className="size-3 animate-spin" />
                Loading…
              </div>
            ) : body ? (
              <div className="prose-md text-[12px] leading-relaxed break-words">
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  components={mdComponents}
                >
                  {body}
                </ReactMarkdown>
              </div>
            ) : (
              <div className="text-[11px] text-muted-foreground/60 italic">
                No description provided.
              </div>
            )}
          </section>

          {/* Checks — shown for every state, not just open */}
          <section className="flex flex-col gap-1.5">
            <h3 className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
              Checks
            </h3>
            {checks.isLoading && (
              <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
                <Loader2 className="size-3 animate-spin" />
                Loading checks…
              </div>
            )}
            {checks.error && (
              <div className="text-[11px] text-state-failed">
                checks unavailable: {(checks.error as Error).message}
              </div>
            )}
            {!checks.isLoading && !checks.error && (
              <div className="flex flex-col gap-0.5 text-[11px]">
                {(checks.data ?? []).length === 0 ? (
                  <div className="text-muted-foreground/60">
                    No checks configured for this PR.
                  </div>
                ) : (
                  <>
                    <div
                      className={cn(
                        'flex items-center gap-1.5',
                        checkSummary.allPassing && 'text-state-running',
                        checkSummary.anyFailed && 'text-state-failed',
                        checkSummary.pending && 'text-muted-foreground',
                      )}
                    >
                      {checkSummary.allPassing && (
                        <CheckCircle2 className="size-3" />
                      )}
                      {checkSummary.anyFailed && <XCircle className="size-3" />}
                      {checkSummary.pending && (
                        <CircleDashed className="size-3 animate-spin" />
                      )}
                      <span>{checkSummary.label}</span>
                    </div>
                    <div className="pl-4 flex flex-col gap-0.5 text-muted-foreground/80">
                      {(checks.data ?? []).map((c) => (
                        <a
                          key={c.name + c.url}
                          href={c.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="flex items-center gap-1.5 hover:text-foreground"
                        >
                          <CheckIcon check={c} />
                          <span className="truncate">{c.name}</span>
                        </a>
                      ))}
                    </div>
                  </>
                )}
              </div>
            )}
          </section>

          {/* Merge — only while the PR is open */}
          {full.state === 'open' && (
            <section className="flex flex-col gap-1.5">
              <h3 className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
                Merge
              </h3>
              <div className="flex flex-wrap items-center gap-2 text-[11px]">
                <select
                  value={method}
                  onChange={(e) =>
                    setMethod(e.target.value as 'squash' | 'merge' | 'rebase')
                  }
                  className="text-[11px] bg-transparent border border-border rounded px-1.5 py-0.5"
                >
                  <option value="squash">squash</option>
                  <option value="merge">merge</option>
                  <option value="rebase">rebase</option>
                </select>
                <label className="flex items-center gap-1 text-muted-foreground">
                  <input
                    type="checkbox"
                    checked={deleteBranch}
                    onChange={(e) => setDeleteBranch(e.target.checked)}
                  />
                  Delete branch
                </label>
                <button
                  type="button"
                  disabled={merge.isPending}
                  onClick={() => {
                    if (
                      !window.confirm(
                        `Merge PR #${full.number} (${method})${
                          deleteBranch ? ' and delete branch' : ''
                        }?`,
                      )
                    ) {
                      return
                    }
                    merge.mutate()
                  }}
                  className={cn(
                    'ml-auto text-[11px] px-2 py-0.5 rounded transition-colors flex items-center gap-1',
                    merge.isPending
                      ? 'bg-muted/30 text-muted-foreground/50'
                      : 'bg-purple-500/80 text-background hover:bg-purple-500',
                  )}
                >
                  {merge.isPending ? (
                    <Loader2 className="size-3 animate-spin" />
                  ) : (
                    <GitMerge className="size-3" />
                  )}
                  Merge
                </button>
              </div>
            </section>
          )}
        </div>
      </div>
    </div>,
    document.body,
  )
}

function CheckIcon({ check }: { check: CheckRun }) {
  if (check.status !== 'completed') {
    return <CircleDashed className="size-3 animate-spin shrink-0" />
  }
  switch (check.conclusion) {
    case 'success':
    case 'neutral':
    case 'skipped':
      return <CheckCircle2 className="size-3 text-state-running shrink-0" />
    case 'failure':
    case 'cancelled':
    case 'timed_out':
    case 'action_required':
      return <XCircle className="size-3 text-state-failed shrink-0" />
    default:
      return <CircleDashed className="size-3 shrink-0" />
  }
}

function aggregateChecks(checks: CheckRun[]): {
  label: string
  allPassing: boolean
  anyFailed: boolean
  pending: boolean
} {
  if (checks.length === 0) {
    return { label: '', allPassing: false, anyFailed: false, pending: false }
  }
  let pending = 0
  let failed = 0
  let passed = 0
  for (const c of checks) {
    if (c.status !== 'completed') {
      pending++
      continue
    }
    if (
      c.conclusion === 'success' ||
      c.conclusion === 'neutral' ||
      c.conclusion === 'skipped'
    ) {
      passed++
    } else {
      failed++
    }
  }
  if (pending > 0) {
    return {
      label: `${pending} check${pending === 1 ? '' : 's'} pending · ${passed} passed${failed > 0 ? ` · ${failed} failed` : ''}`,
      allPassing: false,
      anyFailed: false,
      pending: true,
    }
  }
  if (failed > 0) {
    return {
      label: `${failed} check${failed === 1 ? '' : 's'} failed · ${passed} passed`,
      allPassing: false,
      anyFailed: true,
      pending: false,
    }
  }
  return {
    label: `All ${passed} check${passed === 1 ? '' : 's'} passed`,
    allPassing: true,
    anyFailed: false,
    pending: false,
  }
}

function NeedTokenHint({ host }: { host: string }) {
  return (
    <div className="rounded-md border border-dashed border-border bg-card/40 p-2.5 flex flex-col gap-2">
      <div className="flex items-start gap-2 text-[11px] text-muted-foreground">
        <KeyRound className="size-3.5 mt-0.5 text-muted-foreground/60 shrink-0" />
        <span>
          No token configured for <span className="font-mono">{host}</span>. Add
          one to fetch pull requests.
        </span>
      </div>
      <Link
        to="/plugins"
        className="text-[11px] text-state-running hover:underline self-start"
      >
        Configure git host →
      </Link>
    </div>
  )
}

function relTime(iso: string): string {
  try {
    return formatDistanceToNow(new Date(iso), { addSuffix: true })
  } catch {
    return iso
  }
}
