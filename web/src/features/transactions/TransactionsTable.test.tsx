import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import type { Account, Category, Transaction } from "../../api/types";
import { TransactionsTable } from "./TransactionsTable";

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
  it("renders transactions without a delete action", () => {
    render(<TransactionsTable transactions={[incomeTransaction]} accounts={accounts} categories={categories} />);

    expect(screen.getAllByText("Salary")).toHaveLength(2);
    expect(screen.queryByRole("button", { name: /delete transaction/i })).not.toBeInTheDocument();
  });

  it("renders empty state", () => {
    render(<TransactionsTable transactions={[]} accounts={accounts} categories={categories} />);

    expect(screen.getByText("No transactions")).toBeInTheDocument();
  });

  it("renders chunked account history and reveals more rows on demand", async () => {
    const user = userEvent.setup();
    const transactions = Array.from({ length: 205 }, (_, index): Transaction => ({
      ...incomeTransaction,
      id: `transaction-${index}`,
      description: `Transaction ${index}`,
    }));

    render(<TransactionsTable transactions={transactions} accounts={accounts} categories={categories} chunked />);

    expect(screen.getAllByRole("row")).toHaveLength(81);
    expect(screen.getByText("80 of 205")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Show more" }));

    expect(screen.getAllByRole("row")).toHaveLength(201);
    expect(screen.getByText("200 of 205")).toBeInTheDocument();
  });
});
