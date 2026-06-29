import { useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, ReceiptText } from "lucide-react";
import { api } from "../../api/client";
import { compareMoney, formatMoney, signedAmount } from "../../api/money";
import type { Account, Category, Transaction } from "../../api/types";
import { apiErrorMessages, errorMessage } from "../../shared/api/query";
import { dateLabel } from "../../shared/date";
import { useI18n } from "../../shared/i18n/useI18n";
import { Button } from "../../components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "../../components/ui/command";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "../../components/ui/dialog";
import { TransactionDetails } from "./components/TransactionDetails";
import { CategoryBadge } from "./components/CategoryBadge";

type QuickFilter = "all" | "month" | "transfers" | "categories";

export function TransactionSearchDialog({
  accounts,
  categories,
  onClose,
}: {
  accounts: Account[];
  categories: Category[];
  onClose: () => void;
}) {
  const { t } = useI18n();
  const errorMessages = apiErrorMessages(t);
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<QuickFilter>("all");
  const [selectedTransaction, setSelectedTransaction] =
    useState<Transaction | null>(null);
  const restoreFocusRef = useRef<HTMLElement | null>(null);
  const transactions = useQuery({
    queryKey: ["transactions"],
    queryFn: () => api.transactions(),
    staleTime: 30_000,
  });

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
  const results = useMemo(
    () =>
      filterTransactions({
        transactions: transactions.data ?? [],
        accountNames,
        accountCurrencies,
        categoryNames,
        query,
        filter,
        typeLabels: t.transactions.types,
      }),
    [
      accountCurrencies,
      accountNames,
      categoryNames,
      filter,
      query,
      t.transactions.types,
      transactions.data,
    ],
  );

  useEffect(() => {
    restoreFocusRef.current =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;

    return () => {
      restoreFocusRef.current?.focus();
    };
  }, []);

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="transaction-search-dialog" showCloseButton>
        <DialogHeader>
          <DialogTitle>
            {selectedTransaction
              ? t.transactions.transactionDetails
              : t.shell.transactionSearch}
          </DialogTitle>
          <DialogDescription>
            {selectedTransaction
              ? t.shell.transactionSearchDetailDescription
              : t.shell.transactionSearchDescription}
          </DialogDescription>
        </DialogHeader>

        {selectedTransaction ? (
          <div className="transaction-search-detail">
            <Button
              type="button"
              variant="outline"
              onClick={() => setSelectedTransaction(null)}
            >
              <ArrowLeft data-icon="inline-start" />
              {t.common.back}
            </Button>
            <TransactionDetails
              transaction={selectedTransaction}
              accounts={accounts}
              categories={categories}
            />
          </div>
        ) : (
          <Command className="transaction-search-layout" shouldFilter={false}>
            <CommandInput
              value={query}
              placeholder={
                filter === "categories"
                  ? t.shell.categorySearchPlaceholder
                  : t.shell.transactionSearchPlaceholder
              }
              onValueChange={setQuery}
            />
            <div className="transaction-search-filters" role="group">
              <FilterButton
                active={filter === "all"}
                label={t.shell.filters.all}
                onClick={() => setFilter("all")}
              />
              <FilterButton
                active={filter === "month"}
                label={t.shell.filters.thisMonth}
                onClick={() => setFilter("month")}
              />
              <FilterButton
                active={filter === "transfers"}
                label={t.shell.filters.transfers}
                onClick={() => setFilter("transfers")}
              />
              <FilterButton
                active={filter === "categories"}
                label={t.shell.filters.categories}
                onClick={() => setFilter("categories")}
              />
            </div>
            <CommandList className="transaction-search-results">
              {transactions.isLoading ? (
                <CommandEmpty>{t.transactions.loadingTransactions}</CommandEmpty>
              ) : null}
              {transactions.error ? (
                <CommandEmpty>
                  {errorMessage(transactions.error, errorMessages)}
                </CommandEmpty>
              ) : null}
              {!transactions.isLoading && !transactions.error ? (
                <CommandGroup heading={t.shell.transactionSearchResults}>
                  {results.map((transaction) => (
                    <TransactionResult
                      key={transaction.id}
                      transaction={transaction}
                      accountName={accountNames.get(transaction.account_id)}
                      categoryName={
                        transaction.category_id
                          ? categoryNames.get(transaction.category_id)
                          : undefined
                      }
                      currency={
                        accountCurrencies.get(transaction.account_id) ?? "RUB"
                      }
                      onSelect={() => setSelectedTransaction(transaction)}
                    />
                  ))}
                </CommandGroup>
              ) : null}
              {!transactions.isLoading &&
              !transactions.error &&
              results.length === 0 ? (
                <CommandEmpty>
                  {filter === "categories"
                    ? t.shell.categorySearchEmpty
                    : t.shell.transactionSearchEmpty}
                </CommandEmpty>
              ) : null}
            </CommandList>
          </Command>
        )}
      </DialogContent>
    </Dialog>
  );
}

