import { ChakraProvider, defaultSystem } from "@chakra-ui/react";
import { ThemeProvider } from "next-themes";
import type { ReactNode } from "react";
import { Toaster } from "./toaster";

export function Provider({ children }: { children: ReactNode }) {
  return (
    <ChakraProvider value={defaultSystem}>
      <ThemeProvider attribute="data-theme" storageKey="capitalflow_theme" defaultTheme="light" enableSystem={false} disableTransitionOnChange>
        {children}
        <Toaster />
      </ThemeProvider>
    </ChakraProvider>
  );
}
