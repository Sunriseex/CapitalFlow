import { useMemo, useState } from "react";
import type { ReactNode } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
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
import { Button as ShadcnButton } from "../../components/ui/button";
import { useI18n } from "../../shared/i18n/useI18n";
import { CategoryPickerDialog } from "./CategoryPickerDialog";

type TransactionFormValues = {
  account_id: string;
  type: TransactionType;
  amount: string;
  category_id: string;
  description: string;
  occurred_at: string;
};

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
  const moneyParseMessages = useMemo(
    () => ({
      amountRequired: t.money.amountRequired,
      amountFormat: (scale: number) =>
        t.money.amountFormat.replace("{scale}", String(scale)),
      amountNonNegative: t.money.amountNonNegative,
      amountGreaterThanZero: t.money.amountGreaterThanZero,
    }),
    [t],
  );

  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [categoryPickerOpen, setCategoryPickerOpen] = useState(false);
  const [subscriptionPromptDismissed, setSubscriptionPromptDismissed] =
    useState(false);
  const formDefaults = useMemo<TransactionFormValues>(() => ({
    account_id: accounts[0]?.id ?? "",
    type: fixedType ?? "income",
    amount: "",
    category_id: "",
    description: "",
    occurred_at: today,
  }), [accounts, fixedType]);
  const formSchema = useMemo(
    () => createTransactionFormSchema(accounts, moneyParseMessages),
    [accounts, moneyParseMessages],
  );
  const {
    clearErrors,
    control,
    formState: { errors },
    handleSubmit,
    register,
    setValue,
  } = useForm<TransactionFormValues>({
    defaultValues: formDefaults,
    mode: "onBlur",
    resolver: zodResolver(formSchema),
  });
  const watchedForm = useWatch({ control });
  const form = { ...formDefaults, ...watchedForm };
  const accountOptions = useMemo(
    () =>
      accounts.map((account) => (
        <option key={account.id} value={account.id}>
          {account.name}
        </option>
      )),
    [accounts],
  );
  const selectedCategory = useMemo(
    () => categories.find((category) => category.id === form.category_id),
    [categories, form.category_id],
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
    mutationFn: (values: TransactionFormValues) => {
      const transactionType = values.type;
      const submittedAccount = accounts.find(
        (account) => account.id === values.account_id,
      );
      const amount = parseMoneyToMinorResult(values.amount, {
        required: true,
        positive: transactionType !== "adjustment",
        allowNegative: transactionType === "adjustment",
        currency: submittedAccount?.currency ?? "RUB",
        messages: moneyParseMessages,
      });
      if (!amount.ok) {
        throw new Error(amount.error);
      }

      return api.createTransaction({
        account_id: values.account_id,
        type: transactionType,
        amount: amount.value,
        category_id: values.category_id || null,
        description: values.description,
        occurred_at: values.occurred_at,
      });
    },
    onSuccess: () => {
      invalidateMoney(queryClient);
      onDone();
    },
    onError: (err) => setError(errorMessage(err, errorMessages)),
  });

  const transactionType = form.type as TransactionType;
  const showSubscriptionPrompt =
    transactionType === "expense" &&
    selectedCategory &&
    isSubscriptionCategory(selectedCategory) &&
    !subscriptionPromptDismissed;
  const title = t.transactions.createTypedTransaction.replace(
    "{type}",
    t.transactions.types[transactionType].toLowerCase(),
  );
  const submitForm = handleSubmit((values) => {
    setError("");
    mutation.mutate(values);
  });

  return (
    <FormShell
      title={title}
      error={error}
      onSubmit={submitForm}
      showTitle={showTitle}
    >
      <Field label={t.transactions.account}>
        <Select
          {...register("account_id", {
            onChange: () => clearErrors("amount"),
          })}
        >
          {accountOptions}
        </Select>
      </Field>

      {!fixedType ? (
        <Field label={t.transactions.type}>
          <Select
            {...register("type", {
              onChange: () => clearErrors("amount"),
            })}
          >
            {typeOptions}
          </Select>
        </Field>
      ) : null}

      <ValidatedField
        error={errors.amount?.message}
        errorId="transaction-amount-error"
        label={t.transactions.amount}
      >
        <Input
          aria-describedby={
            errors.amount ? "transaction-amount-error" : undefined
          }
          aria-label={t.transactions.amount}
          aria-invalid={Boolean(errors.amount)}
          required
          inputMode="decimal"
          {...register("amount", {
            onChange: () => clearErrors("amount"),
          })}
        />
        {form.type === "adjustment" ? (
          <small className="field-hint">
            {t.transactions.adjustmentAmountHint}
          </small>
        ) : null}
      </ValidatedField>

      <div className="field category-picker-field">
        <span>{t.transactions.category}</span>
        <ShadcnButton
          className="category-picker-trigger"
          type="button"
          variant="outline"
          aria-haspopup="dialog"
          onClick={() => setCategoryPickerOpen(true)}
        >
          <strong>{selectedCategory?.name ?? t.common.none}</strong>
          <small>{t.transactions.categoryPickerTriggerHint}</small>
        </ShadcnButton>
      </div>

      {showSubscriptionPrompt ? (
        <div className="subscription-suggestion" role="status">
          <strong>{t.transactions.subscriptionPromptTitle}</strong>
          <p>{t.transactions.subscriptionPromptDescription}</p>
          <div>
            <Button type="button" disabled title={t.common.notAvailable}>
              {t.transactions.createSubscription}
            </Button>
            <Button type="button" disabled title={t.common.notAvailable}>
              {t.transactions.linkSubscription}
            </Button>
            <Button
              type="button"
              onClick={() => setSubscriptionPromptDismissed(true)}
            >
              {t.transactions.notNow}
            </Button>
          </div>
        </div>
      ) : null}

      <Field label={t.transactions.date}>
        <Input type="date" {...register("occurred_at")} />
      </Field>

      <Field label={t.transactions.description}>
        <Input {...register("description")} />
      </Field>

      <Button disabled={mutation.isPending}>{t.common.create}</Button>
      {categoryPickerOpen ? (
        <CategoryPickerDialog
          categories={categories}
          selectedCategoryId={form.category_id}
          onClose={() => setCategoryPickerOpen(false)}
          onSelect={(categoryId) => {
            setSubscriptionPromptDismissed(false);
            setValue("category_id", categoryId, {
              shouldDirty: true,
              shouldValidate: true,
            });
          }}
        />
      ) : null}
    </FormShell>
  );
}