function FilterButton({
  active,
  label,
  onClick,
}: {
  active: boolean;
  label: string;
  onClick: () => void;
}) {
  return (
    <Button
      className={active ? "filter-chip is-active" : "filter-chip"}
      type="button"
      variant="ghost"
      aria-pressed={active}
      onClick={onClick}
    >
      {label}
    </Button>
  );
}

function TransactionResult({
  transaction,
  accountName,
  categoryName,
  currency,
  onSelect,
}: {
  transaction: Transaction;
  accountName?: string;
  categoryName?: string;
  currency: string;
  onSelect: () => void;
}) {
  const { t, locale } = useI18n();
  const signed = signedAmount(transaction);
  const isNegative = compareMoney(signed, "0") < 0;
  const title = transaction.description || t.transactions.types[transaction.type];

  return (
    <CommandItem
      className="transaction-search-result"
      value={searchValue(transaction, accountName, categoryName, currency)}
      onSelect={onSelect}
    >
      <ReceiptText aria-hidden="true" />
      <span className="transaction-search-result-main">
        <strong>{title}</strong>
        <small className="transaction-search-result-meta">
          <CategoryBadge
            categoryKey={transaction.category_id ?? "uncategorized"}
            name={categoryName ?? t.common.none}
          />
          <span>
            {accountName ?? transaction.account_id} ·{" "}
            {dateLabel(transaction.occurred_at, locale)}
          </span>
        </small>
      </span>
      {transaction.transfer_id ? (
        <span className="tag info">{t.transactions.transfer}</span>
      ) : null}
      <span
        className={
          isNegative
            ? "transaction-search-amount delta-down"
            : "transaction-search-amount delta-up"
        }
      >
        {formatMoney(signed, currency, locale)}
      </span>
    </CommandItem>
  );
}

function filterTransactions({
  transactions,
  accountNames,
  accountCurrencies,
  categoryNames,
  query,
  filter,
  typeLabels,
}: {
  transactions: Transaction[];
  accountNames: Map<string, string>;
  accountCurrencies: Map<string, string>;
  categoryNames: Map<string, string>;
  query: string;
  filter: QuickFilter;
  typeLabels: Record<Transaction["type"], string>;
}) {
  const normalizedQuery = normalizeSearch(query);
  const monthPrefix = new Date().toISOString().slice(0, 7);

  return transactions
    .filter((transaction) => {
      if (
        filter === "month" &&
        !transaction.occurred_at.startsWith(monthPrefix)
      ) {
        return false;
      }

      if (
        filter === "transfers" &&
        transaction.type !== "transfer_in" &&
        transaction.type !== "transfer_out"
      ) {
        return false;
      }

      const categoryName = transaction.category_id
        ? categoryNames.get(transaction.category_id)
        : undefined;

      if (filter === "categories") {
        if (!categoryName) {
          return false;
        }

        return (
          !normalizedQuery ||
          normalizeSearch(categoryName).includes(normalizedQuery)
        );
      }

      if (!normalizedQuery) {
        return true;
      }

      return normalizeSearch(
        searchValue(
          transaction,
          accountNames.get(transaction.account_id),
          categoryName,
          accountCurrencies.get(transaction.account_id) ?? "",
          typeLabels,
        ),
      ).includes(normalizedQuery);
    })
    .slice(0, 50);
}

function searchValue(
  transaction: Transaction,
  accountName?: string,
  categoryName?: string,
  currency?: string,
  typeLabels?: Record<Transaction["type"], string>,
) {
  return [
    transaction.description,
    transaction.type,
    typeLabels?.[transaction.type],
    transaction.amount,
    currency,
    accountName,
    categoryName,
    transaction.id,
    transaction.transfer_id,
    transaction.related_account_id,
  ]
    .filter(Boolean)
    .join(" ");
}

function normalizeSearch(value: string) {
  return value.trim().toLocaleLowerCase();
}
