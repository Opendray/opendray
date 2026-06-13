// /memory/archived — cross-project "Archived (restorable)" view.
// Soft-archived memories across every project, grouped per project as
// COLLAPSED rows (project name + count); expand one to see and restore
// its memories. Read-only otherwise: there is no approval queue — the
// cleaner auto-applies its verdicts as reversible soft-archives, and
// this view is where the operator undoes one (until the 30-day grace
// window purges it). Mirrors
// app/mobile/lib/features/memory_archived/archived_screen.dart.

import { useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  ChevronDown,
  ChevronRight,
  FolderArchive,
  Globe,
  Loader2,
  RotateCcw,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

import {
  type MemoryRecord,
  listArchived,
  restoreMemory,
} from '@/lib/memory'

interface ArchivedGroup {
  key: string
  scope: string
  scopeKey: string
  label: string
  records: MemoryRecord[]
}

// Last path segment for a readable project name; full cwd stays on hover.
function basename(p: string): string {
  if (!p) return p
  const parts = p.replace(/\/+$/, '').split('/')
  return parts[parts.length - 1] || p
}

export function ArchivedPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  const query = useQuery({
    queryKey: ['archived-memories', 'all'],
    // Both scopes: project-scoped rows (all cwds) AND global-scope
    // rows — global archived memories were invisible here before.
    queryFn: async () => {
      const [proj, glob] = await Promise.all([
        listArchived('project', '', 500),
        listArchived('global', '', 500),
      ])
      return [...proj, ...glob]
    },
    staleTime: 10_000,
  })

  const groups: ArchivedGroup[] = useMemo(() => {
    const m = new Map<string, ArchivedGroup>()
    for (const r of query.data ?? []) {
      const scopeKey = r.scope_key || ''
      const key = `${r.scope}:${scopeKey}`
      if (!m.has(key)) {
        m.set(key, {
          key,
          scope: r.scope,
          scopeKey,
          label:
            r.scope === 'global'
              ? t('web.archived.globalScope')
              : basename(scopeKey),
          records: [],
        })
      }
      m.get(key)!.records.push(r)
    }
    // Most memories first — the projects that need attention float up.
    return [...m.values()].sort((a, b) => b.records.length - a.records.length)
  }, [query.data, t])

  const totalMemories = query.data?.length ?? 0

  const refresh = () =>
    qc.invalidateQueries({ queryKey: ['archived-memories'] })

  const toggle = (key: string) =>
    setExpanded((cur) => {
      const next = new Set(cur)
      next.has(key) ? next.delete(key) : next.add(key)
      return next
    })

  if (query.isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
        <Loader2 className="h-3 w-3 animate-spin" /> {t('web.archived.loading')}
      </div>
    )
  }

  if (groups.length === 0) {
    return (
      <div className="mx-auto max-w-2xl space-y-2 p-12 text-center">
        <h1 className="text-lg font-semibold">
          {t('web.archived.emptyTitle')}
        </h1>
        <p className="text-muted-foreground text-sm">
          {t('web.archived.emptyDescription')}
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl space-y-4 p-6">
      <div>
        <h1 className="text-xl font-semibold">{t('web.archived.title')}</h1>
        <p className="text-muted-foreground mt-0.5 text-sm">
          {t('web.archived.subtitle')}
        </p>
        <p className="text-muted-foreground/70 mt-1.5 text-xs">
          {t('web.archived.summary', {
            projects: groups.length,
            memories: totalMemories,
          })}
        </p>
      </div>

      <div className="overflow-hidden rounded-md border">
        {groups.map((g, i) => (
          <ArchivedProject
            key={g.key}
            group={g}
            open={expanded.has(g.key)}
            onToggle={() => toggle(g.key)}
            onChange={refresh}
            first={i === 0}
          />
        ))}
      </div>
    </div>
  )
}

