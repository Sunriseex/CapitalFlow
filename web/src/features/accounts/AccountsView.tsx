import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { CreditCard } from "lucide-react";
import { api } from "../../api/client";
import type { Account, InterestRule } from "../../api/types";
import { accountTypes } from "../../shared/constants";
import { apiErrorMessages, errorMessage } from "../../shared/api/query";
import {
  Empty,
  EmptyState,
  LoadingSkeleton,
  Panel,
  QueryError,
  Select,
} from "../../shared/ui";
import { AccountsTable } from "./components/AccountsTable";
import { useI18n } from "../../shared/i18n/useI18n";

export function AccountsView({
  accounts,
  isLoading = false,
  error = null,
  onSelect,
  onCreateAccount,
  onRetry,
}: {
  accounts: Account[];
  isLoading?: boolean;
  error?: unknown;
  onSelect: (id: string) => void;
  onCreateAccount?: () => void;
  onRetry?: () => void;
}) {
  const { t } = useI18n();
  const errorMessages = apiErrorMessages(t);

  const [type, setType] = useState("");
  const summary = useQuery({
    queryKey: ["dashboard", "summary"],
    queryFn: api.dashboardSummary,
  });
  const rules = useQuery({
    queryKey: ["interest-rules"],
    queryFn: () => api.interestRules(),
  });
  const balances = useMemo(
    () =>
      new Map(
        (summary.data?.account_balances ?? []).map((account) => [
          account.account_id,
          account,
        ]),
      ),
    [summary.data?.account_balances],
  );
  const activeRules = useMemo(
    () => activeRulesByAccount(rules.data ?? []),
    [rules.data],
  );
  const filtered = useMemo(
    () => accounts.filter((account) => !type || account.type === type),
    [accounts, type],
  );
  const accountTypeOptions = useMemo(
    () =>
      accountTypes.map((accountType) => (
        <option key={accountType} value={accountType}>
          {t.accounts.types[accountType]}
        </option>
      )),
    [t],
  );

  return (
    <Panel
      className="workspace-panel accounts-panel"
      title={t.accounts.title}
      action={
        <Select
          aria-label={t.accounts.filterByType}
          value={type}
          onChange={(event) => setType(event.target.value)}
        >
          <option value="">{t.accounts.allTypes}</option>
          {accountTypeOptions}
        </Select>
      }
    >
      {isLoading ? (
        <LoadingSkeleton label={t.accounts.loadingAccounts} />
      ) : null}{" "}
      {error ? (
        <QueryError
          stale={accounts.length > 0}
          message={errorMessage(error, errorMessages)}
          onRetry={onRetry}
        />
      ) : null}
      {!isLoading &&
      (!error || accounts.length > 0) &&
      accounts.length === 0 ? (
        <EmptyState
          icon={<CreditCard aria-hidden="true" />}
          title={t.accounts.emptyTitle}
          description={t.accounts.emptyDescription}
          primaryAction={
            onCreateAccount
              ? { label: t.accounts.createAccount, onClick: onCreateAccount }
              : undefined
          }
        />
      ) : null}
      {!isLoading &&
      (!error || accounts.length > 0) &&
      accounts.length > 0 &&
      !filtered.length ? (
        <Empty>{t.accounts.noAccounts}</Empty>
      ) : null}
      {!isLoading && (!error || accounts.length > 0) && filtered.length ? (
        <AccountsTable
          accounts={filtered}
          balances={balances}
          activeRules={activeRules}
          rulesLoading={rules.isLoading}
          rulesError={rules.error}
          onSelect={onSelect}
        />
      ) : null}
    </Panel>
  );
}

function activeRulesByAccount(rules: InterestRule[]) {
  const activeRules = new Map<string, InterestRule>();
  const today = localDateString(new Date());
  for (const rule of rules) {
    if (!rule.is_active || !isRuleEffective(rule, today)) {
      continue;
    }
    const current = activeRules.get(rule.account_id);
    if (!current || rule.start_date.localeCompare(current.start_date) > 0) {
      activeRules.set(rule.account_id, rule);
    }
  }
  return activeRules;
}

function isRuleEffective(rule: InterestRule, today: string) {
  return rule.start_date <= today && (!rule.end_date || rule.end_date >= today);
}

function localDateString(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}
