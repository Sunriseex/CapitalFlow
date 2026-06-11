import { ThemeProvider } from "next-themes";
import type { ReactNode } from "react";
import { TooltipProvider } from "./tooltip";
import { Toaster } from "./toaster";

export function Provider({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider
      attribute="data-theme"
      storageKey="capitalflow_theme"
      defaultTheme="light"
      enableSystem={false}
      disableTransitionOnChange
    >
      <TooltipProvider>
        {children}
        <Toaster />
      </TooltipProvider>
    </ThemeProvider>
  );
}
