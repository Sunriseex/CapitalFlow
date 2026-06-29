import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type {
  DashboardCategoryLimitProgress,
  DashboardGoalProgress,
} from "../../../api/types";
import { Provider } from "../../../components/ui/provider";
import { GoalsLimitsCard } from "./GoalsLimitsCard";
import { selectDashboardProgress } from "./goalsLimits";
import { categoryColorClass } from "../../categories/categoryColor";

const goals: DashboardGoalProgress[] = [
  {
    id: "later",
    account_id: "account-1",
    name: "Later goal",
    current_amount: "350",
    target_amount: "1000",
    currency: "RUB",
    target_date: "2027-12-01",
    status: "active",
  },
  {
    id: "nearest",
    account_id: "account-1",
    name: "Nearest goal",
    current_amount: "1000",
    target_amount: "1000",
    currency: "RUB",
    target_date: "2026-12-01",
    status: "active",
  },
];

const limits: DashboardCategoryLimitProgress[] = [
  { id: "safe", category_id: "food", category_name: "Food", current_amount: "45", target_amount: "100", currency: "RUB" },
  { id: "warning", category_id: "transport", category_name: "Transport", current_amount: "83", target_amount: "100", currency: "RUB" },
  { id: "danger", category_id: "subscriptions", category_name: "Subscriptions", current_amount: "110", target_amount: "100", currency: "RUB" },
];

describe("GoalsLimitsCard", () => {
  it("selects two most urgent limits and the nearest goal", () => {
    expect(selectDashboardProgress(goals, limits).map((item) => item.data.id)).toEqual([
      "danger",
      "warning",
      "nearest",
    ]);
  });

  it("fills all rows from the available type", () => {
    expect(selectDashboardProgress([], limits).map((item) => item.data.id)).toEqual([
      "danger",
      "warning",
      "safe",
    ]);
    expect(selectDashboardProgress(goals, []).map((item) => item.data.id)).toEqual([
      "nearest",
      "later",
    ]);
  });

  it("caps accessible progress and opens goals from a row", async () => {
    const onOpen = vi.fn();
    const user = userEvent.setup();
    render(
      <Provider>
        <GoalsLimitsCard
          goals={goals}
          limits={limits}
          locale="en"
          monthLabel="June"
          title="Goals & limits"
          emptyLabel="Nothing here"
          openLabel="Open goals"
          onOpen={onOpen}
        />
      </Provider>,
    );

    expect(screen.getByRole("progressbar", { name: "Subscriptions: 110%" })).toHaveAttribute("aria-valuenow", "100");
    expect(
      screen
        .getByRole("progressbar", { name: "Subscriptions: 110%" })
        .firstElementChild,
    ).toHaveClass(categoryColorClass("subscriptions"));
    expect(screen.getByRole("progressbar", { name: "Transport: 83%" })).toHaveAttribute("aria-valuenow", "83");
    expect(screen.getByRole("progressbar", { name: "Nearest goal: 100%" })).toHaveAttribute("aria-valuenow", "100");
    await user.click(screen.getByRole("button", { name: /Subscriptions/ }));
    expect(onOpen).toHaveBeenCalledOnce();
  });
});
