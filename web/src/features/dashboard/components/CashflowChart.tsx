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
    <ChartShell summary={summary} className="cashflow-chart-shell">
      <LineChart data={data} margin={{ top: 8, right: 10, bottom: 0, left: 0 }}>
        <CartesianGrid {...chartGridProps} />
        <XAxis {...chartAxisProps} dataKey="period" />
        <YAxis
          {...chartAxisProps}
          tickFormatter={(value) => compactMoney(Number(value), currency)}
          width={64}
        />
        <Line
          type="linear"
          dataKey="income"
          stroke="var(--chart-income)"
          strokeWidth={2}
          dot={false}
          activeDot={false}
          isAnimationActive={false}
        />
        <Line
          type="linear"
          dataKey="expense"
          stroke="var(--chart-expense)"
          strokeWidth={2}
          dot={false}
          activeDot={false}
          isAnimationActive={false}
        />
        <Line
          type="linear"
          dataKey="net"
          stroke="var(--chart-net)"
          strokeWidth={2}
          dot={false}
          strokeDasharray="4 4"
          activeDot={false}
          isAnimationActive={false}
        />
      </LineChart>
    </ChartShell>
  );
});
