import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

import en from '../../i18n/en.json'
import zh from '../../i18n/zh.json'
import { useLocale, type Locale } from '@/stores/locale'

void i18n
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      zh: { translation: zh },
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

useLocale.subscribe((s) => {
  if (i18n.language !== s.locale) {
    void i18n.changeLanguage(s.locale)
  }
})

export function setLocale(l: Locale) {
  useLocale.getState().setLocale(l)
}

export default i18n
