import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  Dialog,
  DialogContent,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'
import {
  createRoundTable,
  SEAT_MODEL_PLACEHOLDER,
  SEAT_PROVIDERS,
  SEAT_VENDOR,
  type RoundTable,
  type SeatProvider,
} from '@/lib/roundtable'

interface Props {
  open: boolean
  onClose: () => void
  onCreated: (rt: RoundTable) => void
}

// A round table needs at least two seats to be a discussion.
const DEFAULT_SEATS: SeatProvider[] = ['claude', 'codex', 'antigravity']

export function CreateRoundTableDialog({ open, onClose, onCreated }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [topic, setTopic] = useState('')
  const [cwd, setCwd] = useState('')
  const [seats, setSeats] = useState<SeatProvider[]>(DEFAULT_SEATS)
  // Optional per-seat model override (blank = the CLI's own default).
  const [models, setModels] = useState<Partial<Record<SeatProvider, string>>>(
    {},
  )

  const reset = () => {
    setTopic('')
    setCwd('')
    setSeats(DEFAULT_SEATS)
    setModels({})
  }

  const toggleSeat = (p: SeatProvider) =>
    setSeats((cur) =>
      cur.includes(p) ? cur.filter((s) => s !== p) : [...cur, p],
    )

  const create = useMutation({
    mutationFn: () =>
      createRoundTable({
        topic: topic.trim(),
        cwd: cwd.trim() || undefined,
        seats: seats.map((provider) => ({
          provider,
          model: models[provider]?.trim() || undefined,
        })),
      }),
    onSuccess: (rt) => {
      qc.invalidateQueries({ queryKey: ['round-tables'] })
      toast.success(t('web.roundTable.dialog.created'))
      reset()
      onCreated(rt)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const canSubmit = topic.trim().length > 0 && seats.length >= 2

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) onClose()
      }}
    >
      <DialogContent className="max-w-lg">
        <DialogTitle>{t('web.roundTable.dialog.title')}</DialogTitle>
        <p className="text-xs text-muted-foreground -mt-1">
          {t('web.roundTable.dialog.description')}
        </p>

        <div className="flex flex-col gap-4 mt-2">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="rt-topic">{t('web.roundTable.dialog.topic')}</Label>
            <Textarea
              id="rt-topic"
              value={topic}
              onChange={(e) => setTopic(e.target.value)}
              rows={3}
              placeholder={t('web.roundTable.dialog.topicPlaceholder')}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="rt-cwd">{t('web.roundTable.dialog.cwd')}</Label>
            <Input
              id="rt-cwd"
              value={cwd}
              onChange={(e) => setCwd(e.target.value)}
              placeholder="/path/to/project"
            />
            <span className="text-[11px] text-muted-foreground">
              {t('web.roundTable.dialog.cwdHint')}
            </span>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>{t('web.roundTable.dialog.seats')}</Label>
            <div className="flex flex-col gap-1.5">
              {SEAT_PROVIDERS.map((p) => {
                const on = seats.includes(p)
                return (
                  <div key={p} className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => toggleSeat(p)}
                      className={cn(
                        'flex w-36 shrink-0 flex-col items-start rounded-md border px-3 py-1.5 text-left transition-colors',
                        on
                          ? 'border-accent/40 bg-accent/10 text-foreground'
                          : 'border-border bg-card text-muted-foreground hover:text-foreground',
                      )}
                    >
                      <span className="text-[13px] font-medium capitalize">
                        {p}
                      </span>
                      <span className="text-[10px] text-muted-foreground">
                        {SEAT_VENDOR[p]}
                      </span>
                    </button>
                    {on && (
                      <Input
                        value={models[p] ?? ''}
                        onChange={(e) =>
                          setModels((cur) => ({ ...cur, [p]: e.target.value }))
                        }
                        placeholder={SEAT_MODEL_PLACEHOLDER[p]}
                        className="h-8 flex-1 text-xs"
                        aria-label={`${p} model`}
                      />
                    )}
                  </div>
                )
              })}
            </div>
            <span className="text-[11px] text-muted-foreground">
              {t('web.roundTable.dialog.seatsHint')}
            </span>
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-4">
          <Button variant="ghost" size="sm" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button
            size="sm"
            disabled={!canSubmit || create.isPending}
            onClick={() => create.mutate()}
          >
            {t('web.roundTable.dialog.create')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
