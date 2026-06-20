import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Account } from "../../api/types";
import { I18nProvider } from "../../shared/i18n/I18nProvider";
import { EditAccountForm } from "./EditAccountForm";

const mocks = vi.hoisted(() => ({
  updateAccount: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  ApiClientError: class ApiClientError extends Error {},
  api: {
    updateAccount: mocks.updateAccount,
  },
}));

const account: Account = {
  id: "account-1",
  name: "Daily card",
  bank: "Test Bank",
  type: "card",
  currency: "RUB",
  is_active: true,
  opened_at: "2026-05-17",
  created_at: "2026-05-17T00:00:00Z",
  updated_at: "2026-05-17T00:00:00Z",
};

function renderEditAccountForm(onDone = vi.fn()) {
  render(
    <I18nProvider>
      <QueryClientProvider client={new QueryClient()}>
        <EditAccountForm account={account} onDone={onDone} />
      </QueryClientProvider>
    </I18nProvider>,
  );

  return { onDone };
}

describe("EditAccountForm", () => {
  beforeEach(() => {
    localStorage.setItem("capitalflow_locale", "en");

    vi.clearAllMocks();
    mocks.updateAccount.mockResolvedValue(account);
  });

  it("submits the edited account payload", async () => {
    const user = userEvent.setup();
    const { onDone } = renderEditAccountForm();

    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "Travel card");
    await user.clear(screen.getByLabelText("Bank"));
    await user.type(screen.getByLabelText("Bank"), "Capital Bank");
    await user.selectOptions(screen.getByLabelText("Type"), "savings");
    await user.selectOptions(screen.getByLabelText("Currency"), "USD");
    await user.clear(screen.getByLabelText("Opened"));
    await user.type(screen.getByLabelText("Opened"), "2026-06-01");
    await user.click(screen.getByRole("checkbox", { name: /active/i }));

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(mocks.updateAccount).toHaveBeenCalledWith("account-1", {
        name: "Travel card",
        bank: "Capital Bank",
        type: "savings",
        currency: "USD",
        opened_at: "2026-06-01",
        is_active: false,
      }),
    );
    expect(onDone).toHaveBeenCalled();
  });
});
