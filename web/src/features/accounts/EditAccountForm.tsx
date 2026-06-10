import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import type { Account, AccountType } from "../../api/types";
import {
  apiErrorMessages,
  errorMessage,
  invalidateMoney,
} from "../../shared/api/query";
import { accountTypes } from "../../shared/constants";
import { currencyOptions } from "../../shared/currencies";
import { Button, Field, FormShell, Input, Select } from "../../shared/ui";
import { useI18n } from "../../shared/i18n/useI18n";

export function EditAccountForm({
  account,
  onDone,
}: {
  account: Account;
  onDone: () => void;
}) {
  const { t, locale } = useI18n();
  const errorMessages = apiErrorMessages(t);

  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    name: account.name,
    bank: account.bank ?? "",
    type: account.type,
    currency: account.currency,
    opened_at: account.opened_at.slice(0, 10),
    is_active: account.is_active,
  });
  const mutation = useMutation({
    mutationFn: () => api.updateAccount(account.id, form),
    onSuccess: () => {
      invalidateMoney(queryClient);
      onDone();
    },
    onError: (err) => setError(errorMessage(err, errorMessages)),
  });
  const currencies = useMemo(() => currencyOptions(locale), [locale]);
  const accountTypeOptions = useMemo(
    () =>
      accountTypes.map((type) => (
        <option key={type} value={type}>
          {t.accounts.types[type]}
        </option>
      )),
    [t],
  );
  const currencySelectOptions = useMemo(
    () =>
      currencies.map((currency) => (
        <option key={currency.code} value={currency.code}>
          {currency.label}
        </option>
      )),
    [currencies],
  );

  return (
    <FormShell
      title={t.accounts.editAccount}
      error={error}
      onSubmit={() => mutation.mutate()}
    >
      <Field label={t.accounts.name}>
        <Input
          required
          value={form.name}
          onChange={(event) => setForm({ ...form, name: event.target.value })}
        />
      </Field>

      <Field label={t.accounts.bank}>
        <Input
          value={form.bank}
          onChange={(event) => setForm({ ...form, bank: event.target.value })}
        />
      </Field>

      <Field label={t.accounts.type}>
        <Select
          value={form.type}
          onChange={(event) =>
            setForm({ ...form, type: event.target.value as AccountType })
          }
        >
          {accountTypeOptions}
        </Select>
      </Field>

      <Field label={t.accounts.currency}>
        <Select
          value={form.currency}
          onChange={(event) =>
            setForm({ ...form, currency: event.target.value })
          }
        >
          {currencySelectOptions}
        </Select>
      </Field>

      <Field label={t.accounts.opened}>
        <Input
          type="date"
          value={form.opened_at}
          onChange={(event) =>
            setForm({ ...form, opened_at: event.target.value })
          }
        />
      </Field>

      <label className="checkbox-field">
        <input
          type="checkbox"
          checked={form.is_active}
          onChange={(event) =>
            setForm({ ...form, is_active: event.target.checked })
          }
        />
        <span>{t.accounts.active}</span>
      </label>

      <Button disabled={mutation.isPending}>{t.common.save}</Button>
    </FormShell>
  );
}
