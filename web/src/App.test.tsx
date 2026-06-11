import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClientError } from "./api/client";
import { App } from "./App";
import { Provider } from "./components/ui/provider";
import { I18nProvider } from "./shared/i18n/I18nProvider";

const mocks = vi.hoisted(() => ({
  token: "token",
  authStatus: vi.fn(),
  login: vi.fn(),
  setup: vi.fn(),
  accounts: vi.fn(),
  categories: vi.fn(),
  profile: vi.fn(),
  serviceStatus: vi.fn(),
  dashboardSummary: vi.fn(),
  interestRules: vi.fn(),
  transactions: vi.fn(),
  createAccount: vi.fn(),
  createTransaction: vi.fn(),
  createInterestRule: vi.fn(),
}));

vi.mock("./api/client", () => ({
  ApiClientError: class ApiClientError extends Error {
    status: number;

    constructor(message: string, status: number) {
      super(message);
      this.status = status;
    }
  },
  api: {
    authStatus: mocks.authStatus,
    login: mocks.login,
    setup: mocks.setup,
    accounts: mocks.accounts,
    categories: mocks.categories,
    profile: mocks.profile,
    serviceStatus: mocks.serviceStatus,
    dashboardSummary: mocks.dashboardSummary,
    interestRules: mocks.interestRules,
    transactions: mocks.transactions,
    createAccount: mocks.createAccount,
    createTransaction: mocks.createTransaction,
    createInterestRule: mocks.createInterestRule,
  },
  clearStoredSession: vi.fn(),
  getStoredToken: () => mocks.token,
}));

vi.mock("./features/dashboard/DashboardView", () => ({
  DashboardView: ({
    quickActionsDisabled,
    onQuickAction,
  }: {
    quickActionsDisabled?: boolean;
    onQuickAction?: (
      action: "transaction" | "transfer" | "account" | "import",
    ) => void;
  }) => (
    <div>
      Dashboard mock
      <button
        disabled={quickActionsDisabled}
        onClick={() => onQuickAction?.("transaction")}
      >
        + Transaction
      </button>
      <button
        disabled={quickActionsDisabled}
        onClick={() => onQuickAction?.("transfer")}
      >
        + Transfer
      </button>
      <button onClick={() => onQuickAction?.("import")}>Import</button>
    </div>
  ),
}));

vi.mock("./features/accounts/AccountDetails", () => ({
  AccountDetails: ({
    account,
    onBack,
  }: {
    account: { name: string };
    onBack: () => void;
  }) => (
    <div>
      <h2>{account.name}</h2>
      <button onClick={onBack}>Back to accounts</button>
    </div>
  ),
}));

vi.mock("./features/auth/passkeys", () => ({
  browserSupportsPasskeys: () => true,
  passkeyErrorMessage: (err: unknown) =>
    err instanceof Error ? err.message : "Passkey operation failed",
  signInWithPasskey: vi.fn(),
}));

function renderApp() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <Provider>
      <I18nProvider>
        <QueryClientProvider client={queryClient}>
          <App />
        </QueryClientProvider>
      </I18nProvider>
    </Provider>,
  );

  return { queryClient };
}

