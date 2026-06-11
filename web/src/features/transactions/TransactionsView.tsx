import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus, ReceiptText } from "lucide-react";
import { api } from "../../api/client";
import type { Account, Category, Transaction } from "../../api/types";
import { apiErrorMessages, errorMessage } from "../../shared/api/query";
import { transactionTypes } from "../../shared/constants";
import {
  Button,
  Dialog,
  Empty,
  EmptyState,
  Input,
  Panel,
  Select,
} from "../../shared/ui";
import { TransactionDetails } from "./components/TransactionDetails";
import { TransactionsTable } from "./components/TransactionsTable";
import { TransactionForm } from "./TransactionForm";
import { useI18n } from "../../shared/i18n/useI18n";

export function TransactionsView({
  accounts,
  categories,
  accountsLoading = false,
  accountsError = null,
  categoriesLoading = false,
  categoriesError = null,
  onCreateTransaction,
  onImport,
}: {
  accounts: Account[];
  categories: Category[];
  accountsLoading?: boolean;
  accountsError?: unknown;
  categoriesLoading?: boolean;
  categoriesError?: unknown;
  onCreateTransaction?: () => void;
  onImport?: () => void;
}) {
  const { t } = useI18n();
  const errorMessages = apiErrorMessages(t);
  const transactions = useQuery({
    queryKey: ["transactions"],
    queryFn: () => api.transactions(),
    staleTime: 30_000,
  });
  const [createOpen, setCreateOpen] = useState(false);
  const [selectedTransaction, setSelectedTransaction] =
    useState<Transaction | null>(null);
  const [accountId, setAccountId] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [type, setType] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const filtered = useMemo(
    () =>
      (transactions.data ?? []).filter((transaction) => {
        const day = transaction.occurred_at.slice(0, 10);
        return (
          (!accountId || transaction.account_id === accountId) &&
          (!categoryId || transaction.category_id === categoryId) &&
          (!type || transaction.type === type) &&
          (!from || day >= from) &&
          (!to || day <= to)
        );
      }),
    [accountId, categoryId, from, to, transactions.data, type],
  );
  const accountOptions = useMemo(
    () =>
      accounts.map((account) => (
        <option key={account.id} value={account.id}>
          {account.name}
        </option>
      )),
    [accounts],
  );
  const categoryOptions = useMemo(
    () =>
      categories.map((category) => (
        <option key={category.id} value={category.id}>
          {category.name}
        </option>
      )),
    [categories],
  );
  const typeOptions = useMemo(
    () =>
      transactionTypes.map((transactionType) => (
        <option key={transactionType} value={transactionType}>
          {t.transactions.types[transactionType]}
        </option>
      )),
    [t],
  );

  const disabledCreate =
    accountsLoading || Boolean(accountsError) || accounts.length === 0;

  return (
    <Panel
      className="workspace-panel transactions-panel"
      title={t.transactions.title}
      action={
        <Button onClick={() => setCreateOpen(true)} disabled={disabledCreate}>
          <Plus size={16} /> {t.transactions.adjustment}
        </Button>
      }
    >
      {accountsLoading ? <Empty>{t.accounts.loadingAccounts}</Empty> : null}{" "}
      {accountsError ? (
        <div className="error inline-error">
          {errorMessage(accountsError, errorMessages)}
        </div>
      ) : null}
      {categoriesLoading ? (
        <Empty>{t.transactions.loadingCategories}</Empty>
      ) : null}{" "}
      {categoriesError ? (
        <div className="error inline-error">
          {errorMessage(categoriesError, errorMessages)}
        </div>
      ) : null}
      {transactions.isLoading ? (
        <Empty>{t.transactions.loadingTransactions}</Empty>
      ) : null}{" "}
      {transactions.error ? (
        <div className="error inline-error">
          {errorMessage(transactions.error, errorMessages)}
        </div>
      ) : null}
      <div className="filters workspace-filters transactions-filters">
        <Select
          aria-label={t.transactions.filterByAccount}
          value={accountId}
          disabled={accountsLoading || Boolean(accountsError)}
          onChange={(event) => setAccountId(event.target.value)}
        >
          <option value="">{t.transactions.allAccounts}</option>
          {accountOptions}
        </Select>

        <Select
          aria-label={t.transactions.filterByCategory}
          value={categoryId}
          disabled={categoriesLoading || Boolean(categoriesError)}
          onChange={(event) => setCategoryId(event.target.value)}
        >
          <option value="">{t.transactions.allCategories}</option>
          {categoryOptions}
        </Select>

        <Select
          aria-label={t.transactions.filterByType}
          value={type}
          onChange={(event) => setType(event.target.value)}
        >
          <option value="">{t.accounts.allTypes}</option>
          {typeOptions}
        </Select>

        <Input
          aria-label={t.transactions.filterFromDate}
          type="date"
          value={from}
          onChange={(event) => setFrom(event.target.value)}
        />

        <Input
          aria-label={t.transactions.filterToDate}
          type="date"
          value={to}
          onChange={(event) => setTo(event.target.value)}
        />
      </div>
      {!transactions.isLoading &&
      !transactions.error &&
      (transactions.data?.length ?? 0) === 0 ? (
        <EmptyState
          icon={<ReceiptText aria-hidden="true" />}
          title={t.transactions.emptyTitle}
          description={t.transactions.emptyDescription}
          primaryAction={
            onCreateTransaction
              ? {
                  label: t.dashboard.addTransaction,
                  onClick: onCreateTransaction,
                  disabled: disabledCreate,
                }
              : undefined
          }
          secondaryAction={
            onImport
              ? { label: t.dashboard.importTransactions, onClick: onImport }
              : undefined
          }
        />
      ) : null}
      {!transactions.isLoading &&
      !transactions.error &&
      (transactions.data?.length ?? 0) > 0 ? (
        <TransactionsTable
          transactions={filtered}
          accounts={accounts}
          categories={categories}
          chunked
          onOpenTransaction={setSelectedTransaction}
        />
      ) : null}
      {createOpen ? (
        <Dialog
          title={t.transactions.createAdjustment}
          onClose={() => setCreateOpen(false)}
        >
          {" "}
          <TransactionForm
            accounts={accounts}
            categories={categories}
            fixedType="adjustment"
            showTitle={false}
            onDone={() => setCreateOpen(false)}
          />
        </Dialog>
      ) : null}
      {selectedTransaction ? (
        <Dialog
          title={t.transactions.transactionDetails}
          onClose={() => setSelectedTransaction(null)}
        >
          <TransactionDetails
            transaction={selectedTransaction}
            accounts={accounts}
            categories={categories}
          />
        </Dialog>
      ) : null}
    </Panel>
  );
}
