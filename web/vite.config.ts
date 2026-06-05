/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const apiProxyTarget = process.env.VITE_API_PROXY_TARGET ?? "http://127.0.0.1:18080";

export default defineConfig({
  plugins: [react()],
  build: {
    // zxcvbn ships one large frequency-list module. It is isolated below so the
    // main app stays small, but the lazy vendor chunk is still >500 kB.
    chunkSizeWarningLimit: 900,
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: "react",
              test: /node_modules\/(react|react-dom)\//,
            },
            {
              name: "chakra",
              test: /node_modules\/(@chakra-ui|@emotion|next-themes)\//,
            },
            {
              name: "charts",
              test: /node_modules\/(recharts|d3-|victory-vendor|decimal.js-light)\//,
            },
            {
              name: "icons",
              test: /node_modules\/lucide-react\//,
            },
            {
              name: "auth-strength",
              test: /node_modules\/zxcvbn\//,
            },
          ],
        },
      },
    },
  },
  server: {
    host: "127.0.0.1",
    port: 5173,
    proxy: {
      "/api/v1": apiProxyTarget,
      "/auth": apiProxyTarget,
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    globals: true,
    exclude: ["e2e/**", "node_modules/**", "dist/**"],
  },
});
