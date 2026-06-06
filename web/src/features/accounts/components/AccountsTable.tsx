import { formatMoney } from "../../../api/money";
import type { Account, DashboardAccountBalance, InterestRule } from "../../../api/types";
import { errorMessage } from "../../../shared/api/query";
import { Button } from "../../../shared/ui";

export function AccountsTable({
  accounts,
  balances,
  activeRules,
  rulesLoading,
  rulesError,
  onSelect,
}: {
  accounts: Account[];
  balances: Map<string, DashboardAccountBalance>;
  activeRules: Map<string, InterestRule>;
  rulesLoading: boolean;
  rulesError: unknown;
  onSelect: (id: string) => void;
}) {
  return (
    <div className="table-wrap workspace-table-wrap accounts-table-wrap">
      <table className="workspace-table accounts-table">
        <thead>
          <tr><th>Name</th><th>Bank</th><th>Type</th><th>Balance</th><th>Rate</th><th>Status</th><th></th></tr>
        </thead>
        <tbody>
          {accounts.map((account) => (
            <tr key={account.id}>
              <td data-label="Name">{account.name}</td>
              <td data-label="Bank">{account.bank || "-"}</td>
              <td data-label="Type">{account.type}</td>
              <td data-label="Balance" className="amount">{formatMoney(balances.get(account.id)?.balance ?? "0", account.currency)}</td>
              <td data-label="Rate"><AccountRate rule={activeRules.get(account.id)} isLoading={rulesLoading} error={rulesError} /></td>
              <td data-label="Status">{account.is_active ? "active" : "archived"}</td>
              <td data-label="Action"><Button onClick={() => onSelect(account.id)}>Open</Button></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
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
