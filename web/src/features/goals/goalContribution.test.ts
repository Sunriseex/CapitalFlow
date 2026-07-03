import { describe, expect, it } from "vitest";
import { recommendedMonthlyContribution } from "./goalContribution";

describe("recommendedMonthlyContribution", () => {
  const now = new Date("2026-06-28T12:00:00Z");

  it("returns no recommendation without a deadline", () => {
    expect(
      recommendedMonthlyContribution("200", "1000", null, "RUB", now),
    ).toBeNull();
  });

  it("splits the remaining amount across calendar months", () => {
    expect(
      recommendedMonthlyContribution(
        "210000",
        "300000",
        "2026-09-30",
        "RUB",
        now,
      ),
    ).toEqual({ amount: "30000", months: 3, overdue: false });
  });

  it("rounds to the currency scale", () => {
    expect(
      recommendedMonthlyContribution("0", "100", "2026-09-01", "USD", now),
    ).toEqual({ amount: "33.33", months: 3, overdue: false });
  });

  it("shows zero for a funded goal", () => {
    expect(
      recommendedMonthlyContribution("1200", "1000", "2026-09-01", "RUB", now),
    ).toEqual({ amount: "0", months: 3, overdue: false });
  });

  it("marks a missed deadline and recommends the full remainder", () => {
    expect(
      recommendedMonthlyContribution("250", "1000", "2026-05-31", "RUB", now),
    ).toEqual({ amount: "750", months: 1, overdue: true });
  });
});
