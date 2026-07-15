import { useEffect, useState } from 'react'
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
  SEAT_SUPPORTS_ACCOUNT,
  SEAT_VENDOR,
  type RoundTable,
  type SeatProvider,
} from '@/lib/roundtable'
import { listClaudeAccounts } from '@/lib/claudeAccounts'
import { listAntigravityAccounts } from '@/lib/antigravityAccounts'

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

// Radix Select forbids an empty-string item value, so "" (CLI default) is
// represented by these sentinels in the dropdowns and mapped back to "".
const DEFAULT_MODEL = '__default__'
const DEFAULT_ACCOUNT = '__default__'

// A seat account option, normalised across the claude/antigravity account
// shapes (both share id / display_name / name / token_filled).
interface AccountOption {
  id: string
  label: string
  usable: boolean
}

// Quick-fill role presets for the persona field. The localized label IS the
// persona text sent to the model (models handle any language) — one string,
// editable after tapping.
const PERSONA_PRESET_KEYS = [
  'security',
  'performance',
  'ux',
  'skeptic',
  'pragmatist',
] as const

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

  // Auto-pick the first account for a seated claude/antigravity provider that
  // has 2+ accounts and none chosen yet: with multiple accounts the CLI's
  // "default" is ambiguous (and claude may hit a login prompt), so force a
  // concrete pick — mirrors SpawnDialog's multi-account behaviour.
  useEffect(() => {
    if (!open) return
    setSeatAccounts((cur) => {
      let next = cur
      for (const p of seats) {
        if (!SEAT_SUPPORTS_ACCOUNT.has(p)) continue
        const accts = accountsFor(p)
        if (accts.length >= 2 && !cur[p]) {
          const first = accts.find((a) => a.usable) ?? accts[0]
          next = next === cur ? { ...cur } : next
          next[p] = first.id
        }
      }
      return next
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, seats, claudeAccountsQuery.data, agyAccountsQuery.data])

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
            <div className="flex flex-col gap-1.5">
              {SEAT_PROVIDERS.map((p) => {
                const on = seats.includes(p)
                const options = modelsQuery.data?.[p] ?? []
                const current = models[p] ?? ''
                const accts = accountsFor(p)
                const showAccount =
                  on && SEAT_SUPPORTS_ACCOUNT.has(p) && accts.length > 0
                // With 2+ accounts the "default" is ambiguous, so require a
                // concrete pick (auto-picked above); with a single account,
                // offer Default too.
                const forcePick = accts.length >= 2
                const currentAcct = seatAccounts[p] ?? ''
                return (
                  <div key={p} className="flex flex-col gap-1">
                    <div className="flex items-center gap-2">
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
                    {showAccount && (
                      <div className="flex items-center gap-2 pl-[10.5rem]">
                        <Select
                          value={currentAcct === '' ? DEFAULT_ACCOUNT : currentAcct}
                          onValueChange={(v) =>
                            setSeatAccounts((cur) => ({
                              ...cur,
                              [p]: v === DEFAULT_ACCOUNT ? '' : v,
                            }))
                          }
                        >
                          <SelectTrigger className="h-7 flex-1 text-xs">
                            <SelectValue
                              placeholder={t('web.roundTable.dialog.accountPlaceholder')}
                            />
                          </SelectTrigger>
                          <SelectContent>
                            {!forcePick && (
                              <SelectItem value={DEFAULT_ACCOUNT}>
                                {t('web.roundTable.dialog.accountDefault')}
                              </SelectItem>
                            )}
                            {accts.map((a) => (
                              <SelectItem
                                key={a.id}
                                value={a.id}
                                disabled={!a.usable}
                              >
                                {a.label}
                                {a.usable
                                  ? ''
                                  : ` (${t('web.roundTable.dialog.accountNoToken')})`}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    )}
                    {on && (
                      <div className="flex flex-col gap-1 pl-[10.5rem]">
                        <textarea
                          value={personas[p] ?? ''}
                          onChange={(e) =>
                            setPersonas((cur) => ({ ...cur, [p]: e.target.value }))
                          }
                          placeholder={t('web.roundTable.dialog.personaPlaceholder')}
                          rows={2}
                          className="w-full resize-none rounded-md border border-border bg-background px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
                        />
                        <div className="flex flex-wrap items-center gap-1">
                          <span className="text-[10px] text-muted-foreground">
                            {t('web.roundTable.dialog.personaPresets.label')}
                          </span>
                          {PERSONA_PRESET_KEYS.map((key) => {
                            const label = t(
                              `web.roundTable.dialog.personaPresets.${key}`,
                            )
                            return (
                              <button
                                key={key}
                                type="button"
                                onClick={() =>
                                  setPersonas((cur) => ({ ...cur, [p]: label }))
                                }
                                className="rounded-full border border-border bg-card px-1.5 py-0.5 text-[10px] text-muted-foreground transition-colors hover:text-foreground"
                              >
                                {label}
                              </button>
                            )
                          })}
                        </div>
                      </div>
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
