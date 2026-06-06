import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../api/client";
import type { Account, InterestRule } from "../../api/types";
import { accountTypes } from "../../shared/constants";
import { errorMessage } from "../../shared/api/query";
import { Empty, Panel, Select } from "../../shared/ui";
import { AccountsTable } from "./components/AccountsTable";

export function AccountsView({
  accounts,
  isLoading = false,
  error = null,
  onSelect,
}: {
  accounts: Account[];
  isLoading?: boolean;
  error?: unknown;
  onSelect: (id: string) => void;
}) {
  const [type, setType] = useState("");
  const summary = useQuery({ queryKey: ["dashboard", "summary"], queryFn: api.dashboardSummary });
  const rules = useQuery({ queryKey: ["interest-rules"], queryFn: () => api.interestRules() });
  const balances = useMemo(
    () => new Map((summary.data?.account_balances ?? []).map((account) => [account.account_id, account])),
    [summary.data?.account_balances],
  );
  const activeRules = useMemo(() => activeRulesByAccount(rules.data ?? []), [rules.data]);
  const filtered = useMemo(
    () => accounts.filter((account) => !type || account.type === type),
    [accounts, type],
  );
  const accountTypeOptions = useMemo(
    () => accountTypes.map((accountType) => <option key={accountType}>{accountType}</option>),
    [],
  );

  return (
    <Panel
      className="workspace-panel accounts-panel"
      title="Accounts"
      action={
        <Select aria-label="Filter accounts by type" value={type} onChange={(event) => setType(event.target.value)}>
          <option value="">All types</option>
          {accountTypeOptions}
        </Select>
      }
    >
      {isLoading ? <Empty>Loading accounts</Empty> : null}
      {error ? <div className="error inline-error">{errorMessage(error)}</div> : null}
      {!isLoading && !error && !filtered.length ? <Empty>No accounts</Empty> : null}
      {!isLoading && !error ? (
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
