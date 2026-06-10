import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import {
  convertAmount,
  formatMoney,
  isPositiveMoney,
  parseMoneyToMinorResult,
} from "../../api/money";
import type { Account } from "../../api/types";
import { errorMessage, invalidateMoney } from "../../shared/api/query";
import {
  Button,
  Empty,
  Field,
  FormShell,
  Input,
  Select,
} from "../../shared/ui";
import { useI18n } from "../../shared/i18n/useI18n";

export function TransferForm({
  accounts,
  onDone,
}: {
  accounts: Account[];
  onDone: () => void;
}) {
  const { t, locale } = useI18n();

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
  const [form, setForm] = useState({
    from_account_id: accounts[0]?.id ?? "",
    to_account_id: accounts[1]?.id ?? "",
    amount: "",
    fee_amount: "",
    description: "",
  });

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
    mutationFn: () => {
      const amount = parseMoneyToMinorResult(form.amount, {
        required: true,
        positive: true,
        currency: fromAccount?.currency ?? "RUB",
        messages: moneyParseMessages,
      });

      if (!amount.ok) {
        throw new Error(amount.error);
      }

      const feeAmount = parseMoneyToMinorResult(form.fee_amount, {
        currency: fromAccount?.currency ?? "RUB",
        messages: moneyParseMessages,
      });

      if (!feeAmount.ok) {
        throw new Error(feeAmount.error);
      }

      return api.createTransfer({
        from_account_id: form.from_account_id,
        to_account_id: form.to_account_id,
        amount: amount.value,
        ...(isPositiveMoney(feeAmount.value)
          ? { fee_amount: feeAmount.value }
          : {}),
        description: form.description,
      });
    },
    onSuccess: () => {
      invalidateMoney(queryClient);
      onDone();
    },
    onError: (err) => setError(errorMessage(err)),
  });

  return (
    <FormShell
      title={t.transfers.createTransfer}
      error={error}
      onSubmit={() => mutation.mutate()}
    >
      <Field label={t.transfers.from}>
        <Select
          value={form.from_account_id}
          onChange={(event) =>
            setForm({ ...form, from_account_id: event.target.value })
          }
        >
          {accountOptions}
        </Select>
      </Field>

      <Field label={t.transfers.to}>
        <Select
          value={form.to_account_id}
          onChange={(event) =>
            setForm({ ...form, to_account_id: event.target.value })
          }
        >
          {accountOptions}
        </Select>
      </Field>

      <Field label={t.transactions.amount}>
        <Input
          aria-label={t.transactions.amount}
          required
          inputMode="decimal"
          value={form.amount}
          onChange={(event) => setForm({ ...form, amount: event.target.value })}
        />
      </Field>

      <Field label={t.transfers.fee}>
        <Input
          aria-label={t.transfers.fee}
          inputMode="decimal"
          value={form.fee_amount}
          onChange={(event) =>
            setForm({ ...form, fee_amount: event.target.value })
          }
        />
      </Field>

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

          {rates.error ? <Empty>{errorMessage(rates.error)}</Empty> : null}
        </div>
      ) : null}

      <Field label={t.transactions.description}>
        <Input
          value={form.description}
          onChange={(event) =>
            setForm({ ...form, description: event.target.value })
          }
        />
      </Field>

      <Button disabled={mutation.isPending || cannotConvert}>
        {t.common.create}
      </Button>
    </FormShell>
  );
}
