import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  GitPullRequest,
  GitPullRequestDraft,
  Loader2,
  ExternalLink,
  KeyRound,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'

import { listGitPRs } from '@/lib/githost'
import { cn } from '@/lib/utils'

interface PullRequestsSectionProps {
  cwd: string
}

// PullRequestsSection sits in GitPanel under the working tree. Tries
// `GET /git/prs?path=<cwd>` — backend detects the remote and either
// returns PRs from the matching git_hosts row or `need_token: true`,
// which we render as a deep link to /plugins.
export function PullRequestsSection({ cwd }: PullRequestsSectionProps) {
  const [state, setState] = useState<'open' | 'closed' | 'all'>('open')
  const { data, isLoading, error } = useQuery({
    queryKey: ['git-prs', cwd, state],
    queryFn: () => listGitPRs(cwd, state),
    refetchInterval: 60_000,
    // Treat upstream API failures as data, not query errors — the
    // backend wraps them in {error: string} and a 200 so we can render
    // them inline. The `error` branch here is for transport failures.
  })

  return (
    <section className="flex flex-col gap-1">
      <div className="flex items-center justify-between px-1">
        <div className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium flex items-center gap-1.5">
          <GitPullRequest className="size-3" />
          Pull requests
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
      {data && data.need_token && (
        <NeedTokenHint host={data.remote.host} />
      )}
      {data && !data.error && !data.need_token && data.prs.length === 0 && (
        <div className="text-[11px] text-muted-foreground/60 px-1 py-1">
          No {state === 'all' ? '' : `${state} `}PRs.
        </div>
      )}
      {data && !data.need_token && (
        <div className="flex flex-col">
          {data.prs.map((p) => (
            <a
              key={p.number}
              href={p.url}
              target="_blank"
              rel="noopener noreferrer"
              className="px-1 py-1.5 flex items-start gap-2 hover:bg-card rounded-sm group"
              title={`#${p.number} · ${p.author} · ${p.head} → ${p.base}`}
            >
              {p.draft ? (
                <GitPullRequestDraft className="size-3 mt-0.5 text-muted-foreground/60 shrink-0" />
              ) : (
                <GitPullRequest
                  className={cn(
                    'size-3 mt-0.5 shrink-0',
                    p.state === 'merged'
                      ? 'text-purple-400'
                      : p.state === 'closed'
                        ? 'text-state-failed'
                        : 'text-state-running',
                  )}
                />
              )}
              <div className="flex flex-col min-w-0 flex-1">
                <span className="text-[12px] truncate group-hover:text-foreground">
                  {p.title}
                </span>
                <span className="text-[10px] text-muted-foreground/70 font-mono truncate">
                  #{p.number} · {p.author} · {p.head} → {p.base} ·{' '}
                  {relTime(p.updated_at)}
                </span>
              </div>
              <ExternalLink className="size-3 text-muted-foreground/50 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity mt-0.5" />
            </a>
          ))}
        </div>
      )}
    </section>
  )
}

function NeedTokenHint({ host }: { host: string }) {
  return (
    <div className="rounded-md border border-dashed border-border bg-card/40 p-2.5 flex flex-col gap-2">
      <div className="flex items-start gap-2 text-[11px] text-muted-foreground">
        <KeyRound className="size-3.5 mt-0.5 text-muted-foreground/60 shrink-0" />
        <span>
          No token configured for <span className="font-mono">{host}</span>.
          Add one to fetch pull requests.
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
