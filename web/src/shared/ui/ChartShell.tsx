import type { ReactElement, ReactNode } from "react";
import { ResponsiveContainer } from "recharts";

export function ChartShell({
  children,
  size = "regular",
  title,
  meta,
  summary,
  className = "",
}: {
  children: ReactElement;
  size?: "regular" | "large";
  title?: ReactNode;
  meta?: ReactNode;
  summary?: ReactNode;
  className?: string;
}) {
  return (
    <div className={`chart chart-${size} chart-shell ${className}`.trim()}>
      {title || meta ? (
        <div className="chart-shell-head">
          <strong>{title}</strong>
          {meta ? <span>{meta}</span> : null}
        </div>
      ) : null}
      {summary ? <p className="sr-only">{summary}</p> : null}
      <div className="chart-shell-canvas">
        <ResponsiveContainer width="100%" height="100%" debounce={80}>
          {children}
        </ResponsiveContainer>
      </div>
    </div>
  );
}
