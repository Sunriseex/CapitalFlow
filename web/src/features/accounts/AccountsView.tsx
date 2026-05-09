import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../api/client";
import { formatMoney } from "../../api/money";
import type { Account } from "../../api/types";
import { accountTypes } from "../../shared/constants";
import { Button, Panel, Select } from "../../shared/ui";

export function AccountsView({ accounts, onSelect }: { accounts: Account[]; onSelect: (id: string) => void }) {
  const [type, setType] = useState("");
  const summary = useQuery({ queryKey: ["dashboard", "summary"], queryFn: api.dashboardSummary });
  const balances = new Map((summary.data?.account_balances ?? []).map((account) => [account.account_id, account]));
  const filtered = accounts.filter((account) => !type || account.type === type);

  return (
    <Panel
      title="Accounts"
      action={
        <Select value={type} onChange={(event) => setType(event.target.value)}>
          <option value="">All types</option>
          {accountTypes.map((accountType) => <option key={accountType}>{accountType}</option>)}
        </Select>
      }
    >
      <div className="table-wrap">
        <table>
          <thead>
            <tr><th>Name</th><th>Bank</th><th>Type</th><th>Balance</th><th>Rate</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>
            {filtered.map((account) => (
              <tr key={account.id}>
                <td>{account.name}</td>
                <td>{account.bank || "-"}</td>
                <td>{account.type}</td>
                <td className="amount">{formatMoney(balances.get(account.id)?.balance_minor ?? 0, account.currency)}</td>
                <td><AccountRate accountId={account.id} /></td>
                <td>{account.is_active ? "active" : "archived"}</td>
                <td><Button onClick={() => onSelect(account.id)}>Open</Button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function AccountRate({ accountId }: { accountId: string }) {
  const rules = useQuery({ queryKey: ["interest-rules", accountId], queryFn: () => api.interestRules(accountId) });
  const activeRule = rules.data?.find((rule) => rule.is_active);
  if (!activeRule) {
    return <span>-</span>;
  }
  return <span>{(activeRule.annual_rate_bps / 100).toFixed(2)}%</span>;
}

