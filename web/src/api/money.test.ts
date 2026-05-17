import { describe, expect, it } from "vitest";
import { parseMoneyToMinorResult } from "./money";

describe("parseMoneyToMinorResult", () => {
  it.each([
    ["empty optional amount", "", {}, { ok: true, value: 0 }],
    ["valid integer", "12", {}, { ok: true, value: 1200 }],
    ["valid decimal", "12.34", {}, { ok: true, value: 1234 }],
    ["valid comma decimal", "12,3", {}, { ok: true, value: 1230 }],
    ["alphabetic input", "abc", {}, { ok: false, error: "Amount must be a number with up to 2 decimal places" }],
    ["infinity", "Infinity", {}, { ok: false, error: "Amount must be a number with up to 2 decimal places" }],
    ["too many decimals", "1.234", {}, { ok: false, error: "Amount must be a number with up to 2 decimal places" }],
    ["zero when positive is required", "0", { positive: true }, { ok: false, error: "Amount must be greater than zero" }],
    ["negative by default", "-1", {}, { ok: false, error: "Amount must be non-negative" }],
    ["negative when allowed", "-1.25", { allowNegative: true }, { ok: true, value: -125 }],
  ])("parses %s", (_name, input, options, expected) => {
    expect(parseMoneyToMinorResult(input, options)).toEqual(expected);
  });
});
