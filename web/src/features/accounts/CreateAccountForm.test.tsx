import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CreateAccountForm } from "./CreateAccountForm";
import { I18nProvider } from "../../shared/i18n/I18nProvider";

const mocks = vi.hoisted(() => ({
  createAccount: vi.fn(),
  createTransaction: vi.fn(),
  createInterestRule: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  ApiClientError: class ApiClientError extends Error {},
  api: {
    createAccount: mocks.createAccount,
    createTransaction: mocks.createTransaction,
    createInterestRule: mocks.createInterestRule,
  },
}));

describe("CreateAccountForm", () => {
  beforeEach(() => {
    localStorage.setItem("capitalflow_locale", "en");
    vi.clearAllMocks();
  });

  it("creates an account through the API client", async () => {
    const user = userEvent.setup();
    const onDone = vi.fn();
    mocks.createAccount.mockResolvedValueOnce({ id: "account-1" });

    render(
      <I18nProvider>
        <QueryClientProvider client={new QueryClient()}>
          <CreateAccountForm onDone={onDone} />
        </QueryClientProvider>
      </I18nProvider>,
    );

    await user.type(screen.getByLabelText("Card name"), "Daily card");
    await user.type(screen.getByLabelText("Bank"), "Test Bank");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() =>
      expect(mocks.createAccount).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "Daily card",
          bank: "Test Bank",
          type: "card",
          currency: "RUB",
        }),
      ),
    );
    expect(mocks.createAccount).not.toHaveBeenCalledWith(
      expect.objectContaining({
        cardLast4: expect.anything(),
        includeInBalance: expect.anything(),
        notes: expect.anything(),
      }),
    );
    expect(onDone).toHaveBeenCalled();
  });

  it("shows type cards and preserves hidden interest draft values", async () => {
    const user = userEvent.setup();

    render(
      <I18nProvider>
        <QueryClientProvider client={new QueryClient()}>
          <CreateAccountForm onDone={vi.fn()} />
        </QueryClientProvider>
      </I18nProvider>,
    );

    expect(screen.getByRole("radio", { name: /Card/ })).toBeChecked();
    expect(screen.getByLabelText("Last 4 digits")).toBeDisabled();
    expect(screen.queryByLabelText("Annual rate %")).not.toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: /Savings/ }));
    await user.type(screen.getByLabelText("Annual rate %"), "5");

    await user.click(screen.getByRole("radio", { name: /Card/ }));
    expect(screen.queryByLabelText("Annual rate %")).not.toBeInTheDocument();
    expect(
      screen.getByText(/Interest fields are hidden/),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: /Savings/ }));
    expect(screen.getByLabelText("Annual rate %")).toHaveValue("5");
  });

  it("supports native radio keyboard behavior for account type cards", async () => {
    const user = userEvent.setup();

    render(
      <I18nProvider>
        <QueryClientProvider client={new QueryClient()}>
          <CreateAccountForm onDone={vi.fn()} />
        </QueryClientProvider>
      </I18nProvider>,
    );

    await user.tab();
    expect(screen.getByRole("radio", { name: /Card/ })).toHaveFocus();

    await user.keyboard("{ArrowDown}");
    expect(screen.getByRole("radio", { name: /Cash/ })).toBeChecked();
    expect(screen.queryByLabelText("Bank")).not.toBeInTheDocument();
  });

  it("creates an interest rule only for savings and deposits", async () => {
    const user = userEvent.setup();
    const onDone = vi.fn();
    mocks.createAccount.mockResolvedValueOnce({ id: "account-1" });

    render(
      <I18nProvider>
        <QueryClientProvider client={new QueryClient()}>
          <CreateAccountForm onDone={onDone} />
        </QueryClientProvider>
      </I18nProvider>,
    );

    await user.click(screen.getByRole("radio", { name: /Savings/ }));
    await user.type(screen.getByLabelText("Name"), "Savings");
    await user.type(screen.getByLabelText("Bank"), "Test Bank");
    await user.type(screen.getByLabelText("Annual rate %"), "7.5");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() =>
      expect(mocks.createInterestRule).toHaveBeenCalledWith(
        "account-1",
        expect.objectContaining({
          annual_rate_bps: 750,
        }),
      ),
    );
    expect(onDone).toHaveBeenCalled();
  });

  it("hides bank and interest fields for cash accounts", async () => {
    const user = userEvent.setup();

    render(
      <I18nProvider>
        <QueryClientProvider client={new QueryClient()}>
          <CreateAccountForm onDone={vi.fn()} />
        </QueryClientProvider>
      </I18nProvider>,
    );

    await user.click(screen.getByRole("radio", { name: /Cash/ }));

    expect(screen.queryByLabelText("Bank")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Annual rate %")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Storage place")).toBeDisabled();
  });

  it("shows deposit-only placeholders without changing the API payload", async () => {
    const user = userEvent.setup();
    const onDone = vi.fn();
    mocks.createAccount.mockResolvedValueOnce({ id: "account-1" });

    render(
      <I18nProvider>
        <QueryClientProvider client={new QueryClient()}>
          <CreateAccountForm onDone={onDone} />
        </QueryClientProvider>
      </I18nProvider>,
    );

    await user.click(screen.getByRole("radio", { name: /Term deposit/ }));
    await user.type(screen.getByLabelText("Deposit name"), "Deposit");
    await user.type(screen.getByLabelText("Bank"), "Test Bank");
    expect(screen.getByLabelText("End date")).toBeDisabled();
    expect(screen.getByLabelText("Refill allowed")).toBeDisabled();
    expect(screen.getByLabelText("Partial withdrawal allowed")).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() =>
      expect(mocks.createAccount).toHaveBeenCalledWith({
        name: "Deposit",
        bank: "Test Bank",
        type: "term_deposit",
        currency: "RUB",
        opened_at: expect.any(String),
      }),
    );
  });

  it("does not call the API when initial balance is invalid", async () => {
    const user = userEvent.setup();

    render(
      <I18nProvider>
        <QueryClientProvider client={new QueryClient()}>
          <CreateAccountForm onDone={vi.fn()} />
        </QueryClientProvider>
      </I18nProvider>,
    );

    await user.type(screen.getByLabelText("Card name"), "Daily card");
    await user.type(screen.getByLabelText("Current balance"), "Infinity");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await screen.findByText(
      "Amount must be a number with up to 2 decimal places",
    );
    expect(screen.getByLabelText("Current balance")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    await waitFor(() => {
      expect(mocks.createAccount).not.toHaveBeenCalled();
      expect(mocks.createTransaction).not.toHaveBeenCalled();
      expect(mocks.createInterestRule).not.toHaveBeenCalled();
    });
  });

  it("links interest validation errors to their fields", async () => {
    const user = userEvent.setup();

    render(
      <I18nProvider>
        <QueryClientProvider client={new QueryClient()}>
          <CreateAccountForm onDone={vi.fn()} />
        </QueryClientProvider>
      </I18nProvider>,
    );

    await user.click(screen.getByRole("radio", { name: /Savings/ }));
    await user.type(screen.getByLabelText("Annual rate %"), "-1");
    await user.click(screen.getByRole("button", { name: "Create" }));

    const rate = screen.getByLabelText("Annual rate %");
    expect(rate).toHaveAttribute("aria-invalid", "true");
    expect(rate).toHaveAccessibleDescription(
      "Annual rate must be a non-negative number",
    );
    expect(mocks.createAccount).not.toHaveBeenCalled();
  });
});
