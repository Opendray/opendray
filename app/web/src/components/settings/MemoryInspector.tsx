import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Search as SearchIcon,
  Trash2,
  Loader2,
  Brain,
  CheckCircle2,
  XCircle,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import {
  deleteMemory,
  fetchMemoryStatus,
  listMemories,
  searchMemories,
  testEmbedder,
  type MemoryRecord,
  type SearchHit,
  type Scope,
} from '@/lib/memory'

// MemoryInspector shows the live state of opendray's memory
// subsystem: which embedder is active, how many dims it produces,
// and a browse / search / delete pane over the stored memories.
//
// Targeted scope is "project" by default (matches the system
// behaviour for newly-stored memories). The operator types a
// `scope_key` (a cwd) and we list memories under that scope.
export function MemoryInspector() {
  const qc = useQueryClient()
  const [scope, setScope] = useState<Scope>('project')
  const [scopeKey, setScopeKey] = useState<string>('')
  const [search, setSearch] = useState<string>('')
  const [searchHits, setSearchHits] = useState<SearchHit[] | null>(null)
  const [searchBusy, setSearchBusy] = useState(false)

  const { data: status, isError: statusError } = useQuery({
    queryKey: ['memory-status'],
    queryFn: fetchMemoryStatus,
    refetchInterval: 30_000,
  })

  const browseEnabled = scope === 'global' || !!scopeKey.trim()
  const browse = useQuery({
    queryKey: ['memory-list', scope, scopeKey],
    queryFn: () => listMemories(scope, scopeKey.trim(), 100),
    enabled: browseEnabled,
  })

  // Default scopeKey to a sensible candidate (the working directory
  // of the most-recent claude session in active list) the first time
  // the inspector mounts. Not destructive — operator can change.
  useEffect(() => {
    if (scopeKey) return
    // Lightly probe: pick the first session's cwd as a default.
    // Failing silently is fine; the user can type something.
    fetch('/api/v1/sessions', { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : null))
      .then((d: { sessions?: { cwd?: string }[] } | null) => {
        const cwd = d?.sessions?.[0]?.cwd
        if (cwd) setScopeKey(cwd)
      })
      .catch(() => {})
  }, [scopeKey])

  const runSearch = async () => {
    if (!search.trim()) {
      setSearchHits(null)
      return
    }
    setSearchBusy(true)
    try {
      const hits = await searchMemories({
        query: search.trim(),
        scope,
        scope_key: scopeKey.trim(),
        top_k: 10,
        // -1 disables the threshold so the operator sees raw scores
        // (useful for diagnosing "why didn't this match?")
        min_similarity: -1,
      })
      setSearchHits(hits)
    } catch (err) {
      toast.error('Search failed', { description: (err as Error).message })
      setSearchHits(null)
    } finally {
      setSearchBusy(false)
    }
  }

  const del = useMutation({
    mutationFn: (id: string) => deleteMemory(id),
    onSuccess: () => {
      toast.success('Memory deleted')
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      // Also drop from local search results if present.
      setSearchHits((cur) => cur?.filter((h) => h.memory.id !== arguments[0]) ?? null)
    },
    onError: (err: Error) => toast.error('Delete failed', { description: err.message }),
  })

  const test = useMutation({
    mutationFn: () => testEmbedder('opendray memory subsystem self-test'),
    onSuccess: (r) =>
      toast.success(`Embedder OK: ${r.embedder} · ${r.dim} dimensions`, {
        description: `vector_preview = [${r.vector_preview
          .slice(0, 4)
          .map((v) => v.toFixed(3))
          .join(', ')}…]`,
      }),
    onError: (err: Error) =>
      toast.error('Embedder probe failed', { description: err.message }),
  })

  const records = useMemo(() => {
    if (searchHits) return searchHits.map((h) => ({ memory: h.memory, similarity: h.similarity }))
    return (browse.data ?? []).map((m) => ({ memory: m as MemoryRecord, similarity: undefined }))
  }, [searchHits, browse.data])

  return (
    <div className="flex flex-col gap-4">
      {/* Status strip */}
      <div className="flex items-center gap-3 rounded-md border border-border bg-card/30 px-3 py-2">
        <Brain className="size-4 text-accent shrink-0" />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            {statusError ? (
              <Badge variant="danger">unavailable</Badge>
            ) : status ? (
              <>
                <Badge variant="success">{status.embedder}</Badge>
                <span className="text-[11px] text-muted-foreground">
                  {status.dimensions}-dim · {status.enabled ? 'enabled' : 'disabled'}
                </span>
              </>
            ) : (
              <span className="text-[11px] text-muted-foreground">probing…</span>
            )}
          </div>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-7 text-[11px]"
          onClick={() => test.mutate()}
          disabled={test.isPending}
        >
          {test.isPending ? <Loader2 className="size-3 animate-spin" /> : 'Test embedder'}
        </Button>
      </div>

      {/* Scope selector */}
      <div className="flex items-end gap-2 flex-wrap">
        <div className="space-y-1">
          <label className="text-[10px] text-muted-foreground/80 font-medium uppercase tracking-wider">
            Scope
          </label>
          <select
            value={scope}
            onChange={(e) => {
              setScope(e.target.value as Scope)
              setSearchHits(null)
            }}
            className="h-8 px-2 text-xs rounded border border-border bg-background"
          >
            <option value="project">project</option>
            <option value="session">session</option>
            <option value="global">global</option>
          </select>
        </div>
        <div className="flex-1 space-y-1 min-w-[280px]">
          <label className="text-[10px] text-muted-foreground/80 font-medium uppercase tracking-wider">
            Scope key {scope === 'global' && <span className="opacity-60">(ignored for global)</span>}
          </label>
          <Input
            value={scopeKey}
            onChange={(e) => {
              setScopeKey(e.target.value)
              setSearchHits(null)
            }}
            placeholder={scope === 'project' ? '/path/to/project (cwd)' : 'session id'}
            disabled={scope === 'global'}
            className="h-8 font-mono text-xs"
          />
        </div>
      </div>

      {/* Search */}
      <div className="flex gap-2">
        <div className="relative flex-1">
          <SearchIcon className="absolute left-2 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground/60" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && runSearch()}
            placeholder="Semantic search query (Enter to run; empty = browse)"
            className="h-8 pl-7 text-xs"
          />
        </div>
        <Button
          type="button"
          size="sm"
          onClick={runSearch}
          disabled={searchBusy}
          className="h-8 text-[11px]"
        >
          {searchBusy ? <Loader2 className="size-3 animate-spin" /> : 'Search'}
        </Button>
        {searchHits !== null && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              setSearch('')
              setSearchHits(null)
            }}
            className="h-8 text-[11px]"
          >
            Clear
          </Button>
        )}
      </div>

      {/* Records */}
      <div className="flex flex-col gap-1.5">
        {browse.isLoading && (
          <p className="text-[11px] text-muted-foreground/70 italic">Loading…</p>
        )}
        {!browseEnabled && (
          <p className="text-[11px] text-muted-foreground/70 italic">
            Enter a scope key to browse memories.
          </p>
        )}
        {browseEnabled && !browse.isLoading && records.length === 0 && (
          <p className="text-[11px] text-muted-foreground/70 italic">
            {searchHits !== null
              ? `No matches for "${search}"`
              : 'No memories in this scope yet.'}
          </p>
        )}
        {records.map(({ memory: m, similarity }) => (
          <Row
            key={m.id}
            mem={m}
            similarity={similarity}
            onDelete={() => {
              if (!window.confirm(`Delete memory ${m.id}? This is permanent.`)) return
              del.mutate(m.id)
            }}
          />
        ))}
      </div>
    </div>
  )
}

