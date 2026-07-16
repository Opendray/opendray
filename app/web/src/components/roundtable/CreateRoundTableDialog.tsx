import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { FolderOpen } from 'lucide-react'

import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { FileBrowserDialog } from '@/components/sessions/FileBrowserDialog'
import {
  createRoundTable,
  listSeatModels,
  SEAT_MODEL_DEFAULT,
  SEAT_SUPPORTS_ACCOUNT,
  type RoundTable,
  type SeatProvider,
} from '@/lib/roundtable'
import { listClaudeAccounts } from '@/lib/claudeAccounts'
import { listAntigravityAccounts } from '@/lib/antigravityAccounts'
import { SeatPicker } from './SeatPicker'
import { useAutoPickAccounts, type AccountOption } from './seatPickerHelpers'

interface Props {
  open: boolean
  onClose: () => void
  onCreated: (rt: RoundTable) => void
}

// Seat the three vendors that work out of the box by default. grok and
// opencode are offered as toggles (see SEAT_PROVIDERS) but off initially —
// grok needs the CLI installed + `grok login` on the gateway host, and
// opencode needs its own provider auth, so defaulting them on would create
// failing seats before they're configured.
const DEFAULT_SEATS: SeatProvider[] = ['claude', 'codex', 'antigravity']

export function CreateRoundTableDialog({ open, onClose, onCreated }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [seats, setSeats] = useState<SeatProvider[]>(DEFAULT_SEATS)
  // Per-seat model selection ("" = CLI default). Pre-seeded with sensible
  // defaults (e.g. codex → gpt-5.4-mini) so nothing has to be typed.
  const [models, setModels] = useState<Partial<Record<SeatProvider, string>>>(
    () => ({ ...SEAT_MODEL_DEFAULT }),
  )
  // Per-seat account pin ("" = CLI default). Only claude / antigravity seats
  // use it (SEAT_SUPPORTS_ACCOUNT).
  const [seatAccounts, setSeatAccounts] = useState<
    Partial<Record<SeatProvider, string>>
  >({})
  // Per-seat persona / role (optional). Gives each member a distinct lens on
  // top of its vendor voice.
  const [personas, setPersonas] = useState<Partial<Record<SeatProvider, string>>>(
    {},
  )
  const [cwd, setCwd] = useState('')
  const [framing, setFraming] = useState('')
  const [browserOpen, setBrowserOpen] = useState(false)

  // Selectable models per provider (antigravity enumerated live from the CLI).
  const modelsQuery = useQuery({
    queryKey: ['round-table-models'],
    queryFn: listSeatModels,
    staleTime: 60_000,
    enabled: open,
  })

  // Multi-account sources — claude (config dir + OAuth) and antigravity
  // (dedicated HOME). Only enabled accounts can be pinned to a seat.
  const claudeAccountsQuery = useQuery({
    queryKey: ['claude-accounts'],
    queryFn: listClaudeAccounts,
    enabled: open,
  })
  const agyAccountsQuery = useQuery({
    queryKey: ['antigravity-accounts'],
    queryFn: listAntigravityAccounts,
    enabled: open,
  })
  const accountsFor = (p: SeatProvider): AccountOption[] => {
    const raw =
      p === 'claude'
        ? claudeAccountsQuery.data
        : p === 'antigravity'
          ? agyAccountsQuery.data
          : undefined
    return (raw ?? [])
      .filter((a) => a.enabled)
      .map((a) => ({
        id: a.id,
        label: a.display_name || a.name,
        usable: a.token_filled,
      }))
  }

  // Auto-pick a concrete account for seated claude/antigravity providers with
  // 2+ accounts (shared with the members editor).
  useAutoPickAccounts(open, seats, accountsFor, setSeatAccounts, [
    open,
    seats,
    claudeAccountsQuery.data,
    agyAccountsQuery.data,
  ])

  const reset = () => {
    setSeats(DEFAULT_SEATS)
    setModels({ ...SEAT_MODEL_DEFAULT })
    setSeatAccounts({})
    setPersonas({})
    setCwd('')
    setFraming('')
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
        framing: framing.trim() || undefined,
        seats: seats.map((provider) => ({
          provider,
          model: models[provider]?.trim() || undefined,
          account_id: SEAT_SUPPORTS_ACCOUNT.has(provider)
            ? seatAccounts[provider]?.trim() || undefined
            : undefined,
          persona: personas[provider]?.trim() || undefined,
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
            <SeatPicker
              seats={seats}
              models={models}
              seatAccounts={seatAccounts}
              personas={personas}
              modelOptions={modelsQuery.data ?? {}}
              accountsFor={accountsFor}
              onToggle={toggleSeat}
              onModel={(p, v) => setModels((cur) => ({ ...cur, [p]: v }))}
              onAccount={(p, v) =>
                setSeatAccounts((cur) => ({ ...cur, [p]: v }))
              }
              onPersona={(p, v) =>
                setPersonas((cur) => ({ ...cur, [p]: v }))
              }
            />
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

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="rt-framing">
              {t('web.roundTable.dialog.framing')}
            </Label>
            <textarea
              id="rt-framing"
              value={framing}
              onChange={(e) => setFraming(e.target.value)}
              placeholder={t('web.roundTable.dialog.framingPlaceholder')}
              rows={2}
              className="w-full resize-none rounded-md border border-border bg-background px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
            />
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
