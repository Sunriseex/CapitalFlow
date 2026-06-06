import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { Account, Category, Transaction } from "../../../api/types";
import { TransactionDetails } from "./TransactionDetails";

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
  {
    id: "account-2",
    name: "Savings",
    type: "savings",
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
  related_account_id: "account-2",
  transfer_id: "transfer-1",
  type: "income",
  amount: "100.00",
  category_id: "category-1",
  description: "Salary payment",
  occurred_at: "2026-05-17T00:00:00Z",
  created_at: "2026-05-17T00:00:00Z",
};

describe("TransactionDetails", () => {
  it("renders core transaction fields and related transfer context", () => {
    render(<TransactionDetails transaction={transaction} accounts={accounts} categories={categories} />);

    expect(screen.getAllByText("Salary payment")).toHaveLength(2);
    expect(screen.getByText("Card")).toBeInTheDocument();
    expect(screen.getByText("Salary")).toBeInTheDocument();
    expect(screen.getByText("transaction-1")).toBeInTheDocument();
    expect(screen.getByText("Savings")).toBeInTheDocument();
    expect(screen.getByText("transfer-1")).toBeInTheDocument();
  });
});
