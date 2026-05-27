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
  const accountNames = new Map(accounts.map((account) => [account.id, account.name]));
  const accountCurrencies = new Map(accounts.map((account) => [account.id, account.currency]));
  const categoryNames = new Map(categories.map((category) => [category.id, category.name]));

  if (!transactions.length) {
    return <Empty>No transactions</Empty>;
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr><th>Date</th><th>Type</th>{compact ? null : <th>Account</th>}{compact ? null : <th>Category</th>}<th>Description</th><th>Amount</th></tr>
        </thead>
        <tbody>
          {transactions.map((transaction) => (
            <tr key={transaction.id}>
              <td>{dateLabel(transaction.occurred_at)}</td>
              <td>{transactionTypeLabel(transaction.type)}</td>
              {compact ? null : <td>{accountNames.get(transaction.account_id) ?? transaction.account_id.slice(0, 8)}</td>}
              {compact ? null : <td>{transaction.category_id ? categoryNames.get(transaction.category_id) ?? transaction.category_id.slice(0, 8) : "-"}</td>}
              <td>{transaction.description || "-"}</td>
              <td className={compareMoney(signedAmount(transaction), "0") < 0 ? "amount danger" : "amount"}>
                {formatMoney(signedAmount(transaction), accountCurrencies.get(transaction.account_id) ?? "RUB")}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}


