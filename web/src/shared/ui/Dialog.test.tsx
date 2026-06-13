import { useState } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it } from "vitest";
import { I18nProvider } from "../i18n/I18nProvider";
import { Button, Dialog, Input } from ".";

function renderDialogHarness() {
  return render(
    <I18nProvider>
      <DialogHarness />
    </I18nProvider>,
  );
}

function DialogHarness() {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button onClick={() => setOpen(true)}>Open dialog</Button>
      {open ? (
        <Dialog title="Edit account" onClose={() => setOpen(false)}>
          <label>
            Name
            <Input defaultValue="Savings" />
          </label>
          <Button>Save</Button>
        </Dialog>
      ) : null}
    </>
  );
}

function renderNarrowDialog() {
  return render(
    <I18nProvider>
      <Dialog
        title="Transaction details"
        onClose={() => undefined}
        variant="narrow"
      >
        <p>Details</p>
      </Dialog>
    </I18nProvider>,
  );
}

describe("Dialog", () => {
  beforeEach(() => {
    localStorage.setItem("capitalflow_locale", "en");
  });

  it("sets dialog semantics, closes on Escape, and restores focus", async () => {
    const user = userEvent.setup();
    renderDialogHarness();

    const opener = screen.getByRole("button", { name: "Open dialog" });
    await user.click(opener);

    const dialog = screen.getByRole("dialog", { name: "Edit account" });
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(screen.getByRole("button", { name: "Close dialog" })).toHaveFocus();

    await user.keyboard("{Escape}");

    expect(
      screen.queryByRole("dialog", { name: "Edit account" }),
    ).not.toBeInTheDocument();
    expect(opener).toHaveFocus();
  });

  it("keeps tab focus inside the dialog", async () => {
    const user = userEvent.setup();
    renderDialogHarness();
    await user.click(screen.getByRole("button", { name: "Open dialog" }));
    await user.tab({ shift: true });

    expect(screen.getByRole("button", { name: "Save" })).toHaveFocus();
  });

  it("restores focus when closed by the close button", async () => {
    const user = userEvent.setup();
    renderDialogHarness();
    const opener = screen.getByRole("button", { name: "Open dialog" });
    await user.click(opener);
    await user.click(screen.getByRole("button", { name: "Close dialog" }));

    expect(
      screen.queryByRole("dialog", { name: "Edit account" }),
    ).not.toBeInTheDocument();
    expect(opener).toHaveFocus();
  });

  it("supports a narrow panel variant", () => {
    renderNarrowDialog();

    expect(
      screen.getByRole("dialog", { name: "Transaction details" }),
    ).toHaveClass("dialog-panel-narrow");
  });
});
