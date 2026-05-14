// JournalStalePanel — M-PD bulk-prune view for journal entries
// older than 90 days that aren't tied to any pending conflict.
// Operators click the chevron to expand, tick the entries they
// want gone, and bulk-delete in one shot.

import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { ChevronDown, ChevronRight, Loader2, Trash2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  deleteSessionLog,
  listStaleSessionLogs,
} from '@/lib/projectDocs'

interface JournalStalePanelProps {
  cwd: string
}

const DEFAULT_DAYS = 90

export function JournalStalePanel({ cwd }: JournalStalePanelProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [expanded, setExpanded] = useState(false)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [days, setDays] = useState(DEFAULT_DAYS)

  const staleQuery = useQuery({
    queryKey: ['session-logs-stale', cwd, days],
    queryFn: () => listStaleSessionLogs(cwd, days),
    enabled: !!cwd && expanded,
  })

  const bulkDelete = useMutation({
    mutationFn: async (ids: string[]) => {
      // Sequential rather than parallel so a 5xx on one doesn't
      // leave a partial result in the UI's optimistic state.
      for (const id of ids) {
        await deleteSessionLog(id)
      }
      return ids.length
    },
    onSuccess: (n) => {
      toast.success(t('web.journalStale.deleted', { count: n }))
      setSelected(new Set())
      qc.invalidateQueries({ queryKey: ['session-logs-stale', cwd] })
      qc.invalidateQueries({ queryKey: ['session-logs', cwd] })
    },
    onError: (err) => {
      toast.error(`${err}`)
    },
  })

  const entries = staleQuery.data ?? []
  const allChecked = entries.length > 0 && selected.size === entries.length
  const toggleAll = () => {
    if (allChecked) setSelected(new Set())
    else setSelected(new Set(entries.map((e) => e.id)))
  }
  const toggleOne = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }
  const selectedIds = useMemo(() => Array.from(selected), [selected])

  return (
    <div className="bg-card/30 mb-3 rounded-md border">
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="hover:bg-muted/30 flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-[12px]"
      >
        <span className="flex items-center gap-2">
          {expanded ? (
            <ChevronDown className="size-3.5" />
          ) : (
            <ChevronRight className="size-3.5" />
          )}
          <span className="font-medium">{t('web.journalStale.title')}</span>
          <span className="text-muted-foreground">
            {t('web.journalStale.subtitle', { days })}
          </span>
        </span>
        {expanded && staleQuery.data && entries.length > 0 && (
          <Badge variant="muted">{entries.length}</Badge>
        )}
      </button>

      {expanded && (
        <div className="border-t px-3 py-2">
          <div className="mb-2 flex items-center gap-2">
            <label className="text-muted-foreground text-[11px]">
              {t('web.journalStale.daysLabel')}
            </label>
            <input
              type="number"
              min={1}
              max={3650}
              value={days}
              onChange={(e) => setDays(Math.max(1, Number(e.target.value)))}
              className="border-border bg-background h-7 w-16 rounded border px-2 text-[12px]"
            />
            <Button
              size="sm"
              variant="ghost"
              onClick={toggleAll}
              disabled={entries.length === 0}
              className="h-7 text-[11px]"
            >
              {allChecked
                ? t('web.journalStale.deselectAll')
                : t('web.journalStale.selectAll')}
            </Button>
            <Button
              size="sm"
              variant="destructive"
              onClick={() => bulkDelete.mutate(selectedIds)}
              disabled={selected.size === 0 || bulkDelete.isPending}
              className="ml-auto h-7 text-[11px]"
            >
              {bulkDelete.isPending ? (
                <Loader2 className="mr-1 size-3 animate-spin" />
              ) : (
                <Trash2 className="mr-1 size-3" />
              )}
              {t('web.journalStale.deleteSelected', { count: selected.size })}
            </Button>
          </div>

          {staleQuery.isLoading && (
            <div className="text-muted-foreground flex items-center gap-2 text-[11px]">
              <Loader2 className="size-3 animate-spin" />
              {t('web.journalStale.loading')}
            </div>
          )}
          {!staleQuery.isLoading && entries.length === 0 && (
            <p className="text-muted-foreground text-[11px] italic">
              {t('web.journalStale.empty')}
            </p>
          )}
          <ul className="space-y-1">
            {entries.map((e) => (
              <li
                key={e.id}
                className="flex items-start gap-2 rounded border p-2 text-[12px]"
              >
                <input
                  type="checkbox"
                  className="mt-0.5"
                  checked={selected.has(e.id)}
                  onChange={() => toggleOne(e.id)}
                />
                <div className="min-w-0 flex-1">
                  <div className="text-muted-foreground flex items-center gap-2 text-[10px]">
                    <span className="font-mono">{e.id}</span>
                    <span>{new Date(e.created_at).toLocaleDateString()}</span>
                    {e.kind && (
                      <Badge variant="outline" className="font-mono">
                        {e.kind}
                      </Badge>
                    )}
                  </div>
                  {e.title && <div className="font-semibold">{e.title}</div>}
                  <p className="text-muted-foreground line-clamp-2 text-[11px]">
                    {e.content}
                  </p>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
