import { useMemo } from "react";
import { compareMoney, formatMoney, signedAmount, transactionTypeLabel } from "../../api/money";
import type { Account, Category, Transaction } from "../../api/types";
import { dateLabel } from "../../shared/date";
import { Empty } from "../../shared/ui";

export function TransactionsTable({
  transactions,
  accounts,
  categories,
  compact = false,
}: {
  transactions: Transaction[];
  accounts: Account[];
  categories: Category[];
  compact?: boolean;
}) {
  const accountNames = useMemo(() => new Map(accounts.map((account) => [account.id, account.name])), [accounts]);
  const accountCurrencies = useMemo(() => new Map(accounts.map((account) => [account.id, account.currency])), [accounts]);
  const categoryNames = useMemo(() => new Map(categories.map((category) => [category.id, category.name])), [categories]);

  if (!transactions.length) {
    return <Empty>No transactions</Empty>;
  }

  return (
    <div className={`table-wrap workspace-table-wrap transactions-table-wrap${compact ? " is-compact" : ""}`}>
      <table className="workspace-table transactions-table">
        <thead>
          <tr><th>Date</th><th>Type</th>{compact ? null : <th>Account</th>}{compact ? null : <th>Category</th>}<th>Description</th><th>Amount</th></tr>
        </thead>
        <tbody>
          {transactions.map((transaction) => (
            <tr key={transaction.id}>
              <td data-label="Date">{dateLabel(transaction.occurred_at)}</td>
              <td data-label="Type">{transactionTypeLabel(transaction.type)}</td>
              {compact ? null : <td data-label="Account">{accountNames.get(transaction.account_id) ?? transaction.account_id.slice(0, 8)}</td>}
              {compact ? null : <td data-label="Category">{transaction.category_id ? categoryNames.get(transaction.category_id) ?? transaction.category_id.slice(0, 8) : "-"}</td>}
              <td data-label="Description">{transaction.description || "-"}</td>
              <td data-label="Amount" className={compareMoney(signedAmount(transaction), "0") < 0 ? "amount danger" : "amount"}>
                {formatMoney(signedAmount(transaction), accountCurrencies.get(transaction.account_id) ?? "RUB")}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
