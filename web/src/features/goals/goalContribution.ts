import {
  compareMoney,
  divideMoneyByInteger,
  subtractMoney,
} from "../../api/money";

export type GoalContribution = {
  amount: string;
  months: number;
  overdue: boolean;
};

export function recommendedMonthlyContribution(
  currentAmount: string,
  targetAmount: string,
  targetDate: string | null | undefined,
  currency: string,
  now = new Date(),
): GoalContribution | null {
  if (!targetDate) return null;

  const deadline = parseDateOnly(targetDate);
  if (!deadline) return null;

  const remaining =
    compareMoney(targetAmount, currentAmount) > 0
      ? subtractMoney(targetAmount, currentAmount)
      : "0";
  const today = new Date(
    Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()),
  );
  const monthDifference =
    (deadline.getUTCFullYear() - today.getUTCFullYear()) * 12 +
    deadline.getUTCMonth() -
    today.getUTCMonth();
  const overdue = deadline.getTime() < today.getTime();
  const months = Math.max(1, monthDifference);

  return {
    amount: divideMoneyByInteger(remaining, months, currency),
    months,
    overdue,
  };
}

function parseDateOnly(value: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return null;
  const parsed = new Date(`${value}T00:00:00Z`);
  return Number.isNaN(parsed.getTime()) ? null : parsed;
}
