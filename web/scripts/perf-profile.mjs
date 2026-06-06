import { spawn, spawnSync } from "node:child_process";
import { createHash, createHmac, randomBytes, randomUUID } from "node:crypto";
import { existsSync, readFileSync, readdirSync } from "node:fs";
import { join, resolve } from "node:path";
import { chromium } from "playwright";

/* global window, requestAnimationFrame */

const baseURL = process.env.CAPITALFLOW_PERF_BASE_URL ?? "http://127.0.0.1:5173";
const apiHealthURL = process.env.CAPITALFLOW_PERF_API_HEALTH_URL ?? "http://127.0.0.1:18080/health";
const maxFrameGapMs = Number(process.env.CAPITALFLOW_PERF_MAX_FRAME_GAP_MS ?? 32);
const maxLongTasks = Number(process.env.CAPITALFLOW_PERF_MAX_LONG_TASKS ?? 0);
const rootDir = resolve("..");

const vite = await ensureViteServer();
await ensureBackend();
const session = await bootstrapSession();

const browser = await chromium.launch({
  executablePath: findChromiumExecutable(),
});

try {
  const page = await browser.newPage({ viewport: { width: 1440, height: 950 } });
  await page.addInitScript(({ token }) => {
    localStorage.setItem("capitalflow_api_token", token);
    localStorage.setItem("capitalflow_api_base", "/api/v1");
  }, { token: session.accessToken });

  const results = [];
  await gotoApp(page, "/");
  results.push(await measure(page, "dashboard scroll", async () => {
    await smoothScroll(page);
  }));
  results.push(await measure(page, "cashflow chart hover", async () => {
    await hoverAcross(page, ".chart-shell-canvas");
  }));
  results.push(await measure(page, "dashboard recent row hover", async () => {
    await hoverRows(page, ".tx-table tbody tr");
  }));

  results.push(await switchTo(page, "Accounts"));
  results.push(await measure(page, "accounts scroll", async () => {
    await smoothScroll(page);
  }));
  results.push(await measure(page, "accounts table hover", async () => {
    await hoverRows(page, ".accounts-table tbody tr");
  }));

  results.push(await switchTo(page, "Transactions"));
  results.push(await measure(page, "transactions scroll", async () => {
    await smoothScroll(page);
  }));
  results.push(await measure(page, "transactions table hover", async () => {
    await hoverRows(page, ".transactions-table tbody tr", 24);
  }));

  printResults(results);
  const failed = results.filter((result) => result.longTaskCount > maxLongTasks || result.maxFrameGap > maxFrameGapMs);
  if (failed.length) {
    process.exitCode = 1;
  }
} finally {
  await browser.close();
  if (vite) {
    vite.kill("SIGTERM");
  }
}

async function ensureViteServer() {
  if (await reachable(baseURL)) {
    return null;
  }

  const child = spawn("npm", ["run", "dev", "--", "--port", "5173"], {
    cwd: process.cwd(),
    stdio: "inherit",
    env: process.env,
  });

  for (let attempt = 0; attempt < 80; attempt += 1) {
    if (await reachable(baseURL)) {
      return child;
    }
    await delay(250);
  }

  child.kill("SIGTERM");
  throw new Error(`Vite dev server did not start at ${baseURL}`);
}

async function ensureBackend() {
  if (!(await reachable(apiHealthURL))) {
    throw new Error(`API is not reachable at ${apiHealthURL}. Start the VM/dev API first.`);
  }
}

async function gotoApp(page, path) {
  await page.goto(`${baseURL}${path}`, { waitUntil: "networkidle" });
  await page.getByRole("heading", { name: "Overview" }).first().waitFor({ state: "visible", timeout: 15_000 });
}

async function switchTo(page, name) {
  return measure(page, `switch to ${name.toLowerCase()}`, async () => {
    await page.getByRole("button", { name }).click();
    await page.getByRole("heading", { name: name === "Accounts" ? "Accounts" : name }).first().waitFor({ state: "visible" });
  });
}

async function measure(page, name, action) {
  await page.evaluate(() => {
    const previous = window.__capitalflowPerf;
    previous?.stop?.();

    const samples = [];
    const longTasks = [];
    let running = true;
    let last = performance.now();
    let observer;

    const tick = (now) => {
      samples.push(now - last);
      last = now;
      if (running) {
        requestAnimationFrame(tick);
      }
    };

    if (PerformanceObserver.supportedEntryTypes?.includes("longtask")) {
      observer = new PerformanceObserver((list) => {
        longTasks.push(...list.getEntries().map((entry) => entry.duration));
      });
      observer.observe({ type: "longtask" });
    }

    requestAnimationFrame(tick);
    window.__capitalflowPerf = {
      stop() {
        running = false;
        observer?.disconnect();
        return {
          samples,
          longTasks,
          maxFrameGap: samples.length ? Math.max(...samples) : 0,
          p95FrameGap: percentile(samples, 0.95),
          longTaskCount: longTasks.length,
        };
      },
    };

    function percentile(values, quantile) {
      if (!values.length) return 0;
      const sorted = [...values].sort((a, b) => a - b);
      return sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * quantile))];
    }
  });

  const started = Date.now();
  await action();
  await page.waitForTimeout(250);
  const metrics = await page.evaluate(() => window.__capitalflowPerf.stop());
  return {
    name,
    duration: Date.now() - started,
    maxFrameGap: round(metrics.maxFrameGap),
    p95FrameGap: round(metrics.p95FrameGap),
    longTaskCount: metrics.longTaskCount,
  };
}

