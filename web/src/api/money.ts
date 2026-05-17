import type { Amount, CurrencyRateTable, Transaction, TransactionType } from "./types";

export function formatMoney(minor: number, currency = "RUB") {
  return new Intl.NumberFormat("ru-RU", { style: "currency", currency }).format(minor / 100);
}

type MoneyParseOptions = {
  required?: boolean;
  positive?: boolean;
  allowNegative?: boolean;
};

export type MoneyParseResult = { ok: true; value: number } | { ok: false; error: string };

export function parseMoneyToMinorResult(value: string, options: MoneyParseOptions = {}): MoneyParseResult {
  const normalized = value.trim().replace(",", ".");
  if (normalized === "") {
    if (options.required) {
      return { ok: false, error: "Amount is required" };
    }
    return { ok: true, value: 0 };
  }

  if (!/^-?\d+(?:\.\d{1,2})?$/.test(normalized)) {
    return { ok: false, error: "Amount must be a number with up to 2 decimal places" };
  }

  const isNegative = normalized.startsWith("-");
  if (isNegative && options.allowNegative !== true) {
    return { ok: false, error: "Amount must be non-negative" };
  }

  const unsigned = isNegative ? normalized.slice(1) : normalized;
  const [whole, fraction = ""] = unsigned.split(".");
  const minor = (Number(whole) * 100) + Number(fraction.padEnd(2, "0"));
  const signedMinor = isNegative ? -minor : minor;

  if (!Number.isSafeInteger(signedMinor)) {
    return { ok: false, error: "Amount is too large" };
  }

  if (options.positive && signedMinor <= 0) {
    return { ok: false, error: "Amount must be greater than zero" };
  }

  return { ok: true, value: signedMinor };
}

export function parseMoneyToMinor(value: string) {
  const parsed = parseMoneyToMinorResult(value);
  if (!parsed.ok) {
    throw new Error(parsed.error);
  }
  return parsed.value;
}

export function amountFor(amounts: Amount[] | null | undefined, currency = "RUB") {
  return amounts?.find((amount) => amount.currency === currency)?.amount_minor ?? 0;
}

export function convertMinor(amountMinor: number, from: string, to: string, table?: CurrencyRateTable) {
  if (from === to) {
    return amountMinor;
  }
  if (!table || table.base !== to) {
    return 0;
  }
  const rate = table.rates[from];
  if (!rate || rate <= 0) {
    return 0;
  }
  return Math.round(amountMinor / rate);
}

export function sumConverted(amounts: Amount[] | null | undefined, to: string, table?: CurrencyRateTable) {
  return (amounts ?? []).reduce((total, amount) => total + convertMinor(amount.amount_minor, amount.currency, to, table), 0);
}

export function signedAmount(transaction: Transaction) {
  switch (transaction.type) {
    case "expense":
    case "transfer_out":
      return -transaction.amount_minor;
    default:
      return transaction.amount_minor;
  }
}

export function transactionTypeLabel(type: TransactionType) {
  return {
    initial_balance: "Initial",
    income: "Income",
    expense: "Expense",
    transfer_in: "Transfer in",
    transfer_out: "Transfer out",
    interest_income: "Interest",
    adjustment: "Adjustment",
  }[type];
}
