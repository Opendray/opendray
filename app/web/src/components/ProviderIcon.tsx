// ProviderIcon — renders the per-provider brand mark for the spawn
// dialog and provider rail. Uses the curated SVGs at
// app/web/public/icons/*.svg (served under /admin/icons/* in prod)
// so the marks are pixel-exact to the upstream brand assets, not the
// abbreviated simple-icons paths the rest of the admin uses.
//
// Monochrome marks (openai, shell) are authored as black-fill SVGs
// for max contrast on light backgrounds. The admin currently runs
// dark-only, so those get a CSS `invert` filter; full-colour marks
// (claude, gemini) ship as-is.

import { cn } from '@/lib/utils'

// Map opendray provider id → SVG filename (sans `.svg`). Stays
// separate from providerIconKey() so adding/replacing a per-provider
// asset doesn't accidentally affect the legacy BrandIcon fallback
// used elsewhere.
const PROVIDER_ICON_MAP: Record<string, string> = {
  claude: 'claude',
  codex: 'openai',
  gemini: 'gemini',
  shell: 'shell',
}

// IDs whose source SVG is a monochrome black mark. These need
// `invert` on dark themes so the glyph reads as light on dark.
const MONOCHROME_DARK_INVERT = new Set(['openai', 'shell'])

export interface ProviderIconProps {
  providerId: string | undefined
  // Pixel size of the rendered icon. Default matches the Sessions
  // list row glyph; SpawnDialog uses a larger 32 for the card.
  size?: number
  className?: string
  // When the provider id is unknown to PROVIDER_ICON_MAP, the caller
  // can supply a fallback letter for a neutral disc.
  fallbackLetter?: string
  title?: string
}

export function ProviderIcon({
  providerId,
  size = 24,
  className,
  fallbackLetter,
  title,
}: ProviderIconProps) {
  const key = providerId
    ? PROVIDER_ICON_MAP[providerId.toLowerCase()]
    : undefined
  if (!key) {
    // Unknown provider → letter disc. Keeps SpawnDialog usable when
    // an operator adds a custom provider id we haven't seeded an
    // asset for yet.
    const letter = (fallbackLetter ?? providerId ?? '?')
      .slice(0, 1)
      .toUpperCase()
    return (
      <div
        className={cn(
          'shrink-0 rounded-full bg-muted text-muted-foreground flex items-center justify-center',
          className,
        )}
        style={{ width: size, height: size }}
        aria-label={title}
        title={title}
      >
        <span
          className="font-semibold leading-none"
          style={{ fontSize: Math.round(size * 0.45) }}
        >
          {letter}
        </span>
      </div>
    )
  }
  const invert = MONOCHROME_DARK_INVERT.has(key)
  return (
    <img
      src={`/admin/icons/${key}.svg`}
      width={size}
      height={size}
      alt={title ?? key}
      title={title}
      className={cn(
        'shrink-0 object-contain',
        // Monochrome black SVGs invert on dark theme so the mark
        // reads against the popover surface. Brightness clamp keeps
        // the inverted white from looking glaring.
        invert && 'dark:invert dark:brightness-90',
        className,
      )}
      style={{ width: size, height: size }}
    />
  )
}
