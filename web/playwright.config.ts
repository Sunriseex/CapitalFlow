import { defineConfig, devices } from "@playwright/test";
import { existsSync, readdirSync } from "node:fs";
import { join } from "node:path";

const chromiumExecutablePath = findChromiumExecutable();

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  use: {
    baseURL: "http://127.0.0.1:5173",
    trace: "on-first-retry",
  },
  webServer: {
    command: "npm run dev",
    url: "http://127.0.0.1:5173",
    reuseExistingServer: !process.env.CI,
  },
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        launchOptions: chromiumExecutablePath ? { executablePath: chromiumExecutablePath } : undefined,
      },
    },
  ],
});

function findChromiumExecutable() {
  for (const candidate of [process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH, process.env.CHROMIUM_EXECUTABLE_PATH]) {
    if (candidate && existsSync(candidate)) {
      return candidate;
    }
  }

  try {
    for (const entry of readdirSync("/nix/store")) {
      if (!entry.endsWith("-playwright-chromium")) {
        continue;
      }

      const candidate = join("/nix/store", entry, "chrome-linux64", "chrome");
      if (existsSync(candidate)) {
        return candidate;
      }
    }
  } catch {
    // Non-NixOS systems should use Playwright's bundled browser.
  }

  return undefined;
}
