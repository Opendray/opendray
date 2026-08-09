import { useEffect, useState } from 'react'

// Phone breakpoint. Below this every side panel (nav, list, inspector,
// outline) renders as a slide-over drawer so the middle workbench keeps
// the full width.
const MOBILE_QUERY = '(max-width: 767px)'

// Tablet-and-below breakpoint. Between this and MOBILE_QUERY there is
// room for exactly two columns — a list plus its detail. Anything that
// would be a *third* column (inspector, outline, graph side panel)
// becomes a drawer here, while the two primary columns stay inline.
const COMPACT_QUERY = '(max-width: 1023px)'

function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia(query).matches : false,
  )
  useEffect(() => {
    const mql = window.matchMedia(query)
    const onChange = () => setMatches(mql.matches)
    onChange()
    mql.addEventListener('change', onChange)
    return () => mql.removeEventListener('change', onChange)
  }, [query])
  return matches
}

/** True on phones (< 768px) — no room for any side column. */
export function useIsMobile(): boolean {
  return useMediaQuery(MOBILE_QUERY)
}

/** True on tablets and phones (< 1024px) — room for at most two columns. */
export function useIsCompact(): boolean {
  return useMediaQuery(COMPACT_QUERY)
}
