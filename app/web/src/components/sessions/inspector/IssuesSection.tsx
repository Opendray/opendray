import { type ComponentPropsWithoutRef, useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  CircleDot,
  CheckCircle2,
  Loader2,
  ExternalLink,
  KeyRound,
  X,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import ReactMarkdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'

import {
  type GitIssue as Issue,
  type GitLabel,
  type PRComment,
  getGitIssue,
  getIssueComments,
  listGitIssues,
} from '@/lib/githost'
import { cn } from '@/lib/utils'

interface IssuesSectionProps {
  cwd: string
}

// IssuesSection sits in GitPanel directly below PullRequestsSection. It
// mirrors the PR surface but read-only: `GET /git/issues?path=<cwd>`
// returns issues from the matching git_hosts row or `need_token: true`,
// rendered as a deep link to /plugins. GitHub/Gitea expose issues and
// PRs through one endpoint; the backend filters PRs out, so this list is
// issues only.
//
// Each row opens a right-side detail drawer (description + labels +
// comment thread). There is no create / close / comment action — issues
// are view-only here.
export function IssuesSection({ cwd }: IssuesSectionProps) {
  const [state, setState] = useState<'open' | 'closed' | 'all'>('open')
  // Hold the row object (not just the number) so the drawer survives the
  // 60s list refetch — see PullRequestsSection for the rationale.
  const [detailIssue, setDetailIssue] = useState<Issue | null>(null)

  const { data, isLoading, error } = useQuery({
    queryKey: ['git-issues', cwd, state],
    queryFn: () => listGitIssues(cwd, state),
    refetchInterval: 60_000,
  })

  return (
    <section className="flex flex-col gap-1">
      <div className="flex items-center justify-between px-1">
        <div className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium flex items-center gap-1.5">
          <CircleDot className="size-3" />
          Issues
        </div>
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
      {data && !data.error && !data.need_token && data.issues.length === 0 && (
        <div className="text-[11px] text-muted-foreground/60 px-1 py-1">
          No {state === 'all' ? '' : `${state} `}issues.
        </div>
      )}
      {data && !data.need_token && (
        <div className="flex flex-col">
          {data.issues.map((it) => (
            <IssueRow
              key={it.number}
              issue={it}
              onOpen={() => setDetailIssue(it)}
            />
          ))}
        </div>
      )}

      {detailIssue && (
        <IssueDetailDrawer
          cwd={cwd}
          issue={detailIssue}
          onClose={() => setDetailIssue(null)}
        />
      )}
    </section>
  )
}

// IssueRow is the per-issue card. Click opens the detail drawer; the
// external-link icon jumps to the host's web view.
function IssueRow({ issue, onOpen }: { issue: Issue; onOpen: () => void }) {
  return (
    <div className="border-b border-border/30 last:border-b-0">
      <button
        type="button"
        onClick={onOpen}
        className="w-full px-1 py-1.5 flex items-start gap-2 hover:bg-card rounded-sm group text-left"
        title={`#${issue.number} · ${issue.author}`}
      >
        <IssueStateIcon issue={issue} className="mt-0.5" />
        <div className="flex flex-col min-w-0 flex-1 gap-0.5">
          <span className="text-[12px] truncate group-hover:text-foreground">
            {issue.title}
          </span>
          <span className="text-[10px] text-muted-foreground/70 font-mono truncate">
            #{issue.number} · {issue.author} · {relTime(issue.updated_at)}
          </span>
          {issue.labels.length > 0 && <LabelChips labels={issue.labels} />}
        </div>
        <a
          href={issue.url}
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

// IssueStateIcon renders the issue glyph coloured by state. Open issues
// use the green open-circle; closed issues use a purple check (matching
// the host convention of "closed = resolved").
function IssueStateIcon({
  issue,
  className,
}: {
  issue: Issue
  className?: string
}) {
  if (issue.state === 'closed') {
    return (
      <CheckCircle2 className={cn('size-3 text-purple-400 shrink-0', className)} />
    )
  }
  return (
    <CircleDot className={cn('size-3 text-state-running shrink-0', className)} />
  )
}

// StateBadge is the textual status pill in the drawer header.
function StateBadge({ issue }: { issue: Issue }) {
  const { label, cls } =
    issue.state === 'closed'
      ? { label: 'Closed', cls: 'text-purple-400 border-purple-400/40' }
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

// LabelChips renders the issue's labels as small pills. GitHub/Gitea
// supply a hex colour (no leading '#'); GitLab leaves it empty, in which
// case we fall back to a muted border-only chip.
function LabelChips({ labels }: { labels: GitLabel[] }) {
  return (
    <div className="flex flex-wrap gap-1">
      {labels.map((l) => {
        const color = l.color ? `#${l.color}` : undefined
        return (
          <span
            key={l.name}
            className={cn(
              'text-[9px] px-1.5 py-px rounded-full border leading-tight',
              !color && 'border-border text-muted-foreground/80',
            )}
            style={
              color
                ? {
                    borderColor: color,
                    color,
                    backgroundColor: `${color}1a`, // ~10% alpha
                  }
                : undefined
            }
          >
            {l.name}
          </span>
        )
      })}
    </div>
  )
}

// mdComponents keeps rendered markdown inside the panel: links open in a
// new tab, code blocks scroll, images are width-constrained. Mirrors the
// PR drawer's overrides.
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

// IssueDetailDrawer is the right-side panel shown when an issue row is
// clicked. Read-only: a single scrolling pane with the description
// (markdown) followed by the comment thread. No tabs, no footer action.
//
// Mounted into a portal on document.body so it overlays the whole
// viewport rather than being clipped by the inspector sidebar.
function IssueDetailDrawer({
  cwd,
  issue,
  onClose,
}: {
  cwd: string
  issue: Issue
  onClose: () => void
}) {
  // Close on Escape.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  // Full issue incl. body. Seeded from the list row so the header paints
  // instantly; the body fills in when the single-issue fetch resolves
  // (the list endpoint omits body to stay lean). See PR drawer for why
  // placeholderData (not initialData) is used.
  const detail = useQuery({
    queryKey: ['git-issue', cwd, issue.number],
    queryFn: () => getGitIssue(cwd, issue.number),
    placeholderData: issue,
  })
  const full = detail.data ?? issue

  const comments = useQuery({
    queryKey: ['git-issue-comments', cwd, issue.number],
    queryFn: () => getIssueComments(cwd, issue.number),
    staleTime: 5 * 60 * 1000,
  })

  const body = full.body?.trim() ?? ''
  const bodyPending = detail.isPlaceholderData
  const detailError = detail.isError ? (detail.error as Error).message : null
  const list = comments.data ?? []

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
        className="relative h-full w-full max-w-2xl bg-background border-l border-border shadow-2xl flex flex-col"
      >
        {/* Header */}
        <div className="flex items-start gap-2 px-3 py-2.5 border-b border-border">
          <IssueStateIcon issue={full} className="mt-1" />
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
          <StateBadge issue={full} />
          {full.labels.length > 0 && <LabelChips labels={full.labels} />}
          <span className="text-muted-foreground/50">
            · {relTime(full.updated_at)}
          </span>
        </div>

        {/* Body: description + comment thread */}
        <div className="flex-1 overflow-y-auto px-3 py-3">
          <div className="flex flex-col gap-3">
            <div className="rounded border border-border/50 bg-card/30">
              <div className="px-2.5 py-1.5 border-b border-border/40 text-[10px] text-muted-foreground/80">
                <span className="font-mono text-foreground/90">
                  {full.author}
                </span>{' '}
                opened this issue
              </div>
              <div className="px-2.5 py-2">
                {detailError ? (
                  <div className="text-[11px] text-state-failed">
                    Couldn't load details: {detailError}
                  </div>
                ) : bodyPending ? (
                  <TabLoading />
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
              </div>
            </div>

            {comments.isLoading && <TabLoading text="Loading comments…" />}
            {comments.error && (
              <div className="text-[11px] text-state-failed py-2 break-words">
                comments unavailable: {(comments.error as Error).message}
              </div>
            )}
            {!comments.isLoading && !comments.error && list.length === 0 && (
              <div className="text-[11px] text-muted-foreground/60 py-2">
                No comments yet.
              </div>
            )}
            {list.map((c, i) => (
              <CommentItem
                key={c.url ?? `${c.author}-${c.created_at}-${i}`}
                c={c}
              />
            ))}
          </div>
        </div>
      </div>
    </div>,
    document.body,
  )
}

function TabLoading({ text = 'Loading…' }: { text?: string }) {
  return (
    <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground py-2">
      <Loader2 className="size-3 animate-spin" />
      {text}
    </div>
  )
}

function CommentItem({ c }: { c: PRComment }) {
  return (
    <div className="rounded border border-border/50 bg-card/30">
      <div className="flex items-center gap-2 px-2.5 py-1.5 border-b border-border/40 text-[10px] text-muted-foreground/80">
        <span className="font-mono text-foreground/90">{c.author}</span>
        <span className="ml-auto">{relTime(c.created_at)}</span>
      </div>
      <div className="prose-md text-[12px] leading-relaxed break-words px-2.5 py-2">
        <ReactMarkdown remarkPlugins={[remarkGfm]} components={mdComponents}>
          {c.body}
        </ReactMarkdown>
      </div>
    </div>
  )
}

function NeedTokenHint({ host }: { host: string }) {
  return (
    <div className="rounded-md border border-dashed border-border bg-card/40 p-2.5 flex flex-col gap-2">
      <div className="flex items-start gap-2 text-[11px] text-muted-foreground">
        <KeyRound className="size-3.5 mt-0.5 text-muted-foreground/60 shrink-0" />
        <span>
          No token configured for <span className="font-mono">{host}</span>. Add
          one to fetch issues.
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
