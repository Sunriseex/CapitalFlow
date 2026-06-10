import { describe, expect, it } from "vitest";
import { convertAmount, formatMoney, parseMoneyToMinorResult } from "./money";

describe("parseMoneyToMinorResult", () => {
  it.each([
    ["empty optional amount", "", {}, { ok: true, value: "0" }],
    ["valid integer", "12", {}, { ok: true, value: "12" }],
    ["valid decimal", "12.34", {}, { ok: true, value: "12.34" }],
    [
      "valid KWD decimal",
      "12.345",
      { currency: "KWD" },
      { ok: true, value: "12.345" },
    ],
    [
      "valid CLF decimal",
      "12.3456",
      { currency: "CLF" },
      { ok: true, value: "12.3456" },
    ],
    [
      "valid USDT decimal",
      "12.345678",
      { currency: "USDT" },
      { ok: true, value: "12.345678" },
    ],
    ["valid JPY integer", "12", { currency: "JPY" }, { ok: true, value: "12" }],
    ["valid comma decimal", "12,3", {}, { ok: true, value: "12.3" }],
    [
      "alphabetic input",
      "abc",
      {},
      {
        ok: false,
        error: "Amount must be a number with up to 2 decimal places",
      },
    ],
    [
      "infinity",
      "Infinity",
      {},
      {
        ok: false,
        error: "Amount must be a number with up to 2 decimal places",
      },
    ],
    [
      "too many decimals",
      "1.234",
      {},
      {
        ok: false,
        error: "Amount must be a number with up to 2 decimal places",
      },
    ],
    [
      "too many KWD decimals",
      "1.2345",
      { currency: "KWD" },
      {
        ok: false,
        error: "Amount must be a number with up to 3 decimal places",
      },
    ],
    [
      "too many USDT decimals",
      "1.2345678",
      { currency: "USDT" },
      {
        ok: false,
        error: "Amount must be a number with up to 6 decimal places",
      },
    ],
    [
      "JPY decimal",
      "1.2",
      { currency: "JPY" },
      {
        ok: false,
        error: "Amount must be a number with up to 0 decimal places",
      },
    ],
    [
      "zero when positive is required",
      "0",
      { positive: true },
      { ok: false, error: "Amount must be greater than zero" },
    ],
    [
      "negative by default",
      "-1",
      {},
      { ok: false, error: "Amount must be non-negative" },
    ],
    [
      "negative when allowed",
      "-1.25",
      { allowNegative: true },
      { ok: true, value: "-1.25" },
    ],
  ])("parses %s", (_name, input, options, expected) => {
    expect(parseMoneyToMinorResult(input, options)).toEqual(expected);
  });
});

describe("convertAmount", () => {
  it("converts decimal strings without losing cents", () => {
    expect(
      convertAmount("123.45", "RUB", "USD", {
        base: "USD",
        date: "2026-05-23",
        provider: "test",
        rates: { RUB: 100 },
        fetched_at: "2026-05-23T00:00:00Z",
      }),
    ).toBe("1.23");
  });

  it("rounds half up at the currency boundary", () => {
    expect(
      convertAmount("125.00", "RUB", "USD", {
        base: "USD",
        date: "2026-05-23",
        provider: "test",
        rates: { RUB: 100 },
        fetched_at: "2026-05-23T00:00:00Z",
      }),
    ).toBe("1.25");
  });

  it("rounds converted amount to the target currency scale", () => {
    expect(
      convertAmount("1", "RUB", "KWD", {
        base: "KWD",
        date: "2026-05-23",
        provider: "test",
        rates: { RUB: 3 },
        fetched_at: "2026-05-23T00:00:00Z",
      }),
    ).toBe("0.333");
  });

  it("handles rates serialized with exponent notation", () => {
    expect(
      convertAmount("1000", "RUB", "USD", {
        base: "USD",
        date: "2026-05-23",
        provider: "test",
        rates: { RUB: 1e-7 },
        fetched_at: "2026-05-23T00:00:00Z",
      }),
    ).toBe("10000000000");
  });
});

describe("formatMoney", () => {
  it("rounds fractional digits to cents", () => {
    expect(formatMoney("1.999")).toBe("2,00\u00a0₽");
    expect(formatMoney("-1.995")).toBe("-2,00\u00a0₽");
  });

  it("uses display symbols for common currencies", () => {
    expect(formatMoney("1234.56", "RUB")).toBe("1\u00a0234,56\u00a0₽");
    expect(formatMoney("1234.56", "USD")).toBe("1\u00a0234,56\u00a0$");
    expect(formatMoney("1234.56", "EUR")).toBe("1\u00a0234,56\u00a0€");
  });

  it("uses the currency minor unit scale", () => {
    expect(formatMoney("1234.56", "JPY")).toBe("1\u00a0235\u00a0¥");
    expect(formatMoney("1.2345", "KWD")).toBe("1,235\u00a0KWD");
  });

  it("normalizes currency codes before formatting", () => {
    expect(formatMoney("10", "rub")).toBe("10,00\u00a0₽");
  });
  it("formats money with English separators and symbol position", () => {
    expect(formatMoney("1234.56", "USD", "en")).toBe("$1,234.56");
    expect(formatMoney("1234.56", "EUR", "en")).toBe("€1,234.56");
    expect(formatMoney("1234.56", "RUB", "en")).toBe("₽1,234.56");
    expect(formatMoney("1234.56", "JPY", "en")).toBe("¥1,235");
  });
});
