import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface LayoutState {
  /** Global left navigation collapsed to icon-only mode. */
  sidebarCollapsed: boolean
  /** Sessions inner list panel hidden so the workbench takes full width. */
  sessionListCollapsed: boolean
  /** Right-side inspector panel (Plugins / MCP / Files / Logs). */
  inspectorOpen: boolean
  /** UI scale applied via CSS zoom on <body>. 1 = default. */
  fontScale: number

  toggleSidebar: () => void
  toggleSessionList: () => void
  toggleInspector: () => void
  setFontScale: (v: number) => void
}

const FONT_SCALE_MIN = 0.7
const FONT_SCALE_MAX = 1.5

function clampScale(v: number): number {
  if (!Number.isFinite(v)) return 1
  return Math.min(FONT_SCALE_MAX, Math.max(FONT_SCALE_MIN, v))
}

// Apply on <body> rather than <html>: keeps `100svh`/`100vh` correct
// (zoom on <html> shifts the viewport math), but still scales every
// descendant — including hardcoded `text-[12px]`-style px values that
// won't respond to root font-size changes.
function applyFontScale(scale: number) {
  if (typeof document === 'undefined') return
  document.body.style.zoom = String(scale)
}

export const useLayout = create<LayoutState>()(
  persist(
    (set, get) => ({
      sidebarCollapsed: false,
      sessionListCollapsed: false,
      inspectorOpen: true,
      fontScale: 1,

      toggleSidebar: () =>
        set({ sidebarCollapsed: !get().sidebarCollapsed }),
      toggleSessionList: () =>
        set({ sessionListCollapsed: !get().sessionListCollapsed }),
      toggleInspector: () =>
        set({ inspectorOpen: !get().inspectorOpen }),
      setFontScale: (v) => {
        const next = clampScale(v)
        set({ fontScale: next })
        applyFontScale(next)
      },
    }),
    {
      name: 'opendray.layout',
      onRehydrateStorage: () => (state) => {
        if (state) applyFontScale(clampScale(state.fontScale))
      },
    },
  ),
)

// Apply on first load (before React mounts) so the initial paint is
// already at the persisted scale — no flash at default size.
if (typeof window !== 'undefined') {
  applyFontScale(clampScale(useLayout.getState().fontScale))
}
