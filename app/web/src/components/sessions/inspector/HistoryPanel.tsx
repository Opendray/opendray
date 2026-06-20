import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Copy, Send, Search as SearchIcon, History as HistoryIcon } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { fetchSessionHistory, resendInput, type HistoryEntry } from '@/lib/sessions'
import type { Session } from '@/lib/types'
import { copyText } from '@/lib/clipboard'

// HistoryPanel — Inspector tab. Lists every prompt the operator has
// sent in this *project* (cwd), pooled across every Claude session
// ever spawned there. Useful for vibe-coding: copy a past prompt,
// or resend it into the live session with one click.
//
// Source of truth is Claude's own JSONL transcripts under
// ~/.claude/projects/<encoded-cwd>/*.jsonl, which already capture
// inputs from every channel (terminal, telegram forward, etc.).
export function HistoryPanel({ session }: { session: Session }) {
  const [filter, setFilter] = useState('')

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['session-history', session.id],
    queryFn: () => fetchSessionHistory(session.id, 200),
    refetchInterval: 10_000,
  })

  const entries = data?.entries ?? []
  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase()
    if (!q) return entries
    return entries.filter((e) => e.text.toLowerCase().includes(q))
  }, [entries, filter])

  if (data?.unsupported_provider) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-12 text-center text-xs text-muted-foreground">
        <HistoryIcon className="size-6 opacity-40" />
        <p>History isn't available for this provider.</p>
        <p className="opacity-60">
          <code>{session.provider_id}</code> doesn't write a structured
          transcript on disk. Supported: claude, codex.
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="relative">
        <SearchIcon className="absolute left-2 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground" />
        <Input
          placeholder="Filter prompts…"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="h-7 pl-7 text-xs"
        />
      </div>

      {isLoading && (
        <p className="text-[11px] text-muted-foreground/70 px-1">Loading history…</p>
      )}
      {isError && (
        <p className="text-[11px] text-destructive px-1">
          Failed to load history: {(error as Error)?.message ?? 'unknown error'}
        </p>
      )}
      {!isLoading && !isError && entries.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-2 py-10 text-center text-xs text-muted-foreground">
          <HistoryIcon className="size-6 opacity-40" />
          <p>No prompts yet for this project.</p>
          <p className="opacity-60">
            Send something to Claude — it will show up here.
          </p>
        </div>
      )}
      {!isLoading && !isError && entries.length > 0 && filtered.length === 0 && (
        <p className="text-[11px] text-muted-foreground/70 px-1">
          No prompts match “{filter}”.
        </p>
      )}

      <ul className="flex flex-col gap-1.5">
        {filtered.map((entry, i) => (
          <HistoryRow
            key={`${entry.session_id}-${entry.ts}-${i}`}
            entry={entry}
            sessionID={session.id}
          />
        ))}
      </ul>
    </div>
  )
}

interface RowProps {
  entry: HistoryEntry
  sessionID: string
}

function HistoryRow({ entry, sessionID }: RowProps) {
  const resend = useMutation({
    mutationFn: () => resendInput(sessionID, entry.text),
    onSuccess: () =>
      toast.success('Resent', {
        description: 'Prompt forwarded to the live session.',
      }),
    onError: (err: Error) =>
      toast.error('Resend failed', { description: err.message }),
  })

  const copy = async () => {
    try {
      if (!(await copyText(entry.text))) throw new Error('clipboard unavailable')
      toast.success('Copied prompt to clipboard')
    } catch (err) {
      toast.error('Copy failed', {
        description: (err as Error)?.message ?? 'clipboard unavailable',
      })
    }
  }

  const relative = entry.ts
    ? formatDistanceToNow(new Date(entry.ts), { addSuffix: true })
    : ''

  return (
    <li className="rounded border border-border bg-card/40 px-2 py-1.5 group">
      <div className="flex items-start justify-between gap-2">
        <pre className="flex-1 text-xs whitespace-pre-wrap break-words leading-snug font-sans text-foreground">
          {entry.text}
        </pre>
        <div className="flex flex-col gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-6"
            title="Copy prompt"
            onClick={copy}
          >
            <Copy className="size-3" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-6"
            title="Resend to current session"
            disabled={resend.isPending}
            onClick={() => resend.mutate()}
          >
            <Send className="size-3" />
          </Button>
        </div>
      </div>
      <div className="flex items-center justify-between gap-2 mt-1 text-[10px] text-muted-foreground/70">
        <span>{relative}</span>
        <span
          className="font-mono opacity-70 truncate max-w-[60%]"
          title={`Claude session ${entry.session_id}`}
        >
          {entry.session_id.slice(0, 8)}
        </span>
      </div>
    </li>
  )
}
