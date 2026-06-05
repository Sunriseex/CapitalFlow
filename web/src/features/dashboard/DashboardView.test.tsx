import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DashboardCashflow, DashboardInterestIncome, DashboardSummary } from "../../api/types";
import { Provider } from "../../components/ui/provider";
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
  balances: [{ currency: "RUB", amount: "0" }],
  monthly_income: [],
  monthly_expense: [],
  monthly_interest_income: [],
  account_balances: [
    {
      account_id: "account-1",
      balance: "0",
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
  buckets: [
    {
      period: "2026-04",
      income: [{ currency: "RUB", amount: "120000" }],
      expense: [{ currency: "RUB", amount: "64000" }],
      net_cashflow: [{ currency: "RUB", amount: "56000" }],
      transaction_count: 4,
    },
    {
      period: "2026-05",
      income: [{ currency: "RUB", amount: "132000" }],
      expense: [{ currency: "RUB", amount: "71000" }],
      net_cashflow: [{ currency: "RUB", amount: "61000" }],
      transaction_count: 6,
    },
  ],
};

const interest: DashboardInterestIncome = {
  generated_at: "2026-05-19T00:00:00Z",
  months: 6,
  total: [],
  buckets: [],
};

function renderDashboardView({
  onOpenAccount = vi.fn<(id: string) => void>(),
  onQuickAction,
  onNavigate,
  primaryCurrency = "RUB",
}: {
  onOpenAccount?: (id: string) => void;
  onQuickAction?: (action: NonNullable<import("../../shared/constants").QuickAction>) => void;
  onNavigate?: (view: import("../../shared/constants").View) => void;
  primaryCurrency?: string;
} = {}) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <Provider>
      <QueryClientProvider client={queryClient}>
        <DashboardView primaryCurrency={primaryCurrency} onOpenAccount={onOpenAccount} onQuickAction={onQuickAction} onNavigate={onNavigate} />
      </QueryClientProvider>
    </Provider>,
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

  it("shows loading and summary API errors", async () => {
    mocks.dashboardSummary.mockReturnValueOnce(new Promise(() => {}));
    renderDashboardView();

    expect(screen.getByText("Loading dashboard")).toBeInTheDocument();

    mocks.dashboardSummary.mockRejectedValueOnce(new Error("Dashboard unavailable"));
    renderDashboardView();

    expect(await screen.findByText("Dashboard unavailable")).toBeInTheDocument();
  });

  it("renders empty balance and transaction states", async () => {
    mocks.dashboardSummary.mockResolvedValueOnce({
      ...summary,
      accounts_count: 0,
      active_accounts_count: 0,
      balances: [],
      account_balances: [],
      recent_transactions: [],
      recent_transactions_returned: 0,
    } satisfies DashboardSummary);

    renderDashboardView();

    expect(await screen.findByText("Total capital")).toBeInTheDocument();
    expect(screen.queryByText(/active accounts across/)).not.toBeInTheDocument();
    expect(screen.getByText("No positive balances")).toBeInTheDocument();
    expect(screen.getByText("No transactions")).toBeInTheDocument();
  });

  it("renders the reference dashboard structure", async () => {
    const onQuickAction = vi.fn();
    renderDashboardView({ onQuickAction });

    expect(await screen.findByText("Total capital")).toBeInTheDocument();
    expect(screen.getByLabelText("Quick actions")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "+ Transaction" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "+ Transfer" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Import" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Accounts" })).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Recent transactions" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Upcoming" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Rates" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Allocation" })).toBeInTheDocument();
  });

  it("wires dashboard buttons to real actions and navigation", async () => {
    const user = userEvent.setup();
    const onQuickAction = vi.fn();
    const onNavigate = vi.fn();
    renderDashboardView({ onQuickAction, onNavigate });

    await user.click(await screen.findByRole("button", { name: "+ Transaction" }));
    expect(onQuickAction).toHaveBeenCalledWith("transaction");

    await user.click(screen.getByRole("button", { name: "+ Transfer" }));
    expect(onQuickAction).toHaveBeenCalledWith("transfer");

    await user.click(screen.getByRole("button", { name: "Import" }));
    expect(onQuickAction).toHaveBeenCalledWith("import");

    await user.click(screen.getByRole("button", { name: "All transactions" }));
    expect(onNavigate).toHaveBeenCalledWith("transactions");

    await user.click(screen.getByRole("button", { name: "Settings" }));
    expect(onNavigate).toHaveBeenCalledWith("settings");
  });

  it("renders cashflow chart from dashboard cashflow API buckets", async () => {
    renderDashboardView();

    expect(await screen.findByText("2 month buckets")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Week" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Month" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Quarter" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Year" })).toBeInTheDocument();
    expect(mocks.dashboardCashflow).toHaveBeenCalled();
    expect(screen.getByLabelText("Income and expense chart")).toBeInTheDocument();
    expect(screen.getByText(/Cashflow chart covers 2 periods/)).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Cashflow data" })).toBeInTheDocument();
  });

  it("switches cashflow between reference periods", async () => {
    const user = userEvent.setup();
    renderDashboardView();

    await user.click(await screen.findByRole("button", { name: "Quarter" }));
    expect(await screen.findByText("1 quarter buckets")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Week" }));
    expect(await screen.findByText("Weekly cashflow unavailable")).toBeInTheDocument();
    expect(screen.getByText("The backend currently returns monthly cashflow buckets.")).toBeInTheDocument();
  });

  it("toggles the right rail without unmounting dashboard content", async () => {
    const user = userEvent.setup();
    renderDashboardView();

    const toggle = await screen.findByRole("button", { name: "Hide insights" });
    const rail = screen.getByLabelText("Right rail summary");
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(rail).toHaveAttribute("aria-hidden", "false");

    await user.click(toggle);

    expect(screen.getByRole("button", { name: "Show insights" })).toHaveAttribute("aria-expanded", "false");
    expect(rail).toHaveAttribute("aria-hidden", "true");
    expect(screen.getByText("Total capital")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Show insights" }));

    expect(screen.getByRole("button", { name: "Hide insights" })).toHaveAttribute("aria-expanded", "true");
    expect(rail).toHaveAttribute("aria-hidden", "false");
  });

  it("formats chart summaries for custom currencies such as USDT", async () => {
    mocks.dashboardCashflow.mockResolvedValueOnce({
      ...cashflow,
      buckets: [
        {
          ...cashflow.buckets[0],
          income: [{ currency: "USDT", amount: "1.25" }],
          expense: [{ currency: "USDT", amount: "0.5" }],
          net_cashflow: [{ currency: "USDT", amount: "0.75" }],
        },
      ],
    } satisfies DashboardCashflow);
    renderDashboardView({ primaryCurrency: "USDT" });

    expect(await screen.findByText(/Cashflow chart covers 1 periods/)).toHaveTextContent("1.25 USDT");
  });

  it("keeps recent transaction rows static and uses a real view button", async () => {
    const onNavigate = vi.fn();
    mocks.dashboardSummary.mockResolvedValueOnce({
      ...summary,
      recent_transactions: [
        {
          id: "tx-1",
          account_id: "account-1",
          type: "expense",
          amount: "25.00",
          category_id: null,
          description: "Coffee",
          occurred_at: "2026-05-19T00:00:00Z",
          created_at: "2026-05-19T00:00:00Z",
        },
      ],
      recent_transactions_returned: 1,
    } satisfies DashboardSummary);
    renderDashboardView({ onNavigate });

    const table = await screen.findByRole("table", { name: "Recent transactions" });
    const row = within(table).getByRole("row", { name: /Coffee/ });
    expect(row).not.toHaveAttribute("tabindex");

    await userEvent.click(within(row).getByRole("button", { name: "Open transaction details" }));
    expect(onNavigate).toHaveBeenCalledWith("transactions");
  });

  it("switches dashboard currency and reloads conversion rates", async () => {
    const user = userEvent.setup();
    mocks.dashboardSummary.mockResolvedValueOnce({
      ...summary,
      balances: [
        { currency: "RUB", amount: "1000.00" },
        { currency: "USD", amount: "100.00" },
      ],
    } satisfies DashboardSummary);

    renderDashboardView();

    await screen.findByRole("button", { name: "USD" });
    expect(mocks.currencyRates).toHaveBeenCalledWith("RUB");

    await user.click(screen.getByRole("button", { name: "USD" }));

    expect(await screen.findByText("Cashflow (USD)")).toBeInTheDocument();
    await waitFor(() => expect(mocks.currencyRates).toHaveBeenCalledWith("USD"));
  });

  it("shows a clear empty state when currency rates are unavailable", async () => {
    mocks.currencyRates.mockRejectedValueOnce(new Error("Rate provider unavailable"));

    renderDashboardView();

    expect(await screen.findAllByText("Rates unavailable")).toHaveLength(2);
    expect(screen.queryByText("Rate provider unavailable")).not.toBeInTheDocument();
  });

  it("shows fixed reference rate pairs for the selected main currency", async () => {
    mocks.currencyRates.mockResolvedValueOnce({
      base: "RUB",
      date: "2026-05-19",
      fetched_at: "2026-06-05T00:02:31Z",
      provider: "test",
      rates: {
        EUR: 0.01,
        USD: 0.011,
        BTC: 0.00000017,
      },
    });

    renderDashboardView();

    const ratesCard = (await screen.findByRole("heading", { name: "Rates" })).closest("article");
    expect(ratesCard).not.toBeNull();
    const labels = within(ratesCard as HTMLElement).getAllByText(/RUB\//).map((node) => node.textContent);
    expect(labels).toEqual(["RUB/USD", "RUB/EUR", "RUB/BTC"]);
    expect(within(ratesCard as HTMLElement).getByText("Fri, 05 Jun 2026 00:02:31")).toBeInTheDocument();
  });

  it("opens account details from the keyboard-accessible allocation action", async () => {
    const user = userEvent.setup();
    const onOpenAccount = vi.fn<(id: string) => void>();
    mocks.dashboardSummary.mockResolvedValueOnce({
      ...summary,
      balances: [{ currency: "RUB", amount: "100" }],
      account_balances: [{ ...summary.account_balances[0], balance: "100" }],
    } satisfies DashboardSummary);
    renderDashboardView({ onOpenAccount });

    const action = await screen.findByRole("button", { name: /Card/ });
    action.focus();

    await user.keyboard("{Enter}");
    expect(onOpenAccount).toHaveBeenCalledWith("account-1");

    onOpenAccount.mockClear();
    await user.click(action);
    expect(onOpenAccount).toHaveBeenCalledWith("account-1");
  });
});
