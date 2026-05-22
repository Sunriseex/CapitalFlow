import { describe, expect, it } from "vitest";
import { convertAmount, parseMoneyToMinorResult } from "./money";

describe("parseMoneyToMinorResult", () => {
  it.each([
    ["empty optional amount", "", {}, { ok: true, value: "0" }],
    ["valid integer", "12", {}, { ok: true, value: "12" }],
    ["valid decimal", "12.34", {}, { ok: true, value: "12.34" }],
    ["valid comma decimal", "12,3", {}, { ok: true, value: "12.3" }],
    ["alphabetic input", "abc", {}, { ok: false, error: "Amount must be a number with up to 2 decimal places" }],
    ["infinity", "Infinity", {}, { ok: false, error: "Amount must be a number with up to 2 decimal places" }],
    ["too many decimals", "1.234", {}, { ok: false, error: "Amount must be a number with up to 2 decimal places" }],
    ["zero when positive is required", "0", { positive: true }, { ok: false, error: "Amount must be greater than zero" }],
    ["negative by default", "-1", {}, { ok: false, error: "Amount must be non-negative" }],
    ["negative when allowed", "-1.25", { allowNegative: true }, { ok: true, value: "-1.25" }],
  ])("parses %s", (_name, input, options, expected) => {
    expect(parseMoneyToMinorResult(input, options)).toEqual(expected);
  });
});

describe("convertAmount", () => {
  it("converts decimal strings without losing cents", () => {
    expect(convertAmount("123.45", "RUB", "USD", {
      base: "USD",
      date: "2026-05-23",
      provider: "test",
      rates: { RUB: 100 },
      fetched_at: "2026-05-23T00:00:00Z",
    })).toBe("1.23");
  });

  it("rounds half up at the currency boundary", () => {
    expect(convertAmount("125.00", "RUB", "USD", {
      base: "USD",
      date: "2026-05-23",
      provider: "test",
      rates: { RUB: 100 },
      fetched_at: "2026-05-23T00:00:00Z",
    })).toBe("1.25");
  });
});


