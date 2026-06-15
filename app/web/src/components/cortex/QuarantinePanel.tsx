// QuarantinePanel — the review queue for memories captured from
// third-party (integration) sessions (Cortex Phase 2). Quarantined
// facts never feed consolidation or spawn injection; the operator
// promotes the valuable ones into durable memory or discards the rest
// (un-reviewed rows expire on their TTL automatically).

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Check, Loader2, ShieldQuestion, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { discardQuarantined, listQuarantined, promoteQuarantined } from '@/lib/cortex'

export function QuarantinePanel() {
  const { t } = useTranslation()
  const qc = useQueryClient()

  const query = useQuery({
    queryKey: ['cortex-quarantine'],
    queryFn: () => listQuarantined(200),
  })
  const refresh = () => {
    qc.invalidateQueries({ queryKey: ['cortex-quarantine'] })
    qc.invalidateQueries({ queryKey: ['cortex-status'] })
    qc.invalidateQueries({ queryKey: ['memories'] })
  }

  const promote = useMutation({
    mutationFn: (id: string) => promoteQuarantined(id),
    onSuccess: () => {
      toast.success(t('web.cortex.quarantine.promotedToast'))
      refresh()
    },
    onError: (e: Error) =>
      toast.error(t('web.cortex.quarantine.actionFailed'), { description: e.message }),
  })
  const discard = useMutation({
    mutationFn: (id: string) => discardQuarantined(id),
    onSuccess: () => {
      toast.success(t('web.cortex.quarantine.discardedToast'))
      refresh()
    },
    onError: (e: Error) =>
      toast.error(t('web.cortex.quarantine.actionFailed'), { description: e.message }),
  })

  const rows = query.data?.memories ?? []

  return (
    <div className="space-y-3 p-4">
      <div>
        <h2 className="flex items-center gap-2 text-base font-semibold">
          <ShieldQuestion className="h-4 w-4" />
          {t('web.cortex.quarantine.title')}
          {query.data && query.data.count > 0 && (
            <Badge variant="warning" className="text-[10px]">
              {query.data.count}
            </Badge>
          )}
        </h2>
        <p className="text-muted-foreground mt-1 text-xs">
          {t('web.cortex.quarantine.subtitle')}
        </p>
      </div>

      {query.isLoading ? (
        <Loader2 className="h-4 w-4 animate-spin" />
      ) : rows.length === 0 ? (
        <p className="text-muted-foreground py-10 text-center text-sm">
          {t('web.cortex.quarantine.empty')}
        </p>
      ) : (
        rows.map((m) => (
          <div key={m.id} className="bg-card rounded-md border p-3">
            <div className="mb-2 flex flex-wrap items-center gap-2 text-[11px]">
              <Badge variant="outline" className="font-mono text-[10px]">
                {m.scope === 'global' ? 'global' : m.scope_key}
              </Badge>
              {typeof m.metadata?.integration_id === 'string' && (
                <Badge variant="muted" className="font-mono text-[10px]">
                  {String(m.metadata.integration_id)}
                </Badge>
              )}
              <span className="text-muted-foreground">
                {new Date(m.created_at).toLocaleString()}
              </span>
              {m.quarantine_expires_at && (
                <span className="text-muted-foreground ml-auto">
                  {t('web.cortex.quarantine.expires', {
                    date: new Date(m.quarantine_expires_at).toLocaleDateString(),
                  })}
                </span>
              )}
            </div>
            <pre className="bg-muted/20 mb-2.5 max-h-32 overflow-auto rounded p-2 font-mono text-[11px] whitespace-pre-wrap">
              {m.text}
            </pre>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="default"
                disabled={promote.isPending || discard.isPending}
                onClick={() => promote.mutate(m.id)}
                title={t('web.cortex.quarantine.promoteHint')}
              >
                <Check className="mr-1 h-3 w-3" />
                {t('web.cortex.quarantine.promote')}
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="text-destructive hover:text-destructive"
                disabled={promote.isPending || discard.isPending}
                onClick={() => discard.mutate(m.id)}
              >
                <Trash2 className="mr-1 h-3 w-3" />
                {t('web.cortex.quarantine.discard')}
              </Button>
            </div>
          </div>
        ))
      )}
    </div>
  )
}
