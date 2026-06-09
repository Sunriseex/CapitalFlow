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
        ,
      </I18nProvider>,
    );

    await user.type(screen.getByLabelText("Name"), "Daily card");
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
    expect(onDone).toHaveBeenCalled();
  });

  it("does not call the API when initial balance is invalid", async () => {
    const user = userEvent.setup();

    render(
      <I18nProvider>
        <QueryClientProvider client={new QueryClient()}>
          <CreateAccountForm onDone={vi.fn()} />
        </QueryClientProvider>
        ,
      </I18nProvider>,
    );

    await user.type(screen.getByLabelText("Name"), "Daily card");
    await user.type(screen.getByLabelText("Initial balance"), "Infinity");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await screen.findByText(
      "Amount must be a number with up to 2 decimal places",
    );
    await waitFor(() => {
      expect(mocks.createAccount).not.toHaveBeenCalled();
      expect(mocks.createTransaction).not.toHaveBeenCalled();
      expect(mocks.createInterestRule).not.toHaveBeenCalled();
    });
  });
});
