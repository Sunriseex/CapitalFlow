import { moneyToNumber, sumConverted } from "../../../api/money";
import type { DashboardCashflowBucket } from "../../../api/types";
import type { CurrencyRateTable } from "../../../api/generated";
import type { Locale } from "../../../shared/i18n/i18n";

export type CashflowPeriod = "week" | "month" | "quarter" | "year";

export type CashflowChartBucket = {
  period: string;
  sourcePeriod: string;
  income: number;
  expense: number;
  net: number;
  transactions: number;
};

export const cashflowPeriods: Array<{ value: CashflowPeriod; label: string }> =
  [
    { value: "week", label: "Week" },
    { value: "month", label: "Month" },
    { value: "quarter", label: "Quarter" },
    { value: "year", label: "Year" },
  ];

export function cashflowBucketsToChart(
  buckets: DashboardCashflowBucket[],
  selectedCurrency: string,
  rateTable?: CurrencyRateTable,
) {
  return buckets.map((bucket) => ({
    period: shortPeriod(bucket.period),
    sourcePeriod: bucket.period,
    income: moneyToNumber(
      sumConverted(bucket.income, selectedCurrency, rateTable),
    ),
    expense: moneyToNumber(
      sumConverted(bucket.expense, selectedCurrency, rateTable),
    ),
    net: moneyToNumber(
      sumConverted(bucket.net_cashflow, selectedCurrency, rateTable),
    ),
    transactions: bucket.transaction_count,
  }));
}

export function groupCashflow(
  buckets: CashflowChartBucket[],
  period: CashflowPeriod,
) {
  if (period === "month") {
    return buckets;
  }

  if (period === "week") {
    return [];
  }

  const grouped = new Map<string, CashflowChartBucket>();
  for (const bucket of buckets) {
    const key =
      period === "quarter"
        ? quarterLabel(bucket.sourcePeriod)
        : bucket.sourcePeriod.slice(0, 4);
    const existing = grouped.get(key);
    if (existing) {
      existing.income += bucket.income;
      existing.expense += bucket.expense;
      existing.net += bucket.net;
      existing.transactions += bucket.transactions;
    } else {
      grouped.set(key, { ...bucket, period: key, sourcePeriod: key });
    }
  }

  return [...grouped.values()];
}

export function compactMoney(value: number, currency: string) {
  if (Math.abs(value) >= 1000000)
    return `${Math.round(value / 1000000)}M ${currency}`;
  if (Math.abs(value) >= 1000)
    return `${Math.round(value / 1000)}K ${currency}`;
  return `${value} ${currency}`;
}

export function formatChartMoney(
  value: number,
  currency: string,
  locale: Locale,
) {
  try {
    return new Intl.NumberFormat(locale, {
      style: "currency",
      currency,
      currencyDisplay: "code",
      maximumFractionDigits: 2,
    }).format(value);
  } catch {
    return `${value.toLocaleString(locale, {
      maximumFractionDigits: 2,
    })} ${currency}`;
  }
}

function shortPeriod(period: string) {
  const [, month] = period.split("-");
  return month ? `${month}/${period.slice(2, 4)}` : period;
}

function quarterLabel(period: string) {
  const [year, month] = period.split("-");
  const monthNumber = Number(month);
  if (!year || !monthNumber) {
    return period;
  }
  return `${year} Q${Math.ceil(monthNumber / 3)}`;
}
