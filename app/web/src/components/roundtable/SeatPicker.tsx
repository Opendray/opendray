import { useTranslation } from 'react-i18next'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import {
  SEAT_PROVIDERS,
  SEAT_SUPPORTS_ACCOUNT,
  SEAT_VENDOR,
  type SeatModelOption,
  type SeatProvider,
} from '@/lib/roundtable'
import {
  DEFAULT_ACCOUNT,
  DEFAULT_MODEL,
  PERSONA_PRESET_KEYS,
  type AccountOption,
} from './seatPickerHelpers'

interface SeatPickerProps {
  // Which providers are currently seated.
  seats: SeatProvider[]
  // Per-seat model / account / persona state ("" = CLI default).
  models: Partial<Record<SeatProvider, string>>
  seatAccounts: Partial<Record<SeatProvider, string>>
  personas: Partial<Record<SeatProvider, string>>
  // Selectable models per provider (enumerated live for some CLIs).
  modelOptions: Record<string, SeatModelOption[]>
  // Enabled accounts for a provider (empty for those without multi-account).
  accountsFor: (p: SeatProvider) => AccountOption[]
  onToggle: (p: SeatProvider) => void
  onModel: (p: SeatProvider, value: string) => void
  onAccount: (p: SeatProvider, value: string) => void
  onPersona: (p: SeatProvider, value: string) => void
}

// SeatPicker renders the per-vendor seat list — a toggle button, and for
// seated members a model dropdown, an optional account dropdown
// (claude/antigravity) and a persona field with role presets. Presentational:
// the parent owns all state so the same picker drives both creating a table
// and editing its members live.
export function SeatPicker({
  seats,
  models,
  seatAccounts,
  personas,
  modelOptions,
  accountsFor,
  onToggle,
  onModel,
  onAccount,
  onPersona,
}: SeatPickerProps) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col gap-1.5">
      {SEAT_PROVIDERS.map((p) => {
        const on = seats.includes(p)
        const options = modelOptions[p] ?? []
        const current = models[p] ?? ''
        const accts = accountsFor(p)
        const showAccount =
          on && SEAT_SUPPORTS_ACCOUNT.has(p) && accts.length > 0
        // With 2+ accounts the "default" is ambiguous, so require a concrete
        // pick (auto-picked by the parent); with a single account, offer
        // Default too.
        const forcePick = accts.length >= 2
        const currentAcct = seatAccounts[p] ?? ''
        return (
          <div key={p} className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => onToggle(p)}
                className={cn(
                  'flex w-36 shrink-0 flex-col items-start rounded-md border px-3 py-1.5 text-left transition-colors',
                  on
                    ? 'border-accent/40 bg-accent/10 text-foreground'
                    : 'border-border bg-card text-muted-foreground hover:text-foreground',
                )}
              >
                <span className="text-[13px] font-medium capitalize">{p}</span>
                <span className="text-[10px] text-muted-foreground">
                  {SEAT_VENDOR[p]}
                </span>
              </button>
              {on && (
                <Select
                  value={current === '' ? DEFAULT_MODEL : current}
                  onValueChange={(v) =>
                    onModel(p, v === DEFAULT_MODEL ? '' : v)
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
                    onAccount(p, v === DEFAULT_ACCOUNT ? '' : v)
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
                      <SelectItem key={a.id} value={a.id} disabled={!a.usable}>
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
                  onChange={(e) => onPersona(p, e.target.value)}
                  placeholder={t('web.roundTable.dialog.personaPlaceholder')}
                  rows={2}
                  className="w-full resize-none rounded-md border border-border bg-background px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
                />
                <div className="flex flex-wrap items-center gap-1">
                  <span className="text-[10px] text-muted-foreground">
                    {t('web.roundTable.dialog.personaPresets.label')}
                  </span>
                  {PERSONA_PRESET_KEYS.map((key) => {
                    const label = t(`web.roundTable.dialog.personaPresets.${key}`)
                    return (
                      <button
                        key={key}
                        type="button"
                        onClick={() => onPersona(p, label)}
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
  )
}
