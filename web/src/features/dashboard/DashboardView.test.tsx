import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DashboardCashflow, DashboardInterestIncome, DashboardSummary } from "../../api/types";
import { DashboardView } from "./DashboardView";

const mocks = vi.hoisted(() => ({
  dashboardSummary: vi.fn(),
  dashboardCashflow: vi.fn(),
  dashboardInterestIncome: vi.fn(),
  currencyRates: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  ApiClientError: class ApiClientError extends Error {},
  api: {
    dashboardSummary: mocks.dashboardSummary,
    dashboardCashflow: mocks.dashboardCashflow,
    dashboardInterestIncome: mocks.dashboardInterestIncome,
    currencyRates: mocks.currencyRates,
  },
}));

const summary: DashboardSummary = {
  generated_at: "2026-05-19T00:00:00Z",
  accounts_count: 1,
  active_accounts_count: 1,
  balances: [{ currency: "RUB", amount_minor: 0 }],
  monthly_income: [],
  monthly_expense: [],
  monthly_interest_income: [],
  account_balances: [
    {
      account_id: "account-1",
      balance_minor: 0,
      transaction_count: 0,
      name: "Card",
      bank: "Bank",
      type: "card",
      currency: "RUB",
      is_active: true,
    },
  ],
  recent_transactions: [],
  recent_transactions_limit: 5,
  recent_transactions_returned: 0,
};

const cashflow: DashboardCashflow = {
  generated_at: "2026-05-19T00:00:00Z",
  months: 6,
  buckets: [],
};

const interest: DashboardInterestIncome = {
  generated_at: "2026-05-19T00:00:00Z",
  months: 6,
  total: [],
  buckets: [],
};

function renderDashboardView(onOpenAccount = vi.fn()) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <DashboardView primaryCurrency="RUB" onOpenAccount={onOpenAccount} />
    </QueryClientProvider>,
  );

  return { onOpenAccount };
}

describe("DashboardView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.dashboardSummary.mockResolvedValue(summary);
    mocks.dashboardCashflow.mockResolvedValue(cashflow);
    mocks.dashboardInterestIncome.mockResolvedValue(interest);
    mocks.currencyRates.mockResolvedValue({
      base: "RUB",
      date: "2026-05-19",
      provider: "test",
      rates: {},
    });
  });

  it("opens account details from a keyboard-accessible account balance action", async () => {
    const user = userEvent.setup();
    const { onOpenAccount } = renderDashboardView();

    const action = await screen.findByRole("button", { name: "Open Card account" });
    action.focus();

    await user.keyboard("{Enter}");
    expect(onOpenAccount).toHaveBeenCalledWith("account-1");

    onOpenAccount.mockClear();
    await user.click(action);
    expect(onOpenAccount).toHaveBeenCalledWith("account-1");
  });
});
