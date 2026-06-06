import { memo } from "react";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";
import { ChartShell } from "../../../shared/ui/ChartShell";
import { chartAxisProps, chartGridProps } from "../../../shared/ui/chartTokens";
import type { CashflowChartBucket } from "../lib/cashflow";
import { compactMoney } from "../lib/cashflow";

export const CashflowChart = memo(function CashflowChart({
  data,
  currency,
  summary,
}: {
  data: CashflowChartBucket[];
  currency: string;
  summary: string;
}) {
  return (
    <ChartShell summary={summary}>
      <LineChart data={data} margin={{ top: 14, right: 18, bottom: 6, left: 0 }}>
        <defs>
          <linearGradient id="cashflowIncomeStroke" x1="0" x2="1" y1="0" y2="0">
            <stop offset="0%" stopColor="var(--chart-income)" stopOpacity={0.72} />
            <stop offset="100%" stopColor="var(--chart-income-strong)" stopOpacity={1} />
          </linearGradient>
          <linearGradient id="cashflowExpenseStroke" x1="0" x2="1" y1="0" y2="0">
            <stop offset="0%" stopColor="var(--chart-expense)" stopOpacity={0.72} />
            <stop offset="100%" stopColor="var(--chart-expense-strong)" stopOpacity={1} />
          </linearGradient>
        </defs>
        <CartesianGrid {...chartGridProps} />
        <XAxis {...chartAxisProps} dataKey="period" />
        <YAxis {...chartAxisProps} tickFormatter={(value) => compactMoney(Number(value), currency)} width={72} />
        <Line type="monotone" dataKey="income" stroke="url(#cashflowIncomeStroke)" strokeWidth={3} dot={false} activeDot={false} isAnimationActive={false} />
        <Line type="monotone" dataKey="expense" stroke="url(#cashflowExpenseStroke)" strokeWidth={3} dot={false} activeDot={false} isAnimationActive={false} />
        <Line type="monotone" dataKey="net" stroke="var(--chart-net)" strokeWidth={2} dot={false} strokeDasharray="5 5" activeDot={false} isAnimationActive={false} />
      </LineChart>
    </ChartShell>
  );
});
