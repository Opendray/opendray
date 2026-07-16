import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  listSeatModels,
  updateRoundTable,
  SEAT_MODEL_DEFAULT,
  SEAT_SUPPORTS_ACCOUNT,
  type RoundTable,
  type Seat,
  type SeatProvider,
} from '@/lib/roundtable'
import { listClaudeAccounts } from '@/lib/claudeAccounts'
import { listAntigravityAccounts } from '@/lib/antigravityAccounts'
import { SeatPicker } from './SeatPicker'
import { useAutoPickAccounts, type AccountOption } from './seatPickerHelpers'

// Live members + roles editor — add or remove members, reassign each seat's
// model/account/persona and the shared framing directive on an ACTIVE round
// table as the topic evolves. Reuses the create dialog's SeatPicker; the
// backend re-reads seats each reply, so a member added here is @mentionable
// on the next turn and a removed one stops replying (its past messages stay).
interface Props {
  rt: RoundTable
  open: boolean
  onClose: () => void
}

export function RolesDialog({ rt, open, onClose }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [framing, setFraming] = useState(rt.framing ?? '')
  // Seed the editable state from the table's current seats.
  const [seats, setSeats] = useState<SeatProvider[]>(() =>
    rt.seats.map((s) => s.provider),
  )
  const [models, setModels] = useState<Partial<Record<SeatProvider, string>>>(
    () => Object.fromEntries(rt.seats.map((s) => [s.provider, s.model ?? ''])),
  )
  const [seatAccounts, setSeatAccounts] = useState<
    Partial<Record<SeatProvider, string>>
  >(() =>
    Object.fromEntries(rt.seats.map((s) => [s.provider, s.account_id ?? ''])),
  )
  const [personas, setPersonas] = useState<Partial<Record<SeatProvider, string>>>(
    () => Object.fromEntries(rt.seats.map((s) => [s.provider, s.persona ?? ''])),
  )

  const modelsQuery = useQuery({
    queryKey: ['round-table-models'],
    queryFn: listSeatModels,
    staleTime: 60_000,
    enabled: open,
  })
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

  // Auto-pick a concrete account for a newly-added claude/antigravity seat
  // with 2+ accounts (shared with the create dialog).
  useAutoPickAccounts(open, seats, accountsFor, setSeatAccounts, [
    open,
    seats,
    claudeAccountsQuery.data,
    agyAccountsQuery.data,
  ])

  // Toggling a member on pre-seeds its default model (e.g. codex → gpt-5.4-mini)
  // the first time, matching the create dialog.
  const toggleSeat = (p: SeatProvider) =>
    setSeats((cur) => {
      if (cur.includes(p)) return cur.filter((s) => s !== p)
      setModels((m) => (p in m ? m : { ...m, [p]: SEAT_MODEL_DEFAULT[p] ?? '' }))
      return [...cur, p]
    })

  const save = useMutation({
    mutationFn: () =>
      updateRoundTable(rt.id, {
        framing: framing.trim(),
        seats: seats.map<Seat>((provider) => ({
          provider,
          model: models[provider]?.trim() || undefined,
          account_id: SEAT_SUPPORTS_ACCOUNT.has(provider)
            ? seatAccounts[provider]?.trim() || undefined
            : undefined,
          persona: personas[provider]?.trim() || undefined,
        })),
      }),
    onSuccess: () => {
      toast.success(t('web.roundTable.detail.rolesSaved'))
      qc.invalidateQueries({ queryKey: ['round-table', rt.id] })
      qc.invalidateQueries({ queryKey: ['round-tables'] })
      onClose()
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const canSave = seats.length >= 1

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) onClose()
      }}
    >
      <DialogContent className="max-w-lg">
        <DialogTitle>{t('web.roundTable.detail.rolesTitle')}</DialogTitle>
        <p className="-mt-1 text-xs text-muted-foreground">
          {t('web.roundTable.detail.rolesHint')}
        </p>

        <div className="mt-2 flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="roles-framing">
              {t('web.roundTable.detail.rolesFraming')}
            </Label>
            <textarea
              id="roles-framing"
              value={framing}
              onChange={(e) => setFraming(e.target.value)}
              placeholder={t('web.roundTable.dialog.framingPlaceholder')}
              rows={3}
              className="w-full resize-none rounded-md border border-border bg-background px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>

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
            {!canSave && (
              <span className="text-[11px] text-state-failed">
                {t('web.roundTable.detail.membersMin')}
              </span>
            )}
          </div>
        </div>

        <div className="mt-4 flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button
            size="sm"
            disabled={!canSave || save.isPending}
            onClick={() => save.mutate()}
          >
            {t('web.roundTable.detail.rolesSave')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
