import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import { parseMoneyToMinorResult } from "../../api/money";
import type { Account, Category, TransactionType } from "../../api/types";
import { errorMessage, invalidateMoney } from "../../shared/api/query";
import { today, transactionTypes } from "../../shared/constants";
import { Button, Field, FormShell, Input, Select } from "../../shared/ui";

export function TransactionForm({ accounts, categories, fixedType, onDone }: { accounts: Account[]; categories: Category[]; fixedType?: TransactionType; onDone: () => void }) {
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
    () => accounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>),
    [accounts],
  );
  const categoryOptions = useMemo(
    () => categories.map((category) => <option key={category.id} value={category.id}>{category.name}</option>),
    [categories],
  );
  const typeOptions = useMemo(
    () => transactionTypes.map((type) => <option key={type}>{type}</option>),
    [],
  );
  const mutation = useMutation({
    mutationFn: () => {
      const transactionType = form.type as TransactionType;
      const amount = parseMoneyToMinorResult(form.amount, {
        required: true,
        positive: transactionType !== "adjustment",
        allowNegative: transactionType === "adjustment",
        currency: selectedAccount?.currency ?? "RUB",
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
    onError: (err) => setError(errorMessage(err)),
  });

  return (
    <FormShell title={`Create ${form.type}`} error={error} onSubmit={() => mutation.mutate()}>
      <Field label="Account"><Select value={form.account_id} onChange={(event) => setForm({ ...form, account_id: event.target.value })}>{accountOptions}</Select></Field>
      {!fixedType ? <Field label="Type"><Select value={form.type} onChange={(event) => setForm({ ...form, type: event.target.value as TransactionType })}>{typeOptions}</Select></Field> : null}
      <Field label="Amount"><Input required inputMode="decimal" value={form.amount} onChange={(event) => setForm({ ...form, amount: event.target.value })} /></Field>
      <Field label="Category"><Select value={form.category_id} onChange={(event) => setForm({ ...form, category_id: event.target.value })}><option value="">None</option>{categoryOptions}</Select></Field>
      <Field label="Date"><Input type="date" value={form.occurred_at} onChange={(event) => setForm({ ...form, occurred_at: event.target.value })} /></Field>
      <Field label="Description"><Input value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} /></Field>
      <Button disabled={mutation.isPending}>Create</Button>
    </FormShell>
  );
}


