import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../api/client";
import { formatMoney } from "../../api/money";
import type { Account, InterestRule } from "../../api/types";
import { accountTypes } from "../../shared/constants";
import { errorMessage } from "../../shared/api/query";
import { Button, Empty, Panel, Select } from "../../shared/ui";

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
      <div className="table-wrap workspace-table-wrap accounts-table-wrap">
        <table className="workspace-table accounts-table">
          <thead>
            <tr><th>Name</th><th>Bank</th><th>Type</th><th>Balance</th><th>Rate</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>
            {!isLoading && !error ? filtered.map((account) => (
              <tr key={account.id}>
                <td data-label="Name">{account.name}</td>
                <td data-label="Bank">{account.bank || "-"}</td>
                <td data-label="Type">{account.type}</td>
                <td data-label="Balance" className="amount">{formatMoney(balances.get(account.id)?.balance ?? "0", account.currency)}</td>
                <td data-label="Rate"><AccountRate rule={activeRules.get(account.id)} isLoading={rules.isLoading} error={rules.error} /></td>
                <td data-label="Status">{account.is_active ? "active" : "archived"}</td>
                <td data-label="Action"><Button onClick={() => onSelect(account.id)}>Open</Button></td>
              </tr>
            )) : null}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function AccountRate({ rule, isLoading, error }: { rule?: InterestRule; isLoading: boolean; error: unknown }) {
  if (isLoading) {
    return <span>Loading</span>;
  }
  if (error) {
    return <span className="error-text">{errorMessage(error)}</span>;
  }
  if (!rule) {
    return <span>-</span>;
  }
  return <span>{(rule.annual_rate_bps / 100).toFixed(2)}%</span>;
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