describe("App auth screens", () => {
  beforeEach(() => {
    window.history.pushState({}, "", "/dashboard");
    localStorage.setItem("capitalflow_theme", "light");
    localStorage.setItem("capitalflow_locale", "en");
    vi.clearAllMocks();
    mocks.token = "";
    mocks.authStatus.mockResolvedValue({ setup_required: false });
    mocks.login.mockResolvedValue({
      user: {
        id: "user-1",
        email: "user@example.com",
        primary_currency: "RUB",
      },
      access_token: "token",
      access_expires_at: "2026-05-19T01:00:00Z",
    });
    mocks.setup.mockResolvedValue({
      user: {
        id: "user-1",
        email: "owner@example.com",
        primary_currency: "RUB",
      },
      access_token: "token",
      access_expires_at: "2026-05-19T01:00:00Z",
    });
    mocks.accounts.mockResolvedValue([]);
    mocks.categories.mockResolvedValue([]);
    mocks.profile.mockResolvedValue({
      user: {
        id: "user-1",
        email: "user@example.com",
        primary_currency: "RUB",
      },
    });
    mocks.serviceStatus.mockResolvedValue({ status: "ok", version: "v0.5.9" });
    mocks.transactions.mockResolvedValue([]);
    mocks.createAccount.mockResolvedValue({
      id: "account-created",
      name: "Brokerage",
      bank: "Bank",
      type: "card",
      currency: "RUB",
      is_active: true,
      opened_at: "2026-05-19",
      created_at: "2026-05-19T00:00:00Z",
      updated_at: "2026-05-19T00:00:00Z",
    });
    mocks.createTransaction.mockResolvedValue({});
    mocks.createInterestRule.mockResolvedValue({});
  });

  it("renders login as a standalone screen with passkey sign-in", async () => {
    renderApp();

    expect(
      await screen.findByRole("heading", { name: "Sign in" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("form", { name: "Login form" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Sign in with passkey" }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("page-transition")).toBeInTheDocument();
  });

  it("toggles password visibility on the login screen", async () => {
    const user = userEvent.setup();
    renderApp();

    const password = await screen.findByLabelText("Password");
    expect(password).toHaveAttribute("type", "password");

    await user.click(screen.getByRole("button", { name: "Show password" }));
    expect(password).toHaveAttribute("type", "text");

    await user.click(screen.getByRole("button", { name: "Hide password" }));
    expect(password).toHaveAttribute("type", "password");
  });

  it("renders initial setup as a separate screen without passkey sign-in", async () => {
    mocks.authStatus.mockResolvedValue({ setup_required: true });

    renderApp();

    expect(
      await screen.findByRole("heading", { name: "Create the owner account" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("form", { name: "Initial setup form" }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Owner email")).toBeInTheDocument();
    expect(screen.getByLabelText("Primary currency")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Sign in with passkey" }),
    ).not.toBeInTheDocument();
  });

  it("checks setup password strength and confirmation before submit", async () => {
    const user = userEvent.setup();
    mocks.authStatus.mockResolvedValue({ setup_required: true });
    renderApp();

    await user.type(
      await screen.findByLabelText("Owner email"),
      "owner@example.com",
    );
    await user.type(screen.getByLabelText("Password"), "abc");
    await user.type(screen.getByLabelText("Confirm password"), "abc");
    await user.click(
      screen.getByRole("button", { name: "Create owner account" }),
    );

    expect(
      await screen.findAllByText(
        "Use a stronger password. Password score must be at least 3 of 4.",
      ),
    ).toHaveLength(1);
    expect(mocks.setup).not.toHaveBeenCalled();

    await user.clear(screen.getByLabelText("Password"));
    await user.type(
      screen.getByLabelText("Password"),
      "correct horse battery staple 2026!",
    );
    await user.clear(screen.getByLabelText("Confirm password"));
    await user.type(
      screen.getByLabelText("Confirm password"),
      "different horse battery staple 2026!",
    );
    await user.click(
      screen.getByRole("button", { name: "Create owner account" }),
    );

    expect(
      await screen.findByText("Password confirmation does not match."),
    ).toBeInTheDocument();
    expect(mocks.setup).not.toHaveBeenCalled();
    expect(
      screen.getByRole("meter", { name: "Password strength score" }),
    ).toHaveAttribute("aria-valuetext", "Strong");

    await user.clear(screen.getByLabelText("Confirm password"));
    await user.type(
      screen.getByLabelText("Confirm password"),
      "correct horse battery staple 2026!",
    );
    await user.click(
      screen.getByRole("button", { name: "Create owner account" }),
    );

    expect(
      await screen.findAllByText(
        "Please confirm the owner account requirement.",
      ),
    ).toHaveLength(1);
    expect(mocks.setup).not.toHaveBeenCalled();

    await user.click(
      screen.getByLabelText(
        /I understand that this account becomes the service owner/,
      ),
    );
    await user.click(
      screen.getByRole("button", { name: "Create owner account" }),
    );

    await waitFor(() => expect(mocks.setup).toHaveBeenCalled());
  });

  it("toggles setup password fields visibility", async () => {
    const user = userEvent.setup();
    mocks.authStatus.mockResolvedValue({ setup_required: true });
    renderApp();

    await screen.findByRole("heading", { name: "Create the owner account" });
    const password = screen.getByLabelText("Password");
    const confirmation = screen.getByLabelText("Confirm password");

    await user.click(screen.getByRole("button", { name: "Show password" }));
    await user.click(
      screen.getByRole("button", { name: "Show password confirmation" }),
    );

    expect(password).toHaveAttribute("type", "text");
    expect(confirmation).toHaveAttribute("type", "text");
  });

  it("authenticates through login and reaches the dashboard", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.type(await screen.findByLabelText("Email"), "user@example.com");
    await user.type(screen.getByLabelText("Password"), "password");
    await user.click(
      screen.getByRole("button", { name: "Sign in with email" }),
    );

    await waitFor(() =>
      expect(mocks.login).toHaveBeenCalledWith({
        email: "user@example.com",
        password: "password",
      }),
    );
    expect(await screen.findByText("Dashboard mock")).toBeInTheDocument();
  });

  it("shows field errors for credential failures and global copy for technical login errors", async () => {
    const user = userEvent.setup();
    mocks.login.mockRejectedValueOnce(
      new ApiClientError("Invalid credentials", 401),
    );
    renderApp();

    await user.type(await screen.findByLabelText("Email"), "bad@example.com");
    await user.type(screen.getByLabelText("Password"), "wrong");
    await user.click(
      screen.getByRole("button", { name: "Sign in with email" }),
    );

    expect(
      await screen.findByText("Check the email address for this sign-in."),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Check the password for this sign-in."),
    ).toBeInTheDocument();
    expect(screen.queryByText("Invalid credentials")).not.toBeInTheDocument();

    mocks.login.mockRejectedValueOnce(
      new ApiClientError("Server unavailable", 503),
    );
    await user.click(
      screen.getByRole("button", { name: "Sign in with email" }),
    );

    expect(await screen.findByText("Server unavailable")).toBeInTheDocument();
    expect(
      screen.queryByText("Check the email address for this sign-in."),
    ).not.toBeInTheDocument();
  });

  it("shows auth status loading and error states before choosing a screen", async () => {
    mocks.authStatus.mockReturnValueOnce(new Promise(() => {}));
    renderApp();

    expect(
      screen.getByRole("heading", { name: "Checking access" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: "Sign in" }),
    ).not.toBeInTheDocument();

    vi.clearAllMocks();
    mocks.authStatus.mockRejectedValueOnce(new Error("Status unavailable"));
    renderApp();

    expect(
      await screen.findByRole("heading", {
        name: "Authentication unavailable",
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("Status unavailable")).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: "Sign in" }),
    ).not.toBeInTheDocument();
  });
});

describe("App query states", () => {
  beforeEach(() => {
    window.history.pushState({}, "", "/dashboard");
    localStorage.setItem("capitalflow_theme", "light");
    vi.clearAllMocks();
    mocks.token = "token";
    mocks.accounts.mockResolvedValue([
      {
        id: "account-1",
        name: "Card",
        bank: "Bank",
        type: "card",
        currency: "RUB",
        is_active: true,
        opened_at: "2026-05-19",
        created_at: "2026-05-19T00:00:00Z",
        updated_at: "2026-05-19T00:00:00Z",
      },
    ]);
    mocks.categories.mockResolvedValue([]);
    mocks.profile.mockResolvedValue({
      user: {
        id: "user-1",
        email: "user@example.com",
        primary_currency: "RUB",
      },
    });
    mocks.serviceStatus.mockResolvedValue({ status: "ok", version: "v0.5.8" });
    mocks.dashboardSummary.mockResolvedValue({
      account_balances: [],
    });
    mocks.interestRules.mockResolvedValue([]);
    mocks.transactions.mockResolvedValue([]);
    mocks.createAccount.mockResolvedValue({
      id: "account-created",
      name: "Brokerage",
      bank: "Bank",
      type: "card",
      currency: "RUB",
      is_active: true,
      opened_at: "2026-05-19",
      created_at: "2026-05-19T00:00:00Z",
      updated_at: "2026-05-19T00:00:00Z",
    });
    mocks.createTransaction.mockResolvedValue({});
    mocks.createInterestRule.mockResolvedValue({});
  });

  it("shows account loading state and disables account-dependent quick actions", async () => {
    const user = userEvent.setup();
    mocks.accounts.mockReturnValue(new Promise(() => {}));

    renderApp();

    expect(
      await screen.findByRole("button", { name: "+ Transaction" }),
    ).toBeDisabled();
    expect(screen.getByRole("button", { name: "+ Transfer" })).toBeDisabled();

    await user.click(screen.getByRole("button", { name: /Accounts/ }));

    expect(await screen.findByText("Loading accounts")).toBeInTheDocument();
  });

  it("uses URL routes for navigation and account details", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(screen.getByRole("button", { name: /Transactions/ }));
    expect(window.location.pathname).toBe("/transactions");
    expect(await screen.findByText("No transactions yet")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Accounts/ }));
    await user.click(await screen.findByRole("button", { name: "Open" }));

    expect(window.location.pathname).toBe("/accounts/account-1");
    await waitFor(() =>
      expect(screen.getAllByText("Card").length).toBeGreaterThan(0),
    );

    await user.click(
      await screen.findByRole("button", { name: "Back to accounts" }),
    );
    expect(window.location.pathname).toBe("/accounts");
  });

  it("shows the service version badge on the dashboard", async () => {
    renderApp();

    expect(
      await screen.findByLabelText("Service version v0.5.8"),
    ).toBeInTheDocument();
  });

  it("toggles dashboard insights from the header", async () => {
    const user = userEvent.setup();
    renderApp();

    const toggle = await screen.findByRole("button", { name: "Hide insights" });
    expect(toggle).toHaveAttribute("aria-expanded", "true");

    await user.click(toggle);

    expect(
      screen.getByRole("button", { name: "Show insights" }),
    ).toHaveAttribute("aria-expanded", "false");
  });

  it("opens command menu with Ctrl+K and toggles theme", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.keyboard("{Control>}k{/Control}");
    expect(
      await screen.findByRole("dialog", { name: "Command menu" }),
    ).toBeInTheDocument();

    await user.keyboard("{Escape}");
    await user.click(
      screen.getByRole("button", { name: "Switch to dark theme" }),
    );
    await waitFor(() =>
      expect(document.documentElement).toHaveAttribute("data-theme", "dark"),
    );
    await waitFor(() =>
      expect(localStorage.getItem("capitalflow_theme")).toBe("dark"),
    );
  });

  it("opens health popover and import placeholder", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(
      await screen.findByRole("button", { name: "Check system health" }),
    );
    expect(
      await screen.findByRole("dialog", { name: "System health" }),
    ).toBeInTheDocument();
    await user.click(
      screen.getByRole("button", { name: "Close system health" }),
    );
    expect(
      screen.queryByRole("dialog", { name: "System health" }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Check system health" }),
    ).toHaveFocus();
    await waitFor(() => expect(mocks.serviceStatus).toHaveBeenCalledTimes(2));

    await user.click(screen.getByRole("button", { name: "Import" }));
    expect(
      await screen.findByRole("dialog", { name: "Import transactions" }),
    ).toBeInTheDocument();
    const importDialog = await screen.findByRole("dialog", {
      name: "Import transactions",
    });

    expect(
      within(importDialog).getByText(
        "Backend import is not available yet. Manual transactions and transfers are ready.",
      ),
    ).toBeInTheDocument();
  });

  it("uses command menu for navigation and quick actions", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(
      await screen.findByRole("button", { name: "Open command menu" }),
    );
    await user.click(
      within(
        await screen.findByRole("dialog", { name: "Command menu" }),
      ).getByRole("option", { name: /Transactions/ }),
    );
    expect(window.location.pathname).toBe("/transactions");

    await user.keyboard("{Control>}k{/Control}");
    await user.click(
      within(
        await screen.findByRole("dialog", { name: "Command menu" }),
      ).getByRole("option", { name: /Add transaction/ }),
    );
    expect(
      await screen.findByRole("dialog", { name: "Create transaction" }),
    ).toBeInTheDocument();
  });

  it("opens transaction search with Ctrl+F and shows details", async () => {
    const user = userEvent.setup();
    mocks.transactions.mockResolvedValue([
      {
        id: "transaction-1",
        account_id: "account-1",
        type: "income",
        amount: "250",
        category_id: null,
        description: "Salary",
        occurred_at: "2026-05-19",
        created_at: "2026-05-19T00:00:00Z",
      },
    ]);

    renderApp();

    await user.keyboard("{Control>}f{/Control}");
    const searchDialog = await screen.findByRole("dialog", {
      name: "Transaction search",
    });
    await user.type(
      within(searchDialog).getByPlaceholderText(
        "Search by description, category, account, amount...",
      ),
      "salary",
    );
    await user.click(within(searchDialog).getByRole("option", { name: /Salary/ }));

    expect(
      await screen.findByRole("dialog", { name: "Transaction details" }),
    ).toBeInTheDocument();
    expect(screen.getAllByText("Salary").length).toBeGreaterThan(0);
  });

  it("invalidates only targeted quick-action query keys after account creation", async () => {
    const user = userEvent.setup();
    const { queryClient } = renderApp();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

    await user.click(
      await screen.findByRole("button", { name: "Open command menu" }),
    );
    await user.click(
      within(
        await screen.findByRole("dialog", { name: "Command menu" }),
      ).getByRole("option", { name: /Create account/ }),
    );
    await user.type(await screen.findByLabelText("Name"), "Brokerage");
    await user.type(screen.getByLabelText("Bank"), "Bank");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => expect(mocks.createAccount).toHaveBeenCalled());
    await waitFor(() =>
      expect(
        screen.queryByRole("dialog", { name: "Create account" }),
      ).not.toBeInTheDocument(),
    );
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["accounts"] });
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["dashboard"] });
    expect(invalidateQueries).not.toHaveBeenCalledWith({
      queryKey: ["balance"],
    });
  });

  it("initializes route state from the URL", async () => {
    window.history.pushState({}, "", "/settings");

    renderApp();

    expect(
      await screen.findByDisplayValue("user@example.com"),
    ).toBeInTheDocument();
  });

  it("ignores malformed account route segments", async () => {
    window.history.pushState({}, "", "/accounts/%");

    expect(() => renderApp()).not.toThrow();

    await waitFor(() =>
      expect(
        screen.getAllByRole("heading", { name: "Accounts" }).length,
      ).toBeGreaterThan(0),
    );
    expect(
      screen.queryByRole("button", { name: "Back to accounts" }),
    ).not.toBeInTheDocument();
  });

  it("shows transaction dependency errors instead of empty filters", async () => {
    const user = userEvent.setup();
    mocks.categories.mockRejectedValue(new Error("Categories unavailable"));

    renderApp();

    await user.click(screen.getByRole("button", { name: /Transactions/ }));

    expect(
      await screen.findByText("Categories unavailable"),
    ).toBeInTheDocument();
    expect(screen.getAllByRole("combobox")[1]).toBeDisabled();
  });

  it("shows profile loading state on settings", async () => {
    const user = userEvent.setup();
    mocks.profile.mockReturnValue(new Promise(() => {}));

    renderApp();

    await user.click(screen.getByRole("button", { name: /Settings/ }));

    expect(await screen.findByText("Loading profile")).toBeInTheDocument();
  });
});
