import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import { parseMoneyToMinorResult } from "../../api/money";
import type { Account, Category, TransactionType } from "../../api/types";
import {
  apiErrorMessages,
  errorMessage,
  invalidateMoney,
} from "../../shared/api/query";
import { today, transactionTypes } from "../../shared/constants";
import { Button, Field, FormShell, Input, Select } from "../../shared/ui";
import { useI18n } from "../../shared/i18n/useI18n";

export function TransactionForm({
  accounts,
  categories,
  fixedType,
  onDone,
  showTitle = true,
}: {
  accounts: Account[];
  categories: Category[];
  fixedType?: TransactionType;
  onDone: () => void;
  showTitle?: boolean;
}) {
  const { t } = useI18n();
  const errorMessages = apiErrorMessages(t);
  const moneyParseMessages = {
    amountRequired: t.money.amountRequired,
    amountFormat: (scale: number) =>
      t.money.amountFormat.replace("{scale}", String(scale)),
    amountNonNegative: t.money.amountNonNegative,
    amountGreaterThanZero: t.money.amountGreaterThanZero,
  };

  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    account_id: accounts[0]?.id ?? "",
    type: fixedType ?? "income",
    amount: "",
    category_id: "",
    description: "",
    occurred_at: today,
  });
  const selectedAccount = useMemo(
    () => accounts.find((account) => account.id === form.account_id),
    [accounts, form.account_id],
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
      transactionTypes.map((type) => (
        <option key={type} value={type}>
          {t.transactions.types[type]}
        </option>
      )),
    [t],
  );
  const mutation = useMutation({
    mutationFn: () => {
      const transactionType = form.type as TransactionType;
      const amount = parseMoneyToMinorResult(form.amount, {
        required: true,
        positive: transactionType !== "adjustment",
        allowNegative: transactionType === "adjustment",
        currency: selectedAccount?.currency ?? "RUB",
        messages: moneyParseMessages,
      });
      if (!amount.ok) {
        throw new Error(amount.error);
      }

      return api.createTransaction({
        account_id: form.account_id,
        type: transactionType,
        amount: amount.value,
        category_id: form.category_id || null,
        description: form.description,
        occurred_at: form.occurred_at,
      });
    },
    onSuccess: () => {
      invalidateMoney(queryClient);
      onDone();
    },
    onError: (err) => setError(errorMessage(err, errorMessages)),
  });

  const transactionType = form.type as TransactionType;
  const title = t.transactions.createTypedTransaction.replace(
    "{type}",
    t.transactions.types[transactionType].toLowerCase(),
  );

  return (
    <FormShell
      title={title}
      error={error}
      onSubmit={() => mutation.mutate()}
      showTitle={showTitle}
    >
      <Field label={t.transactions.account}>
        <Select
          value={form.account_id}
          onChange={(event) =>
            setForm({ ...form, account_id: event.target.value })
          }
        >
          {accountOptions}
        </Select>
      </Field>

      {!fixedType ? (
        <Field label={t.transactions.type}>
          <Select
            value={form.type}
            onChange={(event) =>
              setForm({ ...form, type: event.target.value as TransactionType })
            }
          >
            {typeOptions}
          </Select>
        </Field>
      ) : null}

      <Field label={t.transactions.amount}>
        <Input
          aria-label={t.transactions.amount}
          required
          inputMode="decimal"
          value={form.amount}
          onChange={(event) => setForm({ ...form, amount: event.target.value })}
        />
        {form.type === "adjustment" ? (
          <small className="field-hint">
            {t.transactions.adjustmentAmountHint}
          </small>
        ) : null}
      </Field>

      <Field label={t.transactions.category}>
        <Select
          value={form.category_id}
          onChange={(event) =>
            setForm({ ...form, category_id: event.target.value })
          }
        >
          <option value="">{t.common.none}</option>
          {categoryOptions}
        </Select>
      </Field>

      <Field label={t.transactions.date}>
        <Input
          type="date"
          value={form.occurred_at}
          onChange={(event) =>
            setForm({ ...form, occurred_at: event.target.value })
          }
        />
      </Field>

      <Field label={t.transactions.description}>
        <Input
          value={form.description}
          onChange={(event) =>
            setForm({ ...form, description: event.target.value })
          }
        />
      </Field>

      <Button disabled={mutation.isPending}>{t.common.create}</Button>
    </FormShell>
  );
}
