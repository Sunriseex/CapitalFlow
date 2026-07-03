import { useMemo } from "react";
import type { KeyboardEvent } from "react";
import { formatMoney } from "../../../api/money";
import type { Account, Category, Transaction } from "../../../api/types";
import { useI18n } from "../../../shared/i18n/useI18n";
import { PrimitiveButton as ShadcnButton } from "../../../shared/ui";
import { CategoryBadge } from "../../transactions/components/CategoryBadge";

export function RecentTransactionsTable({
  accounts,
  categories,
  transactions,
  selectedCurrency,
  onOpenTransaction,
}: {
  accounts: Account[];
  categories: Category[];
  transactions: Transaction[];
  selectedCurrency: string;
  onOpenTransaction: (transaction: Transaction) => void;
}) {
  const { t, locale } = useI18n();
  const visibleTransactions = useMemo(
    () => transactions.slice(0, 5),
    [transactions],
  );
  const accountNames = useMemo(
    () => new Map(accounts.map((account) => [account.id, account.name])),
    [accounts],
  );
  const accountCurrencies = useMemo(
    () => new Map(accounts.map((account) => [account.id, account.currency])),
    [accounts],
  );
  const categoryNames = useMemo(
    () => new Map(categories.map((category) => [category.id, category.name])),
    [categories],
  );

  if (!transactions.length) {
    return (
      <div className="empty-state">
        <strong>{t.transactions.noTransactions}</strong>
        <span>{t.transactions.addFirstTransaction}</span>
      </div>
    );
  }

  function openWithKeyboard(
    event: KeyboardEvent<HTMLTableRowElement>,
    transaction: Transaction,
  ) {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }
    event.preventDefault();
    onOpenTransaction(transaction);
  }

  return (
    <div className="table-scroll">
      <table className="tx-table" aria-label={t.dashboard.recentTransactions}>
        <colgroup>
          <col className="col-operation" />
          <col className="col-account" />
          <col className="col-category" />
          <col className="col-amount" />
          <col className="col-view" />
        </colgroup>
        <thead>
          <tr>
            <th scope="col">{t.transactions.operation}</th>
            <th scope="col">{t.transactions.account}</th>
            <th scope="col">{t.transactions.category}</th>
            <th scope="col">{t.transactions.amount}</th>
            <th scope="col">{t.transactions.view}</th>
          </tr>
        </thead>
        <tbody>
          {visibleTransactions.map((transaction) => {
            const negative =
              transaction.type === "expense" ||
              transaction.type === "transfer_out";
            const sign = negative ? "-" : "+";
            return (
              <tr
                className="tx is-clickable-row"
                key={transaction.id}
                tabIndex={0}
                aria-label={`${t.transactions.openTransactionDetails}: ${transaction.description || transaction.id}`}
                onClick={() => onOpenTransaction(transaction)}
                onKeyDown={(event) => openWithKeyboard(event, transaction)}
              >
                <td data-label={t.transactions.operation}>
                  <strong>{transaction.description || transaction.type}</strong>
                  <small>
                    {t.transactions.types[transaction.type]} ·{" "}
                    {t.transactions.ledgerEvent}
                  </small>
                </td>
                <td data-label={t.transactions.account}>
                  {accountNames.get(transaction.account_id) ??
                    transaction.account_id}
                </td>
                <td data-label={t.transactions.category}>
                  <CategoryBadge
                    categoryKey={transaction.category_id ?? "uncategorized"}
                    name={
                      transaction.category_id
                        ? (categoryNames.get(transaction.category_id) ?? "-")
                        : "-"
                    }
                  />
                </td>
                <td
                  data-label={t.transactions.amount}
                  className={negative ? "delta-down" : "delta-up"}
                >
                  {sign}
                  {formatMoney(
                    transaction.amount,
                    accountCurrencies.get(transaction.account_id) ??
                      selectedCurrency,
                    locale,
                  )}
                </td>
                <td data-label={t.transactions.view}>
                  <ShadcnButton
                    className="view-cell"
                    type="button"
                    variant="ghost"
                    aria-label={t.transactions.openTransactionDetails}
                    onClick={(event) => {
                      event.stopPropagation();
                      onOpenTransaction(transaction);
                    }}
                  >
                    {t.transactions.view}
                  </ShadcnButton>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
