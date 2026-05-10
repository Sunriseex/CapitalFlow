import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import { parseMoneyToMinor } from "../../api/money";
import type { Account } from "../../api/types";
import { errorMessage, invalidateMoney } from "../../shared/api/query";
import { Button, Field, FormShell, Input, Select } from "../../shared/ui";

export function TransferForm({ accounts, onDone }: { accounts: Account[]; onDone: () => void }) {
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    from_account_id: accounts[0]?.id ?? "",
    to_account_id: accounts[1]?.id ?? "",
    amount: "",
    description: "",
  });
  const mutation = useMutation({
    mutationFn: () => api.createTransfer({
      from_account_id: form.from_account_id,
      to_account_id: form.to_account_id,
      amount_minor: parseMoneyToMinor(form.amount),
      description: form.description,
    }),
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
      <Field label="Description"><Input value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} /></Field>
      <Button disabled={mutation.isPending}>Create</Button>
    </FormShell>
  );
}

