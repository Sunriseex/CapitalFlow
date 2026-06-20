import { useId, useMemo, useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
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
import {
  Button,
  Field,
  FormShell,
  Input,
  Select,
  ValidatedField,
} from "../../shared/ui";
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

const supportedAccountTypes = [
  "card",
  "cash",
  "savings",
  "term_deposit",
] as const;
const capitalizationValues = [
  "none",
  "daily",
  "monthly",
  "end_of_term",
] as const;

type SupportedAccountType = (typeof supportedAccountTypes)[number];
type CapitalizationValue = (typeof capitalizationValues)[number];

type CreateAccountFormValues = {
  name: string;
  bank: string;
  type: SupportedAccountType;
  currency: string;
  opened_at: string;
  initial: string;
  rate: string;
  promoRate: string;
  promoEndDate: string;
  capitalization: CapitalizationValue;
};

const createAccountFormDefaults: CreateAccountFormValues = {
  name: "",
  bank: "",
  type: "card",
  currency: "RUB",
  opened_at: today,
  initial: "",
  rate: "",
  promoRate: "",
  promoEndDate: "",
  capitalization: "none",
};

function createAccountFormSchema(t: ReturnType<typeof useI18n>["t"]) {
  return z
    .object({
      name: z.string(),
      bank: z.string(),
      type: z.enum(supportedAccountTypes),
      currency: z.string().min(1),
      opened_at: z.string(),
      initial: z.string(),
      rate: z.string(),
      promoRate: z.string(),
      promoEndDate: z.string(),
      capitalization: z.enum(capitalizationValues),
    })
    .superRefine((values, context) => {
      const initial = parseMoneyToMinorResult(values.initial, {
        currency: values.currency,
      });
      if (!initial.ok) {
        context.addIssue({
          code: "custom",
          message: initial.error,
          path: ["initial"],
        });
      }

      if (!isInterestBearing(values.type)) {
        return;
      }

      const rate = parsePercent(values.rate);
      const promoRate = parsePercent(values.promoRate);

      if (Number.isNaN(rate) || rate < 0) {
        context.addIssue({
          code: "custom",
          message: t.accounts.annualRateInvalid,
          path: ["rate"],
        });
      }

      if (Number.isNaN(promoRate) || promoRate < 0) {
        context.addIssue({
          code: "custom",
          message: t.accounts.promoRateInvalid,
          path: ["promoRate"],
        });
      }

      if (rate <= 0 && (promoRate > 0 || values.promoEndDate)) {
        context.addIssue({
          code: "custom",
          message: t.accounts.annualRateRequiredForPromo,
          path: ["rate"],
        });
      }

      if (rate > 0 && promoRate > 0 && !values.promoEndDate) {
        context.addIssue({
          code: "custom",
          message: t.accounts.promoFieldsRequiredTogether,
          path: ["promoEndDate"],
        });
      }

      if (rate > 0 && promoRate <= 0 && values.promoEndDate) {
        context.addIssue({
          code: "custom",
          message: t.accounts.promoFieldsRequiredTogether,
          path: ["promoRate"],
        });
      }
    });
}

export function CreateAccountForm({ onDone }: { onDone: () => void }) {
  const { t, locale } = useI18n();
  const errorMessages = apiErrorMessages(t);
  const errorId = useId();

  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const formSchema = useMemo(() => createAccountFormSchema(t), [t]);
  const {
    clearErrors,
    control,
    formState: { errors },
    handleSubmit,
    register,
    setValue,
  } = useForm<CreateAccountFormValues>({
    defaultValues: createAccountFormDefaults,
    mode: "onBlur",
    resolver: zodResolver(formSchema),
  });
  const watchedForm = useWatch({ control });
  const form = {
    ...createAccountFormDefaults,
    ...watchedForm,
  };
  const mutation = useMutation({
    mutationFn: async (values: CreateAccountFormValues) => {
      const initial = parseValidatedMoney(values.initial, values.currency);
      const interestEnabledForPayload = isInterestBearing(values.type);
      const rate = interestEnabledForPayload ? parsePercent(values.rate) : 0;
      const promoRate = interestEnabledForPayload
        ? parsePercent(values.promoRate)
        : 0;

      const account = await api.createAccount({
        name: values.name,
        bank: values.bank,
        type: values.type,
        currency: values.currency,
        opened_at: values.opened_at,
      });

      if (isPositiveMoney(initial)) {
        await api.createTransaction({
          account_id: account.id,
          type: "initial_balance",
          amount: initial,
          description: t.accounts.initialBalance,
          occurred_at: values.opened_at,
        });
      }

      if (rate > 0) {
        await api.createInterestRule(account.id, {
          annual_rate_bps: Math.round(rate * 100),
          promo_rate_bps: promoRate > 0 ? Math.round(promoRate * 100) : null,
          promo_end_date: promoRate > 0 ? values.promoEndDate : null,
          accrual_frequency: "daily",
          capitalization_frequency: values.capitalization,
          day_count_convention: "actual_365",
          start_date: values.opened_at,
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
      capitalizationValues.map((value) => (
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
  const getFieldErrorId = (field: keyof CreateAccountFormValues) =>
    `${errorId}-${field}-error`;
  const submitForm = handleSubmit((values) => {
    setError("");
    mutation.mutate(values);
  });

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
                    setValue("type", option.type as SupportedAccountType, {
                      shouldDirty: true,
                      shouldValidate: true,
                    });
                    clearErrors();
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
                <Input required {...register("name")} />
              </Field>

              <Field label={t.accounts.currency}>
                <Select
                  {...register("currency", {
                    onChange: () => clearErrors("initial"),
                  })}
                >
                  {currencySelectOptions}
                </Select>
              </Field>

              <Field
                label={isDeposit ? t.accounts.openingDate : t.accounts.opened}
              >
                <Input type="date" {...register("opened_at")} />
              </Field>

              <ValidatedField
                error={errors.initial?.message}
                errorId={getFieldErrorId("initial")}
                label={
                  isDeposit
                    ? t.accounts.openingAmount
                    : t.accounts.currentBalance
                }
              >
                <Input
                  aria-describedby={
                    errors.initial
                      ? getFieldErrorId("initial")
                      : undefined
                  }
                  aria-invalid={Boolean(errors.initial)}
                  inputMode="decimal"
                  {...register("initial", {
                    onChange: () => clearErrors("initial"),
                  })}
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
                    <Input {...register("bank")} />
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
                  error={errors.rate?.message}
                  errorId={getFieldErrorId("rate")}
                  label={t.accounts.annualRate}
                >
                  <Input
                    aria-describedby={
                      errors.rate ? getFieldErrorId("rate") : undefined
                    }
                    aria-invalid={Boolean(errors.rate)}
                    inputMode="decimal"
                    {...register("rate", {
                      onChange: () =>
                        clearErrors(["rate", "promoRate", "promoEndDate"]),
                    })}
                  />
                </ValidatedField>

                <ValidatedField
                  error={errors.promoRate?.message}
                  errorId={getFieldErrorId("promoRate")}
                  label={t.accounts.promoRate}
                >
                  <Input
                    aria-describedby={
                      errors.promoRate
                        ? getFieldErrorId("promoRate")
                        : undefined
                    }
                    aria-invalid={Boolean(errors.promoRate)}
                    inputMode="decimal"
                    {...register("promoRate", {
                      onChange: () =>
                        clearErrors(["rate", "promoRate", "promoEndDate"]),
                    })}
                  />
                </ValidatedField>

                <ValidatedField
                  error={errors.promoEndDate?.message}
                  errorId={getFieldErrorId("promoEndDate")}
                  label={t.accounts.promoEnd}
                >
                  <Input
                    aria-describedby={
                      errors.promoEndDate
                        ? getFieldErrorId("promoEndDate")
                        : undefined
                    }
                    aria-invalid={Boolean(errors.promoEndDate)}
                    type="date"
                    {...register("promoEndDate", {
                      onChange: () =>
                        clearErrors(["rate", "promoRate", "promoEndDate"]),
                    })}
                  />
                </ValidatedField>

                <Field label={t.accounts.capitalization}>
                  <Select {...register("capitalization")}>
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

function parsePercent(value: string) {
  return value ? Number(value.replace(",", ".")) : 0;
}

function parseValidatedMoney(value: string, currency: string) {
  const result = parseMoneyToMinorResult(value, { currency });
  return result.ok ? result.value : "0";
}

function accountTypeLabel(
  type: AccountType | "checking",
  t: ReturnType<typeof useI18n>["t"],
) {
  return type === "checking"
    ? t.accounts.checkingAccount
    : t.accounts.types[type];
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
