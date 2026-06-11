import type { ReactElement } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Account, Category, Transaction } from "../../api/types";
import { I18nProvider } from "../../shared/i18n/I18nProvider";
import { TransactionsTable } from "./TransactionsTable";

function renderWithI18n(ui: ReactElement) {
  return render(<I18nProvider>{ui}</I18nProvider>);
}
const accounts: Account[] = [
  {
    id: "account-1",
    name: "Card",
    type: "card",
    currency: "RUB",
    is_active: true,
    opened_at: "2026-05-17",
    created_at: "2026-05-17T00:00:00Z",
    updated_at: "2026-05-17T00:00:00Z",
  },
];

const categories: Category[] = [
  {
    id: "category-1",
    slug: "salary",
    name: "Salary",
    created_at: "2026-05-17T00:00:00Z",
    updated_at: "2026-05-17T00:00:00Z",
  },
];

const incomeTransaction: Transaction = {
  id: "transaction-1",
  account_id: "account-1",
  type: "income",
  amount: "100.00",
  category_id: "category-1",
  description: "Salary",
  occurred_at: "2026-05-17T00:00:00Z",
  created_at: "2026-05-17T00:00:00Z",
};

describe("TransactionsTable", () => {
  beforeEach(() => {
    localStorage.setItem("capitalflow_locale", "en");
    mockMediaQuery(false);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders transactions without a delete action", () => {
    renderWithI18n(
      <TransactionsTable
        transactions={[incomeTransaction]}
        accounts={accounts}
        categories={categories}
      />,
    );

    expect(screen.getAllByText("Salary")).toHaveLength(2);
    expect(screen.getAllByText("Verified")).toHaveLength(1);
    expect(
      screen.queryByRole("button", { name: /delete transaction/i }),
    ).not.toBeInTheDocument();
  });

  it("renders empty state", () => {
    renderWithI18n(
      <TransactionsTable
        transactions={[]}
        accounts={accounts}
        categories={categories}
      />,
    );

    expect(screen.getByText("No transactions")).toBeInTheDocument();
  });

  it("renders chunked account history and reveals more rows on demand", async () => {
    const user = userEvent.setup();
    const transactions = Array.from(
      { length: 205 },
      (_, index): Transaction => ({
        ...incomeTransaction,
        id: `transaction-${index}`,
        description: `Transaction ${index}`,
      }),
    );

    renderWithI18n(
      <TransactionsTable
        transactions={transactions}
        accounts={accounts}
        categories={categories}
        chunked
      />,
    );

    expect(screen.getAllByRole("row")).toHaveLength(49);
    expect(screen.getByText("48 of 205")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Show more" }));

    expect(screen.getAllByRole("row")).toHaveLength(145);
    expect(screen.getByText("144 of 205")).toBeInTheDocument();
  });

  it("opens transaction details from click and keyboard", async () => {
    const user = userEvent.setup();
    const onOpenTransaction = vi.fn();

    renderWithI18n(
      <TransactionsTable
        transactions={[incomeTransaction]}
        accounts={accounts}
        categories={categories}
        onOpenTransaction={onOpenTransaction}
      />,
    );

    const row = screen.getByRole("row", { name: /Salary/ });
    await user.click(row);
    expect(onOpenTransaction).toHaveBeenCalledWith(incomeTransaction);

    onOpenTransaction.mockClear();
    row.focus();
    await user.keyboard("{Enter}");
    expect(onOpenTransaction).toHaveBeenCalledWith(incomeTransaction);

    onOpenTransaction.mockClear();
    await user.keyboard(" ");
    expect(onOpenTransaction).toHaveBeenCalledWith(incomeTransaction);
  });

  it("opens transaction details from the mobile card", async () => {
    const user = userEvent.setup();
    const onOpenTransaction = vi.fn();
    mockMediaQuery(true);

    renderWithI18n(
      <TransactionsTable
        transactions={[incomeTransaction]}
        accounts={accounts}
        categories={categories}
        onOpenTransaction={onOpenTransaction}
      />,
    );

    await user.click(
      screen.getByRole("button", {
        name: /Open transaction details: Salary/,
      }),
    );

    expect(onOpenTransaction).toHaveBeenCalledWith(incomeTransaction);
    expect(screen.queryByRole("row", { name: /Salary/ })).not.toBeInTheDocument();
  });
});

function mockMediaQuery(matches: boolean) {
  vi.stubGlobal(
    "matchMedia",
    vi.fn().mockImplementation((query: string) => ({
      matches,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  );
}
