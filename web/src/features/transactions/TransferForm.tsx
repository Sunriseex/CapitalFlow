import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import { convertAmount, formatMoney, isPositiveMoney, parseMoneyToMinorResult } from "../../api/money";
import type { Account } from "../../api/types";
import { errorMessage, invalidateMoney } from "../../shared/api/query";
import { Button, Empty, Field, FormShell, Input, Select } from "../../shared/ui";

export function TransferForm({ accounts, onDone }: { accounts: Account[]; onDone: () => void }) {
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    from_account_id: accounts[0]?.id ?? "",
    to_account_id: accounts[1]?.id ?? "",
    amount: "",
    fee_amount: "",
    description: "",
  });
  const fromAccount = accounts.find((account) => account.id === form.from_account_id);
  const toAccount = accounts.find((account) => account.id === form.to_account_id);
  const previewAmount = parseMoneyToMinorResult(form.amount, { currency: fromAccount?.currency ?? "RUB" });
  const amount = previewAmount.ok ? previewAmount.value : "0";
  const rates = useQuery({
    queryKey: ["currency-rates", fromAccount?.currency],
    queryFn: () => api.currencyRates(fromAccount?.currency ?? "RUB"),
    enabled: Boolean(fromAccount?.currency && toAccount?.currency && fromAccount.currency !== toAccount.currency),
    staleTime: 1000 * 60 * 60,
  });
  const needsConversion = Boolean(fromAccount && toAccount && fromAccount.currency !== toAccount.currency);
  const rate = toAccount?.currency ? rates.data?.rates[toAccount.currency] : undefined;
  const convertedAmount = isPositiveMoney(amount) && rate ? convertAmount(amount, fromAccount?.currency ?? "RUB", toAccount?.currency ?? "RUB", {
    base: toAccount?.currency ?? "RUB",
    date: "",
    provider: "",
    fetched_at: "",
    rates: { [fromAccount?.currency ?? "RUB"]: 1 / rate },
  }) : "0";
  const cannotConvert = needsConversion && (!rate || rates.isLoading || Boolean(rates.error));
  const mutation = useMutation({
    mutationFn: () => {
      const amount = parseMoneyToMinorResult(form.amount, { required: true, positive: true, currency: fromAccount?.currency ?? "RUB" });
      if (!amount.ok) {
        throw new Error(amount.error);
      }
      const feeAmount = parseMoneyToMinorResult(form.fee_amount, { currency: fromAccount?.currency ?? "RUB" });
      if (!feeAmount.ok) {
        throw new Error(feeAmount.error);
      }

      return api.createTransfer({
        from_account_id: form.from_account_id,
        to_account_id: form.to_account_id,
        amount: amount.value,
        ...(isPositiveMoney(feeAmount.value) ? { fee_amount: feeAmount.value } : {}),
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
    <FormShell title="Create transfer" error={error} onSubmit={() => mutation.mutate()}>
      <Field label="From"><Select value={form.from_account_id} onChange={(event) => setForm({ ...form, from_account_id: event.target.value })}>{accounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}</Select></Field>
      <Field label="To"><Select value={form.to_account_id} onChange={(event) => setForm({ ...form, to_account_id: event.target.value })}>{accounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}</Select></Field>
      <Field label="Amount"><Input required inputMode="decimal" value={form.amount} onChange={(event) => setForm({ ...form, amount: event.target.value })} /></Field>
      <Field label="Fee"><Input inputMode="decimal" value={form.fee_amount} onChange={(event) => setForm({ ...form, fee_amount: event.target.value })} /></Field>
      {needsConversion && fromAccount && toAccount ? (
        <div className="conversion-preview">
          <span>{fromAccount.currency} to {toAccount.currency}</span>
          {rates.isLoading ? <strong>Loading rate</strong> : null}
          {rate ? (
            <strong>
              {formatMoney(amount, fromAccount.currency)} = {formatMoney(convertedAmount, toAccount.currency)}
            </strong>
          ) : null}
          {rates.error ? <Empty>{errorMessage(rates.error)}</Empty> : null}
        </div>
      ) : null}
      <Field label="Description"><Input value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} /></Field>
      <Button disabled={mutation.isPending || cannotConvert}>Create</Button>
    </FormShell>
  );
}


