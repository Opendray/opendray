import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  type MemoryRecord,
  type SearchHit,
  listMemories,
  MobileAPIError,
  searchMemories,
} from '../lib/api'

interface Props {
  serverURL: string
  token: string
  onAuthExpired: () => void
}

type Mode =
  | { kind: 'list'; loading: boolean; items: MemoryRecord[]; error?: string }
  | { kind: 'search'; loading: boolean; hits: SearchHit[]; error?: string }

// Memory facts viewer + search. Read-only on mobile (creates / edits
// happen on desktop or via the agent's MCP tools). Default scope is
// `global` — that's what most user-typed facts end up in.
export function MemoryScreen({ serverURL, token, onAuthExpired }: Props) {
  const [mode, setMode] = useState<Mode>({ kind: 'list', loading: true, items: [] })
  const [query, setQuery] = useState('')

  const loadList = useCallback(async () => {
    setMode({ kind: 'list', loading: true, items: [] })
    try {
      const items = await listMemories(serverURL, token, { scope: 'global', n: 100 })
      setMode({ kind: 'list', loading: false, items })
    } catch (err) {
      if (err instanceof MobileAPIError && err.status === 401) {
        onAuthExpired()
        return
      }
      setMode({
        kind: 'list',
        loading: false,
        items: [],
        error: err instanceof Error ? err.message : 'Failed to load memories',
      })
    }
  }, [serverURL, token, onAuthExpired])

  useEffect(() => {
    void loadList()
  }, [loadList])

  async function runSearch(e: React.FormEvent) {
    e.preventDefault()
    const q = query.trim()
    if (!q) {
      void loadList()
      return
    }
    setMode({ kind: 'search', loading: true, hits: [] })
    try {
      const hits = await searchMemories(serverURL, token, q, 'global', 30)
      setMode({ kind: 'search', loading: false, hits })
    } catch (err) {
      if (err instanceof MobileAPIError && err.status === 401) {
        onAuthExpired()
        return
      }
      setMode({
        kind: 'search',
        loading: false,
        hits: [],
        error: err instanceof Error ? err.message : 'Search failed',
      })
    }
  }

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <header className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-4 py-3">
        <h1 className="text-lg font-semibold leading-tight">Memory</h1>
        <form onSubmit={runSearch} className="mt-2 flex gap-2">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search facts…"
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
            inputMode="search"
          />
          <Button type="submit" variant="outline" size="sm">
            {query ? 'Search' : 'List'}
          </Button>
        </form>
      </header>
      <main className="flex-1 px-4 py-3">
        <Body mode={mode} onRetry={loadList} />
      </main>
    </div>
  )
}

function Body({ mode, onRetry }: { mode: Mode; onRetry: () => void }) {
  if (mode.loading) {
    return <div className="text-sm text-muted-foreground">Loading…</div>
  }
  if (mode.error) {
    return (
      <div className="space-y-3">
        <div className="rounded-md border border-destructive/30 bg-destructive/10 text-destructive text-sm px-3 py-2">
          {mode.error}
        </div>
        <Button variant="outline" size="sm" onClick={onRetry}>
          Retry
        </Button>
      </div>
    )
  }
  if (mode.kind === 'list') {
    if (mode.items.length === 0) {
      return (
        <Empty>
          No global memories yet. Talk to a CLI session and ask the agent to
          remember something — it&rsquo;ll show up here.
        </Empty>
      )
    }
    return (
      <ul className="space-y-2">
        {mode.items.map((m) => (
          <FactCard key={m.id} text={m.text} updated={m.updated_at} />
        ))}
      </ul>
    )
  }
  // search
  if (mode.hits.length === 0) {
    return <Empty>No matches.</Empty>
  }
  return (
    <ul className="space-y-2">
      {mode.hits.map((h) => (
        <FactCard
          key={h.record.id}
          text={h.record.text}
          updated={h.record.updated_at}
          similarity={h.similarity}
        />
      ))}
    </ul>
  )
}

function FactCard({
  text,
  updated,
  similarity,
}: {
  text: string
  updated: string
  similarity?: number
}) {
  return (
    <li className="rounded-md border border-border bg-card p-3 text-sm space-y-1">
      <div className="whitespace-pre-wrap break-words">{text}</div>
      <div className="text-[11px] text-muted-foreground flex items-center justify-between gap-2">
        <span>{formatRelative(updated)}</span>
        {similarity !== undefined && (
          <span className="tabular-nums">
            {(similarity * 100).toFixed(0)}% match
          </span>
        )}
      </div>
    </li>
  )
}

function Empty({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-md border border-border bg-card p-6 text-center text-sm text-muted-foreground">
      {children}
    </div>
  )
}

function formatRelative(iso: string): string {
  const ts = Date.parse(iso)
  if (Number.isNaN(ts)) return iso
  const seconds = Math.round((Date.now() - ts) / 1000)
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.round(hours / 24)
  return `${days}d ago`
}
