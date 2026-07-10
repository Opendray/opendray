// Provider visual identity — single source of truth so the session
// list, workbench header, and any future surfaces (notifications,
// activity feed) use the same color + initial for a given provider.
//
// Colors are intentionally saturated tile-style backgrounds with
// white-ish text — they read as identity badges, not chrome.

export interface ProviderVisual {
  /** Tailwind background classes, e.g. "bg-amber-500" — applied to the avatar tile. */
  bg: string
  /** Tailwind text class for letter on top of `bg`. */
  fg: string
  /** Capital initial shown inside the avatar. */
  letter: string
  /** Human-readable label used in subtitles when no manifest is loaded. */
  name: string
}

const palette: Record<string, Omit<ProviderVisual, 'letter'>> = {
  claude: { bg: 'bg-orange-600', fg: 'text-white', name: 'Claude Code' },
  codex: { bg: 'bg-emerald-600', fg: 'text-white', name: 'Codex' },
  shell: { bg: 'bg-slate-600', fg: 'text-white', name: 'Shell' },
  grok: { bg: 'bg-neutral-900', fg: 'text-white', name: 'Grok' },
}

const fallback: Omit<ProviderVisual, 'letter'> = {
  bg: 'bg-zinc-700',
  fg: 'text-white',
  name: 'Provider',
}

export function providerVisual(id: string): ProviderVisual {
  const base = palette[id] ?? fallback
  const letter = (id?.[0] ?? '?').toUpperCase()
  return { ...base, letter }
}

// Trims a working directory to its trailing segment for compact
// display, e.g. /Users/me/work/billing-svc → "billing-svc".
export function cwdTail(cwd: string): string {
  const parts = cwd.split('/').filter(Boolean)
  return parts.length ? parts[parts.length - 1] : cwd
}
