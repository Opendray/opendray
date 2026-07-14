import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import { listProjects } from '@/lib/projectDocs'
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

// Start with every vendor at the table by default.
const DEFAULT_SEATS: SeatProvider[] = ['claude', 'codex', 'antigravity']

// Sentinel for "no project" in the Select (empty string isn't a valid value).
const NO_PROJECT = '__none__'

export function CreateRoundTableDialog({ open, onClose, onCreated }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [seats, setSeats] = useState<SeatProvider[]>(DEFAULT_SEATS)
  const [models, setModels] = useState<Partial<Record<SeatProvider, string>>>(
    {},
  )
  const [cwd, setCwd] = useState<string>(NO_PROJECT)

  // Known projects for the optional binding dropdown (memory recall).
  const projects = useQuery({
    queryKey: ['known-projects'],
    queryFn: () => listProjects(),
    staleTime: 30_000,
    enabled: open,
  })

  const reset = () => {
    setSeats(DEFAULT_SEATS)
    setModels({})
    setCwd(NO_PROJECT)
  }

  const toggleSeat = (p: SeatProvider) =>
    setSeats((cur) =>
      cur.includes(p) ? cur.filter((s) => s !== p) : [...cur, p],
    )

  const create = useMutation({
    mutationFn: () =>
      createRoundTable({
        // No topic — the chat names itself from the first message.
        cwd: cwd === NO_PROJECT ? undefined : cwd,
        seats: seats.map((provider) => ({
          provider,
          model: models[provider]?.trim() || undefined,
        })),
      }),
    onSuccess: (rt) => {
      qc.invalidateQueries({ queryKey: ['round-tables'] })
      reset()
      onCreated(rt)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const canSubmit = seats.length >= 1

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

          <div className="flex flex-col gap-1.5">
            <Label>{t('web.roundTable.dialog.project')}</Label>
            <Select value={cwd} onValueChange={setCwd}>
              <SelectTrigger className="h-9">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NO_PROJECT}>
                  {t('web.roundTable.dialog.projectNone')}
                </SelectItem>
                {(projects.data ?? []).map((p) => (
                  <SelectItem key={p.cwd} value={p.cwd}>
                    {p.cwd}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <span className="text-[11px] text-muted-foreground">
              {t('web.roundTable.dialog.cwdHint')}
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
            {t('web.roundTable.dialog.start')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
