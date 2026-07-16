import { useEffect } from 'react'

import { SEAT_SUPPORTS_ACCOUNT, type SeatProvider } from '@/lib/roundtable'

// Non-component helpers for the Round Table seat picker, kept out of the
// component file so react-refresh only sees component exports there.

// Radix Select forbids an empty-string item value, so "" (CLI default) is
// represented by these sentinels in the dropdowns and mapped back to "".
export const DEFAULT_MODEL = '__default__'
export const DEFAULT_ACCOUNT = '__default__'

// A seat account option, normalised across the claude/antigravity account
// shapes (both share id / display_name / name / token_filled).
export interface AccountOption {
  id: string
  label: string
  usable: boolean
}

// Quick-fill role presets for the persona field. The localized label IS the
// persona text sent to the model (models handle any language) — one string,
// editable after tapping.
export const PERSONA_PRESET_KEYS = [
  'security',
  'performance',
  'ux',
  'skeptic',
  'pragmatist',
] as const

// useAutoPickAccounts forces a concrete account choice for any seated
// claude/antigravity provider that has 2+ accounts and none chosen yet: with
// multiple accounts the CLI's "default" is ambiguous (and claude may hit a
// login prompt). Mirrors SpawnDialog's multi-account behaviour. Shared by the
// create dialog (new table) and the members editor (seat added mid-chat).
export function useAutoPickAccounts(
  enabled: boolean,
  seats: SeatProvider[],
  accountsFor: (p: SeatProvider) => AccountOption[],
  setSeatAccounts: React.Dispatch<
    React.SetStateAction<Partial<Record<SeatProvider, string>>>
  >,
  deps: unknown[],
) {
  useEffect(() => {
    if (!enabled) return
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
  }, deps)
}
