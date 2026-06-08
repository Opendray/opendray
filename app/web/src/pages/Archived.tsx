// /memory/archived — cross-project "Archived (restorable)" view.
// Lists the soft-archived memories the auto-cleaner / lifecycle pass
// removed across every project, grouped by scope, with one-click
// restore. Read-only otherwise: there is no approval queue anymore —
// the cleaner auto-applies its verdicts as reversible soft-archives,
// and this view is where the operator undoes one if needed (until the
// 30-day grace window purges it). Mirrors
// app/mobile/lib/features/memory_archived/archived_screen.dart.

import { useMemo } from 'react'
import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { ChevronRight, Loader2, RotateCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

import {
  type MemoryRecord,
  listArchived,
  restoreMemory,
} from '@/lib/memory'

export function ArchivedPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()

  const query = useQuery({
    queryKey: ['archived-memories', 'all'],
    queryFn: () => listArchived('project', '', 500),
    staleTime: 10_000,
  })

  const grouped = useMemo(() => {
    const m = new Map<string, MemoryRecord[]>()
    for (const r of query.data ?? []) {
      const key = `${r.scope}:${r.scope_key || t('web.archived.globalScope')}`
      if (!m.has(key)) m.set(key, [])
      m.get(key)!.push(r)
    }
    return [...m.entries()].sort((a, b) => a[0].localeCompare(b[0]))
  }, [query.data, t])

  const refresh = () =>
    qc.invalidateQueries({ queryKey: ['archived-memories'] })

  if (query.isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
        <Loader2 className="h-3 w-3 animate-spin" /> {t('web.archived.loading')}
      </div>
    )
  }

  if ((query.data ?? []).length === 0) {
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
    <div className="mx-auto max-w-4xl space-y-6 p-6">
      <div>
        <h1 className="text-xl font-semibold">{t('web.archived.title')}</h1>
        <p className="text-muted-foreground text-sm">
          {t('web.archived.subtitle')}
        </p>
      </div>

      {grouped.map(([key, records]) => {
        const [scope, scopeKey] = key.split(':', 2)
        return (
          <section key={key} className="space-y-3">
            <header className="flex items-center justify-between border-b pb-2">
              <div className="flex items-center gap-2">
                <Badge variant="outline">{scope}</Badge>
                <span className="font-mono text-xs">{scopeKey}</span>
              </div>
              {scope === 'project' && scopeKey && (
                <Link
                  to="/notes"
                  search={{ mode: 'project', cwd: scopeKey }}
                  className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs"
                >
                  {t('web.archived.openProject')}{' '}
                  <ChevronRight className="h-3 w-3" />
                </Link>
              )}
            </header>
            {records.map((r) => (
              <ArchivedRow key={r.id} record={r} onChange={refresh} />
            ))}
          </section>
        )
      })}
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
          <Badge variant="muted">{record.archived_reason}</Badge>
        )}
        {record.archived_at && (
          <span className="text-muted-foreground text-[11px]">
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
