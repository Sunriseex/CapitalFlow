import { createContext } from "react";
import { en } from "./dictionaries/en";
import { ru } from "./dictionaries/ru";
import type { TranslationDictionary } from "./dictionaries/ru";

export const dictionaries = {
  ru,
  en,
} as const satisfies Record<string, TranslationDictionary>;

export type Locale = keyof typeof dictionaries;

export type I18nContextValue = {
  locale: Locale;
  t: TranslationDictionary;
  setLocale: (locale: Locale) => void;
  toggleLocale: () => void;
};

export const defaultLocale: Locale = "ru";

export const localeStorageKey = "capitalflow_locale";

export const supportedLocales = Object.keys(dictionaries) as Locale[];

export const I18nContext = createContext<I18nContextValue | null>(null);

export function isLocale(value: unknown): value is Locale {
  return typeof value === "string" && value in dictionaries;
}

export function getDictionary(locale: Locale): TranslationDictionary {
  return dictionaries[locale];
}

export function nextLocale(locale: Locale): Locale {
  return locale === "ru" ? "en" : "ru";
}
