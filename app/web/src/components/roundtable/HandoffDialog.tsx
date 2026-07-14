import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { FolderOpen } from 'lucide-react'

import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { FileBrowserDialog } from '@/components/sessions/FileBrowserDialog'
import { cn } from '@/lib/utils'
import { handoffRoundTable, type RoundTable } from '@/lib/roundtable'

interface Props {
  rt: RoundTable
  open: boolean
  onClose: () => void
  onDone: (sessionId: string) => void
}

// Default executor: claude if it's at the table (strongest for file edits),
// else the first seat.
function defaultExecutor(rt: RoundTable): string {
  if (rt.seats.some((s) => s.provider === 'claude')) return 'claude'
  return rt.seats[0]?.provider ?? 'claude'
}

export function HandoffDialog({ rt, open, onClose, onDone }: Props) {
  const { t } = useTranslation()
  const [provider, setProvider] = useState(() => defaultExecutor(rt))
  const [cwd, setCwd] = useState(rt.cwd ?? '')
  const [browserOpen, setBrowserOpen] = useState(false)

  const run = useMutation({
    mutationFn: () => handoffRoundTable(rt.id, { provider, cwd: cwd.trim() }),
    onSuccess: ({ session_id }) => {
      toast.success(t('web.roundTable.handoff.started'))
      onDone(session_id)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const canRun = provider !== '' && cwd.trim() !== ''

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) onClose()
      }}
    >
      <DialogContent className="max-w-lg">
        <DialogTitle>{t('web.roundTable.handoff.title')}</DialogTitle>
        <p className="text-xs text-muted-foreground -mt-1">
          {t('web.roundTable.handoff.description')}
        </p>

        <div className="mt-2 flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label>{t('web.roundTable.handoff.executor')}</Label>
            <div className="flex flex-wrap gap-2">
              {rt.seats.map((s) => (
                <button
                  key={s.provider}
                  type="button"
                  onClick={() => setProvider(s.provider)}
                  className={cn(
                    'rounded-md border px-3 py-1.5 text-[13px] font-medium capitalize transition-colors',
                    provider === s.provider
                      ? 'border-accent/40 bg-accent/10 text-foreground'
                      : 'border-border bg-card text-muted-foreground hover:text-foreground',
                  )}
                >
                  {s.provider}
                </button>
              ))}
            </div>
            <span className="text-[11px] text-muted-foreground">
              {t('web.roundTable.handoff.executorHint')}
            </span>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="handoff-cwd">
              {t('web.roundTable.handoff.project')}
            </Label>
            <div className="flex gap-1.5">
              <Input
                id="handoff-cwd"
                value={cwd}
                onChange={(e) => setCwd(e.target.value)}
                placeholder={t('web.roundTable.handoff.projectPlaceholder')}
                className="flex-1 font-mono text-xs"
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setBrowserOpen(true)}
                className="shrink-0 gap-1"
              >
                <FolderOpen className="size-3.5" />
                {t('web.roundTable.dialog.browse')}
              </Button>
            </div>
            <span className="text-[11px] text-muted-foreground">
              {t('web.roundTable.handoff.projectHint')}
            </span>
          </div>
        </div>

        <div className="mt-4 flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button
            size="sm"
            disabled={!canRun || run.isPending}
            onClick={() => run.mutate()}
          >
            {t('web.roundTable.handoff.run')}
          </Button>
        </div>

        <FileBrowserDialog
          open={browserOpen}
          onOpenChange={setBrowserOpen}
          initialPath={cwd.trim() || undefined}
          onSelect={(p) => setCwd(p)}
        />
      </DialogContent>
    </Dialog>
  )
}
