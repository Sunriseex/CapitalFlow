import { memo, useCallback, useEffect, useMemo, useState } from "react";
import type { KeyboardEvent } from "react";
import { compareMoney, formatMoney, signedAmount } from "../../../api/money";
import type { Account, Category, Transaction } from "../../../api/types";
import { dateLabel } from "../../../shared/date";
import { Empty } from "../../../shared/ui";
import { useI18n } from "../../../shared/i18n/useI18n";

const initialChunkSize = 48;
const nextChunkSize = 96;

export const TransactionsTable = memo(function TransactionsTable({
  transactions,
  accounts,
  categories,
  compact = false,
  chunked = false,
  onOpenTransaction,
}: {
  transactions: Transaction[];
  accounts: Account[];
  categories: Category[];
  compact?: boolean;
  chunked?: boolean;
  onOpenTransaction?: (transaction: Transaction) => void;
}) {
  const { t } = useI18n();
  const [visibleCount, setVisibleCount] = useState(initialChunkSize);
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
  const visibleTransactions = useMemo(
    () => (chunked ? transactions.slice(0, visibleCount) : transactions),
    [chunked, transactions, visibleCount],
  );
  const hasMore = chunked && visibleTransactions.length < transactions.length;
  const transactionWindowKey = `${transactions.length}:${transactions[0]?.id ?? ""}:${transactions.at(-1)?.id ?? ""}`;
  const openTransaction = useCallback(
    (transaction: Transaction) => {
      onOpenTransaction?.(transaction);
    },
    [onOpenTransaction],
  );

  const openWithKeyboard = useCallback(
    (event: KeyboardEvent<HTMLTableRowElement>, transaction: Transaction) => {
      if (!onOpenTransaction || (event.key !== "Enter" && event.key !== " ")) {
        return;
      }
      event.preventDefault();
      onOpenTransaction(transaction);
    },
    [onOpenTransaction],
  );

  useEffect(() => {
    if (chunked) {
      setVisibleCount(initialChunkSize);
    }
  }, [chunked, transactionWindowKey]);

  if (!transactions.length) {
    return <Empty>{t.transactions.noTransactions}</Empty>;
  }

  return (
    <div
      className={`table-wrap workspace-table-wrap transactions-table-wrap${compact ? " is-compact" : ""}`}
    >
      <table className="workspace-table transactions-table">
        <thead>
          <tr>
            <th>{t.transactions.date}</th>
            <th>{t.transactions.type}</th>
            {compact ? null : <th>{t.transactions.account}</th>}
            {compact ? null : <th>{t.transactions.category}</th>}
            <th>{t.transactions.description}</th>
            <th>{t.transactions.amount}</th>
          </tr>
        </thead>
        <tbody>
          {visibleTransactions.map((transaction) => (
            <TransactionRow
              key={transaction.id}
              transaction={transaction}
              compact={compact}
              accountNames={accountNames}
              accountCurrencies={accountCurrencies}
              categoryNames={categoryNames}
              isInteractive={Boolean(onOpenTransaction)}
              onOpenTransaction={openTransaction}
              onKeyOpen={openWithKeyboard}
              t={t}
            />
          ))}
        </tbody>
      </table>
      {hasMore ? (
        <div className="table-more">
          <button
            className="button"
            type="button"
            onClick={() => setVisibleCount((count) => count + nextChunkSize)}
          >
            {t.transactions.showMore}{" "}
          </button>
          <span>
            {t.transactions.visibleOfTotal
              .replace("{visible}", String(visibleTransactions.length))
              .replace("{total}", String(transactions.length))}
          </span>
        </div>
      ) : null}
    </div>
  );
});

const TransactionRow = memo(function TransactionRow({
  transaction,
  compact,
  accountNames,
  accountCurrencies,
  categoryNames,
  isInteractive,
  onOpenTransaction,
  onKeyOpen,
  t,
}: {
  transaction: Transaction;
  compact: boolean;
  accountNames: Map<string, string>;
  accountCurrencies: Map<string, string>;
  categoryNames: Map<string, string>;
  isInteractive: boolean;
  onOpenTransaction: (transaction: Transaction) => void;
  onKeyOpen: (
    event: KeyboardEvent<HTMLTableRowElement>,
    transaction: Transaction,
  ) => void;
  t: ReturnType<typeof useI18n>["t"];
}) {
  const { locale } = useI18n();
  const signed = signedAmount(transaction);

  return (
    <tr
      className={isInteractive ? "is-clickable-row" : undefined}
      tabIndex={isInteractive ? 0 : undefined}
      aria-label={
        isInteractive
          ? `${t.transactions.openTransactionDetails}: ${transaction.description || transaction.id}`
          : undefined
      }
      onClick={isInteractive ? () => onOpenTransaction(transaction) : undefined}
      onKeyDown={(event) => onKeyOpen(event, transaction)}
    >
      <td data-label={t.transactions.date}>
        {dateLabel(transaction.occurred_at, locale)}
      </td>
      <td data-label={t.transactions.type}>
        {t.transactions.types[transaction.type]}
      </td>
      {compact ? null : (
        <td data-label={t.transactions.account}>
          {accountNames.get(transaction.account_id) ??
            transaction.account_id.slice(0, 8)}
        </td>
      )}
      {compact ? null : (
        <td data-label={t.transactions.category}>
          {transaction.category_id
            ? (categoryNames.get(transaction.category_id) ??
              transaction.category_id.slice(0, 8))
            : "-"}
        </td>
      )}
      <td data-label={t.transactions.description}>
        {transaction.description || "-"}
      </td>
      <td
        data-label={t.transactions.amount}
        className={compareMoney(signed, "0") < 0 ? "amount danger" : "amount"}
      >
        {formatMoney(
          signed,
          accountCurrencies.get(transaction.account_id) ?? "RUB",
          locale,
        )}
      </td>
    </tr>
  );
});
