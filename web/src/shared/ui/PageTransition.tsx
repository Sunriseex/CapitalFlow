import type { ReactNode } from "react";

export function PageTransition({ children }: { children: ReactNode }) {
  return (
    <div className="page-transition is-static" data-testid="page-transition">
      {children}
    </div>
  );
}
