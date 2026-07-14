import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { FolderOpen } from 'lucide-react'

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
import { FileBrowserDialog } from '@/components/sessions/FileBrowserDialog'
import { cn } from '@/lib/utils'
import {
  createRoundTable,
  listSeatModels,
  SEAT_MODEL_DEFAULT,
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

// Radix Select forbids an empty-string item value, so "" (CLI default) is
// represented by this sentinel in the dropdown and mapped back to "".
const DEFAULT_MODEL = '__default__'

export function CreateRoundTableDialog({ open, onClose, onCreated }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [seats, setSeats] = useState<SeatProvider[]>(DEFAULT_SEATS)
  // Per-seat model selection ("" = CLI default). Pre-seeded with sensible
  // defaults (e.g. codex → gpt-5.4-mini) so nothing has to be typed.
  const [models, setModels] = useState<Partial<Record<SeatProvider, string>>>(
    () => ({ ...SEAT_MODEL_DEFAULT }),
  )
  const [cwd, setCwd] = useState('')
  const [browserOpen, setBrowserOpen] = useState(false)

  // Selectable models per provider (antigravity enumerated live from the CLI).
  const modelsQuery = useQuery({
    queryKey: ['round-table-models'],
    queryFn: listSeatModels,
    staleTime: 60_000,
    enabled: open,
  })

  const reset = () => {
    setSeats(DEFAULT_SEATS)
    setModels({ ...SEAT_MODEL_DEFAULT })
    setCwd('')
  }

  const toggleSeat = (p: SeatProvider) =>
    setSeats((cur) =>
      cur.includes(p) ? cur.filter((s) => s !== p) : [...cur, p],
    )

  const create = useMutation({
    mutationFn: () =>
      createRoundTable({
        // No topic — the chat names itself from the first message.
        cwd: cwd.trim() || undefined,
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
                const options = modelsQuery.data?.[p] ?? []
                const current = models[p] ?? ''
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
                      <Select
                        value={current === '' ? DEFAULT_MODEL : current}
                        onValueChange={(v) =>
                          setModels((cur) => ({
                            ...cur,
                            [p]: v === DEFAULT_MODEL ? '' : v,
                          }))
                        }
                      >
                        <SelectTrigger className="h-8 flex-1 text-xs">
                          <SelectValue
                            placeholder={t('web.roundTable.dialog.modelPlaceholder')}
                          />
                        </SelectTrigger>
                        <SelectContent>
                          {options.length === 0 && (
                            <SelectItem value={DEFAULT_MODEL}>
                              {t('web.roundTable.dialog.modelLoading')}
                            </SelectItem>
                          )}
                          {options.map((m) => (
                            <SelectItem
                              key={m.value || DEFAULT_MODEL}
                              value={m.value || DEFAULT_MODEL}
                            >
                              {m.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
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
            <Label htmlFor="rt-cwd">{t('web.roundTable.dialog.project')}</Label>
            <div className="flex gap-1.5">
              <Input
                id="rt-cwd"
                value={cwd}
                onChange={(e) => setCwd(e.target.value)}
                placeholder={t('web.roundTable.dialog.cwdPlaceholder')}
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
