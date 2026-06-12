import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import { isPositiveMoney, parseMoneyToMinorResult } from "../../api/money";
import type { AccountType } from "../../api/types";
import {
  apiErrorMessages,
  errorMessage,
  invalidateMoney,
} from "../../shared/api/query";
import { currencyOptions } from "../../shared/currencies";
import { today } from "../../shared/constants";
import { Button, Field, FormShell, Input, Select } from "../../shared/ui";
import { useI18n } from "../../shared/i18n/useI18n";

const createAccountTypes: Array<{
  key: AccountType | "checking";
  type?: AccountType;
  disabled?: boolean;
}> = [
  { key: "card", type: "card" },
  { key: "cash", type: "cash" },
  { key: "checking", disabled: true },
  { key: "savings", type: "savings" },
  { key: "term_deposit", type: "term_deposit" },
];

export function CreateAccountForm({ onDone }: { onDone: () => void }) {
  const { t, locale } = useI18n();
  const errorMessages = apiErrorMessages(t);

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

      const interestEnabled = isInterestBearing(form.type);
      const rate = interestEnabled ? Number(form.rate.replace(",", ".")) : 0;
      const promoRate = interestEnabled
        ? Number(form.promoRate.replace(",", "."))
        : 0;

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
    onError: (err) => setError(errorMessage(err, errorMessages)),
  });
  const currencies = useMemo(() => currencyOptions(locale), [locale]);
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
  const interestEnabled = isInterestBearing(form.type);
  const showBankField = form.type !== "cash";
  const hasHiddenInterestDraft =
    !interestEnabled &&
    Boolean(
      form.rate || form.promoRate || form.promoEndDate || form.capitalization !== "none",
    );

  return (
    <FormShell
      title={t.accounts.createAccount}
      error={error}
      onSubmit={() => mutation.mutate()}
    >
      <div
        className="account-type-picker"
        role="radiogroup"
        aria-label={t.accounts.type}
      >
        {createAccountTypes.map((option) => (
          <button
            key={option.key}
            className={
              form.type === option.type
                ? "account-type-card is-selected"
                : "account-type-card"
            }
            disabled={option.disabled}
            type="button"
            role="radio"
            aria-checked={form.type === option.type}
            onClick={() => {
              if (option.type) {
                setForm({ ...form, type: option.type });
              }
            }}
          >
            <span className="account-type-icon" aria-hidden="true">
              {accountTypeLabel(option.key, t).slice(0, 1)}
            </span>
            <span>
              <strong>{accountTypeLabel(option.key, t)}</strong>
              <small>
                {accountTypeDescription(option.key, t)}
                {option.disabled ? ` · ${t.accounts.unsupportedAccountType}` : ""}
              </small>
            </span>
          </button>
        ))}
      </div>

      <div className="account-type-help">
        <strong>
          {form.type === "term_deposit"
            ? t.accounts.depositConditions
            : interestEnabled
              ? t.accounts.interestSettings
              : t.accounts.types[form.type]}
        </strong>
        <span>{t.accounts.typeDescriptions[form.type]}</span>
      </div>

      {hasHiddenInterestDraft ? (
        <p className="form-note">{t.accounts.hiddenTypeFieldsNotice}</p>
      ) : null}

      <Field label={t.accounts.name}>
        <Input
          required
          value={form.name}
          onChange={(event) => setForm({ ...form, name: event.target.value })}
        />
      </Field>

      {showBankField ? (
        <Field label={t.accounts.bank}>
          <Input
            value={form.bank}
            onChange={(event) => setForm({ ...form, bank: event.target.value })}
          />
        </Field>
      ) : null}

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

      {interestEnabled ? (
        <div className="interest-fieldset">
          <h3>
            {form.type === "term_deposit"
              ? t.accounts.depositConditions
              : t.accounts.interestSettings}
          </h3>
          <Field label={t.accounts.annualRate}>
            <Input
              inputMode="decimal"
              value={form.rate}
              onChange={(event) =>
                setForm({ ...form, rate: event.target.value })
              }
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
        </div>
      ) : null}

      <Button disabled={mutation.isPending}>{t.common.create}</Button>
    </FormShell>
  );
}

function isInterestBearing(type: AccountType) {
  return type === "savings" || type === "term_deposit";
}

function accountTypeLabel(
  type: AccountType | "checking",
  t: ReturnType<typeof useI18n>["t"],
) {
  return type === "checking" ? t.accounts.checkingAccount : t.accounts.types[type];
}

function accountTypeDescription(
  type: AccountType | "checking",
  t: ReturnType<typeof useI18n>["t"],
) {
  return type === "checking"
    ? t.accounts.checkingAccountDescription
    : t.accounts.typeDescriptions[type];
}
