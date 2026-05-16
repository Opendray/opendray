import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface LayoutState {
  /** Global left navigation collapsed to icon-only mode. */
  sidebarCollapsed: boolean
  /** Sessions inner list panel hidden so the workbench takes full width. */
  sessionListCollapsed: boolean
  /** Right-side inspector panel (Plugins / MCP / Files / Logs). */
  inspectorOpen: boolean
  /** Inspector width in CSS pixels. User-resizable via the drag
   *  handle on the inspector's left edge. */
  inspectorWidth: number
  /** UI scale applied via CSS zoom on <body>. 1 = default. */
  fontScale: number

  toggleSidebar: () => void
  toggleSessionList: () => void
  toggleInspector: () => void
  setInspectorWidth: (v: number) => void
  setFontScale: (v: number) => void
}

const FONT_SCALE_MIN = 0.7
const FONT_SCALE_MAX = 1.5

// Inspector drag range. 320 matches the old hardcoded `w-80` so
// existing sessions don't visibly shift on first load; 900 keeps
// the terminal usable on a 1440px workspace.
export const INSPECTOR_WIDTH_MIN = 320
export const INSPECTOR_WIDTH_MAX = 900
export const INSPECTOR_WIDTH_DEFAULT = 320

function clampScale(v: number): number {
  if (!Number.isFinite(v)) return 1
  return Math.min(FONT_SCALE_MAX, Math.max(FONT_SCALE_MIN, v))
}

function clampInspectorWidth(v: number): number {
  if (!Number.isFinite(v)) return INSPECTOR_WIDTH_DEFAULT
  return Math.min(
    INSPECTOR_WIDTH_MAX,
    Math.max(INSPECTOR_WIDTH_MIN, Math.round(v)),
  )
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
      inspectorWidth: INSPECTOR_WIDTH_DEFAULT,
      fontScale: 1,

      toggleSidebar: () =>
        set({ sidebarCollapsed: !get().sidebarCollapsed }),
      toggleSessionList: () =>
        set({ sessionListCollapsed: !get().sessionListCollapsed }),
      toggleInspector: () =>
        set({ inspectorOpen: !get().inspectorOpen }),
      setInspectorWidth: (v) =>
        set({ inspectorWidth: clampInspectorWidth(v) }),
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
