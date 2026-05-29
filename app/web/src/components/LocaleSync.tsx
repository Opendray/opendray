import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { useLocale } from '@/stores/locale'

// Bridges the zustand locale store → i18next.changeLanguage.
//
// Previously this lived as a module-level useLocale.subscribe(...) in
// i18n.ts, which fired before React mounted and could miss updates in
// React 19 StrictMode / Vite HMR — the dropdown's checkmark moved but
// the rendered translations didn't. Running the bridge as a React
// effect uses the same lifecycle as every other useTranslation()
// consumer, so they update in lockstep.
export function LocaleSync() {
  const locale = useLocale((s) => s.locale)
  const { i18n } = useTranslation()

  useEffect(() => {
    if (i18n.language !== locale) {
      void i18n.changeLanguage(locale)
    }
  }, [locale, i18n])

  return null
}
