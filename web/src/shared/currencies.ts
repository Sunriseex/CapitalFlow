import type { Locale } from "./i18n/i18n";

const fallbackCurrencyCodes = [
  "AED",
  "ARS",
  "AUD",
  "BRL",
  "CAD",
  "CHF",
  "CNY",
  "EUR",
  "GBP",
  "HKD",
  "INR",
  "JPY",
  "KRW",
  "MXN",
  "RUB",
  "SGD",
  "TRY",
  "USD",
];

const customCurrencies: Record<string, Record<Locale, string>> = {
  USDT: {
    ru: "Tether USD",
    en: "Tether USD",
  },
};

const currencyNameLocales: Record<Locale, string> = {
  ru: "ru-RU",
  en: "en-US",
};

export type CurrencyOption = {
  code: string;
  label: string;
};

export function currencyOptions(locale: Locale = "en") {
  const codes =
    typeof Intl.supportedValuesOf === "function"
      ? Intl.supportedValuesOf("currency")
      : fallbackCurrencyCodes;

  return Array.from(new Set([...codes, ...Object.keys(customCurrencies)])).map(
    (code) => ({
      code,
      label: currencyLabel(code, locale),
    }),
  );
}

export function currencyLabel(code: string, locale: Locale = "en") {
  const normalized = code.trim().toUpperCase();
  const customName = customCurrencies[normalized]?.[locale];

  if (customName) {
    return `${normalized} - ${customName}`;
  }

  try {
    return `${normalized} - ${
      new Intl.DisplayNames([currencyNameLocales[locale]], {
        type: "currency",
      }).of(normalized) ?? normalized
    }`;
  } catch {
    return normalized;
  }
}
