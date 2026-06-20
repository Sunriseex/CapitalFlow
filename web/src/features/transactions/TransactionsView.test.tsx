import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Account, Category, Transaction } from "../../api/types";
import { TransactionsView } from "./TransactionsView";
import { I18nProvider } from "../../shared/i18n/I18nProvider";

const mocks = vi.hoisted(() => ({
  transactions: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  api: {
    transactions: mocks.transactions,
  },
}));

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

const transaction: Transaction = {
  id: "transaction-1",
  account_id: "account-1",
  type: "income",
  amount: "100.00",
  category_id: "category-1",
  description: "Salary",
  occurred_at: "2026-05-17T00:00:00Z",
  created_at: "2026-05-17T00:00:00Z",
};

function renderTransactionsView() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <I18nProvider>
      <QueryClientProvider client={queryClient}>
        <TransactionsView accounts={accounts} categories={categories} />
      </QueryClientProvider>
    </I18nProvider>,
  );
}

describe("TransactionsView", () => {
  beforeEach(() => {
    localStorage.setItem("capitalflow_locale", "en");
    vi.clearAllMocks();
    mocks.transactions.mockResolvedValue([]);
  });

  it("exposes accessible names for all filters", () => {
    renderTransactionsView();

    expect(
      screen.getByLabelText("Filter transactions by account"),
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText("Filter transactions by category"),
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText("Filter transactions by type"),
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText("Filter transactions from date"),
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText("Filter transactions to date"),
    ).toBeInTheDocument();
  });

  it("opens the adjustment dialog with a single visible title and returns focus on Escape", async () => {
    const user = userEvent.setup();
    renderTransactionsView();

    const trigger = screen.getByRole("button", { name: "Adjustment" });
    await user.click(trigger);

    expect(
      await screen.findByRole("dialog", { name: "Create adjustment" }),
    ).toBeInTheDocument();
    expect(
      screen
        .getByRole("dialog", { name: "Create adjustment" })
        .closest(".transactions-panel"),
    ).toBeNull();
    expect(
      document.body.contains(
        screen.getByRole("dialog", { name: "Create adjustment" }),
      ),
    ).toBe(true);
    expect(
      screen.getAllByRole("heading", { name: "Create adjustment" }),
    ).toHaveLength(1);
    expect(
      screen.getByText("Adjustment accepts positive or negative values."),
    ).toBeInTheDocument();

    await user.keyboard("{Escape}");

    expect(
      screen.queryByRole("dialog", { name: "Create adjustment" }),
    ).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it("opens transaction details from a transaction row", async () => {
    const user = userEvent.setup();
    mocks.transactions.mockResolvedValueOnce([transaction]);
    renderTransactionsView();

    const row = await screen.findByRole("row", { name: /Salary/ });
    await user.click(row);

    expect(
      await screen.findByRole("dialog", { name: "Transaction details" }),
    ).toBeInTheDocument();
    expect(screen.getByText("transaction-1")).toBeInTheDocument();
  });
});
