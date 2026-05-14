import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type Locale = 'en' | 'zh'

export const SUPPORTED_LOCALES: Locale[] = ['en', 'zh']

interface LocaleState {
  locale: Locale
  setLocale: (l: Locale) => void
}

function detectDefault(): Locale {
  if (typeof navigator === 'undefined') return 'en'
  const lang = navigator.language || (navigator.languages && navigator.languages[0]) || 'en'
  return lang.toLowerCase().startsWith('zh') ? 'zh' : 'en'
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
