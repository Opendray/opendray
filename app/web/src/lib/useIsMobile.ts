import { useEffect, useState } from 'react'

// Phone breakpoint. Below this the 3-pane workbench renders the side
// panels (nav, inspector) as slide-over drawers instead of inline
// columns, so the middle workbench keeps full width.
const MOBILE_QUERY = '(max-width: 767px)'

export function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia(MOBILE_QUERY).matches : false,
  )
  useEffect(() => {
    const mql = window.matchMedia(MOBILE_QUERY)
    const onChange = () => setIsMobile(mql.matches)
    onChange()
    mql.addEventListener('change', onChange)
    return () => mql.removeEventListener('change', onChange)
  }, [])
  return isMobile
}
