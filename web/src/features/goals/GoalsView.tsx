import { useMemo, useState } from "react";
import type { ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Gauge, Target } from "lucide-react";
import { api } from "../../api/client";
import { formatMoney } from "../../api/money";
import type {
  Account,
  Category,
  CategoryLimit,
  FinancialGoal,
} from "../../api/types";
import { apiErrorMessages, errorMessage } from "../../shared/api/query";
import { useI18n } from "../../shared/i18n/useI18n";
import {
  Empty,
  EmptyState,
  Panel,
  PrimitiveButton as Button,
  PrimitiveInput as Input,
  Select,
} from "../../shared/ui";
import { recommendedMonthlyContribution } from "./goalContribution";
import { categoryColorClass } from "../categories/categoryColor";
import { CategoryBadge } from "../transactions/components/CategoryBadge";

type GoalDraft = {
  id: string;
  accountID: string;
  name: string;
  amount: string;
  targetDate: string;
  status: FinancialGoal["status"];
};

type LimitDraft = {
  id: string;
  categoryID: string;
  amount: string;
  currency: string;
  isActive: boolean;
};

type GoalsSection = "goals" | "limits";

export function GoalsView({
  accounts,
  categories,
  primaryCurrency,
}: {
  accounts: Account[];
  categories: Category[];
  primaryCurrency: string;
}) {
  const { locale, t } = useI18n();
  const queryClient = useQueryClient();
  const errors = apiErrorMessages(t);
  const [goalFormOpen, setGoalFormOpen] = useState(false);
  const [limitFormOpen, setLimitFormOpen] = useState(false);
  const [goalName, setGoalName] = useState("");
  const [goalAmount, setGoalAmount] = useState("");
  const [goalAccountID, setGoalAccountID] = useState("");
  const [targetDate, setTargetDate] = useState("");
  const [limitCategoryID, setLimitCategoryID] = useState("");
  const [limitAmount, setLimitAmount] = useState("");
  const [limitCurrency, setLimitCurrency] = useState(primaryCurrency);
  const [goalDraft, setGoalDraft] = useState<GoalDraft | null>(null);
  const [limitDraft, setLimitDraft] = useState<LimitDraft | null>(null);
  const [activeSection, setActiveSection] = useState<GoalsSection>("goals");

  const goals = useQuery({
    queryKey: ["financial-goals"],
    queryFn: api.financialGoals,
  });
  const limits = useQuery({
    queryKey: ["category-limits"],
    queryFn: api.categoryLimits,
  });
  const summary = useQuery({
    queryKey: ["dashboard", "summary"],
    queryFn: api.dashboardSummary,
  });
  const accountNames = useMemo(
    () => new Map(accounts.map((account) => [account.id, account.name])),
    [accounts],
  );
  const categoryNames = useMemo(
    () => new Map(categories.map((category) => [category.id, category.name])),
    [categories],
  );
  const goalProgress = useMemo(
    () =>
      new Map(
        (summary.data?.financial_goals ?? []).map((goal) => [goal.id, goal]),
      ),
    [summary.data?.financial_goals],
  );
  const limitProgress = useMemo(
    () =>
      new Map(
        (summary.data?.category_limits ?? []).map((limit) => [limit.id, limit]),
      ),
    [summary.data?.category_limits],
  );

  const createGoal = useMutation({
    mutationFn: api.createFinancialGoal,
    onSuccess: async () => {
      setGoalName("");
      setGoalAmount("");
      setGoalAccountID("");
      setTargetDate("");
      setGoalFormOpen(false);
      await refreshGoalData(queryClient);
    },
  });
  const updateGoal = useMutation({
    mutationFn: ({
      id,
      input,
    }: {
      id: string;
      input: Parameters<typeof api.updateFinancialGoal>[1];
    }) => api.updateFinancialGoal(id, input),
    onSuccess: async () => refreshGoalData(queryClient),
  });
  const createLimit = useMutation({
    mutationFn: api.createCategoryLimit,
    onSuccess: async () => {
      setLimitCategoryID("");
      setLimitAmount("");
      setLimitFormOpen(false);
      await refreshGoalData(queryClient);
    },
  });
  const updateLimit = useMutation({
    mutationFn: ({
      id,
      input,
    }: {
      id: string;
      input: Parameters<typeof api.updateCategoryLimit>[1];
    }) => api.updateCategoryLimit(id, input),
    onSuccess: async () => refreshGoalData(queryClient),
  });
  const mutationError =
    createGoal.error ??
    updateGoal.error ??
    createLimit.error ??
    updateLimit.error;
  const queryError = goals.error ?? limits.error ?? summary.error;
  const loading = goals.isLoading || limits.isLoading || summary.isLoading;

  return (
    <Panel className="workspace-panel goals-panel" title={t.goals.title}>
      {mutationError || queryError ? (
        <div className="error inline-error" role="alert">
          {errorMessage(mutationError ?? queryError, errors)}
        </div>
      ) : null}
      {loading ? <Empty>{t.goals.loading}</Empty> : null}

      <div
        className="goals-workspace-tabs"
        role="tablist"
        aria-label={t.goals.sections}
      >
        <Button
          id="goals-tab"
          className={
            activeSection === "goals"
              ? "goals-workspace-tab is-active"
              : "goals-workspace-tab"
          }
          type="button"
          variant="ghost"
          role="tab"
          aria-selected={activeSection === "goals"}
          aria-controls="goals-panel"
          onClick={() => {
            setActiveSection("goals");
            setLimitDraft(null);
          }}
        >
          <Target aria-hidden="true" />
          <span>{t.goals.savingsGoals}</span>
          <span className="goals-tab-count">{goals.data?.length ?? 0}</span>
        </Button>
        <Button
          id="limits-tab"
          className={
            activeSection === "limits"
              ? "goals-workspace-tab is-active"
              : "goals-workspace-tab"
          }
          type="button"
          variant="ghost"
          role="tab"
          aria-selected={activeSection === "limits"}
          aria-controls="limits-panel"
          onClick={() => {
            setActiveSection("limits");
            setGoalDraft(null);
          }}
        >
          <Gauge aria-hidden="true" />
          <span>{t.goals.monthlyLimits}</span>
          <span className="goals-tab-count">{limits.data?.length ?? 0}</span>
        </Button>
      </div>

      <section
        id="goals-panel"
        className="goals-management-section"
        role="tabpanel"
        aria-labelledby="goals-tab"
        hidden={activeSection !== "goals"}
      >
        <div className="goals-section-head">
          <div>
            <h2 id="savings-goals-title">{t.goals.savingsGoals}</h2>
            <p>{t.goals.savingsGoalsDescription}</p>
          </div>
          <Button
            type="button"
            onClick={() => setGoalFormOpen((open) => !open)}
          >
            {goalFormOpen ? t.common.cancel : t.goals.create}
          </Button>
        </div>

        {goalFormOpen ? (
          <form
            className="goal-create-form"
            onSubmit={(event) => {
              event.preventDefault();
              if (!goalName.trim() || !goalAmount || !goalAccountID) return;
              createGoal.mutate({
                account_id: goalAccountID,
                name: goalName.trim(),
                target_amount: goalAmount,
                ...(targetDate ? { target_date: targetDate } : {}),
              });
            }}
          >
            <Field label={t.goals.name} htmlFor="goal-name">
              <Input
                id="goal-name"
                value={goalName}
                maxLength={100}
                onChange={(event) => setGoalName(event.target.value)}
              />
            </Field>
            <AccountField
              id="goal-account"
              label={t.goals.account}
              chooseLabel={t.goals.chooseAccount}
              accounts={accounts}
              value={goalAccountID}
              onChange={setGoalAccountID}
            />
            <Field label={t.goals.targetAmount} htmlFor="goal-amount">
              <Input
                id="goal-amount"
                value={goalAmount}
                type="number"
                min="0.01"
                step="0.01"
                onChange={(event) => setGoalAmount(event.target.value)}
              />
            </Field>
            <Field label={t.goals.targetDate} htmlFor="goal-date">
              <Input
                id="goal-date"
                value={targetDate}
                type="date"
                onChange={(event) => setTargetDate(event.target.value)}
              />
            </Field>
            <Button
              type="submit"
              disabled={
                !goalName.trim() ||
                !goalAmount ||
                !goalAccountID ||
                createGoal.isPending
              }
            >
              {createGoal.isPending ? t.goals.creating : t.goals.save}
            </Button>
          </form>
        ) : null}

        {!loading && !goals.data?.length ? (
          <EmptyState
            icon={<Target aria-hidden="true" />}
            title={t.goals.emptyTitle}
            description={t.goals.emptyDescription}
            primaryAction={{
              label: t.goals.create,
              onClick: () => setGoalFormOpen(true),
            }}
          />
        ) : null}
        <ul className="goals-list management-list">
          {goals.data?.map((goal) => {
            const progress = goalProgress.get(goal.id);
            const current = progress?.current_amount ?? "0";
            const percent = ratio(current, goal.target_amount);
            const contribution =
              goal.status === "active"
                ? recommendedMonthlyContribution(
                    current,
                    goal.target_amount,
                    goal.target_date,
                    goal.currency,
                  )
                : null;
            const editing = goalDraft?.id === goal.id;
            return (
              <li key={goal.id}>
                <div className="management-item-main">
                  <div className="management-item-copy">
                    <strong>{goal.name}</strong>
                    <div className="management-meta">
                      <span>
                        {goal.account_id
                          ? (accountNames.get(goal.account_id) ??
                            t.goals.accountNotLinked)
                          : t.goals.accountNotLinked}
                      </span>
                      <span>
                        {goal.target_date
                          ? formatGoalDate(goal.target_date, locale)
                          : t.goals.noDeadline}
                      </span>
                      <StatusLabel status={goal.status} labels={t.goals} />
                    </div>
                  </div>
                  <strong className="goal-target">
                    {formatMoney(current, goal.currency, locale)} /{" "}
                    {formatMoney(goal.target_amount, goal.currency, locale)}
                  </strong>
                </div>
                <Progress
                  value={percent}
                  label={`${goal.name}: ${Math.round(percent)}%`}
                  tone={percent >= 100 ? "success" : "default"}
                />
                {contribution ? (
                  <div className="goal-contribution">
                    <span>
                      {contribution.overdue
                        ? t.goals.deadlinePassed
                        : t.goals.monthlyContribution}
                    </span>
                    <strong>
                      {formatMoney(contribution.amount, goal.currency, locale)}
                      {contribution.overdue ? null : ` / ${t.goals.month}`}
                    </strong>
                  </div>
                ) : null}

                {editing ? (
                  <form
                    className="management-edit-form goal-edit-form"
                    aria-label={`${t.goals.edit}: ${goal.name}`}
                    onSubmit={(event) => {
                      event.preventDefault();
                      if (
                        !goalDraft.name.trim() ||
                        !goalDraft.amount ||
                        !goalDraft.accountID
                      ) {
                        return;
                      }
                      updateGoal.mutate(
                        {
                          id: goal.id,
                          input: {
                            account_id: goalDraft.accountID,
                            name: goalDraft.name.trim(),
                            target_amount: goalDraft.amount,
                            target_date: goalDraft.targetDate,
                            status: goalDraft.status,
                          },
                        },
                        { onSuccess: () => setGoalDraft(null) },
                      );
                    }}
                  >
                    <Field
                      label={t.goals.name}
                      htmlFor={`goal-edit-name-${goal.id}`}
                    >
                      <Input
                        id={`goal-edit-name-${goal.id}`}
                        value={goalDraft.name}
                        maxLength={100}
                        onChange={(event) =>
                          setGoalDraft({
                            ...goalDraft,
                            name: event.target.value,
                          })
                        }
                      />
                    </Field>
                    <AccountField
                      id={`goal-edit-account-${goal.id}`}
                      label={t.goals.account}
                      chooseLabel={t.goals.chooseAccount}
                      accounts={accounts}
                      value={goalDraft.accountID}
                      onChange={(accountID) =>
                        setGoalDraft({ ...goalDraft, accountID })
                      }
                    />
                    <Field
                      label={t.goals.targetAmount}
                      htmlFor={`goal-edit-amount-${goal.id}`}
                    >
                      <Input
                        id={`goal-edit-amount-${goal.id}`}
                        value={goalDraft.amount}
                        type="number"
                        min="0.01"
                        step="0.01"
                        onChange={(event) =>
                          setGoalDraft({
                            ...goalDraft,
                            amount: event.target.value,
                          })
                        }
                      />
                    </Field>
                    <Field
                      label={t.goals.targetDate}
                      htmlFor={`goal-edit-date-${goal.id}`}
                    >
                      <Input
                        id={`goal-edit-date-${goal.id}`}
                        value={goalDraft.targetDate}
                        type="date"
                        onChange={(event) =>
                          setGoalDraft({
                            ...goalDraft,
                            targetDate: event.target.value,
                          })
                        }
                      />
                    </Field>
                    <Field
                      label={t.goals.status}
                      htmlFor={`goal-edit-status-${goal.id}`}
                    >
                      <Select
                        id={`goal-edit-status-${goal.id}`}
                        value={goalDraft.status}
                        onChange={(event) =>
                          setGoalDraft({
                            ...goalDraft,
                            status: event.target
                              .value as FinancialGoal["status"],
                          })
                        }
                      >
                        <option value="active">{t.goals.statusActive}</option>
                        <option value="completed">
                          {t.goals.statusCompleted}
                        </option>
                        <option value="archived">
                          {t.goals.statusArchived}
                        </option>
                      </Select>
                    </Field>
                    <div className="management-form-actions">
                      <Button
                        type="button"
                        variant="ghost"
                        onClick={() => setGoalDraft(null)}
                      >
                        {t.common.cancel}
                      </Button>
                      <Button type="submit" disabled={updateGoal.isPending}>
                        {t.goals.saveChanges}
                      </Button>
                    </div>
                  </form>
                ) : (
                  <div className="management-actions">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => {
                        setLimitDraft(null);
                        setGoalDraft(goalToDraft(goal));
                      }}
                    >
                      {goal.account_id ? t.goals.edit : t.goals.linkAccount}
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      disabled={updateGoal.isPending}
                      onClick={() =>
                        updateGoal.mutate({
                          id: goal.id,
                          input: {
                            status:
                              goal.status === "archived"
                                ? "active"
                                : "archived",
                          },
                        })
                      }
                    >
                      {goal.status === "archived"
                        ? t.goals.activate
                        : t.goals.archive}
                    </Button>
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      </section>

      <section
        id="limits-panel"
        className="goals-management-section"
        role="tabpanel"
        aria-labelledby="limits-tab"
        hidden={activeSection !== "limits"}
      >
        <div className="goals-section-head">
          <div>
            <h2 id="category-limits-title">{t.goals.monthlyLimits}</h2>
            <p>{t.goals.monthlyLimitsDescription}</p>
          </div>
          <Button
            type="button"
            variant="outline"
            onClick={() => setLimitFormOpen((open) => !open)}
          >
            {limitFormOpen ? t.common.cancel : t.goals.createLimit}
          </Button>
        </div>
        {limitFormOpen ? (
          <form
            className="goal-create-form limit-create-form"
            onSubmit={(event) => {
              event.preventDefault();
              if (!limitCategoryID || !limitAmount || !limitCurrency) return;
              createLimit.mutate({
                category_id: limitCategoryID,
                amount: limitAmount,
                currency: limitCurrency,
              });
            }}
          >
            <CategoryField
              id="limit-category"
              label={t.transactions.category}
              chooseLabel={t.goals.chooseCategory}
              categories={categories}
              value={limitCategoryID}
              onChange={setLimitCategoryID}
            />
            <Field label={t.goals.limitAmount} htmlFor="limit-amount">
              <Input
                id="limit-amount"
                value={limitAmount}
                type="number"
                min="0.01"
                step="0.01"
                onChange={(event) => setLimitAmount(event.target.value)}
              />
            </Field>
            <Field label={t.goals.currency} htmlFor="limit-currency">
              <Input
                id="limit-currency"
                value={limitCurrency}
                maxLength={3}
                onChange={(event) =>
                  setLimitCurrency(event.target.value.toUpperCase())
                }
              />
            </Field>
            <Button
              type="submit"
              disabled={
                !limitCategoryID ||
                !limitAmount ||
                !limitCurrency ||
                createLimit.isPending
              }
            >
              {createLimit.isPending ? t.goals.creating : t.goals.saveLimit}
            </Button>
          </form>
        ) : null}
        {!loading && !limits.data?.length ? (
          <EmptyState
            icon={<Gauge aria-hidden="true" />}
            title={t.goals.noLimits}
            description={t.goals.noLimitsDescription}
            primaryAction={{
              label: t.goals.createLimit,
              onClick: () => setLimitFormOpen(true),
            }}
          />
        ) : null}
        <ul className="goals-list management-list">
          {limits.data?.map((limit) => {
            const progress = limitProgress.get(limit.id);
            const current = progress?.current_amount ?? "0";
            const percent = ratio(current, limit.amount);
            const editing = limitDraft?.id === limit.id;
            const categoryName =
              categoryNames.get(limit.category_id) ?? t.goals.unknownCategory;
            return (
              <li key={limit.id}>
                <div className="management-item-main">
                  <div className="management-item-copy">
                    <CategoryBadge
                      categoryKey={limit.category_id}
                      name={categoryName}
                    />
                    <div className="management-meta">
                      <span>{limit.currency}</span>
                      <span>{t.goals.repeatsMonthly}</span>
                      <span
                        className={`management-status is-${limit.is_active ? "active" : "archived"}`}
                      >
                        {limit.is_active
                          ? t.goals.statusActive
                          : t.goals.inactive}
                      </span>
                    </div>
                  </div>
                  <strong className="goal-target">
                    {formatMoney(current, limit.currency, locale)} /{" "}
                    {formatMoney(limit.amount, limit.currency, locale)}
                  </strong>
                </div>
                <Progress
                  value={percent}
                  label={`${categoryName}: ${Math.round(percent)}%`}
                  tone={
                    percent >= 100
                      ? "danger"
                      : percent >= 80
                        ? "warning"
                        : "default"
                  }
                  categoryKey={limit.category_id}
                />

                {editing ? (
                  <form
                    className="management-edit-form limit-edit-form"
                    aria-label={`${t.goals.edit}: ${categoryName}`}
                    onSubmit={(event) => {
                      event.preventDefault();
                      if (
                        !limitDraft.categoryID ||
                        !limitDraft.amount ||
                        !limitDraft.currency
                      ) {
                        return;
                      }
                      updateLimit.mutate(
                        {
                          id: limit.id,
                          input: {
                            category_id: limitDraft.categoryID,
                            amount: limitDraft.amount,
                            currency: limitDraft.currency,
                            is_active: limitDraft.isActive,
                          },
                        },
                        { onSuccess: () => setLimitDraft(null) },
                      );
                    }}
                  >
                    <CategoryField
                      id={`limit-edit-category-${limit.id}`}
                      label={t.transactions.category}
                      chooseLabel={t.goals.chooseCategory}
                      categories={categories}
                      value={limitDraft.categoryID}
                      onChange={(categoryID) =>
                        setLimitDraft({ ...limitDraft, categoryID })
                      }
                    />
                    <Field
                      label={t.goals.limitAmount}
                      htmlFor={`limit-edit-amount-${limit.id}`}
                    >
                      <Input
                        id={`limit-edit-amount-${limit.id}`}
                        value={limitDraft.amount}
                        type="number"
                        min="0.01"
                        step="0.01"
                        onChange={(event) =>
                          setLimitDraft({
                            ...limitDraft,
                            amount: event.target.value,
                          })
                        }
                      />
                    </Field>
                    <Field
                      label={t.goals.currency}
                      htmlFor={`limit-edit-currency-${limit.id}`}
                    >
                      <Input
                        id={`limit-edit-currency-${limit.id}`}
                        value={limitDraft.currency}
                        maxLength={3}
                        onChange={(event) =>
                          setLimitDraft({
                            ...limitDraft,
                            currency: event.target.value.toUpperCase(),
                          })
                        }
                      />
                    </Field>
                    <Field
                      label={t.goals.status}
                      htmlFor={`limit-edit-status-${limit.id}`}
                    >
                      <Select
                        id={`limit-edit-status-${limit.id}`}
                        value={limitDraft.isActive ? "active" : "inactive"}
                        onChange={(event) =>
                          setLimitDraft({
                            ...limitDraft,
                            isActive: event.target.value === "active",
                          })
                        }
                      >
                        <option value="active">{t.goals.statusActive}</option>
                        <option value="inactive">{t.goals.inactive}</option>
                      </Select>
                    </Field>
                    <div className="management-form-actions">
                      <Button
                        type="button"
                        variant="ghost"
                        onClick={() => setLimitDraft(null)}
                      >
                        {t.common.cancel}
                      </Button>
                      <Button type="submit" disabled={updateLimit.isPending}>
                        {t.goals.saveChanges}
                      </Button>
                    </div>
                  </form>
                ) : (
                  <div className="management-actions">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => {
                        setGoalDraft(null);
                        setLimitDraft(limitToDraft(limit));
                      }}
                    >
                      {t.goals.edit}
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      disabled={updateLimit.isPending}
                      onClick={() =>
                        updateLimit.mutate({
                          id: limit.id,
                          input: { is_active: !limit.is_active },
                        })
                      }
                    >
                      {limit.is_active ? t.goals.deactivate : t.goals.activate}
                    </Button>
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      </section>
    </Panel>
  );
}

function Field({
  label,
  htmlFor,
  children,
}: {
  label: string;
  htmlFor: string;
  children: ReactNode;
}) {
  return (
    <div className="field">
      <label htmlFor={htmlFor}>{label}</label>
      {children}
    </div>
  );
}

function AccountField({
  id,
  label,
  chooseLabel,
  accounts,
  value,
  onChange,
}: {
  id: string;
  label: string;
  chooseLabel: string;
  accounts: Account[];
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field label={label} htmlFor={id}>
      <Select
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        <option value="">{chooseLabel}</option>
        {accounts.map((account) => (
          <option key={account.id} value={account.id}>
            {account.name} · {account.currency}
          </option>
        ))}
      </Select>
    </Field>
  );
}

function CategoryField({
  id,
  label,
  chooseLabel,
  categories,
  value,
  onChange,
}: {
  id: string;
  label: string;
  chooseLabel: string;
  categories: Category[];
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field label={label} htmlFor={id}>
      <Select
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        <option value="">{chooseLabel}</option>
        {categories.map((category) => (
          <option key={category.id} value={category.id}>
            {category.name}
          </option>
        ))}
      </Select>
    </Field>
  );
}

function StatusLabel({
  status,
  labels,
}: {
  status: FinancialGoal["status"];
  labels: {
    statusActive: string;
    statusCompleted: string;
    statusArchived: string;
  };
}) {
  const label =
    status === "completed"
      ? labels.statusCompleted
      : status === "archived"
        ? labels.statusArchived
        : labels.statusActive;
  return <span className={`management-status is-${status}`}>{label}</span>;
}

function Progress({
  value,
  label,
  tone,
  categoryKey,
}: {
  value: number;
  label: string;
  tone: "default" | "success" | "warning" | "danger";
  categoryKey?: string;
}) {
  const bounded = Math.min(100, Math.max(0, Math.round(value)));
  return (
    <span
      className="budget-progress"
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={bounded}
    >
      <span
        className={
          categoryKey
            ? `budget-progress-bar category-progress ${categoryColorClass(categoryKey)}`
            : `budget-progress-bar is-${tone}`
        }
        style={{ transform: `scaleX(${bounded / 100})` }}
      />
    </span>
  );
}

function goalToDraft(goal: FinancialGoal): GoalDraft {
  return {
    id: goal.id,
    accountID: goal.account_id ?? "",
    name: goal.name,
    amount: goal.target_amount,
    targetDate: goal.target_date ?? "",
    status: goal.status,
  };
}

function limitToDraft(limit: CategoryLimit): LimitDraft {
  return {
    id: limit.id,
    categoryID: limit.category_id,
    amount: limit.amount,
    currency: limit.currency,
    isActive: limit.is_active,
  };
}

function formatGoalDate(value: string, locale: string) {
  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  }).format(new Date(`${value}T00:00:00Z`));
}

function ratio(current: string, target: string) {
  const targetValue = Number(target);
  return targetValue > 0 ? (Number(current) / targetValue) * 100 : 0;
}

async function refreshGoalData(queryClient: ReturnType<typeof useQueryClient>) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ["financial-goals"] }),
    queryClient.invalidateQueries({ queryKey: ["category-limits"] }),
    queryClient.invalidateQueries({ queryKey: ["dashboard"] }),
  ]);
}
