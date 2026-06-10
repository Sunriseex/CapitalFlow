import { useMemo } from "react";
import { compareMoney, formatMoney, signedAmount } from "../../../api/money";
import type { Account, Category, Transaction } from "../../../api/types";
import { dateLabel } from "../../../shared/date";
import { useI18n } from "../../../shared/i18n/useI18n";

export function TransactionDetails({
  transaction,
  accounts,
  categories = [],
}: {
  transaction: Transaction;
  accounts: Account[];
  categories?: Category[];
}) {
  const { t, locale } = useI18n();

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
  const amount = signedAmount(transaction);
  const currency = accountCurrencies.get(transaction.account_id) ?? "RUB";
  const typeLabel = t.transactions.types[transaction.type];

  return (
    <div className="transaction-detail">
      <header className="transaction-detail-hero">
        <span className="transaction-detail-icon" aria-hidden="true">
          {typeLabel.slice(0, 1)}
        </span>
        <div>
          <strong>{transaction.description || typeLabel}</strong>
          <small>
            {typeLabel} · {dateLabel(transaction.occurred_at, locale)}
          </small>
        </div>
        <span className="tag info">
          {transaction.transfer_id
            ? t.transactions.transfer
            : t.transactions.posted}
        </span>
      </header>

      <div className="transaction-detail-total">
        <span
          className={compareMoney(amount, "0") < 0 ? "delta-down" : "delta-up"}
        >
          {formatMoney(amount, currency)}
        </span>
      </div>

      <dl className="transaction-detail-list">
        <DetailItem
          label={t.transactions.date}
          value={dateLabel(transaction.occurred_at, locale)}
        />
        <DetailItem label={t.transactions.type} value={typeLabel} />
        <DetailItem
          label={t.transactions.account}
          value={
            accountNames.get(transaction.account_id) ?? transaction.account_id
          }
        />
        <DetailItem
          label={t.transactions.category}
          value={
            transaction.category_id
              ? (categoryNames.get(transaction.category_id) ??
                transaction.category_id)
              : "-"
          }
        />
        <DetailItem
          label={t.transactions.description}
          value={transaction.description || "-"}
        />
        <DetailItem
          label={t.transactions.transactionId}
          value={transaction.id}
        />
        {transaction.related_account_id ? (
          <DetailItem
            label={t.transactions.relatedAccount}
            value={
              accountNames.get(transaction.related_account_id) ??
              transaction.related_account_id
            }
          />
        ) : null}
        {transaction.transfer_id ? (
          <DetailItem
            label={t.transactions.transferId}
            value={transaction.transfer_id}
          />
        ) : null}
      </dl>
    </div>
  );
}

function DetailItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="transaction-detail-row">
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}
