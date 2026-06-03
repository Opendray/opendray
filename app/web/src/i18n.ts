import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

import en from '../../i18n/en.json'
import zh from '../../i18n/zh.json'
import es from '../../i18n/es.json'
import { useLocale } from '@/stores/locale'

// Initialise with the persisted (or detected) locale. After this point
// the running language is owned by i18next and kept in sync with the
// zustand store through <LocaleSync /> in the React tree — see
// components/LocaleSync.tsx for the rationale.
void i18n
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      zh: { translation: zh },
      es: { translation: es },
    },
    lng: useLocale.getState().locale,
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false,
      prefix: '{',
      suffix: '}',
    },
    returnNull: false,
  })

export default i18n
