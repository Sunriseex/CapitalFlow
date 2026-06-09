import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PasskeysPanel } from "./PasskeysPanel";
import { I18nProvider } from "../../shared/i18n/I18nProvider";

const mocks = vi.hoisted(() => ({
  passkeys: vi.fn(),
  renamePasskey: vi.fn(),
  deletePasskey: vi.fn(),
  registerPasskey: vi.fn(),
  browserSupportsPasskeys: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  api: {
    passkeys: mocks.passkeys,
    renamePasskey: mocks.renamePasskey,
    deletePasskey: mocks.deletePasskey,
  },
}));

vi.mock("../auth/passkeys", () => ({
  browserSupportsPasskeys: mocks.browserSupportsPasskeys,
  passkeyErrorMessage: (err: unknown) =>
    err instanceof Error ? err.message : "Passkey operation failed",
  registerPasskey: mocks.registerPasskey,
}));

function renderPasskeysPanel() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <I18nProvider>
      <QueryClientProvider client={queryClient}>
        <PasskeysPanel />
      </QueryClientProvider>
    </I18nProvider>,
  );
}

describe("PasskeysPanel", () => {
  beforeEach(() => {
    localStorage.setItem("capitalflow_locale", "en");

    vi.clearAllMocks();
    mocks.browserSupportsPasskeys.mockReturnValue(true);
    mocks.passkeys.mockResolvedValue([
      {
        id: "passkey-1",
        name: "Laptop",
        backup_eligible: true,
        backup_state: true,
        last_used_at: null,
        created_at: "2026-06-04T10:00:00Z",
      },
    ]);
    mocks.registerPasskey.mockResolvedValue({});
    mocks.renamePasskey.mockResolvedValue({});
    mocks.deletePasskey.mockResolvedValue(undefined);
  });

  it("adds a passkey with password confirmation", async () => {
    const user = userEvent.setup();
    renderPasskeysPanel();

    await user.type(
      screen.getByLabelText("Password confirmation"),
      "correct password",
    );
    await user.click(screen.getByRole("button", { name: /Add passkey/ }));

    await waitFor(() =>
      expect(mocks.registerPasskey).toHaveBeenCalledWith("correct password"),
    );
  });

  it("preserves password confirmation whitespace", async () => {
    const user = userEvent.setup();
    renderPasskeysPanel();

    await user.type(
      screen.getByLabelText("Password confirmation"),
      " correct password ",
    );
    await user.click(screen.getByRole("button", { name: /Add passkey/ }));

    await waitFor(() =>
      expect(mocks.registerPasskey).toHaveBeenCalledWith(" correct password "),
    );
  });

  it("does not add a passkey without password confirmation", async () => {
    const user = userEvent.setup();
    renderPasskeysPanel();

    const button = screen.getByRole("button", { name: /Add passkey/ });
    expect(button).toBeDisabled();

    await user.click(button);

    expect(mocks.registerPasskey).not.toHaveBeenCalled();
  });

  it("renames and deletes passkeys", async () => {
    const user = userEvent.setup();
    renderPasskeysPanel();

    await user.click(
      await screen.findByRole("button", { name: "Rename passkey" }),
    );
    await user.clear(screen.getByLabelText("Passkey name"));
    await user.type(screen.getByLabelText("Passkey name"), "Phone");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(mocks.renamePasskey).toHaveBeenCalledWith("passkey-1", {
        name: "Phone",
      }),
    );

    await user.click(screen.getByRole("button", { name: "Delete passkey" }));

    await waitFor(() =>
      expect(mocks.deletePasskey).toHaveBeenCalledWith("passkey-1"),
    );
  });

  it("shows unsupported browser state", () => {
    mocks.browserSupportsPasskeys.mockReturnValue(false);

    renderPasskeysPanel();

    expect(
      screen.getByText("This browser does not support passkeys"),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Add passkey/ })).toBeDisabled();
  });
});
