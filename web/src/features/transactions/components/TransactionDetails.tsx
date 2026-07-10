import { useMemo } from "react";
import { compareMoney, formatMoney, signedAmount } from "../../../api/money";
import type { Account, Category, Transaction } from "../../../api/types";
import { dateLabel } from "../../../shared/date";
import { Button } from "../../../shared/ui";
import { useI18n } from "../../../shared/i18n/useI18n";
import type { TranslationDictionary } from "../../../shared/i18n/dictionaries/ru";

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
  const accountName =
    accountNames.get(transaction.account_id) ?? transaction.account_id;
  const categoryName = transaction.category_id
    ? (categoryNames.get(transaction.category_id) ?? transaction.category_id)
    : t.common.none;
  const sourceLabel = sourceForTransaction(transaction, t);
  const statusLabel = t.transactions.statuses[transaction.status ?? "confirmed"];

  return (
    <div className="transaction-detail">
      <header className="transaction-detail-hero">
        <div className="transaction-detail-title-row">
          <div className="transaction-detail-title-copy">
            <strong>{transaction.description || typeLabel}</strong>
            <small>
              {typeLabel} · {categoryName} · {accountName}
            </small>
          </div>
          <span className="tag info">
            {statusLabel}
          </span>
        </div>
        <p
          className={
            compareMoney(amount, "0") < 0
              ? "transaction-detail-amount delta-down"
              : "transaction-detail-amount delta-up"
          }
        >
          {formatMoney(amount, currency, locale)}
        </p>
        <p className="transaction-detail-meta">
          {dateLabel(transaction.occurred_at, locale)}
        </p>
      </header>

      <div className="transaction-detail-actions">
        <Button type="button" disabled title={t.common.notAvailable}>
          {t.common.edit}
        </Button>
        <Button type="button" disabled title={t.common.notAvailable}>
          {t.transactions.changeCategory}
        </Button>
        <Button type="button" disabled title={t.common.notAvailable}>
          {t.transactions.createRule}
        </Button>
        <Button type="button" disabled title={t.common.notAvailable}>
          {t.transactions.duplicate}
        </Button>
        <Button
          className="danger-action"
          type="button"
          disabled
          title={t.common.notAvailable}
        >
          {t.common.delete}
        </Button>
      </div>

      <section className="transaction-detail-section">
        <h3>{t.transactions.mainDetails}</h3>
        <dl className="transaction-detail-list">
          <DetailItem label={t.transactions.type} value={typeLabel} />
          <DetailItem label={t.transactions.category} value={categoryName} />
          <DetailItem label={t.transactions.account} value={accountName} />
          <DetailItem
            label={t.transactions.date}
            value={dateLabel(transaction.occurred_at, locale)}
          />
          <DetailItem
            label={t.transactions.amount}
            value={formatMoney(amount, currency, locale)}
          />
          <DetailItem label={t.accounts.currency} value={currency} />
          <DetailItem label={t.transactions.status} value={statusLabel} />
          <DetailItem
            label={t.transactions.description}
            value={transaction.description || "-"}
          />
        </dl>
      </section>

      <section className="transaction-detail-section">
        <h3>{t.transactions.source}</h3>
        <dl className="transaction-detail-list">
          <DetailItem label={t.transactions.source} value={sourceLabel} />
          <DetailItem
            label={t.transactions.createdAt}
            value={dateLabel(transaction.created_at, locale)}
          />
          <DetailItem
            label={t.transactions.transactionId}
            value={transaction.id}
          />
        </dl>
      </section>

      <section className="transaction-detail-section">
        <h3>{t.transactions.relations}</h3>
        <dl className="transaction-detail-list">
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
          {!transaction.related_account_id && !transaction.transfer_id ? (
            <DetailItem
              label={t.transactions.relations}
              value={t.transactions.noRelations}
            />
          ) : null}
        </dl>
      </section>

      <section className="transaction-detail-section">
        <h3>{t.transactions.auditTimeline}</h3>
        <ol className="transaction-audit-list timeline">
          <li>
            <strong>{t.transactions.auditCreated}</strong>
            <span>{dateLabel(transaction.created_at, locale)}</span>
          </li>
          <li>
            <strong>{t.transactions.auditSourceCaptured}</strong>
            <span>{sourceLabel}</span>
          </li>
        </ol>
      </section>
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
