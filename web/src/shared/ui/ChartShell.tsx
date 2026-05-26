import type { ReactElement } from "react";
import { ResponsiveContainer } from "recharts";

export function ChartShell({ children, size = "regular" }: { children: ReactElement; size?: "regular" | "large" }) {
  return (
    <div className={`chart chart-${size}`}>
      <ResponsiveContainer width="100%" height="100%">
        {children}
      </ResponsiveContainer>
    </div>
  );
}
