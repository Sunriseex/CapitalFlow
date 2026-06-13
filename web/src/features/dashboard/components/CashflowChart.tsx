import { memo } from "react";
import {
  Bar,
  CartesianGrid,
  ComposedChart,
  Line,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { ChartShell } from "../../../shared/ui/ChartShell";
import { chartAxisProps, chartGridProps } from "../../../shared/ui/chartTokens";
import type { Locale } from "../../../shared/i18n/i18n";
import type { CashflowChartBucket } from "../lib/cashflow";
import { compactMoney, formatChartMoney } from "../lib/cashflow";

type CashflowChartLabels = {
  income: string;
  expense: string;
  net: string;
  transactions: string;
};

type TooltipPayloadItem = {
  dataKey?: string | number;
  value?: number | string;
  color?: string;
  payload?: CashflowChartBucket;
};

export const CashflowChart = memo(function CashflowChart({
  data,
  currency,
  labels,
  locale,
  summary,
}: {
  data: CashflowChartBucket[];
  currency: string;
  labels: CashflowChartLabels;
  locale: Locale;
  summary: string;
}) {
  return (
    <ChartShell summary={summary} className="cashflow-chart-shell">
      <ComposedChart
        data={data}
        barCategoryGap="30%"
        barGap={3}
        margin={{ top: 4, right: 8, bottom: 0, left: 0 }}
      >
        <CartesianGrid {...chartGridProps} />
        <XAxis {...chartAxisProps} dataKey="period" />
        <YAxis
          {...chartAxisProps}
          tickFormatter={(value) => compactMoney(Number(value), currency)}
          width={64}
        />
        <Tooltip
          content={
            <CashflowTooltip
              currency={currency}
              labels={labels}
              locale={locale}
            />
          }
          cursor={{ fill: "var(--chart-cursor)" }}
          isAnimationActive={false}
          wrapperStyle={{ outline: "none" }}
        />
        <Bar
          dataKey="income"
          fill="var(--chart-income)"
          radius={[3, 3, 0, 0]}
          maxBarSize={20}
          isAnimationActive={false}
        />
        <Bar
          dataKey="expense"
          fill="var(--chart-expense)"
          radius={[3, 3, 0, 0]}
          maxBarSize={20}
          isAnimationActive={false}
        />
        <Line
          type="monotone"
          dataKey="net"
          stroke="var(--chart-net)"
          strokeWidth={1.75}
          dot={false}
          activeDot={{ r: 3, strokeWidth: 0, fill: "var(--chart-net)" }}
          isAnimationActive={false}
        />
      </ComposedChart>
    </ChartShell>
  );
});

function CashflowTooltip({
  active,
  currency,
  label,
  labels,
  locale,
  payload,
}: {
  active?: boolean;
  currency: string;
  label?: string;
  labels: CashflowChartLabels;
  locale: Locale;
  payload?: TooltipPayloadItem[];
}) {
  if (!active || !payload?.length) {
    return null;
  }

  const bucket = payload[0]?.payload;
  const values = new Map(
    payload.map((item) => [String(item.dataKey), Number(item.value ?? 0)]),
  );
  const rows = [
    { key: "income", label: labels.income, tone: "income" },
    { key: "expense", label: labels.expense, tone: "expense" },
    { key: "net", label: labels.net, tone: "net" },
  ];

  return (
    <div className="chart-tooltip" role="presentation">
      <strong>{label}</strong>
      {rows.map((row) => (
        <span className="chart-tooltip-row" key={row.key}>
          <span
            className={`chart-tooltip-dot ${row.tone}`}
            aria-hidden="true"
          />
          <span>{row.label}</span>
          <b>{formatChartMoney(values.get(row.key) ?? 0, currency, locale)}</b>
        </span>
      ))}
      {bucket ? (
        <span className="chart-tooltip-meta">
          {labels.transactions}: {bucket.transactions}
        </span>
      ) : null}
    </div>
  );
}
