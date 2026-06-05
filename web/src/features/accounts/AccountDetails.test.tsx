import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Account, Transaction } from "../../api/types";
import { AccountDetails } from "./AccountDetails";

const chartMocks = vi.hoisted(() => ({
  lineChart: vi.fn(({ children }: { children?: ReactNode }) => <div data-testid="line-chart">{children}</div>),
}));

const apiMocks = vi.hoisted(() => ({
  transactions: vi.fn(),
  accountBalance: vi.fn(),
  interestRules: vi.fn(),
  accrueInterest: vi.fn(),
  archiveAccount: vi.fn(),
}));

vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  LineChart: chartMocks.lineChart,
  CartesianGrid: () => null,
  Line: () => null,
  Tooltip: () => null,
  XAxis: () => null,
  YAxis: () => null,
}));

vi.mock("../../api/client", () => ({
  api: {
    transactions: apiMocks.transactions,
    accountBalance: apiMocks.accountBalance,
    interestRules: apiMocks.interestRules,
    accrueInterest: apiMocks.accrueInterest,
    archiveAccount: apiMocks.archiveAccount,
  },
}));

const account: Account = {
  id: "account-1",
  name: "Card",
  bank: "Bank",
  type: "card",
  currency: "RUB",
  is_active: true,
  opened_at: "2026-05-17",
  created_at: "2026-05-17T00:00:00Z",
  updated_at: "2026-05-17T00:00:00Z",
};

function renderAccountDetails() {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <AccountDetails account={account} onBack={vi.fn()} />
    </QueryClientProvider>,
  );
}

describe("AccountDetails", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiMocks.accountBalance.mockResolvedValue({ balance: "0", currency: "RUB" });
    apiMocks.interestRules.mockResolvedValue([]);
  });

  it("renders account summary before chart and table heavy content", async () => {
    apiMocks.transactions.mockResolvedValue([]);

    renderAccountDetails();

    expect(screen.getByText("Account summary")).toBeInTheDocument();
    expect(screen.getByText("Preparing chart")).toBeInTheDocument();
    expect(screen.getByText("Preparing transactions")).toBeInTheDocument();

    expect(await screen.findByText("Running balance chart has no transactions.")).toHaveClass("sr-only");
  });

  it("caps running balance chart points for large transaction histories", async () => {
    apiMocks.transactions.mockResolvedValue(Array.from({ length: 1000 }, (_, index): Transaction => ({
      id: `tx-${index}`,
      account_id: account.id,
      type: "income",
      amount: "1.00",
      category_id: null,
      description: `Transaction ${index}`,
      occurred_at: `2026-01-${String((index % 28) + 1).padStart(2, "0")}T00:00:00Z`,
      created_at: "2026-01-01T00:00:00Z",
    })));

    renderAccountDetails();

    await waitFor(() => {
      const latestProps = chartMocks.lineChart.mock.calls.at(-1)?.[0] as { data: unknown[] };
      expect(latestProps.data).toHaveLength(240);
    });
    expect(screen.getByText(/Running balance chart covers 1000 transactions/)).toHaveClass("sr-only");
  });
});