async function smoothScroll(page) {
  await page.evaluate(async () => {
    for (const top of [160, 360, 620, 900, 1240, 700, 240, 0]) {
      window.scrollTo({ top, behavior: "instant" });
      await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
    }
  });
}

async function hoverAcross(page, selector) {
  const box = await page.locator(selector).first().boundingBox();
  if (!box) return;
  for (let index = 0; index <= 16; index += 1) {
    await page.mouse.move(box.x + (box.width * index) / 16, box.y + box.height / 2, { steps: 2 });
  }
}

async function hoverRows(page, selector, limit = 12) {
  const rows = await page.locator(selector).evaluateAll((elements, rowLimit) =>
    elements.slice(0, rowLimit).map((element) => {
      const rect = element.getBoundingClientRect();
      return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
    }), limit);

  for (const row of rows) {
    await page.mouse.move(row.x, row.y);
  }
}

async function reachable(url) {
  try {
    const response = await fetch(url);
    return response.ok;
  } catch {
    return false;
  }
}

async function bootstrapSession() {
  const env = loadEnv(resolve(rootDir, "configs/.env"));
  const apiEnv = loadAPIProcessEnv();
  const databaseURL = process.env.DATABASE_URL ?? apiEnv.DATABASE_URL ?? env.DATABASE_URL;
  const jwtSecret = process.env.JWT_SECRET ?? apiEnv.JWT_SECRET ?? env.JWT_SECRET;

  if (!databaseURL || !jwtSecret) {
    throw new Error("DATABASE_URL and JWT_SECRET are required for perf auth bootstrap.");
  }

  const userOutput = psql(databaseURL, "SELECT id, email FROM users ORDER BY created_at LIMIT 1");
  const [userID, email] = userOutput.split("|");
  if (!userID || !email) {
    throw new Error("No dev user found. Complete auth setup before running perf profile.");
  }

  const now = Math.floor(Date.now() / 1000);
  const sessionID = randomUUID();
  const refreshToken = randomBytes(32).toString("base64url");
  const tokenHash = createHash("sha256").update(refreshToken).digest("hex");
  psql(databaseURL, `
    INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, revoked_reason, created_at)
    VALUES ('${sessionID}', '${userID}', '${tokenHash}', now() + interval '30 minutes', NULL, NULL, now())
  `);

  return {
    accessToken: signJWT({
      user_id: userID,
      email,
      session_id: sessionID,
      token_type: "access",
      iss: "capitalflow",
      sub: userID,
      exp: now + 900,
      iat: now,
      jti: randomUUID(),
    }, jwtSecret),
  };
}

function psql(databaseURL, sql) {
  const result = spawnSync("psql", [databaseURL, "-Atc", sql], {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
  if (result.status !== 0) {
    throw new Error(result.stderr.trim() || "psql failed");
  }
  return result.stdout.trim();
}

function signJWT(payload, secret) {
  const header = { alg: "HS256", typ: "JWT" };
  const encodedHeader = base64url(JSON.stringify(header));
  const encodedPayload = base64url(JSON.stringify(payload));
  const signature = createHmac("sha256", secret).update(`${encodedHeader}.${encodedPayload}`).digest("base64url");
  return `${encodedHeader}.${encodedPayload}.${signature}`;
}

function base64url(value) {
  return Buffer.from(value).toString("base64url");
}

function loadEnv(path) {
  if (!existsSync(path)) {
    return {};
  }

  return Object.fromEntries(
    readFileSync(path, "utf8")
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter((line) => line && !line.startsWith("#") && line.includes("="))
      .map((line) => {
        const index = line.indexOf("=");
        return [line.slice(0, index), line.slice(index + 1)];
      }),
  );
}

function loadAPIProcessEnv() {
  const result = spawnSync("ss", ["-ltnp", "sport", "=", ":18080"], {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "ignore"],
  });
  const pid = result.stdout.match(/pid=(\d+)/)?.[1];
  if (!pid) {
    return {};
  }

  try {
    return Object.fromEntries(
      readFileSync(`/proc/${pid}/environ`, "utf8")
        .split("\0")
        .filter((entry) => entry.includes("="))
        .map((entry) => {
          const index = entry.indexOf("=");
          return [entry.slice(0, index), entry.slice(index + 1)];
        }),
    );
  } catch {
    return {};
  }
}

function findChromiumExecutable() {
  for (const candidate of [process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH, process.env.CHROMIUM_EXECUTABLE_PATH]) {
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
        if ((entry.endsWith("-playwright-chromium") || entry.endsWith("-playwright-browsers")) && existsSync(candidate)) {
          return candidate;
        }
      }
    }
  } catch {
    return undefined;
  }

  return undefined;
}

function printResults(results) {
  console.table(results);
  const failed = results.filter((result) => result.longTaskCount > maxLongTasks || result.maxFrameGap > maxFrameGapMs);
  if (failed.length) {
    console.error(`Performance budget failed: maxFrameGap <= ${maxFrameGapMs}ms, longTaskCount <= ${maxLongTasks}`);
  } else {
    console.log(`Performance budget passed: maxFrameGap <= ${maxFrameGapMs}ms, longTaskCount <= ${maxLongTasks}`);
  }
}

function round(value) {
  return Math.round(value * 10) / 10;
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
