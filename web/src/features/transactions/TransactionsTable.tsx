import { memo, useEffect, useMemo, useState } from "react";
import { compareMoney, formatMoney, signedAmount, transactionTypeLabel } from "../../api/money";
import type { Account, Category, Transaction } from "../../api/types";
import { dateLabel } from "../../shared/date";
import { Empty } from "../../shared/ui";

const initialChunkSize = 80;
const nextChunkSize = 120;

export const TransactionsTable = memo(function TransactionsTable({
  transactions,
  accounts,
  categories,
  compact = false,
  chunked = false,
}: {
  transactions: Transaction[];
  accounts: Account[];
  categories: Category[];
  compact?: boolean;
  chunked?: boolean;
}) {
  const [visibleCount, setVisibleCount] = useState(initialChunkSize);
  const accountNames = useMemo(() => new Map(accounts.map((account) => [account.id, account.name])), [accounts]);
  const accountCurrencies = useMemo(() => new Map(accounts.map((account) => [account.id, account.currency])), [accounts]);
  const categoryNames = useMemo(() => new Map(categories.map((category) => [category.id, category.name])), [categories]);
  const visibleTransactions = useMemo(
    () => chunked ? transactions.slice(0, visibleCount) : transactions,
    [chunked, transactions, visibleCount],
  );
  const hasMore = chunked && visibleTransactions.length < transactions.length;
  const transactionWindowKey = `${transactions.length}:${transactions[0]?.id ?? ""}:${transactions.at(-1)?.id ?? ""}`;

  useEffect(() => {
    if (chunked) {
      setVisibleCount(initialChunkSize);
    }
  }, [chunked, transactionWindowKey]);

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
          {visibleTransactions.map((transaction) => (
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
      {hasMore ? (
        <div className="table-more">
          <button className="button" type="button" onClick={() => setVisibleCount((count) => count + nextChunkSize)}>
            Show more
          </button>
          <span>{visibleTransactions.length} of {transactions.length}</span>
        </div>
      ) : null}
    </div>
  );
});
