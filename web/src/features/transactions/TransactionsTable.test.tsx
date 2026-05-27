import { render, screen } from "@testing-library/react";
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
});