function createTransactionFormSchema(
  accounts: Account[],
  moneyParseMessages: {
    amountRequired: string;
    amountFormat: (scale: number) => string;
    amountNonNegative: string;
    amountGreaterThanZero: string;
  },
) {
  return z
    .object({
      account_id: z.string(),
      type: z.enum(transactionTypes),
      amount: z.string(),
      category_id: z.string(),
      description: z.string(),
      occurred_at: z.string(),
    })
    .superRefine((values, context) => {
      const selectedAccount = accounts.find(
        (account) => account.id === values.account_id,
      );
      const amount = parseMoneyToMinorResult(values.amount, {
        required: true,
        positive: values.type !== "adjustment",
        allowNegative: values.type === "adjustment",
        currency: selectedAccount?.currency ?? "RUB",
        messages: moneyParseMessages,
      });

      if (!amount.ok) {
        context.addIssue({
          code: "custom",
          message: amount.error,
          path: ["amount"],
        });
      }
    });
}

function ValidatedField({
  children,
  error,
  errorId,
  label,
}: {
  children: ReactNode;
  error?: string;
  errorId: string;
  label: string;
}) {
  return (
    <div className="field">
      <label className="field-control">
        <span>{label}</span>
        {children}
      </label>
      {error ? (
        <span className="field-error" id={errorId}>
          {error}
        </span>
      ) : null}
    </div>
  );
}

function isSubscriptionCategory(category: Category) {
  const value = `${category.name} ${category.slug}`.toLocaleLowerCase();
  return (
    value.includes("subscription") ||
    value.includes("subscriptions") ||
    value.includes("подпис")
  );
}