function Row({
  mem,
  similarity,
  onDelete,
}: {
  mem: MemoryRecord
  similarity?: number
  onDelete: () => void
}) {
  const [expanded, setExpanded] = useState(false)
  const source = (mem.metadata?.source as string | undefined) ?? null
  const sourcePath = (mem.metadata?.source_path as string | undefined) ?? null

  return (
    <div className="rounded-md border border-border bg-card/30 px-3 py-2 group">
      <div className="flex items-start gap-2">
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          className="flex-1 text-left min-w-0"
        >
          <div className="flex items-center gap-2 flex-wrap mb-1">
            <span className="text-[10px] font-mono text-muted-foreground/60">{mem.id}</span>
            {similarity !== undefined && (
              <span
                className={cn(
                  'text-[10px] px-1.5 py-0.5 rounded border font-mono',
                  similarity > 0.5
                    ? 'border-emerald-500/30 text-emerald-300 bg-emerald-500/10'
                    : similarity > 0.2
                      ? 'border-amber-500/30 text-amber-300 bg-amber-500/10'
                      : 'border-border text-muted-foreground/60',
                )}
              >
                sim {similarity.toFixed(3)}
              </span>
            )}
            {source && (
              <span className="text-[10px] text-muted-foreground/60">
                <CheckCircle2 className="inline size-2.5 mr-0.5" />
                {source}
              </span>
            )}
            <span className="text-[10px] text-muted-foreground/50 ml-auto">
              {formatDistanceToNow(new Date(mem.created_at), { addSuffix: true })}
            </span>
          </div>
          <pre
            className={cn(
              'text-xs whitespace-pre-wrap break-words leading-snug font-sans text-foreground',
              !expanded && 'line-clamp-3',
            )}
          >
            {mem.text}
          </pre>
          {sourcePath && expanded && (
            <p className="text-[10px] font-mono text-muted-foreground/50 mt-1.5 break-all">
              {sourcePath}
            </p>
          )}
        </button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive"
          onClick={onDelete}
          title="Delete this memory"
        >
          <Trash2 className="size-3" />
        </Button>
      </div>
    </div>
  )
}

// Re-export ad-hoc types so the host page doesn't have to dual-import.
export { type Scope } from '@/lib/memory'

// Suppress unused — XCircle imported but only used conditionally.
const _unused = XCircle
void _unused
