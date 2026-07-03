import { useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
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
  const [page, setPage] = useState(1);
  const [selectedTransaction, setSelectedTransaction] =
    useState<Transaction | null>(null);
  const restoreFocusRef = useRef<HTMLElement | null>(null);
  const deferredQuery = useDeferredValue(query);
  const pageSize = 50;
  const transactionFilters = useMemo(() => {
    const month = currentMonthRange();
    return {
      search: deferredQuery || undefined,
      types:
        filter === "transfers"
          ? (["transfer_in", "transfer_out"] as Transaction["type"][])
          : undefined,
      categorized: filter === "categories" || undefined,
      fromDate: filter === "month" ? month.from : undefined,
      toDate: filter === "month" ? month.to : undefined,
      limit: pageSize + 1,
      offset: (page - 1) * pageSize,
    };
  }, [deferredQuery, filter, page]);
  const transactions = useQuery({
    queryKey: ["transactions", "search", transactionFilters],
    queryFn: () => api.transactions(transactionFilters),
    placeholderData: (previous) => previous,
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
  const results = (transactions.data ?? []).slice(0, pageSize);

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
              onValueChange={(value) => {
                setQuery(value);
                setPage(1);
              }}
            />
            <div className="transaction-search-filters" role="group">
              <FilterButton
                active={filter === "all"}
                label={t.shell.filters.all}
                onClick={() => {
                  setFilter("all");
                  setPage(1);
                }}
              />
              <FilterButton
                active={filter === "month"}
                label={t.shell.filters.thisMonth}
                onClick={() => {
                  setFilter("month");
                  setPage(1);
                }}
              />
              <FilterButton
                active={filter === "transfers"}
                label={t.shell.filters.transfers}
                onClick={() => {
                  setFilter("transfers");
                  setPage(1);
                }}
              />
              <FilterButton
                active={filter === "categories"}
                label={t.shell.filters.categories}
                onClick={() => {
                  setFilter("categories");
                  setPage(1);
                }}
              />
            </div>
            <CommandList className="transaction-search-results">
              {transactions.isLoading ? (
                <CommandEmpty>
                  {t.transactions.loadingTransactions}
                </CommandEmpty>
              ) : null}
              {transactions.error ? (
                <div className="query-error search-query-error" role="alert">
                  <span>{errorMessage(transactions.error, errorMessages)}</span>
                  <Button
                    type="button"
                    onClick={() => void transactions.refetch()}
                  >
                    {t.common.retry}
                  </Button>
                </div>
              ) : null}
              {!transactions.isLoading &&
              (!transactions.error || transactions.data) ? (
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
              (!transactions.error || Boolean(transactions.data)) &&
              results.length === 0 ? (
                <CommandEmpty>
                  {filter === "categories"
                    ? t.shell.categorySearchEmpty
                    : t.shell.transactionSearchEmpty}
                </CommandEmpty>
              ) : null}
            </CommandList>
            <div className="transaction-search-pagination">
              <Button
                type="button"
                variant="outline"
                disabled={page === 1 || transactions.isFetching}
                onClick={() => setPage((current) => Math.max(1, current - 1))}
              >
                {t.transactions.previousPage}
              </Button>
              <span aria-live="polite">
                {t.transactions.pageLabel.replace("{page}", String(page))}
              </span>
              <Button
                type="button"
                variant="outline"
                disabled={
                  (transactions.data?.length ?? 0) <= pageSize ||
                  transactions.isFetching
                }
                onClick={() => setPage((current) => current + 1)}
              >
                {t.transactions.nextPage}
              </Button>
            </div>
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
  const title =
    transaction.description || t.transactions.types[transaction.type];

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

function currentMonthRange() {
  const now = new Date();
  const from = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1));
  const to = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 0));
  return {
    from: from.toISOString().slice(0, 10),
    to: to.toISOString().slice(0, 10),
  };
}
