import { memo } from "react";
import {
  Bar,
  CartesianGrid,
  ComposedChart,
  Line,
  XAxis,
  YAxis,
} from "recharts";
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
    <ChartShell summary={summary} className="cashflow-chart-shell">
      <ComposedChart
        data={data}
        barCategoryGap="24%"
        barGap={4}
        margin={{ top: 8, right: 10, bottom: 0, left: 0 }}
      >
        <CartesianGrid {...chartGridProps} />
        <XAxis {...chartAxisProps} dataKey="period" />
        <YAxis
          {...chartAxisProps}
          tickFormatter={(value) => compactMoney(Number(value), currency)}
          width={64}
        />
        <Bar
          dataKey="income"
          fill="var(--chart-income)"
          radius={[4, 4, 0, 0]}
          maxBarSize={26}
          isAnimationActive={false}
        />
        <Bar
          dataKey="expense"
          fill="var(--chart-expense)"
          radius={[4, 4, 0, 0]}
          maxBarSize={26}
          isAnimationActive={false}
        />
        <Line
          type="monotone"
          dataKey="net"
          stroke="var(--chart-net)"
          strokeWidth={2}
          dot={false}
          strokeDasharray="4 4"
          activeDot={false}
          isAnimationActive={false}
        />
      </ComposedChart>
    </ChartShell>
  );
});
