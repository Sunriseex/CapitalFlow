import { useMemo } from "react";
import type { KeyboardEvent } from "react";
import { formatMoney, transactionTypeLabel } from "../../../api/money";
import type { Account, Transaction } from "../../../api/types";

export function RecentTransactionsTable({
  accounts,
  transactions,
  selectedCurrency,
  onOpenTransaction,
}: {
  accounts: Account[];
  transactions: Transaction[];
  selectedCurrency: string;
  onOpenTransaction: (transaction: Transaction) => void;
}) {
  const visibleTransactions = useMemo(() => transactions.slice(0, 5), [transactions]);
  const accountNames = useMemo(() => new Map(accounts.map((account) => [account.id, account.name])), [accounts]);
  const accountCurrencies = useMemo(() => new Map(accounts.map((account) => [account.id, account.currency])), [accounts]);

  if (!transactions.length) {
    return <div className="empty-state"><strong>No transactions</strong><span>Add the first transaction or import a bank statement.</span></div>;
  }

  function openWithKeyboard(event: KeyboardEvent<HTMLTableRowElement>, transaction: Transaction) {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }
    event.preventDefault();
    onOpenTransaction(transaction);
  }

  return (
    <div className="table-scroll">
      <table className="tx-table" aria-label="Recent transactions">
        <colgroup>
          <col className="col-operation" />
          <col className="col-account" />
          <col className="col-category" />
          <col className="col-amount" />
          <col className="col-view" />
        </colgroup>
        <thead>
          <tr>
            <th scope="col">Operation</th>
            <th scope="col">Account</th>
            <th scope="col">Category</th>
            <th scope="col">Amount</th>
            <th scope="col">View</th>
          </tr>
        </thead>
        <tbody>
          {visibleTransactions.map((transaction) => {
            const negative = transaction.type === "expense" || transaction.type === "transfer_out";
            const sign = negative ? "-" : "+";
            return (
              <tr
                className="tx is-clickable-row"
                key={transaction.id}
                tabIndex={0}
                aria-label={`Open transaction details for ${transaction.description || transaction.id}`}
                onClick={() => onOpenTransaction(transaction)}
                onKeyDown={(event) => openWithKeyboard(event, transaction)}
              >
                <td data-label="Operation"><strong>{transaction.description || transaction.type}</strong><small>{transactionTypeLabel(transaction.type)} · ledger event</small></td>
                <td data-label="Account">{accountNames.get(transaction.account_id) ?? transaction.account_id}</td>
                <td data-label="Category">{transaction.category_id ?? "-"}</td>
                <td data-label="Amount" className={negative ? "delta-down" : "delta-up"}>
                  {sign}{formatMoney(transaction.amount, accountCurrencies.get(transaction.account_id) ?? selectedCurrency)}
                </td>
                <td data-label="View">
                  <button
                    className="view-cell"
                    type="button"
                    aria-label="Open transaction details"
                    onClick={(event) => {
                      event.stopPropagation();
                      onOpenTransaction(transaction);
                    }}
                  >
                    View
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
