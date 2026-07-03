import { useMemo, useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { api } from "../../api/client";
import {
  convertAmount,
  formatMoney,
  isPositiveMoney,
  parseMoneyToMinorResult,
} from "../../api/money";
import type { Account } from "../../api/types";
import {
  apiErrorMessages,
  errorMessage,
  invalidateMoney,
} from "../../shared/api/query";
import {
  Button,
  Empty,
  Field,
  FormShell,
  Input,
  Select,
  ValidatedField,
} from "../../shared/ui";
import { useI18n } from "../../shared/i18n/useI18n";

type TransferFormValues = {
  from_account_id: string;
  to_account_id: string;
  amount: string;
  fee_amount: string;
  description: string;
};

export function TransferForm({
  accounts,
  onDone,
}: {
  accounts: Account[];
  onDone: () => void;
}) {
  const { t, locale } = useI18n();
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
  const formDefaults = useMemo<TransferFormValues>(
    () => ({
      from_account_id: accounts[0]?.id ?? "",
      to_account_id: accounts[1]?.id ?? "",
      amount: "",
      fee_amount: "",
      description: "",
    }),
    [accounts],
  );
  const formSchema = useMemo(
    () => createTransferFormSchema(accounts, moneyParseMessages),
    [accounts, moneyParseMessages],
  );
  const {
    clearErrors,
    control,
    formState: { errors },
    handleSubmit,
    register,
  } = useForm<TransferFormValues>({
    defaultValues: formDefaults,
    mode: "onBlur",
    resolver: zodResolver(formSchema),
  });
  const watchedForm = useWatch({ control });
  const form = { ...formDefaults, ...watchedForm };

  const fromAccount = useMemo(
    () => accounts.find((account) => account.id === form.from_account_id),
    [accounts, form.from_account_id],
  );

  const toAccount = useMemo(
    () => accounts.find((account) => account.id === form.to_account_id),
    [accounts, form.to_account_id],
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

  const previewAmount = useMemo(
    () =>
      parseMoneyToMinorResult(form.amount, {
        currency: fromAccount?.currency ?? "RUB",
        messages: moneyParseMessages,
      }),
    [form.amount, fromAccount?.currency, moneyParseMessages],
  );
  const amount = previewAmount.ok ? previewAmount.value : "0";
  const hasAmount = form.amount.trim().length > 0;
  const rates = useQuery({
    queryKey: ["currency-rates", fromAccount?.currency],
    queryFn: () => api.currencyRates(fromAccount?.currency ?? "RUB"),
    enabled: Boolean(
      hasAmount &&
      fromAccount?.currency &&
      toAccount?.currency &&
      fromAccount.currency !== toAccount.currency,
    ),
    staleTime: 1000 * 60 * 60,
  });
  const needsConversion = Boolean(
    fromAccount && toAccount && fromAccount.currency !== toAccount.currency,
  );
  const rate = toAccount?.currency
    ? rates.data?.rates[toAccount.currency]
    : undefined;
  const convertedAmount = useMemo(
    () =>
      isPositiveMoney(amount) && rate
        ? convertAmount(
            amount,
            fromAccount?.currency ?? "RUB",
            toAccount?.currency ?? "RUB",
            {
              base: toAccount?.currency ?? "RUB",
              date: "",
              provider: "",
              fetched_at: "",
              rates: { [fromAccount?.currency ?? "RUB"]: 1 / rate },
            },
          )
        : "0",
    [amount, fromAccount?.currency, rate, toAccount?.currency],
  );
  const cannotConvert =
    needsConversion && (!rate || rates.isLoading || Boolean(rates.error));
  const mutation = useMutation({
    mutationFn: (values: TransferFormValues) => {
      const submittedFromAccount = accounts.find(
        (account) => account.id === values.from_account_id,
      );
      const amount = parseMoneyToMinorResult(values.amount, {
        required: true,
        positive: true,
        currency: submittedFromAccount?.currency ?? "RUB",
        messages: moneyParseMessages,
      });

      if (!amount.ok) {
        throw new Error(amount.error);
      }

      const feeAmount = parseMoneyToMinorResult(values.fee_amount, {
        currency: submittedFromAccount?.currency ?? "RUB",
        messages: moneyParseMessages,
      });

      if (!feeAmount.ok) {
        throw new Error(feeAmount.error);
      }

      return api.createTransfer({
        from_account_id: values.from_account_id,
        to_account_id: values.to_account_id,
        amount: amount.value,
        ...(isPositiveMoney(feeAmount.value)
          ? { fee_amount: feeAmount.value }
          : {}),
        description: values.description,
      });
    },
    onSuccess: () => {
      invalidateMoney(queryClient);
      onDone();
    },
    onError: (err) => setError(errorMessage(err, errorMessages)),
  });
  const submitForm = handleSubmit((values) => {
    setError("");
    mutation.mutate(values);
  });

  return (
    <FormShell
      title={t.transfers.createTransfer}
      error={error}
      onSubmit={submitForm}
    >
      <Field label={t.transfers.from}>
        <Select
          {...register("from_account_id", {
            onChange: () => clearErrors(["amount", "fee_amount"]),
          })}
        >
          {accountOptions}
        </Select>
      </Field>

      <Field label={t.transfers.to}>
        <Select {...register("to_account_id")}>{accountOptions}</Select>
      </Field>

      <ValidatedField
        error={errors.amount?.message}
        errorId="transfer-amount-error"
        label={t.transactions.amount}
      >
        <Input
          aria-describedby={errors.amount ? "transfer-amount-error" : undefined}
          aria-label={t.transactions.amount}
          aria-invalid={Boolean(errors.amount)}
          required
          inputMode="decimal"
          {...register("amount", {
            onChange: () => clearErrors("amount"),
          })}
        />
      </ValidatedField>

      <ValidatedField
        error={errors.fee_amount?.message}
        errorId="transfer-fee-error"
        label={t.transfers.fee}
      >
        <Input
          aria-describedby={
            errors.fee_amount ? "transfer-fee-error" : undefined
          }
          aria-label={t.transfers.fee}
          aria-invalid={Boolean(errors.fee_amount)}
          inputMode="decimal"
          {...register("fee_amount", {
            onChange: () => clearErrors("fee_amount"),
          })}
        />
      </ValidatedField>

      {needsConversion && fromAccount && toAccount ? (
        <div className="conversion-preview">
          <span>
            {t.transfers.conversionPair
              .replace("{fromCurrency}", fromAccount.currency)
              .replace("{toCurrency}", toAccount.currency)}
          </span>

          {rates.isLoading ? <strong>{t.transfers.loadingRate}</strong> : null}

          {rate ? (
            <strong>
              {formatMoney(amount, fromAccount.currency, locale)} ={" "}
              {formatMoney(convertedAmount, toAccount.currency, locale)}
            </strong>
          ) : null}

          {rates.error ? (
            <Empty>{errorMessage(rates.error, errorMessages)}</Empty>
          ) : null}
        </div>
      ) : null}

      <Field label={t.transactions.description}>
        <Input {...register("description")} />
      </Field>

      <Button disabled={mutation.isPending || cannotConvert}>
        {t.common.create}
      </Button>
    </FormShell>
  );
}

function createTransferFormSchema(
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
      from_account_id: z.string(),
      to_account_id: z.string(),
      amount: z.string(),
      fee_amount: z.string(),
      description: z.string(),
    })
    .superRefine((values, context) => {
      const fromAccount = accounts.find(
        (account) => account.id === values.from_account_id,
      );
      const amount = parseMoneyToMinorResult(values.amount, {
        required: true,
        positive: true,
        currency: fromAccount?.currency ?? "RUB",
        messages: moneyParseMessages,
      });
      const feeAmount = parseMoneyToMinorResult(values.fee_amount, {
        currency: fromAccount?.currency ?? "RUB",
        messages: moneyParseMessages,
      });

      if (!amount.ok) {
        context.addIssue({
          code: "custom",
          message: amount.error,
          path: ["amount"],
        });
      }

      if (!feeAmount.ok) {
        context.addIssue({
          code: "custom",
          message: feeAmount.error,
          path: ["fee_amount"],
        });
      }
    });
}
