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
  Archive,
  ShieldQuestion,
  CheckCircle2,
  ChevronDown,
  AlertTriangle,
  RefreshCw,
  Activity,
  FolderSync,
  FolderSearch,
  EraserIcon,
  Plus,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { toast } from 'sonner'
import { Trans, useTranslation } from 'react-i18next'

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
  archiveMemory,
  deleteMemoriesByScope,
  deleteMemory,
  quarantineMemory,
  fetchEmbedderStats,
  fetchMemoryStatus,
  listMemories,
  listScopeKeys,
  mirrorCwd,
  reembedAll,
  searchMemories,
  storeMemory,
  updateMemory,
  type EmbedderStats,
  type MemoryRecord,
  type ReembedReport,
  type SearchHit,
  type Scope,
} from '@/lib/memory'
import { rankingBreakdown } from '@/lib/memoryRanking'
import { listSessions } from '@/lib/sessions'
import { FileBrowserDialog } from '@/components/sessions/FileBrowserDialog'

// MemoryInspector shows the live state of opendray's memory
// subsystem: which embedder is active, how many dims it produces,
// and a browse / search / edit / delete pane over the stored
// memories.
//
// Targeted scope is "project" by default (matches the system
// behaviour for newly-stored memories). The operator types or picks
// a `scope_key` (a cwd) and we list memories under that scope.
export function MemoryInspector() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [scope, setScope] = useState<Scope>('project')
  const [scopeKey, setScopeKey] = useState<string>('')
  const [scopeBrowserOpen, setScopeBrowserOpen] = useState(false)
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
      toast.error(t('web.memoryInspector.search.failedToast'), {
        description: (err as Error).message,
      })
      setSearchHits(null)
    } finally {
      setSearchBusy(false)
    }
  }

  const del = useMutation({
    mutationFn: (id: string) => deleteMemory(id).then(() => id),
    onSuccess: (id) => {
      toast.success(t('web.memoryInspector.toasts.deleted'))
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      qc.invalidateQueries({ queryKey: ['memory-scope-keys'] })
      setSearchHits((cur) => cur?.filter((h) => h.memory.id !== id) ?? null)
    },
    onError: (err: Error) =>
      toast.error(t('web.memoryInspector.toasts.deleteFailed'), {
        description: err.message,
      }),
  })

  // Manual archive — reversible (Archived view) until grace purges it.
  const archive = useMutation({
    mutationFn: (id: string) => archiveMemory(id).then(() => id),
    onSuccess: (id) => {
      toast.success(t('web.memoryInspector.toasts.archived'))
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      qc.invalidateQueries({ queryKey: ['archived-memories'] })
      setSearchHits((cur) => cur?.filter((h) => h.memory.id !== id) ?? null)
    },
    onError: (err: Error) =>
      toast.error(t('web.memoryInspector.toasts.archiveFailed'), {
        description: err.message,
      }),
  })

  // Manual quarantine — moves the row to the Cortex review queue;
  // promote it back from there or let the TTL expire it.
  const quarantine = useMutation({
    mutationFn: (id: string) => quarantineMemory(id).then(() => id),
    onSuccess: (id) => {
      toast.success(t('web.memoryInspector.toasts.quarantined'))
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      qc.invalidateQueries({ queryKey: ['cortex-quarantine'] })
      qc.invalidateQueries({ queryKey: ['cortex-status'] })
      setSearchHits((cur) => cur?.filter((h) => h.memory.id !== id) ?? null)
    },
    onError: (err: Error) =>
      toast.error(t('web.memoryInspector.toasts.quarantineFailed'), {
        description: err.message,
      }),
  })

  const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false)
  const bulkDel = useMutation({
    mutationFn: () => deleteMemoriesByScope(scope, scopeKey.trim()),
    onSuccess: (n) => {
      toast.success(
        t('web.memoryInspector.toasts.bulkDeleted', { count: n }),
      )
      setBulkDeleteOpen(false)
      setSearchHits(null)
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      qc.invalidateQueries({ queryKey: ['memory-scope-keys'] })
      qc.invalidateQueries({ queryKey: ['memory-embedder-stats'] })
    },
    onError: (err: Error) =>
      toast.error(t('web.memoryInspector.toasts.bulkDeleteFailed'), {
        description: err.message,
      }),
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
      toast.success(t('web.memoryInspector.toasts.created'))
      setAddMemOpen(false)
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      qc.invalidateQueries({ queryKey: ['memory-scope-keys'] })
    },
    onError: (err: Error) =>
      toast.error(t('web.memoryInspector.toasts.createFailed'), {
        description: err.message,
      }),
  })

  const edit = useMutation({
    mutationFn: ({ id, text }: { id: string; text: string }) =>
      updateMemory(id, text).then(() => ({ id, text })),
    onSuccess: ({ id, text }) => {
      toast.success(t('web.memoryInspector.toasts.updated'))
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      // Reflect the edit immediately in any open search results.
      setSearchHits((cur) =>
        cur?.map((h) =>
          h.memory.id === id ? { ...h, memory: { ...h.memory, text } } : h,
        ) ?? null,
      )
    },
    onError: (err: Error) =>
      toast.error(t('web.memoryInspector.toasts.updateFailed'), {
        description: err.message,
      }),
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
      toast.success(
        t('web.memoryInspector.toasts.migrated', {
          reembed: r.reembed,
          examined: r.examined,
          to: r.to,
        }),
      )
    },
    onError: (err: Error) =>
      toast.error(t('web.memoryInspector.toasts.migrationFailed'), {
        description: err.message,
      }),
  })

  const sync = useMutation({
    mutationFn: () => mirrorCwd(scopeKey.trim()),
    onSuccess: (r) => {
      qc.invalidateQueries({ queryKey: ['memory-list'] })
      qc.invalidateQueries({ queryKey: ['memory-embedder-stats'] })
      qc.invalidateQueries({ queryKey: ['memory-scope-keys'] })
      if (r.ingested > 0) {
        toast.success(
          t('web.memoryInspector.toasts.syncIngested', { count: r.ingested }),
        )
      } else {
        toast.message(t('web.memoryInspector.toasts.syncEmpty'), {
          description: t('web.memoryInspector.toasts.syncEmptyDescription'),
        })
      }
    },
    onError: (err: Error) =>
      toast.error(t('web.memoryInspector.toasts.syncFailed'), {
        description: err.message,
      }),
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
              {t('web.memoryInspector.status.label')}
            </span>
            {statusError ? (
              <Badge variant="danger">
                {t('web.memoryInspector.status.unavailable')}
              </Badge>
            ) : status ? (
              <>
                <Badge variant="success" className="font-mono">
                  {status.effective_embedder ?? status.embedder}
                </Badge>
                <span className="text-[11px] text-muted-foreground">
                  {t('web.memoryInspector.status.dimensions', {
                    dim: status.dimensions,
                    state: status.enabled
                      ? t('web.memoryInspector.status.enabled')
                      : t('web.memoryInspector.status.disabled'),
                  })}
                </span>
              </>
            ) : (
              <span className="text-[11px] text-muted-foreground">
                {t('web.memoryInspector.status.probing')}
              </span>
            )}
          </div>
          {status &&
            (status.is_floor || status.degraded) &&
            (() => {
              const model = status.configured_dense?.model ?? ''
              let msg = ''
              if (status.is_floor && status.configured_dense) {
                msg =
                  status.dense_reachable === false
                    ? t('web.memoryInspector.status.denseUnreachableFloor', {
                        model,
                      })
                    : t(
                        'web.memoryInspector.status.denseConfiguredPendingRestart',
                        { model },
                      )
              } else if (status.is_floor) {
                msg = t('web.memoryInspector.status.floorNoModel')
              } else {
                msg = t('web.memoryInspector.status.denseDegraded')
              }
              return (
                <p className="text-[10px] text-amber-400/90 leading-snug">
                  {msg}
                </p>
              )
            })()}
        </div>
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
            {t('web.memoryInspector.scope.label')}
          </label>
          <select
            value={scope}
            onChange={(e) => {
              setScope(e.target.value as Scope)
              setSearchHits(null)
            }}
            className="h-8 px-2 text-xs rounded border border-border bg-background"
          >
            <option value="project">
              {t('web.memoryInspector.scope.values.project')}
            </option>
            <option value="global">
              {t('web.memoryInspector.scope.values.global')}
            </option>
          </select>
        </div>
        <div className="flex-1 space-y-1 min-w-[280px]">
          <label className="text-[10px] text-muted-foreground/80 font-medium uppercase tracking-wider">
            {t('web.memoryInspector.scope.scopeKey')}{' '}
            {scope === 'global' ? (
              <span className="opacity-60">
                {t('web.memoryInspector.scope.scopeKeyIgnored')}
              </span>
            ) : (
              <span className="opacity-60">
                {t('web.memoryInspector.scope.scopeKeyCwd')}
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
              placeholder={t('web.memoryInspector.scope.placeholderProject')}
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
                onClick={() => setScopeBrowserOpen(true)}
                className="h-8 text-[11px] gap-1"
                title={t('web.memoryInspector.scope.browseTooltip')}
              >
                <FolderSearch className="size-3" />
                {t('web.memoryInspector.scope.browse')}
              </Button>
            )}
            {scope === 'project' && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => sync.mutate()}
                disabled={!scopeKey.trim() || sync.isPending}
                className="h-8 text-[11px] gap-1"
                title={t('web.memoryInspector.scope.syncTooltip')}
              >
                {sync.isPending ? (
                  <Loader2 className="size-3 animate-spin" />
                ) : (
                  <FolderSync className="size-3" />
                )}
                {t('web.memoryInspector.scope.syncMd')}
              </Button>
            )}
          </div>
        </div>
      </div>
      <FileBrowserDialog
        open={scopeBrowserOpen}
        onOpenChange={setScopeBrowserOpen}
        initialPath={scopeKey.trim() || undefined}
        onSelect={(path) => {
          setScopeKey(path)
          setSearchHits(null)
        }}
      />

      {/* Search */}
      <div className="flex gap-2">
        <div className="relative flex-1">
          <SearchIcon className="absolute left-2 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground/60" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && runSearch()}
            placeholder={t('web.memoryInspector.search.placeholder')}
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
          {searchBusy ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            t('web.memoryInspector.search.run')
          )}
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
            {t('web.memoryInspector.search.clear')}
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
                ? t('web.memoryInspector.records.noMemories')
                : searchHits !== null
                  ? t('web.memoryInspector.records.matches', {
                      count: records.length,
                    })
                  : t('web.memoryInspector.records.memories', {
                      count: records.length,
                    })}
              {scope === 'global'
                ? t('web.memoryInspector.records.scopeGlobalSuffix')
                : t('web.memoryInspector.records.scopeInSuffix', { scope })}
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
                title={t('web.memoryInspector.records.addTooltip')}
              >
                <Plus className="size-3" />
                {t('web.memoryInspector.records.addButton')}
              </Button>
              {searchHits === null && records.length > 0 && (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-7 text-[11px] gap-1 border-destructive/40 text-destructive hover:bg-destructive/10"
                  onClick={() => setBulkDeleteOpen(true)}
                  title={t('web.memoryInspector.records.deleteAllTooltip')}
                >
                  <EraserIcon className="size-3" />
                  {t('web.memoryInspector.records.deleteAll')}
                </Button>
              )}
            </div>
          </div>
        )}
        {browse.isLoading && (
          <p className="text-[11px] text-muted-foreground/70 italic">
            {t('web.memoryInspector.records.loading')}
          </p>
        )}
        {!browseEnabled && (
          <p className="text-[11px] text-muted-foreground/70 italic">
            {t('web.memoryInspector.records.enterScopeKeyHint')}
          </p>
        )}
        {browseEnabled && !browse.isLoading && records.length === 0 && (
          <p className="text-[11px] text-muted-foreground/70 italic">
            {searchHits !== null
              ? t('web.memoryInspector.records.noMatchesForQuery', { query: search })
              : t('web.memoryInspector.records.noMemoriesInScope')}
          </p>
        )}
        {records.map(({ memory: m, similarity }) => (
          <Row
            key={m.id}
            mem={m}
            similarity={similarity}
            onArchive={() => archive.mutate(m.id)}
            onQuarantine={() => quarantine.mutate(m.id)}
            onDelete={() => {
              if (
                !window.confirm(
                  t('web.memoryInspector.row.deleteConfirm', { id: m.id }),
                )
              )
                return
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
            <DialogTitle>
              {t('web.memoryInspector.bulkDelete.title')}
            </DialogTitle>
            <DialogDescription>
              <Trans
                i18nKey="web.memoryInspector.bulkDelete.description"
                components={{ 1: <code className="font-mono mx-1" /> }}
              />
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2 text-[12px] py-2">
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground/80">
                {t('web.memoryInspector.bulkDelete.scope')}
              </span>
              <Badge variant="outline" className="font-mono">
                {scope}
              </Badge>
            </div>
            {scope !== 'global' && (
              <div className="flex items-start gap-2">
                <span className="text-muted-foreground/80 shrink-0">
                  {t('web.memoryInspector.bulkDelete.scopeKey')}
                </span>
                <code className="font-mono text-[11px] break-all">
                  {scopeKey.trim()}
                </code>
              </div>
            )}
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground/80">
                {t('web.memoryInspector.bulkDelete.currentlyVisible')}
              </span>
              <span>
                {t('web.memoryInspector.bulkDelete.items', {
                  count: records.length,
                })}
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
              {t('web.memoryInspector.bulkDelete.cancel')}
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
              {t('web.memoryInspector.bulkDelete.deleteAll')}
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
            <DialogTitle>{t('web.memoryInspector.addMem.title')}</DialogTitle>
            <DialogDescription>
              <Trans
                i18nKey="web.memoryInspector.addMem.description"
                components={{ 1: <code className="font-mono" /> }}
              />
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
                  {t('web.memoryInspector.scope.label')}
                </label>
                <select
                  value={addMemScope}
                  onChange={(e) => setAddMemScope(e.target.value as Scope)}
                  className="h-8 px-2 text-xs rounded border border-border bg-background"
                  disabled={addMem.isPending}
                >
                  <option value="project">
                    {t('web.memoryInspector.scope.values.project')}
                  </option>
                  <option value="global">
                    {t('web.memoryInspector.scope.values.global')}
                  </option>
                </select>
              </div>
              <div className="flex-1 space-y-1 min-w-[220px]">
                <label className="text-[10px] text-muted-foreground/80 font-medium uppercase tracking-wider">
                  {t('web.memoryInspector.scope.scopeKey')}{' '}
                  {addMemScope === 'global' ? (
                    <span className="opacity-60">
                      {t('web.memoryInspector.scope.scopeKeyIgnored')}
                    </span>
                  ) : (
                    <span className="opacity-60">
                      {t('web.memoryInspector.scope.scopeKeyCwd')}
                    </span>
                  )}
                </label>
                <Input
                  value={addMemScope === 'global' ? '' : addMemScopeKey}
                  onChange={(e) => setAddMemScopeKey(e.target.value)}
                  placeholder={t('web.memoryInspector.scope.placeholderProject')}
                  disabled={addMemScope === 'global' || addMem.isPending}
                  className="h-8 font-mono text-xs"
                />
              </div>
            </div>
            <div className="space-y-1">
              <label className="text-[10px] text-muted-foreground/80 font-medium uppercase tracking-wider">
                {t('web.memoryInspector.addMem.textLabel')}
              </label>
              <textarea
                value={addMemText}
                onChange={(e) => setAddMemText(e.target.value)}
                placeholder={t('web.memoryInspector.addMem.textPlaceholder')}
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
                {t('web.memoryInspector.addMem.cancel')}
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
                {t('web.memoryInspector.addMem.create')}
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
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  // Two data sources, both opt-in (only fire when picker is open):
  //   1. Distinct scope_keys we've already stored memories under
  //      (the "Saved" group — definitive but starts empty).
  //   2. Active sessions — their cwd (for scope=project). Lets the
  //      operator pick a project they *intend to* store memories for,
  //      even if none are saved yet.
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
        title={t('web.memoryInspector.picker.buttonTooltip')}
      >
        {t('web.memoryInspector.picker.button')}{' '}
        <ChevronDown className="size-3" />
      </Button>
      {open && (
        <div className="absolute right-0 top-full mt-1 z-20 min-w-[320px] max-w-[480px] rounded-md border border-border bg-popover shadow-lg">
          <div className="p-1 max-h-72 overflow-y-auto">
            {isLoading && (
              <p className="text-[11px] text-muted-foreground/70 italic px-2 py-1.5">
                {t('web.memoryInspector.picker.loading')}
              </p>
            )}
            {!isLoading && !hasAnything && (
              <p className="text-[11px] text-muted-foreground/70 italic px-2 py-1.5">
                {t('web.memoryInspector.picker.empty', { scope })}
              </p>
            )}

            {(savedKeys.data?.length ?? 0) > 0 && (
              <>
                <div className="px-2 pt-1 pb-0.5 text-[10px] uppercase tracking-wider text-muted-foreground/60">
                  {t('web.memoryInspector.picker.savedHeader')}
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
                  {t('web.memoryInspector.picker.activeHeader')}
                </div>
                {sessionCandidates.map((cwd) => (
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
  onArchive,
  onQuarantine,
  onDelete,
  onSave,
}: {
  mem: MemoryRecord
  similarity?: number
  onArchive: () => void
  onQuarantine: () => void
  onDelete: () => void
  onSave: (text: string) => Promise<void>
}) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(mem.text)
  // M-PD — compute the same effective-score the backend ranking
  // uses so operators can see WHY a row sits where it does. When
  // there's no active similarity (browse view) we pass 1.0 to get
  // the "this row's intrinsic boost" reading.
  const rank = useMemo(
    () => rankingBreakdown(mem, similarity ?? 1),
    [mem, similarity],
  )
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
      toast.error(t('web.memoryInspector.row.emptyError'))
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
                {t('web.memoryInspector.row.simBadge', {
                  value: similarity.toFixed(3),
                })}
              </span>
            )}
            <span
              className="text-[10px] px-1.5 py-0.5 rounded border border-border text-muted-foreground/80 font-mono"
              title={t('web.memoryInspector.row.rankTooltip', {
                similarity: rank.similarity.toFixed(3),
                age: rank.ageMultiplier.toFixed(2),
                hits: rank.hitMultiplier.toFixed(2),
                confidence: rank.confidenceMultiplier.toFixed(2),
                effective: rank.effectiveScore.toFixed(3),
                days: Math.round(rank.ageDays),
              })}
            >
              {t('web.memoryInspector.row.rankBadge', {
                value: rank.effectiveScore.toFixed(2),
              })}
            </span>
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
                    ? t('web.memoryInspector.row.lastHitTooltip', {
                        relative: formatDistanceToNow(
                          new Date(mem.last_hit_at),
                          { addSuffix: true },
                        ),
                      })
                    : ''
                }
              >
                <Activity className="size-2.5" />
                {t('web.memoryInspector.row.hits', { count: mem.hit_count })}
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
              placeholder={t('web.memoryInspector.row.editPlaceholder')}
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
                title={t('web.memoryInspector.row.saveTooltip')}
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
                title={t('web.memoryInspector.row.cancelTooltip')}
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
                title={t('web.memoryInspector.row.editTooltip')}
              >
                <Pencil className="size-3" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7 opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-foreground"
                onClick={onArchive}
                title={t('web.memoryInspector.row.archiveTooltip')}
              >
                <Archive className="size-3" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7 opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-amber-400"
                onClick={onQuarantine}
                title={t('web.memoryInspector.row.quarantineTooltip')}
              >
                <ShieldQuestion className="size-3" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7 opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive"
                onClick={onDelete}
                title={t('web.memoryInspector.row.deleteTooltip')}
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
  const { t } = useTranslation()
  const summary = mismatched.breakdown
    .map((b) =>
      t('web.memoryInspector.migrationBanner.summaryItem', {
        count: b.count,
        name: b.name,
      }),
    )
    .join(', ')
  return (
    <div className="flex items-start gap-3 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2.5">
      <AlertTriangle className="size-4 text-amber-400 shrink-0 mt-0.5" />
      <div className="flex-1 min-w-0">
        <p className="text-[12px] font-medium text-amber-200">
          {t('web.memoryInspector.migrationBanner.headline', {
            count: mismatched.total,
          })}
        </p>
        <p className="text-[11px] text-muted-foreground mt-0.5">
          <Trans
            i18nKey="web.memoryInspector.migrationBanner.subtitle"
            values={{ summary, current }}
            components={{ 1: <code className="font-mono text-foreground" /> }}
          />
        </p>
      </div>
      <Button
        type="button"
        size="sm"
        variant="outline"
        className="h-7 text-[11px] border-amber-500/40 hover:bg-amber-500/20 shrink-0"
        onClick={onOpenDialog}
      >
        <RefreshCw className="size-3" />{' '}
        {t('web.memoryInspector.migrationBanner.migrateButton')}
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
  const { t } = useTranslation()
  const total = breakdown.reduce((acc, b) => acc + b.count, 0)
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="text-base">
            {t('web.memoryInspector.reembed.title')}
          </DialogTitle>
          <DialogDescription>
            {t('web.memoryInspector.reembed.description')}
          </DialogDescription>
        </DialogHeader>

        <div className="text-[12px] flex flex-col gap-2 max-h-[50vh] overflow-y-auto pr-1">
          <div className="rounded-md border border-border bg-card/30 p-3 space-y-1.5">
            <div className="flex items-center justify-between">
              <span className="text-[10px] uppercase tracking-wider text-muted-foreground/70">
                {t('web.memoryInspector.reembed.targetEmbedder')}
              </span>
              <code className="font-mono text-[11px] text-foreground">{current}</code>
            </div>
            {breakdown.map((b) => (
              <div key={b.name} className="flex items-center justify-between">
                <span className="text-muted-foreground">
                  {t('web.memoryInspector.reembed.onName')}{' '}
                  <code className="font-mono">{b.name}</code>
                </span>
                <span className="font-mono">{b.count}</span>
              </div>
            ))}
            <div className="flex items-center justify-between border-t border-border/50 pt-1.5 mt-1.5">
              <span className="font-medium">
                {t('web.memoryInspector.reembed.totalToReembed')}
              </span>
              <span className="font-mono font-medium">{total}</span>
            </div>
          </div>

          <p className="text-[11px] text-muted-foreground">
            {t('web.memoryInspector.reembed.explainer')}
          </p>

          {report && (
            <div className="rounded-md border border-emerald-500/30 bg-emerald-500/5 p-3 space-y-1 text-[11px]">
              <div className="flex justify-between">
                <span className="text-muted-foreground">
                  {t('web.memoryInspector.reembed.reportExamined')}
                </span>
                <span className="font-mono">{report.examined}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">
                  {t('web.memoryInspector.reembed.reportReembedded')}
                </span>
                <span className="font-mono text-emerald-300">{report.reembed}</span>
              </div>
              {report.failed > 0 && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">
                    {t('web.memoryInspector.reembed.reportFailed')}
                  </span>
                  <span className="font-mono text-rose-300">{report.failed}</span>
                </div>
              )}
              <div className="flex justify-between">
                <span className="text-muted-foreground">
                  {t('web.memoryInspector.reembed.reportFrom')}
                </span>
                <span className="font-mono">{report.from.join(', ') || '—'}</span>
              </div>
              {(report.errors?.length ?? 0) > 0 && (
                <details className="mt-1.5">
                  <summary className="cursor-pointer text-rose-300">
                    {t('web.memoryInspector.reembed.errors', {
                      count: report.errors?.length ?? 0,
                    })}
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
            {report
              ? t('web.memoryInspector.reembed.done')
              : t('web.memoryInspector.reembed.cancel')}
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
                  <Loader2 className="size-3 animate-spin" />{' '}
                  {t('web.memoryInspector.reembed.reembedding')}
                </>
              ) : (
                <>
                  <RefreshCw className="size-3" />{' '}
                  {t('web.memoryInspector.reembed.reembedTotal', { total })}
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
