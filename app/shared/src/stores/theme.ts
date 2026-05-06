import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type ThemeMode = 'light' | 'dark' | 'system'

interface ThemeState {
  mode: ThemeMode
  setMode: (m: ThemeMode) => void
  applied: () => 'light' | 'dark'
}

function effective(mode: ThemeMode): 'light' | 'dark' {
  if (mode === 'system') {
    if (typeof window === 'undefined') return 'dark'
    return window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light'
  }
  return mode
}

function apply(theme: 'light' | 'dark') {
  if (typeof document === 'undefined') return
  document.documentElement.classList.toggle('dark', theme === 'dark')
}

export const useTheme = create<ThemeState>()(
  persist(
    (set, get) => ({
      mode: 'dark',
      setMode: (m) => {
        set({ mode: m })
        apply(effective(m))
      },
      applied: () => effective(get().mode),
    }),
    {
      name: 'opendray.theme',
      onRehydrateStorage: () => (state) => {
        if (state) apply(effective(state.mode))
      },
    },
  ),
)

// Apply on first load (before React mounts).
if (typeof window !== 'undefined') {
  apply(effective(useTheme.getState().mode))

  // Track system changes when in 'system' mode.
  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  mq.addEventListener('change', () => {
    if (useTheme.getState().mode === 'system') {
      apply(effective('system'))
    }
  })
}
