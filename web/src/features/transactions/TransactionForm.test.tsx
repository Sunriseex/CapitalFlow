import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Account } from "../../api/types";
import { TransactionForm } from "./TransactionForm";

const mocks = vi.hoisted(() => ({
  createTransaction: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  ApiClientError: class ApiClientError extends Error {},
  api: {
    createTransaction: mocks.createTransaction,
  },
}));

describe("TransactionForm", () => {
  it("does not call the API when amount is invalid", async () => {
    const user = userEvent.setup();
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

    render(
      <QueryClientProvider client={new QueryClient()}>
        <TransactionForm
          accounts={[account]}
          categories={[]}
          onDone={vi.fn()}
        />
      </QueryClientProvider>,
    );

    await user.type(screen.getByLabelText("Amount"), "abc");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await screen.findByText("Amount must be a number with up to 2 decimal places");
    await waitFor(() => expect(mocks.createTransaction).not.toHaveBeenCalled());
  });
});
