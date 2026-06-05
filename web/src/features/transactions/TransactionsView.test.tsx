import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Account, Category } from "../../api/types";
import { TransactionsView } from "./TransactionsView";

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

function renderTransactionsView() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <TransactionsView accounts={accounts} categories={categories} />
    </QueryClientProvider>,
  );
}

describe("TransactionsView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.transactions.mockResolvedValue([]);
  });

  it("exposes accessible names for all filters", () => {
    renderTransactionsView();

    expect(screen.getByLabelText("Filter transactions by account")).toBeInTheDocument();
    expect(screen.getByLabelText("Filter transactions by category")).toBeInTheDocument();
    expect(screen.getByLabelText("Filter transactions by type")).toBeInTheDocument();
    expect(screen.getByLabelText("Filter transactions from date")).toBeInTheDocument();
    expect(screen.getByLabelText("Filter transactions to date")).toBeInTheDocument();
  });
});
