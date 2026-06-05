import type { CSSProperties } from "react";

export const chartGridProps = {
  stroke: "var(--chart-grid)",
  strokeDasharray: "4 7",
  vertical: false,
};

export const chartAxisProps = {
  tickLine: false,
  axisLine: false,
  stroke: "var(--chart-axis)",
  tick: { fill: "var(--chart-axis)", fontSize: 12, fontWeight: 700 },
};

export const chartTooltipProps = {
  contentStyle: {
    border: "1px solid var(--chart-tooltip-border)",
    borderRadius: 12,
    background: "var(--chart-tooltip-bg)",
    boxShadow: "var(--shadow)",
    color: "var(--text)",
    backdropFilter: "blur(16px)",
  } satisfies CSSProperties,
  labelStyle: {
    color: "var(--text)",
    fontWeight: 900,
    marginBottom: 6,
  } satisfies CSSProperties,
  itemStyle: {
    color: "var(--text)",
    fontWeight: 800,
  } satisfies CSSProperties,
  cursor: {
    stroke: "var(--chart-cursor)",
    strokeWidth: 1,
  },
};
