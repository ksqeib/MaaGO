import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";

import zhCN from "@/assets/locales/zh_CN.json";
import enUS from "@/assets/locales/en_US.json";

export const i18nReady = i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      zh_CN: { translation: zhCN },
      en_US: { translation: enUS },
    },
    fallbackLng: "en_US",
    supportedLngs: ["zh_CN", "zh-CN", "zh", "en_US", "en-US", "en"],
    nonExplicitSupportedLngs: true,
    detection: {
      order: ["localStorage", "navigator"],
      caches: ["localStorage"],
      lookupLocalStorage: "maago.language",
    },
    interpolation: {
      escapeValue: false,
    },
  })
  .then((result) => {
    document.documentElement.lang = String(i18n.resolvedLanguage || "en-US")
      .replace("_", "-");
    return result;
  });

i18n.on("languageChanged", (language) => {
  document.documentElement.lang = String(language).replace("_", "-");
});

export default i18n;
