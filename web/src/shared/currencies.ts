const fallbackCurrencyCodes = [
  "AED", "ARS", "AUD", "BRL", "CAD", "CHF", "CNY", "EUR", "GBP", "HKD", "INR", "JPY", "KRW", "MXN", "RUB", "SGD", "TRY", "USD",
];

const customCurrencies: Record<string, string> = {
  USDT: "Tether USD",
};

export function currencyOptions() {
  const codes = typeof Intl.supportedValuesOf === "function"
    ? Intl.supportedValuesOf("currency")
    : fallbackCurrencyCodes;

  return Array.from(new Set([...codes, ...Object.keys(customCurrencies)])).map((code) => ({
    code,
    label: currencyLabel(code),
  }));
}

export function currencyLabel(code: string) {
  const normalized = code.trim().toUpperCase();
  const customName = customCurrencies[normalized];
  if (customName) return `${normalized} - ${customName}`;

  try {
    return `${normalized} - ${new Intl.DisplayNames(["en"], { type: "currency" }).of(normalized) ?? normalized}`;
  } catch {
    return normalized;
  }
}



