import { render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { I18nProvider } from "../../../shared/i18n/I18nProvider";
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
  source_type: "transfer",
  source_metadata: {},
  related_account_id: "account-2",
  transfer_id: "transfer-1",
  type: "income",
  status: "confirmed",
  amount: "100.00",
  category_id: "category-1",
  description: "Salary payment",
  occurred_at: "2026-05-17T00:00:00Z",
  created_at: "2026-05-17T00:00:00Z",
};

describe("TransactionDetails", () => {
  beforeEach(() => {
    localStorage.setItem("capitalflow_locale", "en");
  });

  it("renders core transaction fields and related transfer context", () => {
    render(
      <I18nProvider>
        <TransactionDetails
          transaction={transaction}
          accounts={accounts}
          categories={categories}
        />
      </I18nProvider>,
    );

    expect(screen.getAllByText("Salary payment")).toHaveLength(2);
    expect(screen.getByText("Card")).toBeInTheDocument();
    expect(screen.getByText("Salary")).toBeInTheDocument();
    expect(screen.getByText("transaction-1")).toBeInTheDocument();
    expect(screen.getByText("Savings")).toBeInTheDocument();
    expect(screen.getByText("transfer-1")).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Main details" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Source" })).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Relations" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Audit timeline" }),
    ).toBeInTheDocument();
  });

  it("keeps transaction detail hero and audit sections in reference order", () => {
    render(
      <I18nProvider>
        <TransactionDetails
          transaction={transaction}
          accounts={accounts}
          categories={categories}
        />
      </I18nProvider>,
    );

    const hero = document.querySelector(
      ".transaction-detail-hero",
    ) as HTMLElement;
    expect(within(hero).getByText("Salary payment")).toBeInTheDocument();
    expect(within(hero).getByText(/100\.00/)).toBeInTheDocument();

    const headings = screen
      .getAllByRole("heading")
      .map((heading) => heading.textContent);
    expect(headings).toEqual([
      "Main details",
      "Source",
      "Relations",
      "Audit timeline",
    ]);
  });
});
