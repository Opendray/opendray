import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  GitBranch,
  GitCommit,
  ArrowUp,
  ArrowDown,
  Loader2,
} from 'lucide-react'

import { getGitStatus, getGitLog } from '@/lib/git'
import type { GitStatusFile } from '@/lib/git'
import { cn } from '@/lib/utils'

import { BranchControls } from './BranchControls'
import { CommitForm } from './CommitForm'
import { DiffViewer } from './DiffViewer'
import { PullRequestsSection } from './PullRequestsSection'

interface GitPanelProps {
  cwd: string
}

type DiffTarget =
  | { kind: 'file'; cwd: string; file: string }
  | { kind: 'commit'; cwd: string; hash: string; subject?: string }

// GitPanel surfaces a read-only view of the repo at session.cwd.
// Backend shells out to `git status --porcelain=v1 -b -z` and
// `git log --pretty=format:...`. Refreshes on a slow poll so the
// panel reflects what's happening in the terminal next door.
export function GitPanel({ cwd }: GitPanelProps) {
  const [target, setTarget] = useState<DiffTarget | null>(null)
  const status = useQuery({
    queryKey: ['git', 'status', cwd],
    queryFn: () => getGitStatus(cwd),
    refetchInterval: 8_000,
  })
  const log = useQuery({
    queryKey: ['git', 'log', cwd],
    queryFn: () => getGitLog(cwd, 15),
    refetchInterval: 30_000,
    enabled: status.data?.is_repo === true,
  })

  if (status.isLoading) {
    return (
      <div className="flex items-center gap-2 text-[12px] text-muted-foreground py-3">
        <Loader2 className="size-3 animate-spin" />
        Reading repo…
      </div>
    )
  }
  if (status.error) {
    return (
      <div className="text-[12px] text-state-failed py-3">
        {(status.error as Error).message}
      </div>
    )
  }
  if (!status.data) return null
  if (!status.data.is_repo) {
    return (
      <div className="flex flex-col items-center text-center gap-2 py-6 text-muted-foreground">
        <GitBranch className="size-5 opacity-40" strokeWidth={1.5} />
        <div className="text-[12px]">Not a git repository.</div>
        <div className="text-[10px] opacity-70 max-w-[220px] font-mono">
          {cwd}
        </div>
      </div>
    )
  }

  const s = status.data
  return (
    <>
    <div className="flex flex-col gap-3">
      <section className="rounded-md border border-border/60 bg-card/40 p-3 flex flex-col gap-2">
        <div className="flex items-center gap-2 text-[12px]">
          <GitBranch className="size-3.5 text-muted-foreground" />
          <span className="font-medium font-mono">{s.branch || '—'}</span>
          {s.upstream && (
            <span className="text-[10px] text-muted-foreground/70 font-mono truncate">
              ↦ {s.upstream}
            </span>
          )}
          <div className="flex-1" />
          {s.ahead > 0 && (
            <span className="flex items-center gap-0.5 text-[10px] text-state-running font-mono">
              <ArrowUp className="size-3" />
              {s.ahead}
            </span>
          )}
          {s.behind > 0 && (
            <span className="flex items-center gap-0.5 text-[10px] text-state-idle font-mono">
              <ArrowDown className="size-3" />
              {s.behind}
            </span>
          )}
        </div>
        <FileSummary files={s.files} />
        <BranchControls cwd={cwd} ahead={s.ahead} upstream={s.upstream} />
      </section>

      {s.files.length > 0 && (
        <section className="flex flex-col gap-1">
          <div className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium px-1">
            Working tree
          </div>
          <div className="flex flex-col">
            {s.files.slice(0, 50).map((f) => (
              <FileRow
                key={f.path}
                f={f}
                onClick={() =>
                  setTarget({ kind: 'file', cwd, file: f.path })
                }
              />
            ))}
            {s.files.length > 50 && (
              <div className="text-[10px] text-muted-foreground/60 px-1 py-1">
                +{s.files.length - 50} more
              </div>
            )}
          </div>
          <CommitForm cwd={cwd} files={s.files} />
        </section>
      )}

      <PullRequestsSection cwd={cwd} />

      <section className="flex flex-col gap-1">
        <div className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium px-1">
          Recent commits
        </div>
        {log.isLoading ? (
          <div className="flex items-center gap-2 text-[11px] text-muted-foreground px-1 py-1">
            <Loader2 className="size-3 animate-spin" />
            Loading…
          </div>
        ) : (
          <div className="flex flex-col">
            {(log.data?.commits ?? []).map((c) => (
              <button
                key={c.hash}
                type="button"
                onClick={() =>
                  setTarget({
                    kind: 'commit',
                    cwd,
                    hash: c.hash,
                    subject: c.subject,
                  })
                }
                className="px-1 py-1 flex items-start gap-2 hover:bg-card rounded-sm text-left"
                title={`${c.hash}\n${c.author} · ${c.when}`}
              >
                <GitCommit className="size-3 mt-0.5 text-muted-foreground/60 shrink-0" />
                <div className="flex flex-col min-w-0 flex-1">
                  <span className="text-[12px] truncate">{c.subject}</span>
                  <span className="text-[10px] text-muted-foreground/60 font-mono">
                    {c.short_hash} · {c.author} · {c.when}
                  </span>
                </div>
              </button>
            ))}
            {log.data && log.data.commits.length === 0 && (
              <div className="text-[11px] text-muted-foreground/60 px-1 py-1">
                No commits.
              </div>
            )}
          </div>
        )}
      </section>
    </div>
    <DiffViewer
      open={target != null}
      onOpenChange={(v) => !v && setTarget(null)}
      target={target}
    />
    </>
  )
}

function FileSummary({ files }: { files: GitStatusFile[] }) {
  let staged = 0
  let modified = 0
  let untracked = 0
  for (const f of files) {
    const x = f.xy[0]
    const y = f.xy[1]
    if (x === '?' && y === '?') untracked++
    else {
      if (x !== ' ' && x !== '?') staged++
      if (y !== ' ' && y !== '?') modified++
    }
  }
  if (files.length === 0) {
    return (
      <div className="text-[11px] text-muted-foreground">Working tree clean.</div>
    )
  }
  return (
    <div className="flex items-center gap-3 text-[11px] font-mono">
      {staged > 0 && (
        <span className="text-state-running">{staged} staged</span>
      )}
      {modified > 0 && (
        <span className="text-state-idle">{modified} modified</span>
      )}
      {untracked > 0 && (
        <span className="text-muted-foreground">{untracked} untracked</span>
      )}
    </div>
  )
}

function FileRow({
  f,
  onClick,
}: {
  f: GitStatusFile
  onClick: () => void
}) {
  const x = f.xy[0]
  const y = f.xy[1]
  const isUntracked = x === '?' && y === '?'
  const colorIdx = isUntracked
    ? 'text-muted-foreground'
    : x !== ' '
      ? 'text-state-running'
      : 'text-state-idle'
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex items-center gap-2 px-1 py-0.5 hover:bg-card rounded-sm text-left"
      title={`${f.xy} ${f.path}${f.old_path ? ` (was ${f.old_path})` : ''}`}
    >
      <span
        className={cn(
          'shrink-0 inline-block w-5 text-center text-[10px] font-mono font-semibold',
          colorIdx,
        )}
      >
        {f.xy.replace(/ /g, '·')}
      </span>
      <span className="text-[11px] font-mono truncate flex-1">{f.path}</span>
    </button>
  )
}
