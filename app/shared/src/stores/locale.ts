import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type Locale = 'en' | 'zh' | 'es'

export const SUPPORTED_LOCALES: Locale[] = ['en', 'zh', 'es']

interface LocaleState {
  locale: Locale
  setLocale: (l: Locale) => void
}

function detectDefault(): Locale {
  if (typeof navigator === 'undefined') return 'en'
  const lang = navigator.language || (navigator.languages && navigator.languages[0]) || 'en'
  const l = lang.toLowerCase()
  if (l.startsWith('zh')) return 'zh'
  if (l.startsWith('es')) return 'es'
  return 'en'
}

export const useLocale = create<LocaleState>()(
  persist(
    (set) => ({
      locale: detectDefault(),
      setLocale: (l) => set({ locale: l }),
    }),
    { name: 'opendray.locale' },
  ),
)
