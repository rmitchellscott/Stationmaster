import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

// Import all language files
import en from '../../../locales/en.json';
import de from '../../../locales/de.json';
import es from '../../../locales/es.json';
import fr from '../../../locales/fr.json';
import it from '../../../locales/it.json';
import ja from '../../../locales/ja.json';
import ko from '../../../locales/ko.json';
import nl from '../../../locales/nl.json';
import pl from '../../../locales/pl.json';
import pt from '../../../locales/pt.json';
import sv from '../../../locales/sv.json';
import fi from '../../../locales/fi.json';
import no from '../../../locales/no.json';
import da from '../../../locales/da.json';
import zhCN from '../../../locales/zh-CN.json';

const resources = {
  en: { translation: en },
  de: { translation: de },
  es: { translation: es },
  fr: { translation: fr },
  it: { translation: it },
  ja: { translation: ja },
  ko: { translation: ko },
  nl: { translation: nl },
  pl: { translation: pl },
  pt: { translation: pt },
  sv: { translation: sv },
  fi: { translation: fi },
  no: { translation: no },
  da: { translation: da },
  'zh-CN': { translation: zhCN },
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false,
    },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage'],
      lookupLocalStorage: 'i18nextLng',
    },
  });

export default i18n;