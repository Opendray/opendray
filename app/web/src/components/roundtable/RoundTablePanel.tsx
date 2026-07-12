import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Loader2, Plus, Users } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { listRoundTables, type RoundTableStatus } from '@/lib/roundtable'
import { CreateRoundTableDialog } from './CreateRoundTableDialog'
import { RoundTableDetail } from './RoundTableDetail'

const STATUS_VARIANT: Record<
  RoundTableStatus,
  'muted' | 'accent' | 'success' | 'danger' | 'outline'
> = {
  draft: 'muted',
  running: 'accent',
  awaiting_verdict: 'success',
  failed: 'danger',
  closed: 'outline',
}

export function RoundTablePanel() {
  const { t } = useTranslation()
  const [selected, setSelected] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)

  const list = useQuery({
    queryKey: ['round-tables'],
    queryFn: () => listRoundTables(),
    // Keep statuses fresh while a discussion runs somewhere in the list.
    refetchInterval: (q) =>
      (q.state.data ?? []).some((rt) => rt.status === 'running') ? 4000 : false,
  })

  const tables = list.data ?? []

  return (
    <div className="flex h-full min-h-0 gap-4">
      {/* List column */}
      <div className="flex w-72 shrink-0 flex-col gap-2">
        <Button size="sm" onClick={() => setDialogOpen(true)}>
          <Plus className="size-3.5" />
          {t('web.roundTable.new')}
        </Button>

        {list.isLoading ? (
          <div className="flex items-center gap-2 p-4 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin" />
            {t('web.roundTable.loading')}
          </div>
        ) : tables.length === 0 ? (
          <div className="flex flex-col items-center gap-2 rounded-md border border-dashed border-border p-6 text-center">
            <Users className="size-6 text-muted-foreground" />
            <div className="text-xs text-muted-foreground">
              {t('web.roundTable.empty')}
            </div>
          </div>
        ) : (
          <div className="flex flex-col gap-1 overflow-y-auto">
            {tables.map((rt) => (
              <button
                key={rt.id}
                type="button"
                onClick={() => setSelected(rt.id)}
                className={cn(
                  'flex flex-col gap-1 rounded-md border px-3 py-2 text-left transition-colors',
                  selected === rt.id
                    ? 'border-accent/40 bg-accent/10'
                    : 'border-border bg-card/40 hover:bg-card',
                )}
              >
                <div className="flex items-center gap-1.5">
                  <span className="flex-1 truncate text-[13px] font-medium">
                    {rt.topic}
                  </span>
                  <Badge variant={STATUS_VARIANT[rt.status]}>
                    {t(`web.roundTable.status.${rt.status}`)}
                  </Badge>
                </div>
                <div className="flex items-center gap-1 text-[10px] text-muted-foreground">
                  {rt.seats.map((s) => s.provider).join(' · ')}
                </div>
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Detail column */}
      <div className="min-w-0 flex-1 overflow-y-auto">
        {selected ? (
          <RoundTableDetail id={selected} />
        ) : (
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
            {t('web.roundTable.selectHint')}
          </div>
        )}
      </div>

      <CreateRoundTableDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        onCreated={(rt) => {
          setDialogOpen(false)
          setSelected(rt.id)
        }}
      />
    </div>
  )
}
