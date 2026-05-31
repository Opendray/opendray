import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface OpenTab {
  id: string
  name?: string
}

interface SessionTabsState {
  tabs: OpenTab[]
  currentId: string | null
  // recents maps session id -> last-opened epoch ms (this browser).
  // Drives "most recently used on top" ordering in the session list.
  recents: Record<string, number>
  open: (tab: OpenTab) => void
  close: (id: string) => void
  closeAll: () => void
  setCurrent: (id: string | null) => void
  setTabName: (id: string, name?: string) => void
}

export const useSessionTabs = create<SessionTabsState>()(
  persist(
    (set, get) => ({
      tabs: [],
      currentId: null,
      recents: {},

      open: (tab) =>
        set((s) => {
          const recents = { ...s.recents, [tab.id]: Date.now() }
          if (s.tabs.some((t) => t.id === tab.id)) {
            return { currentId: tab.id, recents }
          }
          return { tabs: [...s.tabs, tab], currentId: tab.id, recents }
        }),

      close: (id) =>
        set((s) => {
          const idx = s.tabs.findIndex((t) => t.id === id)
          if (idx === -1) return s
          const tabs = s.tabs.filter((t) => t.id !== id)
          let currentId: string | null = s.currentId
          if (currentId === id) {
            const next = tabs[idx] ?? tabs[idx - 1] ?? null
            currentId = next?.id ?? null
          }
          return { tabs, currentId }
        }),

      closeAll: () => set({ tabs: [], currentId: null }),

      setCurrent: (id) => {
        const tab = get().tabs.find((t) => t.id === id)
        if (!tab && id !== null) return
        set((s) => ({
          currentId: id,
          recents: id ? { ...s.recents, [id]: Date.now() } : s.recents,
        }))
      },

      setTabName: (id, name) =>
        set((s) => ({
          tabs: s.tabs.map((t) => (t.id === id ? { ...t, name } : t)),
        })),
    }),
    {
      name: 'opendray.sessionTabs',
      partialize: (s) => ({
        tabs: s.tabs,
        currentId: s.currentId,
        recents: s.recents,
      }),
    },
  ),
)
