import type {
  DashboardCategoryLimitProgress,
  DashboardGoalProgress,
} from "../../../api/types";

export type DashboardProgressItem =
  | { kind: "goal"; data: DashboardGoalProgress }
  | { kind: "limit"; data: DashboardCategoryLimitProgress };

export function selectDashboardProgress(
  goals: DashboardGoalProgress[],
  limits: DashboardCategoryLimitProgress[],
) {
  const sortedLimits = [...limits].sort(
    (a, b) =>
      progressRatio(b.current_amount, b.target_amount) -
      progressRatio(a.current_amount, a.target_amount),
  );
  const sortedGoals = [...goals].sort((a, b) => {
    const aDate = a.target_date ?? "9999-12-31";
    const bDate = b.target_date ?? "9999-12-31";
    return (
      aDate.localeCompare(bDate) ||
      progressRatio(b.current_amount, b.target_amount) -
        progressRatio(a.current_amount, a.target_amount)
    );
  });

  const selected: DashboardProgressItem[] = [];
  if (sortedLimits.length && sortedGoals.length) {
    selected.push(
      ...sortedLimits
        .slice(0, 2)
        .map((data) => ({ kind: "limit" as const, data })),
    );
    selected.push({ kind: "goal", data: sortedGoals[0] });
  } else if (sortedLimits.length) {
    selected.push(
      ...sortedLimits
        .slice(0, 3)
        .map((data) => ({ kind: "limit" as const, data })),
    );
  } else {
    selected.push(
      ...sortedGoals
        .slice(0, 3)
        .map((data) => ({ kind: "goal" as const, data })),
    );
  }

  if (selected.length < 3 && sortedGoals.length > 1) {
    const used = new Set(
      selected.map((item) => `${item.kind}:${item.data.id}`),
    );
    for (const data of sortedGoals) {
      if (selected.length >= 3) break;
      if (!used.has(`goal:${data.id}`)) {
        selected.push({ kind: "goal", data });
      }
    }
  }
  return selected.slice(0, 3);
}

function progressRatio(current: string, target: string) {
  const targetValue = Number(target);
  return targetValue > 0 ? Number(current) / targetValue : 0;
}
