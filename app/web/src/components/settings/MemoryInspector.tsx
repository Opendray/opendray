import { useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Search as SearchIcon,
  Trash2,
  Pencil,
  Check,
  X,
  Loader2,
  Brain,
  CheckCircle2,
  ChevronDown,
  AlertTriangle,
  RefreshCw,
  Activity,
  FolderSync,
  EraserIcon,
  Plus,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import {
  deleteMemoriesByScope,
  deleteMemory,
  fetchEmbedderStats,
  fetchMemoryStatus,
  listMemories,
  listScopeKeys,
  mirrorCwd,
  reembedAll,
  searchMemories,
  storeMemory,
  testEmbedder,
  updateMemory,
  type EmbedderStats,
  type MemoryRecord,
  type ReembedReport,
  type SearchHit,
  type Scope,
} from '@/lib/memory'
import { listSessions } from '@/lib/sessions'

// MemoryInspector shows the live state of opendray's memory
// subsystem: which embedder is active, how many dims it produces,
// and a browse / search / edit / delete pane over the stored
// memories.
//
// Targeted scope is "project" by default (matches the system
// behaviour for newly-stored memories). The operator types or picks
// a `scope_key` (a cwd) and we list memories under that scope.
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

  // Default scopeKey to a sensible candidate (the cwd of the most-
  // recent active session) the first time the inspector mounts. Not
  // destructive — operator can change.
  useEffect(() => {
    if (scopeKey) return
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
    mutationFn: (id: string) => deleteMemory(id).then(() => id),
    onSuccess: (id) => {
      toast.success('Memory deleted')
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      qc.invalidateQueries({ queryKey: ['memory-scope-keys'] })
      setSearchHits((cur) => cur?.filter((h) => h.memory.id !== id) ?? null)
    },
    onError: (err: Error) => toast.error('Delete failed', { description: err.message }),
  })

  const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false)
  const bulkDel = useMutation({
    mutationFn: () => deleteMemoriesByScope(scope, scopeKey.trim()),
    onSuccess: (n) => {
      toast.success(
        `Deleted ${n} ${n === 1 ? 'memory' : 'memories'} from this scope`,
      )
      setBulkDeleteOpen(false)
      setSearchHits(null)
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      qc.invalidateQueries({ queryKey: ['memory-scope-keys'] })
      qc.invalidateQueries({ queryKey: ['memory-embedder-stats'] })
    },
    onError: (err: Error) =>
      toast.error('Bulk delete failed', { description: err.message }),
  })

  // Add memory dialog state. Pre-fills its scope/scope_key from the
  // currently-browsed scope so the common case (operator browsing
  // a project, wants to add a fact for it) is one click.
  const [addMemOpen, setAddMemOpen] = useState(false)
  const [addMemScope, setAddMemScope] = useState<Scope>(scope)
  const [addMemScopeKey, setAddMemScopeKey] = useState<string>(scopeKey)
  const [addMemText, setAddMemText] = useState<string>('')
  const openAddMem = () => {
    setAddMemScope(scope)
    setAddMemScopeKey(scopeKey)
    setAddMemText('')
    setAddMemOpen(true)
  }
  const addMem = useMutation({
    mutationFn: () =>
      storeMemory(
        addMemText.trim(),
        addMemScope,
        addMemScope === 'global' ? '' : addMemScopeKey.trim(),
      ),
    onSuccess: () => {
      toast.success('Memory created')
      setAddMemOpen(false)
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      qc.invalidateQueries({ queryKey: ['memory-scope-keys'] })
    },
    onError: (err: Error) =>
      toast.error('Create failed', { description: err.message }),
  })

  const edit = useMutation({
    mutationFn: ({ id, text }: { id: string; text: string }) =>
      updateMemory(id, text).then(() => ({ id, text })),
    onSuccess: ({ id, text }) => {
      toast.success('Memory updated')
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      // Reflect the edit immediately in any open search results.
      setSearchHits((cur) =>
        cur?.map((h) =>
          h.memory.id === id ? { ...h, memory: { ...h.memory, text } } : h,
        ) ?? null,
      )
    },
    onError: (err: Error) => toast.error('Update failed', { description: err.message }),
  })

  const stats = useQuery<EmbedderStats>({
    queryKey: ['memory-embedder-stats'],
    queryFn: fetchEmbedderStats,
    refetchInterval: 60_000,
  })

  // Mismatched memories: anything stored under an embedder name
  // that isn't the current one. They still exist in the DB but
  // pgvector won't return them for searches by the new embedder
  // until a reembed pass.
  const mismatched = useMemo(() => {
    if (!stats.data) return { total: 0, breakdown: [] as { name: string; count: number }[] }
    const breakdown = Object.entries(stats.data.counts ?? {})
      .filter(([name]) => name !== stats.data!.current)
      .map(([name, count]) => ({ name, count }))
    const total = breakdown.reduce((acc, b) => acc + b.count, 0)
    return { total, breakdown }
  }, [stats.data])

  const [migrateOpen, setMigrateOpen] = useState(false)
  const [migrateReport, setMigrateReport] = useState<ReembedReport | null>(null)
  const reembed = useMutation({
    mutationFn: () => reembedAll(),
    onSuccess: (r) => {
      setMigrateReport(r)
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      qc.invalidateQueries({ queryKey: ['memory-embedder-stats'] })
      qc.invalidateQueries({ queryKey: ['memory-scope-keys'] })
      toast.success(`Migrated ${r.reembed}/${r.examined} memories to ${r.to}`)
    },
    onError: (err: Error) =>
      toast.error('Migration failed', { description: err.message }),
  })

  const sync = useMutation({
    mutationFn: () => mirrorCwd(scopeKey.trim()),
    onSuccess: (r) => {
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      qc.invalidateQueries({ queryKey: ['memory-embedder-stats'] })
      qc.invalidateQueries({ queryKey: ['memory-scope-keys'] })
      if (r.ingested > 0) {
        toast.success(
          `Ingested ${r.ingested} new memory file${r.ingested === 1 ? '' : 's'}`,
        )
      } else {
        toast.message('No new .md files to sync', {
          description: 'Already in sync, or no Claude memory dir for this cwd.',
        })
      }
    },
    onError: (err: Error) =>
      toast.error('Sync failed', { description: err.message }),
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
      <div className="flex items-start gap-3 rounded-md border border-border bg-card/30 px-3 py-2">
        <Brain className="size-4 text-accent shrink-0 mt-0.5" />
        <div className="flex-1 min-w-0 flex flex-col gap-1">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-[10px] text-muted-foreground/70 font-medium uppercase tracking-wider">
              Active embedder
            </span>
            {statusError ? (
              <Badge variant="danger">unavailable</Badge>
            ) : status ? (
              <>
                <Badge variant="success" className="font-mono">
                  {status.embedder}
                </Badge>
                <span className="text-[11px] text-muted-foreground">
                  {status.dimensions}-dim · {status.enabled ? 'enabled' : 'disabled'}
                </span>
              </>
            ) : (
              <span className="text-[11px] text-muted-foreground">probing…</span>
            )}
          </div>
          <p className="text-[10px] text-muted-foreground/70 leading-snug">
            {`This is the embedder the gateway is currently using for every
            `}
            <code className="font-mono mx-1">memory_search</code> /
            <code className="font-mono mx-1">memory_store</code>
            {` call. If this doesn't match the configuration above, you have
            unsaved changes — click Save then Restart server to apply.`}
          </p>
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

      {/* Cross-embedder migration banner — only shown when there are
          memories under an embedder that isn't the active one. */}
      {mismatched.total > 0 && (
        <MigrationBanner
          mismatched={mismatched}
          current={stats.data?.current ?? '?'}
          onOpenDialog={() => {
            setMigrateReport(null)
            setMigrateOpen(true)
          }}
        />
      )}

      <ReembedDialog
        open={migrateOpen}
        onOpenChange={(v) => {
          setMigrateOpen(v)
          if (!v) setMigrateReport(null)
        }}
        current={stats.data?.current ?? '?'}
        breakdown={mismatched.breakdown}
        report={migrateReport}
        running={reembed.isPending}
        onRun={() => reembed.mutate()}
      />

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
            Scope key{' '}
            {scope === 'global' ? (
              <span className="opacity-60">(ignored for global)</span>
            ) : (
              <span className="opacity-60">
                ({scope === 'project' ? 'cwd of the project' : 'session id'})
              </span>
            )}
          </label>
          <div className="flex gap-2">
            <Input
              value={scopeKey}
              onChange={(e) => {
                setScopeKey(e.target.value)
                setSearchHits(null)
              }}
              placeholder={scope === 'project' ? '/path/to/project (cwd)' : 'session id'}
              disabled={scope === 'global'}
              className="h-8 font-mono text-xs flex-1"
            />
            {scope !== 'global' && (
              <ScopeKeyPicker
                scope={scope}
                onPick={(k) => {
                  setScopeKey(k)
                  setSearchHits(null)
                }}
              />
            )}
            {scope === 'project' && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => sync.mutate()}
                disabled={!scopeKey.trim() || sync.isPending}
                className="h-8 text-[11px] gap-1"
                title="Re-ingest Claude's <cwd>/.claude/memory/*.md files into pgvector"
              >
                {sync.isPending ? (
                  <Loader2 className="size-3 animate-spin" />
                ) : (
                  <FolderSync className="size-3" />
                )}
                Sync .md
              </Button>
            )}
          </div>
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
        {/* Records header — always rendered when browseEnabled so the
            "+ Add" affordance is reachable even from an empty scope.
            "Delete all" is gated on records.length > 0 + not currently
            displaying semantic-search hits (we don't want the operator
            to nuke a project just because their last search returned
            two hits — they'd lose every other memory under that key). */}
        {browseEnabled && !browse.isLoading && (
          <div className="flex items-center justify-between gap-2 px-1">
            <span className="text-[11px] text-muted-foreground/80">
              {records.length === 0
                ? 'No memories yet'
                : searchHits !== null
                  ? `${records.length} match${records.length === 1 ? '' : 'es'}`
                  : `${records.length} ${records.length === 1 ? 'memory' : 'memories'}`}
              {scope === 'global' ? ' (global)' : ` in ${scope}: `}
              {scope !== 'global' && (
                <code className="text-[10.5px] font-mono">
                  {truncMid(scopeKey.trim(), 48)}
                </code>
              )}
            </span>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-7 text-[11px] gap-1"
                onClick={openAddMem}
                title="Manually create a memory in this scope"
              >
                <Plus className="size-3" />
                Add memory
              </Button>
              {searchHits === null && records.length > 0 && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-7 text-[11px] gap-1 border-destructive/40 text-destructive hover:bg-destructive/10"
                  onClick={() => setBulkDeleteOpen(true)}
                  title="Delete every memory under this scope/scope_key"
                >
                  <EraserIcon className="size-3" />
                  Delete all
                </Button>
              )}
            </div>
          </div>
        )}
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
            onSave={(text) =>
              new Promise<void>((resolve, reject) => {
                edit.mutate(
                  { id: m.id, text },
                  {
                    onSuccess: () => resolve(),
                    onError: (e) => reject(e),
                  },
                )
              })
            }
          />
        ))}
      </div>

      {/* Bulk delete confirm dialog — shown when the operator clicks
          "Delete all" in the records header. Server enforces the
          scope-vs-scope_key invariants but we restate them in the
          copy so the operator knows what's about to disappear. */}
      <Dialog open={bulkDeleteOpen} onOpenChange={setBulkDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete every memory in this scope?</DialogTitle>
            <DialogDescription>
              This is a single SQL operation — all memories under the
              specified scope are removed atomically. Memories that
              were ingested via the Claude mirror reappear on the next
              <code className="font-mono mx-1">Sync .md</code>{' '}
              run; everything else is gone for good.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2 text-[12px] py-2">
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground/80">Scope</span>
              <Badge variant="outline" className="font-mono">
                {scope}
              </Badge>
            </div>
            {scope !== 'global' && (
              <div className="flex items-start gap-2">
                <span className="text-muted-foreground/80 shrink-0">
                  Scope key
                </span>
                <code className="font-mono text-[11px] break-all">
                  {scopeKey.trim()}
                </code>
              </div>
            )}
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground/80">Currently visible</span>
              <span>
                {records.length}{' '}
                {records.length === 1 ? 'memory item' : 'memory items'}
              </span>
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setBulkDeleteOpen(false)}
              disabled={bulkDel.isPending}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => bulkDel.mutate()}
              disabled={bulkDel.isPending}
            >
              {bulkDel.isPending && (
                <Loader2 className="size-3.5 animate-spin" />
              )}
              Delete all
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add memory dialog — manually inserts a fact via /memory/store.
          Same shape mobile uses on its FAB → New memory sheet, with
          a scope select that defaults to whichever scope the operator
          is currently browsing so the common case is one click. */}
      <Dialog open={addMemOpen} onOpenChange={setAddMemOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add memory</DialogTitle>
            <DialogDescription>
              Manually create a memory. Agents create these automatically
              via the <code className="font-mono">memory_store</code> MCP
              tool — this form is for cases where the operator wants to
              seed a fact without going through an agent.
            </DialogDescription>
          </DialogHeader>
          <form
            onSubmit={(e) => {
              e.preventDefault()
              if (!addMemText.trim()) return
              if (addMemScope !== 'global' && !addMemScopeKey.trim()) return
              addMem.mutate()
            }}
            className="flex flex-col gap-3"
          >
            <div className="flex items-end gap-2 flex-wrap">
              <div className="space-y-1">
                <label className="text-[10px] text-muted-foreground/80 font-medium uppercase tracking-wider">
                  Scope
                </label>
                <select
                  value={addMemScope}
                  onChange={(e) => setAddMemScope(e.target.value as Scope)}
                  className="h-8 px-2 text-xs rounded border border-border bg-background"
                  disabled={addMem.isPending}
                >
                  <option value="project">project</option>
                  <option value="session">session</option>
                  <option value="global">global</option>
                </select>
              </div>
              <div className="flex-1 space-y-1 min-w-[220px]">
                <label className="text-[10px] text-muted-foreground/80 font-medium uppercase tracking-wider">
                  Scope key{' '}
                  {addMemScope === 'global' ? (
                    <span className="opacity-60">(ignored for global)</span>
                  ) : (
                    <span className="opacity-60">
                      ({addMemScope === 'project' ? 'cwd of the project' : 'session id'})
                    </span>
                  )}
                </label>
                <Input
                  value={addMemScope === 'global' ? '' : addMemScopeKey}
                  onChange={(e) => setAddMemScopeKey(e.target.value)}
                  placeholder={
                    addMemScope === 'project'
                      ? '/path/to/project (cwd)'
                      : 'session id'
                  }
                  disabled={addMemScope === 'global' || addMem.isPending}
                  className="h-8 font-mono text-xs"
                />
              </div>
            </div>
            <div className="space-y-1">
              <label className="text-[10px] text-muted-foreground/80 font-medium uppercase tracking-wider">
                Text
              </label>
              <textarea
                value={addMemText}
                onChange={(e) => setAddMemText(e.target.value)}
                placeholder="Plain prose. The embedder turns this into a vector at store time; agents will retrieve it via memory_search."
                rows={6}
                disabled={addMem.isPending}
                className={cn(
                  'w-full rounded-md border border-border bg-background',
                  'px-3 py-2 text-xs leading-relaxed',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                )}
              />
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setAddMemOpen(false)}
                disabled={addMem.isPending}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant="accent"
                size="sm"
                disabled={
                  addMem.isPending ||
                  !addMemText.trim() ||
                  (addMemScope !== 'global' && !addMemScopeKey.trim())
                }
              >
                {addMem.isPending && (
                  <Loader2 className="size-3.5 animate-spin" />
                )}
                Create
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// truncMid shortens a path-like string by replacing its middle with
// "…". Used in the records header so a long cwd doesn't push the
// "Delete all" button off the line.
function truncMid(s: string, max: number): string {
  if (s.length <= max) return s
  const head = Math.ceil((max - 1) / 2)
  const tail = Math.floor((max - 1) / 2)
  return `${s.slice(0, head)}…${s.slice(s.length - tail)}`
}

