import type { ComponentProps } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Account, Category } from "../../api/types";
import { TransactionForm } from "./TransactionForm";
import { I18nProvider } from "../../shared/i18n/I18nProvider";

const mocks = vi.hoisted(() => ({
  createTransaction: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  ApiClientError: class ApiClientError extends Error {},
  api: {
    createTransaction: mocks.createTransaction,
  },
}));

const account: Account = {
  id: "account-1",
  name: "Card",
  type: "card",
  currency: "RUB",
  is_active: true,
  opened_at: "2026-05-17",
  created_at: "2026-05-17T00:00:00Z",
  updated_at: "2026-05-17T00:00:00Z",
};

const categories: Category[] = [
  {
    id: "category-subscriptions",
    slug: "subscriptions",
    name: "Subscriptions",
    created_at: "2026-05-17T00:00:00Z",
    updated_at: "2026-05-17T00:00:00Z",
  },
  {
    id: "category-groceries",
    slug: "groceries",
    name: "Groceries",
    created_at: "2026-05-17T00:00:00Z",
    updated_at: "2026-05-17T00:00:00Z",
  },
];

function renderTransactionForm(
  props: Partial<ComponentProps<typeof TransactionForm>> = {},
) {
  const onDone = vi.fn();

  render(
    <I18nProvider>
      <QueryClientProvider client={new QueryClient()}>
        <TransactionForm
          accounts={[account]}
          categories={categories}
          onDone={onDone}
          {...props}
        />
      </QueryClientProvider>
    </I18nProvider>,
  );

  return { onDone };
}

describe("TransactionForm", () => {
  beforeEach(() => {
    localStorage.setItem("capitalflow_locale", "en");

    mocks.createTransaction.mockReset();
    mocks.createTransaction.mockResolvedValue({});
  });

  it("does not call the API when amount is invalid", async () => {
    const user = userEvent.setup();

    renderTransactionForm();

    await user.type(screen.getByLabelText("Amount"), "abc");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await screen.findByText(
      "Amount must be a number with up to 2 decimal places",
    );
    await waitFor(() => expect(mocks.createTransaction).not.toHaveBeenCalled());
  });

  it("allows negative adjustment amounts", async () => {
    const user = userEvent.setup();
    const { onDone } = renderTransactionForm({ fixedType: "adjustment" });

    await user.type(screen.getByLabelText("Amount"), "-10");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() =>
      expect(mocks.createTransaction).toHaveBeenCalledWith(
        expect.objectContaining({
          account_id: "account-1",
          type: "adjustment",
          amount: "-10",
        }),
      ),
    );
    await waitFor(() => expect(onDone).toHaveBeenCalled());
  });

  it("rejects negative non-adjustment amounts", async () => {
    const user = userEvent.setup();
    renderTransactionForm();

    await user.type(screen.getByLabelText("Amount"), "-10");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await screen.findByText("Amount must be non-negative");
    await waitFor(() => expect(mocks.createTransaction).not.toHaveBeenCalled());
  });

  it("selects a category from the command picker", async () => {
    const user = userEvent.setup();
    const { onDone } = renderTransactionForm();

    await user.type(screen.getByLabelText("Amount"), "25");
    await user.click(screen.getByRole("button", { name: /Open category picker/ }));

    expect(
      await screen.findByRole("dialog", { name: "Categories" }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("option", { name: /Groceries/ }));
    expect(screen.queryByRole("dialog", { name: "Categories" })).toBeNull();
    expect(
      screen.getByRole("button", { name: /Groceries/ }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() =>
      expect(mocks.createTransaction).toHaveBeenCalledWith(
        expect.objectContaining({
          category_id: "category-groceries",
        }),
      ),
    );
    await waitFor(() => expect(onDone).toHaveBeenCalled());
  });

  it("shows a non-blocking subscription suggestion for subscription expenses", async () => {
    const user = userEvent.setup();
    renderTransactionForm();

    await user.selectOptions(screen.getByLabelText("Type"), "expense");
    await user.click(screen.getByRole("button", { name: /Open category picker/ }));
    await user.click(screen.getByRole("option", { name: /Subscriptions/ }));

    expect(
      screen.getByText("This looks like a regular payment."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create subscription" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Link" })).toBeDisabled();

    await user.click(screen.getByRole("button", { name: "Not now" }));

    expect(
      screen.queryByText("This looks like a regular payment."),
    ).not.toBeInTheDocument();
  });
});
