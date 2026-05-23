import type { Amount, CurrencyRateTable, Transaction, TransactionType } from "./types";

type MoneyParseOptions = {
  required?: boolean;
  positive?: boolean;
  allowNegative?: boolean;
};

export type MoneyParseResult = { ok: true; value: string } | { ok: false; error: string };

const moneyPattern = /^-?\d+(?:\.\d{1,2})?$/;
const moneyFormatError = "Amount must be a number with up to 2 decimal places";

export function normalizeMoney(value: string) {
  const normalized = value.trim().replace(",", ".");
  if (normalized === "" || !moneyPattern.test(normalized)) {
    return normalized;
  }
  const sign = normalized.startsWith("-") ? "-" : "";
  const unsigned = sign ? normalized.slice(1) : normalized;
  const [whole, fraction = ""] = unsigned.split(".");
  const cleanWhole = whole.replace(/^0+(?=\d)/, "") || "0";
  const cleanFraction = fraction.replace(/0+$/, "");
  return `${sign}${cleanWhole}${cleanFraction ? `.${cleanFraction}` : ""}`;
}

export function isZeroMoney(value: string) {
  return decimalParts(value).units === 0n;
}

export function isPositiveMoney(value: string) {
  return decimalParts(value).units > 0n;
}

export function negateMoney(value: string) {
  const normalized = normalizeMoney(value);
  if (isZeroMoney(normalized)) return "0";
  return normalized.startsWith("-") ? normalized.slice(1) : `-${normalized}`;
}

export function addMoney(a: string, b: string) {
  const left = decimalParts(a);
  const right = decimalParts(b);
  const scale = left.scale > right.scale ? left.scale : right.scale;
  const units = scaleUnits(left, scale) + scaleUnits(right, scale);
  return unitsToDecimal(units, scale);
}

export function compareMoney(a: string, b: string) {
  const left = decimalParts(a);
  const right = decimalParts(b);
  const scale = left.scale > right.scale ? left.scale : right.scale;
  const leftUnits = scaleUnits(left, scale);
  const rightUnits = scaleUnits(right, scale);
  return leftUnits === rightUnits ? 0 : leftUnits > rightUnits ? 1 : -1;
}

export function moneyToNumber(amount: string) {
  return Number(normalizeMoney(amount));
}

export function formatMoney(amount: string, currency = "RUB") {
  const units = roundUnits(decimalParts(amount), 2n);
  const sign = units < 0n ? "-" : "";
  const abs = units < 0n ? -units : units;
  const raw = abs.toString().padStart(3, "0");
  const whole = raw.slice(0, -2);
  const fraction = raw.slice(-2);
  const grouped = whole.replace(/\B(?=(\d{3})+(?!\d))/g, "\u00a0");
  return `${sign}${grouped},${fraction}\u00a0${currency}`;
}

export function parseMoneyResult(value: string, options: MoneyParseOptions = {}): MoneyParseResult {
  const normalized = normalizeMoney(value);
  if (normalized === "") {
    if (options.required) return { ok: false, error: "Amount is required" };
    return { ok: true, value: "0" };
  }
  if (!moneyPattern.test(normalized)) {
    return { ok: false, error: moneyFormatError };
  }
  if (normalized.startsWith("-") && options.allowNegative !== true) {
    return { ok: false, error: "Amount must be non-negative" };
  }
  if (options.positive && !isPositiveMoney(normalized)) {
    return { ok: false, error: "Amount must be greater than zero" };
  }
  return { ok: true, value: normalized };
}

export const parseMoneyToMinorResult = parseMoneyResult;

export function parseMoneyToMinor(value: string) {
  const parsed = parseMoneyResult(value);
  if (!parsed.ok) throw new Error(parsed.error);
  return parsed.value;
}

export function amountFor(amounts: Amount[] | null | undefined, currency = "RUB") {
  return amounts?.find((amount) => amount.currency === currency)?.amount ?? "0";
}

export function convertAmount(amount: string, from: string, to: string, table?: CurrencyRateTable) {
  if (from === to) return amount;
  if (!table || table.base !== to) return "0";
  const rate = table.rates[from];
  if (!rate || rate <= 0) return "0";
  return divideMoneyByRate(amount, rate);
}

export const convertMinor = convertAmount;

export function sumConverted(amounts: Amount[] | null | undefined, to: string, table?: CurrencyRateTable) {
  return (amounts ?? []).reduce((total, amount) => addMoney(total, convertAmount(amount.amount, amount.currency, to, table)), "0");
}

export function signedAmount(transaction: Transaction) {
  switch (transaction.type) {
    case "expense":
    case "transfer_out":
      return negateMoney(transaction.amount);
    default:
      return transaction.amount;
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

function decimalParts(value: string) {
  const normalized = normalizeMoney(value);
  const sign = normalized.startsWith("-") ? -1n : 1n;
  const unsigned = sign < 0n ? normalized.slice(1) : normalized;
  const [whole, fraction = ""] = unsigned.split(".");
  return {
    units: sign * BigInt(`${whole || "0"}${fraction}`),
    scale: BigInt(fraction.length),
  };
}

function scaleUnits(value: { units: bigint; scale: bigint }, scale: bigint) {
  return value.units * (10n ** (scale - value.scale));
}

function roundUnits(value: { units: bigint; scale: bigint }, scale: bigint) {
  if (value.scale <= scale) return scaleUnits(value, scale);

  const divisor = 10n ** (value.scale - scale);
  const sign = value.units < 0n ? -1n : 1n;
  const abs = value.units < 0n ? -value.units : value.units;
  const quotient = abs / divisor;
  const remainder = abs % divisor;
  const rounded = remainder * 2n >= divisor ? quotient + 1n : quotient;
  return sign * rounded;
}

function unitsToDecimal(units: bigint, scale: bigint) {
  const sign = units < 0n ? "-" : "";
  const abs = units < 0n ? -units : units;
  const raw = abs.toString().padStart(Number(scale) + 1, "0");
  if (scale === 0n) return `${sign}${raw}`;
  const whole = raw.slice(0, -Number(scale));
  const fraction = raw.slice(-Number(scale)).replace(/0+$/, "");
  return `${sign}${whole}${fraction ? `.${fraction}` : ""}`;
}

function divideMoneyByRate(amount: string, rate: number) {
  const value = decimalParts(amount);
  const rateValue = decimalParts(rate.toString());
  if (rateValue.units <= 0n) return "0";

  const numerator = value.units * 10n ** rateValue.scale * 100n;
  const denominator = rateValue.units * 10n ** value.scale;
  const sign = numerator < 0n ? -1n : 1n;
  const absNumerator = numerator < 0n ? -numerator : numerator;
  const quotient = absNumerator / denominator;
  const remainder = absNumerator % denominator;
  const rounded = remainder * 2n >= denominator ? quotient + 1n : quotient;

  return unitsToDecimal(sign * rounded, 2n);
}