function ScopeKeyPicker({
  scope,
  onPick,
}: {
  scope: Scope
  onPick: (key: string) => void
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  // Two data sources, both opt-in (only fire when picker is open):
  //   1. Distinct scope_keys we've already stored memories under
  //      (the "Saved" group — definitive but starts empty).
  //   2. Active sessions — their cwd (for scope=project) or id
  //      (for scope=session). Lets the operator pick a project they
  //      *intend to* store memories for, even if none are saved yet.
  const savedKeys = useQuery({
    queryKey: ['memory-scope-keys', scope],
    queryFn: () => listScopeKeys(scope),
    enabled: open,
  })
  const sessions = useQuery({
    queryKey: ['memory-picker-sessions'],
    queryFn: listSessions,
    enabled: open && scope !== 'global',
  })

  // Build the suggested-from-sessions list per scope. Dedupe against
  // savedKeys so the same value doesn't appear in both groups.
  const savedSet = new Set(savedKeys.data ?? [])
  const sessionCandidates =
    scope === 'project'
      ? Array.from(
          new Set(
            (sessions.data ?? [])
              .map((s) => (s.cwd ?? '').trim())
              .filter((cwd) => !!cwd && !savedSet.has(cwd)),
          ),
        ).sort()
      : scope === 'session'
        ? (sessions.data ?? [])
            .filter((s) => !savedSet.has(s.id))
            .map((s) => ({
              key: s.id,
              hint: `${s.provider_id ?? '?'} · ${(s.cwd ?? '').replace(/.*\//, '') || '/'}`,
            }))
        : []

  // Close on outside click.
  useEffect(() => {
    if (!open) return
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [open])

  const isLoading = savedKeys.isLoading || sessions.isLoading
  const hasAnything =
    (savedKeys.data?.length ?? 0) > 0 || sessionCandidates.length > 0

  return (
    <div ref={ref} className="relative">
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => setOpen((v) => !v)}
        className="h-8 text-[11px] gap-1"
        title="Pick from saved scope keys or active sessions"
      >
        Pick <ChevronDown className="size-3" />
      </Button>
      {open && (
        <div className="absolute right-0 top-full mt-1 z-20 min-w-[320px] max-w-[480px] rounded-md border border-border bg-popover shadow-lg">
          <div className="p-1 max-h-72 overflow-y-auto">
            {isLoading && (
              <p className="text-[11px] text-muted-foreground/70 italic px-2 py-1.5">
                Loading…
              </p>
            )}
            {!isLoading && !hasAnything && (
              <p className="text-[11px] text-muted-foreground/70 italic px-2 py-1.5">
                No saved keys or active sessions for {scope}.
              </p>
            )}

            {(savedKeys.data?.length ?? 0) > 0 && (
              <>
                <div className="px-2 pt-1 pb-0.5 text-[10px] uppercase tracking-wider text-muted-foreground/60">
                  Saved memories
                </div>
                {savedKeys.data?.map((k) => (
                  <button
                    key={`saved-${k}`}
                    type="button"
                    onClick={() => {
                      onPick(k)
                      setOpen(false)
                    }}
                    className="block w-full text-left px-2 py-1 rounded text-[11px] font-mono hover:bg-accent/30 truncate"
                    title={k}
                  >
                    {k}
                  </button>
                ))}
              </>
            )}

            {sessionCandidates.length > 0 && (
              <>
                <div className="px-2 pt-1.5 pb-0.5 text-[10px] uppercase tracking-wider text-muted-foreground/60">
                  Active sessions
                </div>
                {scope === 'project' &&
                  (sessionCandidates as string[]).map((cwd) => (
                    <button
                      key={`sess-${cwd}`}
                      type="button"
                      onClick={() => {
                        onPick(cwd)
                        setOpen(false)
                      }}
                      className="block w-full text-left px-2 py-1 rounded text-[11px] font-mono hover:bg-accent/30 truncate"
                      title={cwd}
                    >
                      {cwd}
                    </button>
                  ))}
                {scope === 'session' &&
                  (sessionCandidates as { key: string; hint: string }[]).map(
                    ({ key, hint }) => (
                      <button
                        key={`sess-${key}`}
                        type="button"
                        onClick={() => {
                          onPick(key)
                          setOpen(false)
                        }}
                        className="flex w-full text-left px-2 py-1 rounded text-[11px] hover:bg-accent/30 items-center gap-2"
                        title={key}
                      >
                        <span className="font-mono truncate flex-1">{key}</span>
                        <span className="text-muted-foreground/60 shrink-0">
                          {hint}
                        </span>
                      </button>
                    ),
                  )}
              </>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function Row({
  mem,
  similarity,
  onDelete,
  onSave,
}: {
  mem: MemoryRecord
  similarity?: number
  onDelete: () => void
  onSave: (text: string) => Promise<void>
}) {
  const [expanded, setExpanded] = useState(false)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(mem.text)
  const [saving, setSaving] = useState(false)
  const source = (mem.metadata?.source as string | undefined) ?? null
  const sourcePath = (mem.metadata?.source_path as string | undefined) ?? null

  // Keep the textarea synced if the underlying record updates from
  // a re-fetch while we're editing — rare, but avoids ghost state.
  useEffect(() => {
    if (!editing) setDraft(mem.text)
  }, [mem.text, editing])

  const startEdit = () => {
    setDraft(mem.text)
    setEditing(true)
    setExpanded(true)
  }
  const cancelEdit = () => {
    setEditing(false)
    setDraft(mem.text)
  }
  const commitEdit = async () => {
    if (draft.trim() === mem.text.trim()) {
      setEditing(false)
      return
    }
    if (!draft.trim()) {
      toast.error('Memory text cannot be empty')
      return
    }
    setSaving(true)
    try {
      await onSave(draft)
      setEditing(false)
    } catch {
      // toast already shown by the mutation handler
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="rounded-md border border-border bg-card/30 px-3 py-2 group">
      <div className="flex items-start gap-2">
        <div className="flex-1 min-w-0">
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
            {mem.hit_count > 0 && (
              <span
                className="text-[10px] text-muted-foreground/70 inline-flex items-center gap-0.5"
                title={
                  mem.last_hit_at
                    ? `Last hit ${formatDistanceToNow(new Date(mem.last_hit_at), { addSuffix: true })}`
                    : ''
                }
              >
                <Activity className="size-2.5" />
                {mem.hit_count} {mem.hit_count === 1 ? 'hit' : 'hits'}
              </span>
            )}
            <span className="text-[10px] text-muted-foreground/50 ml-auto">
              {formatDistanceToNow(new Date(mem.created_at), { addSuffix: true })}
            </span>
          </div>
          {editing ? (
            <textarea
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Escape') {
                  e.preventDefault()
                  cancelEdit()
                } else if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
                  e.preventDefault()
                  void commitEdit()
                }
              }}
              rows={Math.min(12, Math.max(3, draft.split('\n').length + 1))}
              autoFocus
              className="w-full text-xs leading-snug font-sans rounded border border-accent/40 bg-background px-2 py-1.5 resize-y"
              placeholder="Memory text — Cmd/Ctrl+Enter to save · Esc to cancel"
            />
          ) : (
            <button
              type="button"
              onClick={() => setExpanded((v) => !v)}
              className="text-left w-full"
            >
              <pre
                className={cn(
                  'text-xs whitespace-pre-wrap break-words leading-snug font-sans text-foreground',
                  !expanded && 'line-clamp-3',
                )}
              >
                {mem.text}
              </pre>
            </button>
          )}
          {sourcePath && expanded && !editing && (
            <p className="text-[10px] font-mono text-muted-foreground/50 mt-1.5 break-all">
              {sourcePath}
            </p>
          )}
        </div>
        <div className="flex flex-col gap-1">
          {editing ? (
            <>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7 text-emerald-400 hover:text-emerald-300"
                onClick={() => void commitEdit()}
                disabled={saving}
                title="Save (Cmd/Ctrl+Enter)"
              >
                {saving ? <Loader2 className="size-3 animate-spin" /> : <Check className="size-3" />}
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7 text-muted-foreground hover:text-foreground"
                onClick={cancelEdit}
                disabled={saving}
                title="Cancel (Esc)"
              >
                <X className="size-3" />
              </Button>
            </>
          ) : (
            <>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7 opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-accent"
                onClick={startEdit}
                title="Edit this memory"
              >
                <Pencil className="size-3" />
              </Button>
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
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function MigrationBanner({
  mismatched,
  current,
  onOpenDialog,
}: {
  mismatched: { total: number; breakdown: { name: string; count: number }[] }
  current: string
  onOpenDialog: () => void
}) {
  const summary = mismatched.breakdown
    .map((b) => `${b.count} on ${b.name}`)
    .join(', ')
  return (
    <div className="flex items-start gap-3 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2.5">
      <AlertTriangle className="size-4 text-amber-400 shrink-0 mt-0.5" />
      <div className="flex-1 min-w-0">
        <p className="text-[12px] font-medium text-amber-200">
          {mismatched.total} {mismatched.total === 1 ? 'memory' : 'memories'} won't
          appear in searches
        </p>
        <p className="text-[11px] text-muted-foreground mt-0.5">
          {summary} — current embedder is{' '}
          <code className="font-mono text-foreground">{current}</code>. pgvector
          partitions its similarity index by embedder, so older entries are
          silent until reembedded.
        </p>
      </div>
      <Button
        type="button"
        size="sm"
        variant="outline"
        className="h-7 text-[11px] border-amber-500/40 hover:bg-amber-500/20 shrink-0"
        onClick={onOpenDialog}
      >
        <RefreshCw className="size-3" /> Migrate
      </Button>
    </div>
  )
}

function ReembedDialog({
  open,
  onOpenChange,
  current,
  breakdown,
  report,
  running,
  onRun,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
  current: string
  breakdown: { name: string; count: number }[]
  report: ReembedReport | null
  running: boolean
  onRun: () => void
}) {
  const total = breakdown.reduce((acc, b) => acc + b.count, 0)
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="text-base">Reembed memories</DialogTitle>
          <DialogDescription>
            Recompute vectors for memories stored under a different embedder so
            they become searchable again.
          </DialogDescription>
        </DialogHeader>

        <div className="text-[12px] flex flex-col gap-2 max-h-[50vh] overflow-y-auto pr-1">
          <div className="rounded-md border border-border bg-card/30 p-3 space-y-1.5">
            <div className="flex items-center justify-between">
              <span className="text-[10px] uppercase tracking-wider text-muted-foreground/70">
                Target embedder
              </span>
              <code className="font-mono text-[11px] text-foreground">{current}</code>
            </div>
            {breakdown.map((b) => (
              <div key={b.name} className="flex items-center justify-between">
                <span className="text-muted-foreground">
                  on <code className="font-mono">{b.name}</code>
                </span>
                <span className="font-mono">{b.count}</span>
              </div>
            ))}
            <div className="flex items-center justify-between border-t border-border/50 pt-1.5 mt-1.5">
              <span className="font-medium">Total to reembed</span>
              <span className="font-mono font-medium">{total}</span>
            </div>
          </div>

          <p className="text-[11px] text-muted-foreground">
            Each memory's text gets re-sent to the current embedder; the new
            vector replaces the old one in place. ID, scope, scope_key,
            metadata and timestamps are preserved. Search results take effect
            immediately — no restart needed.
          </p>

          {report && (
            <div className="rounded-md border border-emerald-500/30 bg-emerald-500/5 p-3 space-y-1 text-[11px]">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Examined</span>
                <span className="font-mono">{report.examined}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Reembedded</span>
                <span className="font-mono text-emerald-300">{report.reembed}</span>
              </div>
              {report.failed > 0 && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Failed</span>
                  <span className="font-mono text-rose-300">{report.failed}</span>
                </div>
              )}
              <div className="flex justify-between">
                <span className="text-muted-foreground">From</span>
                <span className="font-mono">{report.from.join(', ') || '—'}</span>
              </div>
              {(report.errors?.length ?? 0) > 0 && (
                <details className="mt-1.5">
                  <summary className="cursor-pointer text-rose-300">
                    {report.errors?.length} error{(report.errors?.length ?? 0) > 1 ? 's' : ''}
                  </summary>
                  <pre className="mt-1 text-[10px] font-mono whitespace-pre-wrap break-all bg-background/50 p-2 rounded max-h-32 overflow-y-auto">
                    {report.errors?.join('\n')}
                  </pre>
                </details>
              )}
            </div>
          )}
        </div>

        <DialogFooter className="gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-8 text-[11px]"
            onClick={() => onOpenChange(false)}
            disabled={running}
          >
            {report ? 'Done' : 'Cancel'}
          </Button>
          {!report && (
            <Button
              type="button"
              size="sm"
              className="h-8 text-[11px]"
              onClick={onRun}
              disabled={running || total === 0}
            >
              {running ? (
                <>
                  <Loader2 className="size-3 animate-spin" /> Reembedding…
                </>
              ) : (
                <>
                  <RefreshCw className="size-3" /> Reembed {total}
                </>
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// Re-export ad-hoc types so the host page doesn't have to dual-import.
export { type Scope } from '@/lib/memory'
