import type { Locale } from "./i18n/i18n";

const dateLocales: Record<Locale, string> = {
  ru: "ru-RU",
  en: "en-US",
};

export function dateLabel(date: string, locale: Locale) {
  return new Date(date).toLocaleDateString(dateLocales[locale]);
}

export function dateTimeLabel(date: string, locale: Locale) {
  return new Date(date).toLocaleString(dateLocales[locale]);
}
