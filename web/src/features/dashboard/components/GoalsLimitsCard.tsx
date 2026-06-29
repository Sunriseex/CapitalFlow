import { Target } from "lucide-react";
import type {
  DashboardCategoryLimitProgress,
  DashboardGoalProgress,
} from "../../../api/types";
import { formatMoney } from "../../../api/money";
import type { Locale } from "../../../shared/i18n/i18n";
import { Button } from "../../../components/ui/button";
import {
  selectDashboardProgress,
  type DashboardProgressItem,
} from "./goalsLimits";
import { categoryColorClass } from "../../categories/categoryColor";

export function GoalsLimitsCard({
  goals,
  limits,
  locale,
  monthLabel,
  title,
  emptyLabel,
  openLabel,
  onOpen,
}: {
  goals: DashboardGoalProgress[];
  limits: DashboardCategoryLimitProgress[];
  locale: Locale;
  monthLabel: string;
  title: string;
  emptyLabel: string;
  openLabel: string;
  onOpen: () => void;
}) {
  const items = selectDashboardProgress(goals, limits);

  return (
    <article className="card rail-card goals-limits-card">
      <div className="card-head">
        <div className="card-title">
          <h2>{title}</h2>
          <span>{monthLabel}</span>
        </div>
        <Target aria-hidden="true" />
      </div>

      {items.length ? (
        <ul className="budget-list">
          {items.map((item) => (
            <ProgressRow
              key={`${item.kind}:${item.data.id}`}
              item={item}
              locale={locale}
              onOpen={onOpen}
            />
          ))}
        </ul>
      ) : (
        <div className="goals-limits-empty">
          <span>{emptyLabel}</span>
          <Button type="button" variant="outline" onClick={onOpen}>
            {openLabel}
          </Button>
        </div>
      )}
    </article>
  );
}

function ProgressRow({
  item,
  locale,
  onOpen,
}: {
  item: DashboardProgressItem;
  locale: Locale;
  onOpen: () => void;
}) {
  const data = item.data;
  const current = Number(data.current_amount);
  const target = Number(data.target_amount);
  const percent = target > 0 ? Math.max(0, (current / target) * 100) : 0;
  const boundedPercent = Math.min(100, Math.round(percent));
  const tone = progressTone(item.kind, percent);
  const name =
    item.kind === "goal" ? item.data.name : item.data.category_name;

  return (
    <li className="budget-item">
      <button className="budget-item-trigger" type="button" onClick={onOpen}>
        <span className="item-row">
          <span className="item-name">{name}</span>
          <span className="item-meta">
            {formatMoney(data.current_amount, data.currency, locale)} /{" "}
            {formatMoney(data.target_amount, data.currency, locale)}
          </span>
        </span>
        <span
          className="budget-progress"
          role="progressbar"
          aria-label={`${name}: ${Math.round(percent)}%`}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={boundedPercent}
        >
          <span
            className={
              item.kind === "limit"
                ? `budget-progress-bar category-progress ${categoryColorClass(item.data.category_id)}`
                : `budget-progress-bar is-${tone}`
            }
            style={{ width: `${boundedPercent}%` }}
          />
        </span>
      </button>
    </li>
  );
}

function progressTone(kind: DashboardProgressItem["kind"], percent: number) {
  if (kind === "goal") return percent >= 100 ? "success" : "default";
  if (percent >= 100) return "danger";
  if (percent >= 80) return "warning";
  return "default";
}
