import type { Amount, CurrencyRateTable, Transaction } from "./types";

type MoneyParseMessages = {
  amountRequired: string;
  amountFormat: (scale: number) => string;
  amountNonNegative: string;
  amountGreaterThanZero: string;
};

type MoneyParseOptions = {
  required?: boolean;
  positive?: boolean;
  allowNegative?: boolean;
  currency?: string;
  messages?: MoneyParseMessages;
};

const defaultMoneyParseMessages: MoneyParseMessages = {
  amountRequired: "Amount is required",
  amountFormat: (scale) =>
    `Amount must be a number with up to ${scale} decimal places`,
  amountNonNegative: "Amount must be non-negative",
  amountGreaterThanZero: "Amount must be greater than zero",
};

const customCurrencyFractionDigits: Record<string, number> = {
  USDT: 6,
};

export type MoneyParseResult =
  | { ok: true; value: string }
  | { ok: false; error: string };

const moneyPattern = /^-?\d+(?:\.\d+)?$/;

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

const currencyDisplaySymbols: Record<string, string> = {
  RUB: "₽",
  USD: "$",
  EUR: "€",
  GBP: "£",
  JPY: "¥",
  CNY: "¥",
  KRW: "₩",
};

export function formatMoney(amount: string, currency = "RUB") {
  const normalizedCurrency = currency.trim().toUpperCase();
  const scale = BigInt(currencyFractionDigits(normalizedCurrency));
  const units = roundUnits(decimalParts(amount), scale);
  const sign = units < 0n ? "-" : "";
  const abs = units < 0n ? -units : units;
  const scaleNumber = Number(scale);
  const raw = abs.toString().padStart(scaleNumber + 1, "0");
  const whole = scale === 0n ? raw : raw.slice(0, -scaleNumber);
  const fraction = scale === 0n ? "" : raw.slice(-scaleNumber);
  const grouped = whole.replace(/\B(?=(\d{3})+(?!\d))/g, "\u00a0");
  const displayCurrency = currencyDisplaySymbol(normalizedCurrency);

  return `${sign}${grouped}${fraction ? `,${fraction}` : ""}\u00a0${displayCurrency}`;
}

export function parseMoneyResult(
  value: string,
  options: MoneyParseOptions = {},
): MoneyParseResult {
  const messages = options.messages ?? defaultMoneyParseMessages;
  const normalized = normalizeMoney(value);
  if (normalized === "") {
    if (options.required) return { ok: false, error: messages.amountRequired };
    return { ok: true, value: "0" };
  }
  const scale = currencyFractionDigits(options.currency ?? "RUB");
  const moneyFormatError = messages.amountFormat(scale);
  if (!moneyPattern.test(normalized)) {
    return { ok: false, error: moneyFormatError };
  }
  const fraction = normalized.split(".")[1] ?? "";
  if (fraction.length > scale) {
    return { ok: false, error: moneyFormatError };
  }
  if (normalized.startsWith("-") && options.allowNegative !== true) {
    return { ok: false, error: messages.amountNonNegative };
  }
  if (options.positive && !isPositiveMoney(normalized)) {
    return { ok: false, error: messages.amountGreaterThanZero };
  }
  return { ok: true, value: normalized };
}

export const parseMoneyToMinorResult = parseMoneyResult;

export function parseMoneyToMinor(value: string) {
  const parsed = parseMoneyResult(value);
  if (!parsed.ok) throw new Error(parsed.error);
  return parsed.value;
}

export function amountFor(
  amounts: Amount[] | null | undefined,
  currency = "RUB",
) {
  return amounts?.find((amount) => amount.currency === currency)?.amount ?? "0";
}

export function convertAmount(
  amount: string,
  from: string,
  to: string,
  table?: CurrencyRateTable,
) {
  if (from === to) return amount;
  if (!table || table.base !== to) return "0";
  const rate = table.rates[from];
  if (!rate || rate <= 0) return "0";
  return divideMoneyByRate(amount, rate, BigInt(currencyFractionDigits(to)));
}

export const convertMinor = convertAmount;

export function sumConverted(
  amounts: Amount[] | null | undefined,
  to: string,
  table?: CurrencyRateTable,
) {
  return (amounts ?? []).reduce(
    (total, amount) =>
      addMoney(total, convertAmount(amount.amount, amount.currency, to, table)),
    "0",
  );
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

function decimalParts(value: string) {
  const normalized = plainDecimalString(normalizeMoney(value));
  const sign = normalized.startsWith("-") ? -1n : 1n;
  const unsigned = sign < 0n ? normalized.slice(1) : normalized;
  const [whole, fraction = ""] = unsigned.split(".");
  return {
    units: sign * BigInt(`${whole || "0"}${fraction}`),
    scale: BigInt(fraction.length),
  };
}

function plainDecimalString(value: string) {
  if (!/[eE]/.test(value)) return value;

  const sign = value.startsWith("-") ? "-" : "";
  const unsigned = sign ? value.slice(1) : value;
  const [coefficient, exponentPart] = unsigned.toLowerCase().split("e");
  const exponent = Number(exponentPart);
  if (!Number.isInteger(exponent)) return value;

  const [whole, fraction = ""] = coefficient.split(".");
  const digits = `${whole}${fraction}`;
  const decimalIndex = whole.length + exponent;
  if (decimalIndex <= 0)
    return `${sign}0.${"0".repeat(-decimalIndex)}${digits}`.replace(
      /\.?0+$/,
      "",
    );
  if (decimalIndex >= digits.length)
    return `${sign}${digits}${"0".repeat(decimalIndex - digits.length)}`;
  return `${sign}${digits.slice(0, decimalIndex)}.${digits.slice(decimalIndex)}`.replace(
    /\.?0+$/,
    "",
  );
}

function currencyDisplaySymbol(currency: string) {
  return currencyDisplaySymbols[currency] ?? currency;
}

function currencyFractionDigits(currency: string) {
  const normalized = currency.trim().toUpperCase();
  const customScale = customCurrencyFractionDigits[normalized];
  if (customScale !== undefined) return customScale;

  try {
    return (
      new Intl.NumberFormat("en", {
        style: "currency",
        currency: normalized,
      }).resolvedOptions().maximumFractionDigits ?? 2
    );
  } catch {
    return 2;
  }
}

function scaleUnits(value: { units: bigint; scale: bigint }, scale: bigint) {
  return value.units * 10n ** (scale - value.scale);
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

function divideMoneyByRate(amount: string, rate: number, scale: bigint) {
  const value = decimalParts(amount);
  const rateValue = decimalParts(rate.toString());
  if (rateValue.units <= 0n) return "0";

  const numerator = value.units * 10n ** rateValue.scale * 10n ** scale;
  const denominator = rateValue.units * 10n ** value.scale;
  const sign = numerator < 0n ? -1n : 1n;
  const absNumerator = numerator < 0n ? -numerator : numerator;
  const quotient = absNumerator / denominator;
  const remainder = absNumerator % denominator;
  const rounded = remainder * 2n >= denominator ? quotient + 1n : quotient;

  return unitsToDecimal(sign * rounded, scale);
}
