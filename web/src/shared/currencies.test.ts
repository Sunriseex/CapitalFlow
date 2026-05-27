import { describe, expect, it } from "vitest";
import { currencyLabel, currencyOptions } from "./currencies";

describe("currencyLabel", () => {
  it("labels custom non-ISO currencies without Intl.DisplayNames", () => {
    expect(currencyLabel("USDT")).toBe("USDT - Tether USD");
  });
});

describe("currencyOptions", () => {
  it("includes USDT even when Intl currency values are not available", () => {
    expect(currencyOptions()).toContainEqual({ code: "USDT", label: "USDT - Tether USD" });
  });
});
