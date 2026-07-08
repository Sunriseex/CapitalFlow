import { expect, test, type Page } from "@playwright/test";

const password = "Correct-Horse-Battery-Staple-2026!";

test("owner can create accounts, record cashflow, transfer money, and see exact balances", async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem("capitalflow_locale", "en");
    localStorage.setItem("capitalflow_theme", "light");
  });

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Create the owner account" })).toBeVisible();
  await page.getByLabel("Owner email").fill("owner@capitalflow.test");
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByLabel("Confirm password").fill(password);
  await page.getByLabel(/I understand that this account/).check();
  await page.getByRole("button", { name: "Create owner account" }).click();
  await expect(page.getByRole("heading", { name: "Overview" }).first()).toBeVisible();

  await createAccount(page, "Cash", "Wallet", "1000");
  await createAccount(page, "Savings", "Bank", "0");
  await createTransaction(page, "income", "250", "Salary");
  await createTransaction(page, "expense", "50", "Groceries");

  await page.getByRole("button", { name: "Create transfer" }).first().click();
  const transfer = page.getByRole("dialog", { name: "Create transfer" });
  await transfer.getByLabel("From").selectOption({ label: "Cash" });
  await transfer.getByLabel("To").selectOption({ label: "Savings" });
  await transfer.getByLabel("Amount").fill("100");
  await transfer.getByLabel("Description").fill("Move to savings");
  await transfer.getByRole("button", { name: "Create", exact: true }).click();
  await expect(transfer).toBeHidden();

  await openWorkspace(page, "Accounts");
  await expect(page.getByRole("row").filter({ hasText: "Cash" })).toContainText("1,100.00");
  await expect(page.getByRole("row").filter({ hasText: "Savings" })).toContainText("100.00");

  await openWorkspace(page, "Transactions");
  await expect(page.getByRole("row", { name: /Groceries/ })).toBeVisible();
  await expect(page.getByRole("row", { name: /Move to savings/ })).toHaveCount(2);
});

async function createAccount(page: Page, name: string, bank: string, balance: string) {
  await page.getByRole("button", { name: "Create account" }).first().click();
  const dialog = page.getByRole("dialog", { name: "Create account" });
  await dialog.getByLabel("Card name", { exact: true }).fill(name);
  await dialog.getByLabel("Bank", { exact: true }).fill(bank);
  await dialog.getByLabel("Current balance", { exact: true }).fill(balance);
  await dialog.getByRole("button", { name: "Create", exact: true }).click();
  await expect(dialog).toBeHidden();
}

async function createTransaction(page: Page, type: "income" | "expense", amount: string, description: string) {
  await page.getByRole("button", { name: "Add transaction" }).first().click();
  const dialog = page.getByRole("dialog", { name: "Create transaction" });
  await dialog.getByRole("combobox", { name: "Type" }).click();
  await page
    .getByRole("option", { name: type === "income" ? "Income" : "Expense" })
    .click();
  await dialog.getByLabel("Amount").fill(amount);
  await dialog.getByLabel("Description").fill(description);
  await dialog.getByRole("button", { name: "Create", exact: true }).click();
  await expect(dialog).toBeHidden();
}

async function openWorkspace(page: Page, name: string) {
  await page
    .getByRole("navigation", { name: "Workspace" })
    .getByRole("button", { name: new RegExp(`^${name}`) })
    .click();
}
