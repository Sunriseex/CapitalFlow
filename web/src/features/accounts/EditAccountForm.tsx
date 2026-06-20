import { useMemo, useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { z } from "zod";
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

const editableAccountTypes = [
  "cash",
  "card",
  "savings",
  "term_deposit",
  "broker",
  "other",
] as const;

type EditAccountFormValues = {
  name: string;
  bank: string;
  type: AccountType;
  currency: string;
  opened_at: string;
  is_active: boolean;
};

const editAccountFormSchema = z.object({
  name: z.string(),
  bank: z.string(),
  type: z.enum(editableAccountTypes),
  currency: z.string().min(1),
  opened_at: z.string(),
  is_active: z.boolean(),
});

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
  const formDefaults = useMemo<EditAccountFormValues>(
    () => ({
      name: account.name,
      bank: account.bank ?? "",
      type: account.type,
      currency: account.currency,
      opened_at: account.opened_at.slice(0, 10),
      is_active: account.is_active,
    }),
    [account],
  );
  const { handleSubmit, register } = useForm<EditAccountFormValues>({
    defaultValues: formDefaults,
    mode: "onBlur",
    resolver: zodResolver(editAccountFormSchema),
  });
  const mutation = useMutation({
    mutationFn: (values: EditAccountFormValues) =>
      api.updateAccount(account.id, values),
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
      onSubmit={handleSubmit((values) => {
        setError("");
        mutation.mutate(values);
      })}
    >
      <Field label={t.accounts.name}>
        <Input required {...register("name")} />
      </Field>

      <Field label={t.accounts.bank}>
        <Input {...register("bank")} />
      </Field>

      <Field label={t.accounts.type}>
        <Select {...register("type")}>
          {accountTypeOptions}
        </Select>
      </Field>

      <Field label={t.accounts.currency}>
        <Select {...register("currency")}>
          {currencySelectOptions}
        </Select>
      </Field>

      <Field label={t.accounts.opened}>
        <Input type="date" {...register("opened_at")} />
      </Field>

      <label className="checkbox-field">
        <input
          type="checkbox"
          {...register("is_active")}
        />
        <span>{t.accounts.active}</span>
      </label>

      <Button disabled={mutation.isPending}>{t.common.save}</Button>
    </FormShell>
  );
}
