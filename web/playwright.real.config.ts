import { defineConfig, devices } from "@playwright/test";
import { existsSync, readdirSync } from "node:fs";
import { join } from "node:path";

const databaseURL = process.env.TEST_DATABASE_URL;
const chromiumExecutablePath = findChromiumExecutable();

if (!databaseURL) {
  throw new Error("TEST_DATABASE_URL is required for the real E2E suite");
}

export default defineConfig({
  testDir: "./e2e-real",
  fullyParallel: false,
  workers: 1,
  timeout: 45_000,
  use: {
    baseURL: "http://127.0.0.1:5174",
    trace: "on-first-retry",
  },
  webServer: [
    {
      command: "go run ./cmd/server --addr :18081",
      cwd: "..",
      env: {
        APP_ENV: "development",
        DATABASE_URL: databaseURL,
        JWT_SECRET: "capitalflow-real-e2e-jwt-secret-0000000000000000",
        API_AUTH_TOKEN: "",
        COOKIE_SECURE: "false",
        CORS_ALLOWED_ORIGINS: "http://127.0.0.1:5174",
        WEBAUTHN_ORIGINS: "http://127.0.0.1:5174",
        RATE_LIMIT_REQUESTS: "1000",
        AUTH_RATE_LIMIT_REQUESTS: "1000",
        MUTATION_RATE_LIMIT_REQUESTS: "1000",
      },
      url: "http://127.0.0.1:18081/ready",
      reuseExistingServer: false,
      timeout: 60_000,
    },
    {
      command: "npm run dev -- --port 5174 --strictPort",
      env: {
        VITE_API_PROXY_TARGET: "http://127.0.0.1:18081",
      },
      url: "http://127.0.0.1:5174",
      reuseExistingServer: false,
      timeout: 60_000,
    },
  ],
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        launchOptions: chromiumExecutablePath
          ? { executablePath: chromiumExecutablePath }
          : undefined,
      },
    },
  ],
});

function findChromiumExecutable() {
  for (const candidate of [
    process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
    process.env.CHROMIUM_EXECUTABLE_PATH,
  ]) {
    if (candidate && existsSync(candidate)) {
      return candidate;
    }
  }

  try {
    for (const entry of readdirSync("/nix/store")) {
      const storePath = join("/nix/store", entry);
      const candidates = [
        join(storePath, "chrome-linux64", "chrome"),
        join(storePath, "chromium-1200", "chrome-linux64", "chrome"),
      ];
      for (const candidate of candidates) {
        if (
          (entry.endsWith("-playwright-chromium") ||
            entry.endsWith("-playwright-browsers")) &&
          existsSync(candidate)
        ) {
          return candidate;
        }
      }
    }
  } catch {
    // Non-NixOS systems should use Playwright's bundled browser.
  }

  return undefined;
}
