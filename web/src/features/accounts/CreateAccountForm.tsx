import { useId, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CreditCard, Landmark, PiggyBank, Timer, Wallet } from "lucide-react";
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

type FieldErrors = Partial<
  Record<"initial" | "rate" | "promoRate" | "promoEndDate", string>
>;

type ValidatedCreateAccount = {
  initial: string;
  rate: number;
  promoRate: number;
};

export function CreateAccountForm({ onDone }: { onDone: () => void }) {
  const { t, locale } = useI18n();
  const errorMessages = apiErrorMessages(t);
  const errorId = useId();

  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
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
    mutationFn: async ({
      initial,
      promoRate,
      rate,
    }: ValidatedCreateAccount) => {
      const account = await api.createAccount({
        name: form.name,
        bank: form.bank,
        type: form.type,
        currency: form.currency,
        opened_at: form.opened_at,
      });

      if (isPositiveMoney(initial)) {
        await api.createTransaction({
          account_id: account.id,
          type: "initial_balance",
          amount: initial,
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
  const isCard = form.type === "card";
  const isCash = form.type === "cash";
  const isSavings = form.type === "savings";
  const isDeposit = form.type === "term_deposit";
  const hasHiddenInterestDraft =
    !interestEnabled &&
    Boolean(
      form.rate ||
        form.promoRate ||
        form.promoEndDate ||
        form.capitalization !== "none",
    );
  const validateForm = () => {
    const nextErrors: FieldErrors = {};
    const initial = parseMoneyToMinorResult(form.initial, {
      currency: form.currency,
    });
    if (!initial.ok) {
      nextErrors.initial = initial.error;
    }

    const rate = interestEnabled ? Number(form.rate.replace(",", ".")) : 0;
    const promoRate = interestEnabled
      ? Number(form.promoRate.replace(",", "."))
      : 0;

    if (Number.isNaN(rate) || rate < 0) {
      nextErrors.rate = t.accounts.annualRateInvalid;
    }

    if (Number.isNaN(promoRate) || promoRate < 0) {
      nextErrors.promoRate = t.accounts.promoRateInvalid;
    }

    if (rate <= 0 && (promoRate > 0 || form.promoEndDate)) {
      nextErrors.rate = t.accounts.annualRateRequiredForPromo;
    }

    if (rate > 0 && promoRate > 0 && !form.promoEndDate) {
      nextErrors.promoEndDate = t.accounts.promoFieldsRequiredTogether;
    }

    if (rate > 0 && promoRate <= 0 && form.promoEndDate) {
      nextErrors.promoRate = t.accounts.promoFieldsRequiredTogether;
    }

    return {
      errors: nextErrors,
      values: {
        initial: initial.ok ? initial.value : "0",
        promoRate,
        rate,
      },
    };
  };
  const validateAndStore = () => {
    const validation = validateForm();
    setFieldErrors(validation.errors);
    return validation;
  };
  const clearFieldError = (field: keyof FieldErrors) => {
    setFieldErrors((errors) => {
      const nextErrors = { ...errors };
      delete nextErrors[field];
      return nextErrors;
    });
  };
  const getFieldErrorId = (field: keyof FieldErrors) =>
    `${errorId}-${field}-error`;
  const submitForm = () => {
    const validation = validateAndStore();
    if (Object.keys(validation.errors).length > 0) {
      setError("");
      return;
    }
    mutation.mutate(validation.values);
  };

  return (
    <FormShell
      title={t.accounts.createAccount}
      error={error}
      onSubmit={submitForm}
    >
      <div className="create-account-layout">
        <fieldset className="account-type-column">
          <legend className="sr-only">{t.accounts.type}</legend>
          {createAccountTypes.map((option) => (
            <label
              key={option.key}
              className={
                [
                  "account-type-card",
                  form.type === option.type ? "is-selected" : "",
                  option.disabled ? "is-disabled" : "",
                ]
                  .filter(Boolean)
                  .join(" ")
              }
            >
              <input
                className="account-type-radio"
                type="radio"
                name="create-account-type"
                value={option.type ?? option.key}
                checked={form.type === option.type}
                disabled={option.disabled}
                onChange={() => {
                  if (option.type) {
                    setForm({ ...form, type: option.type });
                    setFieldErrors({});
                    setError("");
                  }
                }}
              />
              <span className="account-type-icon" aria-hidden="true">
                {accountTypeIcon(option.key)}
              </span>
              <span>
                <strong>{accountTypeLabel(option.key, t)}</strong>
                <small>
                  {accountTypeDescription(option.key, t)}
                  {option.disabled
                    ? ` · ${t.accounts.unsupportedAccountType}`
                    : ""}
                </small>
              </span>
              {option.type && isInterestBearing(option.type) ? (
                <span className="account-type-badge" aria-hidden="true">
                  %
                </span>
              ) : null}
            </label>
          ))}
          <div className="account-type-help" aria-live="polite">
            <strong>
              {isDeposit
                ? t.accounts.depositConditions
                : interestEnabled
                  ? t.accounts.interestSettings
                  : t.accounts.types[form.type]}
            </strong>
            <span>{t.accounts.typeDescriptions[form.type]}</span>
          </div>
        </fieldset>

        <div className="account-form-column">
          {hasHiddenInterestDraft ? (
            <p className="form-note">{t.accounts.hiddenTypeFieldsNotice}</p>
          ) : null}

          <section className="form-section-card">
            <div className="form-section-header">
              <h3 className="form-section-title">{t.accounts.accountSummary}</h3>
              <span className="badge badge-outline">
                {t.accounts.types[form.type]}
              </span>
            </div>
            <div className="account-field-grid">
              <Field label={accountNameLabel(form.type, t)}>
                <Input
                  required
                  value={form.name}
                  onChange={(event) =>
                    setForm({ ...form, name: event.target.value })
                  }
                />
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

              <Field
                label={isDeposit ? t.accounts.openingDate : t.accounts.opened}
              >
                <Input
                  type="date"
                  value={form.opened_at}
                  onChange={(event) =>
                    setForm({ ...form, opened_at: event.target.value })
                  }
                />
              </Field>

              <ValidatedField
                error={fieldErrors.initial}
                errorId={getFieldErrorId("initial")}
                label={
                  isDeposit
                    ? t.accounts.openingAmount
                    : t.accounts.currentBalance
                }
              >
                <Input
                  aria-describedby={
                    fieldErrors.initial
                      ? getFieldErrorId("initial")
                      : undefined
                  }
                  aria-invalid={Boolean(fieldErrors.initial)}
                  inputMode="decimal"
                  value={form.initial}
                  onBlur={validateAndStore}
                  onChange={(event) => {
                    clearFieldError("initial");
                    setForm({ ...form, initial: event.target.value });
                  }}
                />
              </ValidatedField>

              <PlaceholderCheckbox checked label={t.accounts.includeInBalance} />
              <PlaceholderField label={t.accounts.notes} />
            </div>
          </section>

          {showBankField ? (
            <section className="conditional-section is-visible">
              <div className="form-section-card">
                <div className="form-section-header">
                  <h3 className="form-section-title">
                    {isDeposit
                      ? t.accounts.depositConditions
                      : t.accounts.types[form.type]}
                  </h3>
                </div>
                <div className="account-field-grid">
                  <Field label={t.accounts.bank}>
                    <Input
                      value={form.bank}
                      onChange={(event) =>
                        setForm({ ...form, bank: event.target.value })
                      }
                    />
                  </Field>

                  {isCard ? (
                    <>
                      <PlaceholderField label={t.accounts.cardLast4} />
                      <PlaceholderCheckbox label={t.accounts.creditCard} />
                    </>
                  ) : null}

                  {isSavings ? (
                    <>
                      <PlaceholderField label={t.accounts.nextAccrualDate} />
                      <PlaceholderField label={t.accounts.minimumBalance} />
                    </>
                  ) : null}

                  {isDeposit ? (
                    <>
                      <PlaceholderField label={t.accounts.depositEndDate} />
                      <PlaceholderField label={t.accounts.refillAllowed} />
                      <PlaceholderField
                        label={t.accounts.partialWithdrawAllowed}
                      />
                      <PlaceholderField label={t.accounts.maturityAction} />
                    </>
                  ) : null}
                </div>
              </div>
            </section>
          ) : null}

          {isCash ? (
            <section className="conditional-section is-visible">
              <div className="form-section-card">
                <div className="form-section-header">
                  <h3 className="form-section-title">
                    {t.accounts.types.cash}
                  </h3>
                </div>
                <div className="account-field-grid">
                  <PlaceholderField label={t.accounts.storagePlace} />
                </div>
              </div>
            </section>
          ) : null}

          {interestEnabled ? (
            <section className="conditional-section is-visible">
              <div className="interest-fieldset">
                <h3>
                  {isDeposit
                    ? t.accounts.depositConditions
                    : t.accounts.interestSettings}
                </h3>
                <ValidatedField
                  error={fieldErrors.rate}
                  errorId={getFieldErrorId("rate")}
                  label={t.accounts.annualRate}
                >
                  <Input
                    aria-describedby={
                      fieldErrors.rate ? getFieldErrorId("rate") : undefined
                    }
                    aria-invalid={Boolean(fieldErrors.rate)}
                    inputMode="decimal"
                    value={form.rate}
                    onBlur={validateAndStore}
                    onChange={(event) => {
                      clearFieldError("rate");
                      setForm({ ...form, rate: event.target.value });
                    }}
                  />
                </ValidatedField>

                <ValidatedField
                  error={fieldErrors.promoRate}
                  errorId={getFieldErrorId("promoRate")}
                  label={t.accounts.promoRate}
                >
                  <Input
                    aria-describedby={
                      fieldErrors.promoRate
                        ? getFieldErrorId("promoRate")
                        : undefined
                    }
                    aria-invalid={Boolean(fieldErrors.promoRate)}
                    inputMode="decimal"
                    value={form.promoRate}
                    onBlur={validateAndStore}
                    onChange={(event) => {
                      clearFieldError("promoRate");
                      setForm({ ...form, promoRate: event.target.value });
                    }}
                  />
                </ValidatedField>

                <ValidatedField
                  error={fieldErrors.promoEndDate}
                  errorId={getFieldErrorId("promoEndDate")}
                  label={t.accounts.promoEnd}
                >
                  <Input
                    aria-describedby={
                      fieldErrors.promoEndDate
                        ? getFieldErrorId("promoEndDate")
                        : undefined
                    }
                    aria-invalid={Boolean(fieldErrors.promoEndDate)}
                    type="date"
                    value={form.promoEndDate}
                    onBlur={validateAndStore}
                    onChange={(event) => {
                      clearFieldError("promoEndDate");
                      setForm({ ...form, promoEndDate: event.target.value });
                    }}
                  />
                </ValidatedField>

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
            </section>
          ) : null}
        </div>

        <div className="account-dialog-footer">
          <Button disabled={mutation.isPending}>{t.common.create}</Button>
        </div>
      </div>
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

function accountNameLabel(
  type: AccountType,
  t: ReturnType<typeof useI18n>["t"],
) {
  if (type === "card") return t.accounts.cardName;
  if (type === "term_deposit") return t.accounts.depositName;
  return t.accounts.name;
}

function accountTypeIcon(type: AccountType | "checking") {
  if (type === "card") return <CreditCard />;
  if (type === "cash") return <Wallet />;
  if (type === "savings") return <PiggyBank />;
  if (type === "term_deposit") return <Timer />;
  return <Landmark />;
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

function PlaceholderField({ label }: { label: string }) {
  const { t } = useI18n();
  return (
    <Field label={label}>
      <Input disabled placeholder={t.accounts.futureFieldPlaceholder} />
    </Field>
  );
}

function PlaceholderCheckbox({
  checked = false,
  label,
}: {
  checked?: boolean;
  label: string;
}) {
  const { t } = useI18n();
  return (
    <label className="checkbox-field account-placeholder-checkbox">
      <input type="checkbox" checked={checked} disabled readOnly />
      <span>{label}</span>
      <small>{t.accounts.futureFieldPlaceholder}</small>
    </label>
  );
}
