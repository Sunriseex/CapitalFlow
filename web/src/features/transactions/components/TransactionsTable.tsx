import { memo, useCallback, useMemo, useState } from "react";
import type { KeyboardEvent } from "react";
import { compareMoney, formatMoney, signedAmount } from "../../../api/money";
import type { Account, Category, Transaction } from "../../../api/types";
import { dateLabel } from "../../../shared/date";
import { Empty } from "../../../shared/ui";
import { useI18n } from "../../../shared/i18n/useI18n";
import type { TranslationDictionary } from "../../../shared/i18n/dictionaries/ru";

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
  const transactionWindowKey = `${transactions.length}:${transactions[0]?.id ?? ""}:${transactions.at(-1)?.id ?? ""}`;
  const [visibleState, setVisibleState] = useState({
    key: transactionWindowKey,
    count: initialChunkSize,
  });
  const activeVisibleState =
    visibleState.key === transactionWindowKey
      ? visibleState
      : { key: transactionWindowKey, count: initialChunkSize };
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
    () =>
      chunked ? transactions.slice(0, activeVisibleState.count) : transactions,
    [activeVisibleState.count, chunked, transactions],
  );
  const hasMore = chunked && visibleTransactions.length < transactions.length;
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

  if (!transactions.length) {
    return <Empty>{t.transactions.noTransactions}</Empty>;
  }

  return (
    <>
      <div
        className={`table-wrap workspace-table-wrap transactions-table-wrap${compact ? " is-compact" : ""}`}
      >
        <table className="workspace-table transactions-table">
          <thead>
            <tr>
              <th>{t.transactions.operation}</th>
              {compact ? null : <th>{t.transactions.category}</th>}
              {compact ? null : <th>{t.transactions.account}</th>}
              <th>{t.transactions.date}</th>
              {compact ? null : <th>{t.transactions.status}</th>}
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
      </div>
      <div
        className={`transactions-mobile-list${compact ? " is-compact" : ""}`}
      >
        {visibleTransactions.map((transaction) => (
          <TransactionCard
            key={transaction.id}
            transaction={transaction}
            accountNames={accountNames}
            accountCurrencies={accountCurrencies}
            categoryNames={categoryNames}
            isInteractive={Boolean(onOpenTransaction)}
            onOpenTransaction={openTransaction}
            t={t}
          />
        ))}
      </div>
      {hasMore ? (
        <div className="table-more">
          <button
            className="button"
            type="button"
            onClick={() =>
              setVisibleState((state) => ({
                key: transactionWindowKey,
                count:
                  (state.key === transactionWindowKey
                    ? state.count
                    : initialChunkSize) + nextChunkSize,
              }))
            }
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
    </>
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
  const details = transactionPresentation({
    transaction,
    accountNames,
    accountCurrencies,
    categoryNames,
    t,
    locale,
  });

  return (
    <tr
      className={
        isInteractive ? "transaction-row is-clickable-row" : "transaction-row"
      }
      tabIndex={isInteractive ? 0 : undefined}
      aria-label={
        isInteractive
          ? `${t.transactions.openTransactionDetails}: ${details.title}`
          : undefined
      }
      onClick={isInteractive ? () => onOpenTransaction(transaction) : undefined}
      onKeyDown={(event) => onKeyOpen(event, transaction)}
    >
      <td data-label={t.transactions.operation}>
        <div className="transaction-operation-cell">
          <span className="transaction-kind-icon" aria-hidden="true">
            {details.icon}
          </span>
          <span className="transaction-operation-copy">
            <strong>{details.title}</strong>
            <small>{details.secondary}</small>
          </span>
        </div>
      </td>
      {compact ? null : (
        <td data-label={t.transactions.category}>
          {details.categoryName}
        </td>
      )}
      {compact ? null : (
        <td data-label={t.transactions.account}>
          {details.accountName}
        </td>
      )}
      <td data-label={t.transactions.date}>{details.date}</td>
      {compact ? null : (
        <td data-label={t.transactions.status}>
          <StatusBadges badges={details.badges} />
        </td>
      )}
      <td data-label={t.transactions.amount} className={details.amountClass}>
        {details.amount}
      </td>
    </tr>
  );
});

const TransactionCard = memo(function TransactionCard({
  transaction,
  accountNames,
  accountCurrencies,
  categoryNames,
  isInteractive,
  onOpenTransaction,
  t,
}: {
  transaction: Transaction;
  accountNames: Map<string, string>;
  accountCurrencies: Map<string, string>;
  categoryNames: Map<string, string>;
  isInteractive: boolean;
  onOpenTransaction: (transaction: Transaction) => void;
  t: ReturnType<typeof useI18n>["t"];
}) {
  const { locale } = useI18n();
  const details = transactionPresentation({
    transaction,
    accountNames,
    accountCurrencies,
    categoryNames,
    t,
    locale,
  });
  const content = (
    <>
      <span className="transaction-card-top">
        <strong>{details.title}</strong>
        <span className={details.amountClass}>{details.amount}</span>
      </span>
      <span className="transaction-card-meta">
        {details.categoryName} · {details.accountName}
      </span>
      <span className="transaction-card-footer">
        <span>{details.date}</span>
        <StatusBadges badges={details.badges} />
      </span>
    </>
  );

  if (!isInteractive) {
    return <div className="transaction-mobile-card">{content}</div>;
  }

  return (
    <button
      className="transaction-mobile-card is-clickable-card"
      type="button"
      aria-label={`${t.transactions.openTransactionDetails}: ${details.title}`}
      onClick={() => onOpenTransaction(transaction)}
    >
      {content}
    </button>
  );
});

function StatusBadges({ badges }: { badges: TransactionBadge[] }) {
  return (
    <span className="transaction-badges">
      {badges.map((badge) => (
        <span key={badge.label} className={`tag ${badge.tone}`}>
          {badge.label}
        </span>
      ))}
    </span>
  );
}

type TransactionBadge = {
  label: string;
  tone: "info" | "good" | "muted";
};

function transactionPresentation({
  transaction,
  accountNames,
  accountCurrencies,
  categoryNames,
  t,
  locale,
}: {
  transaction: Transaction;
  accountNames: Map<string, string>;
  accountCurrencies: Map<string, string>;
  categoryNames: Map<string, string>;
  t: TranslationDictionary;
  locale: "en" | "ru";
}) {
  const signed = signedAmount(transaction);
  const amountIsNegative = compareMoney(signed, "0") < 0;
  const currency = accountCurrencies.get(transaction.account_id) ?? "RUB";
  const typeLabel = t.transactions.types[transaction.type];
  const title = transaction.description || typeLabel;
  const accountName =
    accountNames.get(transaction.account_id) ??
    transaction.account_id.slice(0, 8);
  const categoryName = transaction.category_id
    ? (categoryNames.get(transaction.category_id) ??
      transaction.category_id.slice(0, 8))
    : t.common.none;
  const sourceLabel = sourceForTransaction(transaction, t);
  const date = dateLabel(transaction.occurred_at, locale);
  const badges = badgesForTransaction(transaction, t);

  return {
    title,
    typeLabel,
    accountName,
    categoryName,
    date,
    badges,
    icon: typeLabel.slice(0, 1),
    secondary: `${sourceLabel} · ${typeLabel}`,
    amount: formatMoney(signed, currency, locale),
    amountClass: amountIsNegative
      ? "transaction-amount delta-down"
      : "transaction-amount delta-up",
  };
}

function sourceForTransaction(
  transaction: Transaction,
  t: TranslationDictionary,
) {
  if (transaction.transfer_id) {
    return t.transactions.sourceTransfer;
  }

  if (transaction.type === "initial_balance") {
    return t.transactions.sourceInitialBalance;
  }

  if (transaction.type === "interest_income") {
    return t.transactions.sourceInterest;
  }

  if (transaction.type === "adjustment") {
    return t.transactions.sourceAdjustment;
  }

  return t.transactions.sourceManual;
}

function badgesForTransaction(
  transaction: Transaction,
  t: TranslationDictionary,
): TransactionBadge[] {
  if (transaction.transfer_id) {
    return [{ label: t.transactions.transfer, tone: "info" }];
  }

  if (transaction.type === "adjustment") {
    return [{ label: t.transactions.adjustment, tone: "muted" }];
  }

  if (transaction.type === "initial_balance") {
    return [{ label: t.transactions.initialBalance, tone: "muted" }];
  }

  if (transaction.type === "interest_income") {
    return [{ label: t.transactions.interest, tone: "good" }];
  }

  return [{ label: t.transactions.verified, tone: "good" }];
}
