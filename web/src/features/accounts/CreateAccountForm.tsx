import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import { isPositiveMoney, parseMoneyToMinorResult } from "../../api/money";
import type { AccountType } from "../../api/types";
import { errorMessage, invalidateMoney } from "../../shared/api/query";
import { currencyOptions } from "../../shared/currencies";
import { accountTypes, today } from "../../shared/constants";
import { Button, Field, FormShell, Input, Select } from "../../shared/ui";
import { useI18n } from "../../shared/i18n/useI18n";

export function CreateAccountForm({ onDone }: { onDone: () => void }) {
  const { t } = useI18n();

  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    name: "",
    bank: "",
    type: "card" as AccountType,
    currency: "RUB",
    opened_at: today,
    initial: "",
    rate: "",
    promoRate: "",
    promoEndDate: "",
    capitalization: "none",
  });
  const mutation = useMutation({
    mutationFn: async () => {
      const initial = parseMoneyToMinorResult(form.initial, {
        currency: form.currency,
      });
      if (!initial.ok) {
        throw new Error(initial.error);
      }

      const rate = Number(form.rate.replace(",", "."));
      const promoRate = Number(form.promoRate.replace(",", "."));

      if (Number.isNaN(rate) || rate < 0) {
        throw new Error(t.accounts.annualRateInvalid);
      }

      if (Number.isNaN(promoRate) || promoRate < 0) {
        throw new Error(t.accounts.promoRateInvalid);
      }

      if (rate <= 0 && (promoRate > 0 || form.promoEndDate)) {
        throw new Error(t.accounts.annualRateRequiredForPromo);
      }

      if (
        rate > 0 &&
        ((promoRate > 0 && !form.promoEndDate) ||
          (promoRate <= 0 && form.promoEndDate))
      ) {
        throw new Error(t.accounts.promoFieldsRequiredTogether);
      }

      const account = await api.createAccount({
        name: form.name,
        bank: form.bank,
        type: form.type,
        currency: form.currency,
        opened_at: form.opened_at,
      });

      if (isPositiveMoney(initial.value)) {
        await api.createTransaction({
          account_id: account.id,
          type: "initial_balance",
          amount: initial.value,
          description: t.accounts.initialBalance,
          occurred_at: form.opened_at,
        });
      }

      if (rate > 0) {
        await api.createInterestRule(account.id, {
          annual_rate_bps: Math.round(rate * 100),
          promo_rate_bps: promoRate > 0 ? Math.round(promoRate * 100) : null,
          promo_end_date: promoRate > 0 ? form.promoEndDate : null,
          accrual_frequency: "daily",
          capitalization_frequency: form.capitalization as
            | "none"
            | "daily"
            | "monthly"
            | "end_of_term",
          day_count_convention: "actual_365",
          start_date: form.opened_at,
        });
      }

      return account;
    },
    onSuccess: () => {
      invalidateMoney(queryClient);
      onDone();
    },
    onError: (err) => setError(errorMessage(err)),
  });
  const currencies = useMemo(() => currencyOptions(), []);
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
  const capitalizationOptions = useMemo(
    () =>
      (["none", "daily", "monthly", "end_of_term"] as const).map((value) => (
        <option key={value} value={value}>
          {t.accounts.capitalizationOptions[value]}
        </option>
      )),
    [t],
  );

  return (
    <FormShell
      title={t.accounts.createAccount}
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

      <Field label={t.accounts.initialBalance}>
        <Input
          inputMode="decimal"
          value={form.initial}
          onChange={(event) =>
            setForm({ ...form, initial: event.target.value })
          }
        />
      </Field>

      <Field label={t.accounts.annualRate}>
        <Input
          inputMode="decimal"
          value={form.rate}
          onChange={(event) => setForm({ ...form, rate: event.target.value })}
        />
      </Field>

      <Field label={t.accounts.promoRate}>
        <Input
          inputMode="decimal"
          value={form.promoRate}
          onChange={(event) =>
            setForm({ ...form, promoRate: event.target.value })
          }
        />
      </Field>

      <Field label={t.accounts.promoEnd}>
        <Input
          type="date"
          value={form.promoEndDate}
          onChange={(event) =>
            setForm({ ...form, promoEndDate: event.target.value })
          }
        />
      </Field>

      <Field label={t.accounts.capitalization}>
        <Select
          value={form.capitalization}
          onChange={(event) =>
            setForm({ ...form, capitalization: event.target.value })
          }
        >
          {capitalizationOptions}
        </Select>
      </Field>

      <Button disabled={mutation.isPending}>{t.common.create}</Button>
    </FormShell>
  );
}
