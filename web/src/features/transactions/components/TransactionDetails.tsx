import { useMemo } from "react";
import { compareMoney, formatMoney, signedAmount, transactionTypeLabel } from "../../../api/money";
import type { Account, Category, Transaction } from "../../../api/types";
import { dateLabel } from "../../../shared/date";

export function TransactionDetails({
  transaction,
  accounts,
  categories = [],
}: {
  transaction: Transaction;
  accounts: Account[];
  categories?: Category[];
}) {
  const accountNames = useMemo(() => new Map(accounts.map((account) => [account.id, account.name])), [accounts]);
  const accountCurrencies = useMemo(() => new Map(accounts.map((account) => [account.id, account.currency])), [accounts]);
  const categoryNames = useMemo(() => new Map(categories.map((category) => [category.id, category.name])), [categories]);
  const amount = signedAmount(transaction);
  const currency = accountCurrencies.get(transaction.account_id) ?? "RUB";
  const typeLabel = transactionTypeLabel(transaction.type);

  return (
    <div className="transaction-detail">
      <header className="transaction-detail-hero">
        <span className="transaction-detail-icon" aria-hidden="true">{typeLabel.slice(0, 1)}</span>
        <div>
          <strong>{transaction.description || typeLabel}</strong>
          <small>{typeLabel} · {dateLabel(transaction.occurred_at)}</small>
        </div>
        <span className="tag info">{transaction.transfer_id ? "Transfer" : "Posted"}</span>
      </header>

      <div className="transaction-detail-total">
        <span className={compareMoney(amount, "0") < 0 ? "delta-down" : "delta-up"}>
          {formatMoney(amount, currency)}
        </span>
      </div>

      <dl className="transaction-detail-list">
        <DetailItem label="Date" value={dateLabel(transaction.occurred_at)} />
        <DetailItem label="Type" value={typeLabel} />
        <DetailItem label="Account" value={accountNames.get(transaction.account_id) ?? transaction.account_id} />
        <DetailItem label="Category" value={transaction.category_id ? categoryNames.get(transaction.category_id) ?? transaction.category_id : "-"} />
        <DetailItem label="Description" value={transaction.description || "-"} />
        <DetailItem label="Transaction ID" value={transaction.id} />
        {transaction.related_account_id ? (
          <DetailItem label="Related account" value={accountNames.get(transaction.related_account_id) ?? transaction.related_account_id} />
        ) : null}
        {transaction.transfer_id ? <DetailItem label="Transfer ID" value={transaction.transfer_id} /> : null}
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
