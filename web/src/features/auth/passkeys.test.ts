import { describe, expect, it } from "vitest";
import { browserSupportsPasskeys, passkeyErrorMessage } from "./passkeys";

describe("passkey browser helpers", () => {
  it("reports unsupported WebAuthn in the default test browser", () => {
    expect(browserSupportsPasskeys()).toBe(false);
  });

  it("maps user cancellation to a friendly message", () => {
    expect(
      passkeyErrorMessage(new DOMException("cancelled", "NotAllowedError")),
    ).toBe("Passkey operation cancelled");
  });
  it("maps user cancellation to a localized friendly message", () => {
    expect(
      passkeyErrorMessage(new DOMException("", "NotAllowedError"), {
        operationCancelled: "Операция с passkey отменена",
        operationFailed: "Операция с passkey не выполнена",
      }),
    ).toBe("Операция с passkey отменена");
  });
});
