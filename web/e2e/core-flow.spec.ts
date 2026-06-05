import { expect, test } from "@playwright/test";

type Account = {
  id: string;
  name: string;
  bank: string;
  type: string;
  currency: string;
  is_active: boolean;
  opened_at: string;
  created_at: string;
  updated_at: string;
};

type Transaction = {
  id: string;
  account_id: string;
  related_account_id?: string | null;
  transfer_id?: string | null;
  type: string;
  amount: string;
  category_id?: string | null;
  description: string;
  occurred_at: string;
  created_at: string;
};

test("setup/login, account, transactions, transfer, dashboard, logout", async ({ page }) => {
  const now = "2026-05-19T00:00:00Z";
  const accounts: Account[] = [];
  const transactions: Transaction[] = [];
  let accountSeq = 0;
  let transactionSeq = 0;

  await page.route("**/auth/status", async (route) => {
    await route.fulfill({ json: { setup_required: false } });
  });
  await page.route("**/auth/login", async (route) => {
    await route.fulfill({
      json: {
        user: { id: "user-1", email: "user@example.com", primary_currency: "RUB" },
        access_token: "e2e-access-token",
        access_expires_at: "2026-05-19T01:00:00Z",
      },
    });
  });
  await page.route("**/auth/logout", async (route) => {
    await route.fulfill({ status: 204 });
  });
  await page.route("**/api/v1/settings/profile", async (route) => {
    await route.fulfill({ json: { user: { id: "user-1", email: "user@example.com", primary_currency: "RUB" } } });
  });
  await page.route("**/api/v1/categories", async (route) => {
    await route.fulfill({ json: [] });
  });
  await page.route("**/api/v1/interest-rules", async (route) => {
    await route.fulfill({ json: [] });
  });
  await page.route("**/api/v1/currency-rates?**", async (route) => {
    await route.fulfill({ json: { base: "RUB", date: "2026-05-19", provider: "e2e", rates: {} } });
  });
  await page.route("**/api/v1/dashboard/**", async (route) => {
    await route.fulfill({ json: dashboardResponse(accounts, transactions, now) });
  });
  await page.route("**/api/v1/accounts", async (route) => {
    if (route.request().method() === "POST") {
      const input = await route.request().postDataJSON();
      const account = {
        id: `account-${++accountSeq}`,
        name: input.name,
        bank: input.bank,
        type: input.type,
        currency: input.currency,
        is_active: true,
        opened_at: input.opened_at,
        created_at: now,
        updated_at: now,
      };
      accounts.push(account);
      await route.fulfill({ status: 201, json: account });
      return;
    }

    await route.fulfill({ json: accounts });
  });
  await page.route("**/api/v1/transactions", async (route) => {
    if (route.request().method() === "POST") {
      const input = await route.request().postDataJSON();
      const transaction = {
        id: `transaction-${++transactionSeq}`,
        account_id: input.account_id,
        type: input.type,
        amount: input.amount,
        category_id: input.category_id,
        description: input.description,
        occurred_at: input.occurred_at,
        created_at: now,
      };
      transactions.push(transaction);
      await route.fulfill({ status: 201, json: transaction });
      return;
    }

    await route.fulfill({ json: transactions });
  });
  await page.route("**/api/v1/transfers", async (route) => {
    const input = await route.request().postDataJSON();
    const transferID = `transfer-${transactionSeq + 1}`;
    const out = {
      id: `transaction-${++transactionSeq}`,
      account_id: input.from_account_id,
      related_account_id: input.to_account_id,
      transfer_id: transferID,
      type: "transfer_out",
      amount: input.amount,
      category_id: null,
      description: input.description,
      occurred_at: "2026-05-19",
      created_at: now,
    };
    const incoming = {
      id: `transaction-${++transactionSeq}`,
      account_id: input.to_account_id,
      related_account_id: input.from_account_id,
      transfer_id: transferID,
      type: "transfer_in",
      amount: input.amount,
      category_id: null,
      description: input.description,
      occurred_at: "2026-05-19",
      created_at: now,
    };
    transactions.push(out, incoming);
    await route.fulfill({ status: 201, json: { out, in: incoming, exchange_rate: "1" } });
  });

  await page.goto("/");
  await page.getByLabel("Email").fill("user@example.com");
  await page.getByLabel("Password", { exact: true }).fill("password");
  await page.getByRole("button", { name: "Sign in with email" }).click();
  await expect(page.getByRole("heading", { name: "Overview" }).first()).toBeVisible();
  await expect(page.getByRole("button", { name: "Open command menu" })).toBeVisible();

  await expectAppTheme(page, "light", "#edf4f2");
  await page.getByRole("button", { name: "Switch to dark theme" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expectAppTheme(page, "dark", "#070d14");
  await page.getByRole("button", { name: "Switch to light theme" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await expectAppTheme(page, "light", "#edf4f2");
  await expect.poll(
    () => page.locator(".toast-card").first().evaluate((element) => getComputedStyle(element).color),
  ).toBe("rgb(16, 24, 32)");
  await page.getByRole("button", { name: "Switch to dark theme" }).click();
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expectAppTheme(page, "dark", "#070d14");

  await createAccount(page, "Cash", "Wallet", "1000");
  await createAccount(page, "Savings", "Bank", "0");
  const referenceSurface = await surfaceSnapshot(page, ".balance-card");

  await page.getByRole("button", { name: "Accounts" }).click();
  await expect(page.locator(".accounts-panel.workspace-panel")).toBeVisible();
  await expectSurface(page, ".accounts-panel", referenceSurface);
  await expect(page.locator(".accounts-table-wrap.workspace-table-wrap")).toBeVisible();
  await expect(page.locator(".accounts-table.workspace-table")).toBeVisible();

  await page.getByRole("button", { name: "Transactions" }).click();
  await expect(page.locator(".transactions-panel.workspace-panel")).toBeVisible();
  await expectSurface(page, ".transactions-panel", referenceSurface);
  await expect(page.locator(".transactions-filters.workspace-filters")).toBeVisible();
  await expect(page.locator(".transactions-table-wrap.workspace-table-wrap")).toBeVisible();
  await expect(page.locator(".transactions-table.workspace-table")).toBeVisible();

  await page.getByRole("button", { name: "Settings" }).click();
  await expect(page.locator(".workspace-settings")).toBeVisible();
  await expect(page.locator(".profile-settings-panel.workspace-panel")).toBeVisible();
  await expectSurface(page, ".profile-settings-panel", referenceSurface);
  await expect(page.locator(".security-settings-panel.workspace-panel")).toBeVisible();
  await expectSurface(page, ".security-settings-panel", referenceSurface);
  await page.getByRole("button", { name: "Overview" }).click();
  await expect(page.getByRole("button", { name: "Hide insights" })).toHaveAttribute("aria-expanded", "true");
  await page.getByRole("button", { name: "Hide insights" }).click();
  await expect(page.getByRole("button", { name: "Show insights" })).toHaveAttribute("aria-expanded", "false");
  await expect(page.locator("#dashboard-right-rail")).toHaveAttribute("aria-hidden", "true");
  await page.getByRole("button", { name: "Show insights" }).click();
  await expect(page.getByRole("button", { name: "Hide insights" })).toHaveAttribute("aria-expanded", "true");

  await page.keyboard.press(process.platform === "darwin" ? "Meta+K" : "Control+K");
  const commandMenu = page.getByRole("dialog", { name: "Command menu" });
  await expect(commandMenu).toBeVisible();
  await commandMenu.getByRole("button", { name: "Transactions" }).click();
  await expect(page).toHaveURL(/\/transactions$/);
  await page.getByRole("button", { name: "Overview" }).click();

  await page.getByRole("button", { name: "+ Transaction" }).click();
  await page.getByLabel("Type").selectOption("income");
  await page.getByLabel("Amount").fill("250");
  await page.getByLabel("Description").fill("Salary");
  await page.getByRole("button", { name: "Create", exact: true }).click();
  await expect(page.getByRole("dialog", { name: "Create transaction" })).toBeHidden();

  await page.getByRole("button", { name: "+ Transaction" }).click();
  await page.getByLabel("Type").selectOption("expense");
  await page.getByLabel("Amount").fill("50");
  await page.getByLabel("Description").fill("Groceries");
  await page.getByRole("button", { name: "Create", exact: true }).click();
  await expect(page.getByRole("dialog", { name: "Create transaction" })).toBeHidden();

  await page.getByRole("button", { name: "+ Transfer" }).click();
  await page.getByLabel("Amount").fill("100");
  await page.getByLabel("Description").fill("Move to savings");
  await page.getByRole("button", { name: "Create", exact: true }).click();
  await expect(page.getByRole("dialog", { name: "Create transfer" })).toBeHidden();

  await page.getByRole("button", { name: "Import" }).click();
  await expect(page.getByRole("dialog", { name: "Import transactions" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "Import transactions" })).toBeHidden();
  await expect(page.getByRole("button", { name: "Import" })).toBeFocused();
  await page.getByRole("button", { name: "Import" }).click();
  await page.getByRole("button", { name: "Open transactions" }).click();
  await expect(page).toHaveURL(/\/transactions$/);
  await page.getByRole("button", { name: "Overview" }).click();

  await expect(page.getByText("Total capital")).toBeVisible();
  await expect(page.getByText(/active accounts across/)).toHaveCount(0);
  await expect(page.getByText("Private local session")).toBeHidden();

  await page.getByRole("button", { name: "Check system health" }).click();
  await expect(page.getByRole("dialog", { name: "System health" })).toBeVisible();
  await page.getByRole("button", { name: "Close system health" }).click();
  await expect(page.getByRole("dialog", { name: "System health" })).toBeHidden();
  await expect(page.getByRole("button", { name: "Check system health" })).toBeFocused();
  await page.getByRole("button", { name: "Check system health" }).click();
  await expect(page.getByRole("dialog", { name: "System health" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "System health" })).toBeHidden();

  for (const width of [320, 768, 1280]) {
    await page.setViewportSize({ width, height: 720 });
    const overflow = await page.evaluate(() => ({
      scrollWidth: document.documentElement.scrollWidth,
      clientWidth: document.documentElement.clientWidth,
      offenders: [...document.querySelectorAll<HTMLElement>("body *")]
        .filter((element) => element.getBoundingClientRect().right > document.documentElement.clientWidth + 1)
        .slice(0, 5)
        .map((element) => ({
          tag: element.tagName,
          className: element.className,
          text: element.textContent?.trim().slice(0, 80),
          right: element.getBoundingClientRect().right,
        })),
    }));
    expect(overflow.scrollWidth).toBeLessThanOrEqual(overflow.clientWidth);
    expect(overflow.offenders).toEqual([]);
    await expect(page.getByRole("button", { name: "Open command menu" })).toBeVisible();
    await expect(page.getByText("Total capital")).toBeVisible();
  }

  await page.getByRole("button", { name: /Logout/ }).click();
  await expect(page.getByRole("button", { name: "Sign in with email" })).toBeVisible();
});

async function createAccount(page: import("@playwright/test").Page, name: string, bank: string, initialBalance: string) {
  await page.keyboard.press(process.platform === "darwin" ? "Meta+K" : "Control+K");
  await page.getByRole("dialog", { name: "Command menu" }).getByRole("button", { name: "Create account" }).click();
  await expect(page.getByRole("dialog", { name: "Create account" })).toBeVisible();
  await page.getByLabel("Name", { exact: true }).fill(name);
  await page.getByLabel("Bank", { exact: true }).fill(bank);
  await page.getByLabel("Initial balance", { exact: true }).fill(initialBalance);
  await page.getByRole("button", { name: "Create", exact: true }).click();
  await expect(page.getByRole("dialog", { name: "Create account" })).toBeHidden();
}

async function expectAppTheme(page: import("@playwright/test").Page, theme: "light" | "dark", expectedBg: string) {
  await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
  await expect.poll(
    () => page.locator(".app").evaluate((element) => getComputedStyle(element).getPropertyValue("--bg").trim()),
  ).toBe(expectedBg);
}

async function surfaceSnapshot(page: import("@playwright/test").Page, selector: string) {
  return page.locator(selector).first().evaluate((element) => {
    const style = getComputedStyle(element);
    return {
      backgroundImage: style.backgroundImage,
      backgroundColor: style.backgroundColor,
      borderColor: style.borderColor,
      color: style.color,
    };
  });
}

async function expectSurface(page: import("@playwright/test").Page, selector: string, expected: Awaited<ReturnType<typeof surfaceSnapshot>>) {
  await expect.poll(() => surfaceSnapshot(page, selector)).toEqual(expected);
}

function dashboardResponse(accounts: Account[], transactions: Transaction[], now: string) {
  const balanceByAccount = new Map<string, number>();
  for (const account of accounts) {
    balanceByAccount.set(account.id, 0);
  }
  for (const transaction of transactions) {
    const current = balanceByAccount.get(transaction.account_id) ?? 0;
    const amount = Number(transaction.amount);
    const signed = transaction.type === "expense" || transaction.type === "transfer_out"
      ? -amount
      : amount;
    balanceByAccount.set(transaction.account_id, current + signed);
  }
  const total = [...balanceByAccount.values()].reduce((sum, value) => sum + value, 0);

  return {
    generated_at: now,
    accounts_count: accounts.length,
    active_accounts_count: accounts.filter((account) => account.is_active).length,
    balances: [{ currency: "RUB", amount: String(total) }],
    monthly_income: [{ currency: "RUB", amount: String(sumByType(transactions, "income")) }],
    monthly_expense: [{ currency: "RUB", amount: String(sumByType(transactions, "expense")) }],
    monthly_interest_income: [{ currency: "RUB", amount: "0" }],
    account_balances: accounts.map((account) => ({
      account_id: account.id,
      balance: String(balanceByAccount.get(account.id) ?? 0),
      transaction_count: transactions.filter((transaction) => transaction.account_id === account.id).length,
      name: account.name,
      bank: account.bank,
      type: account.type,
      currency: account.currency,
      is_active: account.is_active,
    })),
    recent_transactions: transactions.slice(-10),
    recent_transactions_limit: 10,
    recent_transactions_returned: Math.min(transactions.length, 10),
    months: 6,
    total: [{ currency: "RUB", amount: "0" }],
    buckets: [],
  };
}

function sumByType(transactions: Transaction[], type: string) {
  return transactions
    .filter((transaction) => transaction.type === type)
    .reduce((sum, transaction) => sum + Number(transaction.amount), 0);
}