function ArchivedProject({
  group,
  open,
  onToggle,
  onChange,
  first,
}: {
  group: ArchivedGroup
  open: boolean
  onToggle: () => void
  onChange: () => void
  first: boolean
}) {
  const { t } = useTranslation()

  // Bulk restore: iterate the group's rows (counts are small). No
  // general restore-by-scope endpoint — the backend one is reason-
  // scoped — and a project group mixes cleaner / manual / project_
  // archived reasons, so restoring every row here is the right semantic.
  const restoreAll = useMutation({
    mutationFn: async () => {
      const results = await Promise.allSettled(
        group.records.map((r) => restoreMemory(r.id)),
      )
      const ok = results.filter((r) => r.status === 'fulfilled').length
      const failed = results.length - ok
      if (failed > 0) throw new Error(String(failed))
      return ok
    },
    onSuccess: (count) => {
      toast.success(t('web.archived.restoredAllToast', { count }))
      onChange()
    },
    onError: () => {
      toast.error(t('web.archived.restoreFailedToast'))
      onChange()
    },
  })

  return (
    <div className={first ? '' : 'border-t'}>
      {/* Collapsed project header — the whole row toggles. */}
      <div className="hover:bg-card/40 flex items-center gap-2 px-3 py-2.5">
        <button
          onClick={onToggle}
          className="flex min-w-0 flex-1 items-center gap-2 text-left"
          aria-expanded={open}
        >
          {open ? (
            <ChevronDown className="text-muted-foreground h-3.5 w-3.5 shrink-0" />
          ) : (
            <ChevronRight className="text-muted-foreground h-3.5 w-3.5 shrink-0" />
          )}
          {group.scope === 'global' ? (
            <Globe className="text-muted-foreground h-3.5 w-3.5 shrink-0" />
          ) : (
            <FolderArchive className="text-muted-foreground h-3.5 w-3.5 shrink-0" />
          )}
          <span
            className="truncate text-sm font-medium"
            title={group.scopeKey || group.label}
          >
            {group.label}
          </span>
          <Badge variant="muted" className="shrink-0 text-[10px]">
            {t('web.archived.memCount', { count: group.records.length })}
          </Badge>
        </button>

        <div className="flex shrink-0 items-center gap-1">
          {group.scope === 'project' && group.scopeKey && (
            <Button
              asChild
              variant="ghost"
              size="sm"
              className="text-muted-foreground hover:text-foreground h-7 px-2 text-[11px]"
            >
              <Link to="/cortex/memory/project" search={{ cwd: group.scopeKey }}>
                {t('web.archived.openProject')}
                <ChevronRight className="ml-0.5 h-3 w-3" />
              </Link>
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            className="h-7 px-2 text-[11px]"
            disabled={restoreAll.isPending}
            onClick={() => {
              if (
                !window.confirm(
                  t('web.archived.restoreAllConfirm', {
                    count: group.records.length,
                    project: group.label,
                  }),
                )
              )
                return
              restoreAll.mutate()
            }}
            title={t('web.archived.restoreAllTooltip')}
          >
            {restoreAll.isPending ? (
              <Loader2 className="mr-1 h-3 w-3 animate-spin" />
            ) : (
              <RotateCcw className="mr-1 h-3 w-3" />
            )}
            {t('web.archived.restoreAll')}
          </Button>
        </div>
      </div>

      {/* Expanded memory list. */}
      {open && (
        <div className="bg-muted/10 space-y-2 px-3 pb-3 pt-1">
          {group.records.map((r) => (
            <ArchivedRow key={r.id} record={r} onChange={onChange} />
          ))}
        </div>
      )}
    </div>
  )
}

function ArchivedRow({
  record,
  onChange,
}: {
  record: MemoryRecord
  onChange: () => void
}) {
  const { t } = useTranslation()
  const restore = useMutation({
    mutationFn: () => restoreMemory(record.id),
    onSuccess: () => {
      toast.success(t('web.archived.restoredToast'))
      onChange()
    },
    onError: (e: Error) => {
      toast.error(t('web.archived.restoreFailedToast'), {
        description: e.message,
      })
      onChange()
    },
  })

  return (
    <div className="bg-card rounded-md border p-3">
      <div className="mb-2 flex items-center gap-2">
        {record.archived_reason && (
          <Badge variant="muted" className="max-w-full truncate">
            {record.archived_reason}
          </Badge>
        )}
        {record.archived_at && (
          <span className="text-muted-foreground shrink-0 text-[11px]">
            {t('web.archived.archivedAtPrefix')}{' '}
            {new Date(record.archived_at).toLocaleString()}
          </span>
        )}
      </div>
      <pre className="bg-muted/20 mb-3 max-h-32 overflow-auto rounded p-2 font-mono text-[11px] whitespace-pre-wrap">
        {record.text}
      </pre>
      <Button
        size="sm"
        variant="outline"
        onClick={() => restore.mutate()}
        disabled={restore.isPending}
      >
        {restore.isPending ? (
          <Loader2 className="mr-1 h-3 w-3 animate-spin" />
        ) : (
          <RotateCcw className="mr-1 h-3 w-3" />
        )}
        {t('web.archived.restoreButton')}
      </Button>
    </div>
  )
}
