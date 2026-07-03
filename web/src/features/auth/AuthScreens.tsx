import { useEffect, useMemo, useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { Check, Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import type { CurrencyOption } from "../../shared/currencies";
import {
  Button,
  FieldError,
  Input,
  PageTransition,
  PrimitiveButton as ShadcnButton,
  Select,
} from "../../shared/ui";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../components/ui/popover";
import type { Locale } from "../../shared/i18n/i18n";
import { useI18n } from "../../shared/i18n/useI18n";
import { runThemeRipple } from "../../shared/ui/themeTransition";

export type AuthScreenError = {
  message: string;
  kind: "field" | "global";
} | null;

type PasswordStrengthScore = 0 | 1 | 2 | 3 | 4;
type PasswordStrengthState = {
  score: PasswordStrengthScore;
  label: string;
  feedback: string;
  loading: boolean;
};
type PasswordStrengthAnalysis = {
  password: string;
  strength: PasswordStrengthState;
};
type ZxcvbnResult = {
  score: PasswordStrengthScore;
  feedback: {
    suggestions: string[];
    warning: string;
  };
};
type ZxcvbnFn = (password: string) => ZxcvbnResult;

let zxcvbnPromise: Promise<ZxcvbnFn> | null = null;

type LoginScreenProps = {
  error: AuthScreenError;
  passkeyError: string;
  passkeysSupported: boolean;
  passkeyLoading: boolean;
  statusLoading: boolean;
  onSubmit: (values: LoginSubmitValues) => void;
  onPasskeySubmit: () => void;
};

export type LoginSubmitValues = {
  email: string;
  password: string;
};

type InitialSetupScreenProps = {
  currencyOptions: CurrencyOption[];
  error: AuthScreenError;
  statusLoading: boolean;
  onSubmit: (values: InitialSetupSubmitValues) => void;
};

export type InitialSetupSubmitValues = {
  email: string;
  password: string;
  primaryCurrency: string;
};

type InitialSetupFormValues = InitialSetupSubmitValues & {
  ownerName: string;
  passwordConfirm: string;
  setupConfirm: boolean;
};

function AuthHeaderControls() {
  const { theme = "light", setTheme } = useTheme();
  const { locale, setLocale, t } = useI18n();
  const [languageOpen, setLanguageOpen] = useState(false);
  const dark = theme === "dark";

  return (
    <header className="auth-toolbar">
      <a className="brand" href="/" aria-label={t.auth.capitalFlowHome}>
        <span className="brand-mark" aria-hidden="true">
          CF
        </span>
        <span className="brand-name">CapitalFlow</span>
      </a>

      <div className="auth-preferences">
        <ShadcnButton
          className="auth-preference-button"
          type="button"
          size="icon"
          variant="outline"
          aria-label={
            dark ? t.shell.switchToLightTheme : t.shell.switchToDarkTheme
          }
          title={dark ? t.shell.switchToLightTheme : t.shell.switchToDarkTheme}
          aria-pressed={dark}
          onClick={(event) => {
            const next = dark ? "light" : "dark";
            runThemeRipple(event.currentTarget, () => setTheme(next));
          }}
        >
          {dark ? <Moon aria-hidden="true" /> : <Sun aria-hidden="true" />}
        </ShadcnButton>

        <Popover open={languageOpen} onOpenChange={setLanguageOpen}>
          <PopoverTrigger asChild>
            <ShadcnButton
              className="auth-preference-button"
              type="button"
              size="icon"
              variant="outline"
              aria-label={t.shell.chooseLanguage}
              title={t.shell.chooseLanguage}
            >
              <span aria-hidden="true">{locale === "ru" ? "🇷🇺" : "🇬🇧"}</span>
            </ShadcnButton>
          </PopoverTrigger>
          <PopoverContent className="language-popover" align="end" role="menu">
            <p className="language-popover-title">{t.shell.language}</p>
            <AuthLanguageChoice
              locale="ru"
              active={locale === "ru"}
              flag="🇷🇺"
              label="Русский"
              onSelect={(next) => {
                setLocale(next);
                setLanguageOpen(false);
              }}
            />
            <AuthLanguageChoice
              locale="en"
              active={locale === "en"}
              flag="🇬🇧"
              label="English"
              onSelect={(next) => {
                setLocale(next);
                setLanguageOpen(false);
              }}
            />
          </PopoverContent>
        </Popover>
      </div>
    </header>
  );
}

function AuthLanguageChoice({
  locale,
  active,
  flag,
  label,
  onSelect,
}: {
  locale: Locale;
  active: boolean;
  flag: string;
  label: string;
  onSelect: (locale: Locale) => void;
}) {
  return (
    <ShadcnButton
      className="language-choice"
      type="button"
      variant="ghost"
      role="menuitemradio"
      aria-checked={active}
      onClick={() => onSelect(locale)}
    >
      <span className="language-choice-copy">
        <span aria-hidden="true">{flag}</span>
        <span>{label}</span>
      </span>
      {active ? <Check aria-hidden="true" /> : null}
    </ShadcnButton>
  );
}

export function LoginScreen({
  error,
  passkeyError,
  passkeysSupported,
  passkeyLoading,
  statusLoading,
  onSubmit,
  onPasskeySubmit,
}: LoginScreenProps) {
  const [passwordVisible, setPasswordVisible] = useState(false);
  const { t } = useI18n();
  const formSchema = useMemo(() => loginSchema(t), [t]);
  const {
    clearErrors,
    formState: { errors },
    handleSubmit,
    register,
  } = useForm<LoginSubmitValues>({
    defaultValues: {
      email: "",
      password: "",
    },
    mode: "onBlur",
    resolver: zodResolver(formSchema),
  });
  const fieldError = error?.kind === "field";
  const emailError =
    errors.email?.message ?? (fieldError ? t.auth.emailSignInError : "");
  const passwordError =
    errors.password?.message ?? (fieldError ? t.auth.passwordSignInError : "");
  const globalError = error?.kind === "global" ? error.message : "";

  return (
    <main className="auth-page auth-reference-page">
      <PageTransition>
        <div className="auth-stack">
          <AuthHeaderControls />

          <section className="auth-card" aria-labelledby="login-title">
            <header className="auth-header">
              <h1 className="auth-title" id="login-title">
                {t.auth.signIn}
              </h1>
              <p className="auth-description">{t.auth.signInDescription}</p>
            </header>

            <div className="auth-card-content">
              <form
                className="form"
                action="/login"
                method="post"
                noValidate
                onSubmit={handleSubmit(onSubmit)}
                aria-label={t.auth.loginForm}
              >
                <Button
                  id="passkey-button"
                  type="button"
                  aria-describedby="passkey-hint"
                  disabled={!passkeysSupported || passkeyLoading}
                  onClick={onPasskeySubmit}
                >
                  {passkeyLoading
                    ? t.auth.checkingPasskey
                    : t.auth.signInWithPasskey}{" "}
                </Button>
                <p className="form-hint" id="passkey-hint">
                  {t.auth.passkeyHint}
                </p>
                {passkeyError ? (
                  <p className="form-status" role="status">
                    {passkeyError}
                  </p>
                ) : null}
                {!passkeysSupported ? (
                  <p className="form-status" role="status">
                    {t.auth.passkeyUnsupported}{" "}
                  </p>
                ) : null}

                <div className="divider" aria-hidden="true">
                  {t.auth.orUseEmail}
                </div>

                <div className="field">
                  <label htmlFor="email">{t.auth.email}</label>{" "}
                  <Input
                    id="email"
                    type="email"
                    autoComplete="email"
                    placeholder={t.auth.emailPlaceholder}
                    required
                    aria-invalid={Boolean(emailError)}
                    aria-errormessage={emailError ? "email-error" : undefined}
                    {...register("email", {
                      onChange: () => clearErrors("email"),
                    })}
                  />
                  {emailError ? (
                    <FieldError id="email-error">{emailError}</FieldError>
                  ) : null}
                </div>

                <div className="field">
                  <div className="label-row">
                    <label htmlFor="password">{t.auth.password}</label>{" "}
                    <span
                      className="helper-text"
                      aria-label={t.auth.passwordResetUnavailable}
                    >
                      {t.auth.passwordResetUnavailable}{" "}
                    </span>
                  </div>
                  <div className="password-control">
                    <Input
                      id="password"
                      type={passwordVisible ? "text" : "password"}
                      autoComplete="current-password"
                      placeholder={t.auth.passwordPlaceholder}
                      required
                      aria-invalid={Boolean(passwordError)}
                      aria-errormessage={
                        passwordError ? "password-error" : undefined
                      }
                      {...register("password", {
                        onChange: () => clearErrors("password"),
                      })}
                    />
                    <ShadcnButton
                      className="password-toggle"
                      type="button"
                      variant="ghost"
                      aria-label={
                        passwordVisible
                          ? t.auth.hidePassword
                          : t.auth.showPassword
                      }
                      aria-controls="password"
                      aria-pressed={passwordVisible}
                      onClick={() => setPasswordVisible((visible) => !visible)}
                    >
                      <span aria-hidden="true">
                        {passwordVisible
                          ? t.auth.hidePasswordShort
                          : t.auth.showPasswordShort}
                      </span>
                    </ShadcnButton>
                  </div>
                  {passwordError ? (
                    <FieldError id="password-error">{passwordError}</FieldError>
                  ) : null}
                </div>

                <label className="checkbox-label" htmlFor="remember">
                  <input
                    className="checkbox"
                    id="remember"
                    name="remember"
                    type="checkbox"
                  />
                  {t.auth.rememberThisDevice}{" "}
                </label>

                {globalError ? (
                  <p
                    className="form-status"
                    id="form-status"
                    role="status"
                    aria-live="polite"
                  >
                    {globalError}
                  </p>
                ) : null}
                <Button
                  variant="outline"
                  type="submit"
                  disabled={statusLoading}
                >
                  {t.auth.signInWithEmail}{" "}
                </Button>
              </form>

              <p className="footer-text">
                {t.auth.initialSetupAppearsAutomatically}{" "}
              </p>
            </div>
          </section>
        </div>
      </PageTransition>
    </main>
  );
}

export function InitialSetupScreen({
  currencyOptions,
  error,
  statusLoading,
  onSubmit,
}: InitialSetupScreenProps) {
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [confirmPasswordVisible, setConfirmPasswordVisible] = useState(false);
  const { t } = useI18n();
  const [submitError, setSubmitError] = useState<{
    target: "password" | "confirm" | "setup-confirm";
    message: string;
  } | null>(null);
  const formSchema = useMemo(() => initialSetupSchema(t), [t]);
  const {
    clearErrors,
    control,
    formState: { errors, touchedFields },
    handleSubmit,
    register,
  } = useForm<InitialSetupFormValues>({
    defaultValues: {
      ownerName: "",
      email: "",
      password: "",
      passwordConfirm: "",
      primaryCurrency: "RUB",
      setupConfirm: false,
    },
    mode: "onBlur",
    resolver: zodResolver(formSchema),
  });
  const password = useWatch({ control, name: "password" }) ?? "";
  const passwordConfirm = useWatch({ control, name: "passwordConfirm" }) ?? "";
  const setupConfirmed = useWatch({ control, name: "setupConfirm" }) ?? false;
  const strength = usePasswordStrength(password, t);
  const apiFieldError = error?.kind === "field";
  const emailError =
    errors.email?.message ?? (apiFieldError ? t.auth.ownerEmailError : "");
  const passwordError =
    submitError?.target === "password"
      ? submitError.message
      : (errors.password?.message ??
        (touchedFields.password && password && strength.score < 3
          ? t.auth.passwordScoreRequirement
          : ""));
  const confirmError =
    submitError?.target === "confirm"
      ? submitError.message
      : (errors.passwordConfirm?.message ??
        ((touchedFields.passwordConfirm || passwordConfirm) &&
        passwordConfirm !== password
          ? t.auth.passwordsDoNotMatch
          : ""));
  const setupConfirmError =
    submitError?.target === "setup-confirm"
      ? submitError.message
      : errors.setupConfirm?.message;
  const globalError = error?.kind === "global" ? error.message : "";

  const submitSetup = handleSubmit((values) => {
    if (strength.loading) {
      setSubmitError({
        target: "password",
        message: t.auth.passwordStrengthLoading,
      });
      return;
    }

    if (strength.score < 3) {
      setSubmitError({
        target: "password",
        message: t.auth.passwordScoreRequirement,
      });
      return;
    }

    if (values.passwordConfirm !== values.password) {
      setSubmitError({
        target: "confirm",
        message: t.auth.passwordConfirmationDoesNotMatch,
      });
      return;
    }

    if (!setupConfirmed) {
      setSubmitError({
        target: "setup-confirm",
        message: t.auth.ownerAccountRequirementError,
      });
      return;
    }

    setSubmitError(null);
    onSubmit({
      email: values.email,
      password: values.password,
      primaryCurrency: values.primaryCurrency,
    });
  });

  return (
    <main className="setup-page auth-reference-page">
      <PageTransition>
        <div className="auth-stack auth-stack-wide">
          <AuthHeaderControls />

          <section className="setup-card" aria-labelledby="setup-title">
            <header className="setup-header">
              <p className="setup-kicker">{t.auth.oneTimeSetup}</p>{" "}
              <h1 className="setup-title" id="setup-title">
                {t.auth.createOwnerAccountTitle}
              </h1>
              <p className="setup-description">{t.auth.setupDescription}</p>
            </header>

            <div className="auth-card-content">
              <aside
                className="warning-box"
                aria-labelledby="setup-warning-title"
              >
                <h2 className="warning-title" id="setup-warning-title">
                  <span className="warning-icon" aria-hidden="true">
                    !
                  </span>
                  {t.auth.importantBeforeContinuing}
                </h2>
                <p className="warning-text">{t.auth.setupWarning}</p>
              </aside>

              <form
                className="form"
                action="/setup"
                method="post"
                noValidate
                onSubmit={submitSetup}
                aria-label={t.auth.initialSetupForm}
              >
                <div className="field">
                  <label htmlFor="owner-name">{t.auth.ownerName}</label>{" "}
                  <Input
                    id="owner-name"
                    type="text"
                    autoComplete="name"
                    placeholder={t.auth.ownerNamePlaceholder}
                    {...register("ownerName")}
                  />
                </div>

                <div className="field">
                  <label htmlFor="owner-email">{t.auth.ownerEmail}</label>
                  <Input
                    id="owner-email"
                    type="email"
                    autoComplete="email"
                    placeholder={t.auth.emailPlaceholder}
                    required
                    aria-invalid={Boolean(emailError)}
                    aria-errormessage={
                      emailError ? "owner-email-error" : undefined
                    }
                    {...register("email", {
                      onChange: () => clearErrors("email"),
                    })}
                  />
                  {emailError ? (
                    <FieldError id="owner-email-error">{emailError}</FieldError>
                  ) : null}
                </div>

                <div className="field">
                  <label htmlFor="owner-password">{t.auth.password}</label>{" "}
                  <div className="password-control">
                    <Input
                      id="owner-password"
                      type={passwordVisible ? "text" : "password"}
                      autoComplete="new-password"
                      placeholder={t.auth.useStrongPassphrase}
                      required
                      aria-invalid={Boolean(passwordError)}
                      aria-describedby="password-strength-feedback"
                      aria-errormessage={
                        passwordError ? "owner-password-error" : undefined
                      }
                      {...register("password", {
                        onChange: () => {
                          setSubmitError(null);
                          clearErrors(["password", "passwordConfirm"]);
                        },
                      })}
                      onFocus={preloadPasswordStrength}
                    />
                    <ShadcnButton
                      className="password-toggle"
                      type="button"
                      variant="ghost"
                      data-target="owner-password"
                      aria-label={
                        passwordVisible
                          ? t.auth.hidePassword
                          : t.auth.showPassword
                      }
                      aria-controls="owner-password"
                      aria-pressed={passwordVisible}
                      onClick={() => setPasswordVisible((visible) => !visible)}
                    >
                      <span aria-hidden="true">
                        {passwordVisible
                          ? t.auth.hidePasswordShort
                          : t.auth.showPasswordShort}
                      </span>
                    </ShadcnButton>
                  </div>
                  <div
                    className="password-strength"
                    aria-label={t.auth.passwordStrength}
                  >
                    {" "}
                    <div className="strength-row">
                      <span>{t.auth.passwordStrength}</span>{" "}
                      <strong id="password-strength-label">
                        {strength.label}
                      </strong>
                    </div>
                    <div
                      className="strength-track"
                      role="meter"
                      aria-label={t.auth.passwordStrengthScore}
                      aria-valuemin={0}
                      aria-valuemax={4}
                      aria-valuenow={strength.score}
                      aria-valuetext={strength.label}
                    >
                      <div
                        className="strength-bar"
                        id="password-strength-bar"
                        data-score={strength.score}
                      ></div>
                    </div>
                    <p
                      className="strength-feedback"
                      id="password-strength-feedback"
                      aria-live="polite"
                    >
                      {strength.feedback}
                    </p>
                  </div>
                  {passwordError ? (
                    <FieldError id="owner-password-error">
                      {passwordError}
                    </FieldError>
                  ) : null}
                </div>

                <div className="field">
                  <label htmlFor="owner-password-confirm">
                    {t.auth.confirmPassword}
                  </label>{" "}
                  <div className="password-control">
                    <Input
                      id="owner-password-confirm"
                      type={confirmPasswordVisible ? "text" : "password"}
                      autoComplete="new-password"
                      placeholder={t.auth.confirmPasswordPlaceholder}
                      aria-invalid={Boolean(confirmError)}
                      aria-errormessage={
                        confirmError
                          ? "owner-password-confirm-error"
                          : undefined
                      }
                      {...register("passwordConfirm", {
                        onChange: () => {
                          setSubmitError(null);
                          clearErrors("passwordConfirm");
                        },
                      })}
                    />
                    <ShadcnButton
                      className="password-toggle"
                      type="button"
                      variant="ghost"
                      data-target="owner-password-confirm"
                      aria-label={
                        confirmPasswordVisible
                          ? t.auth.hidePasswordConfirmation
                          : t.auth.showPasswordConfirmation
                      }
                      aria-controls="owner-password-confirm"
                      aria-pressed={confirmPasswordVisible}
                      onClick={() =>
                        setConfirmPasswordVisible((visible) => !visible)
                      }
                    >
                      <span aria-hidden="true">
                        {confirmPasswordVisible
                          ? t.auth.hidePasswordShort
                          : t.auth.showPasswordShort}
                      </span>
                    </ShadcnButton>
                  </div>
                  {confirmError ? (
                    <FieldError id="owner-password-confirm-error">
                      {confirmError}
                    </FieldError>
                  ) : null}
                </div>

                <div className="field">
                  <label htmlFor="primary-currency">
                    {t.auth.primaryCurrency}
                  </label>{" "}
                  <Select
                    id="primary-currency"
                    {...register("primaryCurrency")}
                  >
                    {currencyOptions.map((currency) => (
                      <option key={currency.code} value={currency.code}>
                        {currency.label}
                      </option>
                    ))}
                  </Select>
                </div>

                <label className="confirm-label" htmlFor="setup-confirm">
                  <input
                    className="checkbox"
                    id="setup-confirm"
                    type="checkbox"
                    required
                    aria-invalid={Boolean(setupConfirmError)}
                    aria-errormessage={
                      setupConfirmError ? "setup-confirm-error" : undefined
                    }
                    {...register("setupConfirm", {
                      onChange: () => {
                        setSubmitError(null);
                        clearErrors("setupConfirm");
                      },
                    })}
                  />
                  {t.auth.ownerAccountRequirement}
                </label>
                {setupConfirmError ? (
                  <FieldError id="setup-confirm-error">
                    {setupConfirmError}
                  </FieldError>
                ) : null}

                {globalError ? (
                  <p
                    className="form-status"
                    id="form-status-api"
                    role="status"
                    aria-live="polite"
                  >
                    {globalError}
                  </p>
                ) : null}
                <Button
                  id="setup-submit"
                  type="submit"
                  disabled={statusLoading}
                >
                  {t.auth.createOwnerAccount}
                </Button>
              </form>

              <p className="footer-text">
                {t.auth.setupAvailableOnlyWhenOwnerMissing}
              </p>
            </div>
          </section>
        </div>
      </PageTransition>
    </main>
  );
}

export function AuthStatusScreen({
  title,
  message,
  action,
}: {
  title: string;
  message: string;
  action?: { label: string; onClick: () => void };
}) {
  const { t } = useI18n();
  return (
    <main className="auth-page auth-reference-page">
      <PageTransition>
        <div className="auth-stack">
          <div className="brand" aria-label={t.common.appName}>
            <span className="brand-mark" aria-hidden="true">
              CF
            </span>
            <span className="brand-name">CapitalFlow</span>
          </div>

          <section className="auth-card" aria-labelledby="auth-status-title">
            <header className="auth-header">
              <h1 className="auth-title" id="auth-status-title">
                {title}
              </h1>
              <p className="auth-description">{message}</p>
            </header>
            {action ? (
              <div className="auth-card-content">
                <Button type="button" onClick={action.onClick}>
                  {action.label}
                </Button>
              </div>
            ) : null}
          </section>
        </div>
      </PageTransition>
    </main>
  );
}

function loginSchema(t: ReturnType<typeof useI18n>["t"]) {
  return z.object({
    email: z
      .string()
      .trim()
      .min(1, t.auth.emailSignInError)
      .email(t.auth.emailSignInError),
    password: z.string().min(1, t.auth.passwordSignInError),
  });
}

function initialSetupSchema(t: ReturnType<typeof useI18n>["t"]) {
  return z.object({
    ownerName: z.string(),
    email: z
      .string()
      .trim()
      .min(1, t.auth.ownerEmailError)
      .email(t.auth.ownerEmailError),
    password: z.string().min(1, t.auth.passwordScoreRequirement),
    passwordConfirm: z.string(),
    primaryCurrency: z.string().min(1),
    setupConfirm: z.boolean(),
  });
}

function usePasswordStrength(
  password: string,
  t: ReturnType<typeof useI18n>["t"],
) {
  const [analysis, setAnalysis] = useState<PasswordStrengthAnalysis | null>(
    null,
  );

  useEffect(() => {
    let cancelled = false;

    if (!password) {
      return () => {
        cancelled = true;
      };
    }

    void loadZxcvbn()
      .then((zxcvbn) => {
        if (cancelled) {
          return;
        }
        setAnalysis({
          password,
          strength: passwordStrengthFromResult(zxcvbn(password), t),
        });
      })
      .catch(() => {
        if (cancelled) {
          return;
        }
        setAnalysis({
          password,
          strength: {
            score: 0,
            label: t.auth.strength.weak,
            feedback: t.auth.passwordStrengthFallback,
            loading: false,
          },
        });
      });

    return () => {
      cancelled = true;
    };
  }, [password, t]);

  if (!password) {
    return emptyPasswordStrength(t);
  }

  if (analysis?.password === password) {
    return analysis.strength;
  }

  return {
    score: 0,
    label: t.common.loading,
    feedback: t.auth.passwordStrengthLoading,
    loading: true,
  };
}

function loadZxcvbn() {
  zxcvbnPromise ??= import("zxcvbn").then(
    (module) => module.default as ZxcvbnFn,
  );
  return zxcvbnPromise;
}

function preloadPasswordStrength() {
  void loadZxcvbn().catch(() => undefined);
}

function emptyPasswordStrength(
  t: ReturnType<typeof useI18n>["t"],
): PasswordStrengthState {
  return {
    score: 0,
    label: t.auth.strength.empty,
    feedback: t.auth.passwordStrengthEmptyFeedback,
    loading: false,
  };
}

function passwordStrengthFromResult(
  result: ZxcvbnResult,
  t: ReturnType<typeof useI18n>["t"],
): PasswordStrengthState {
  const score = result.score;
  const labels = [
    t.auth.strength.weak,
    t.auth.strength.weak,
    t.auth.strength.fair,
    t.auth.strength.good,
    t.auth.strength.strong,
  ];
  const suggestion = result.feedback.suggestions[0] || result.feedback.warning;
  const feedback =
    score >= 3
      ? t.auth.passwordStrengthAcceptable
      : suggestion || t.auth.passwordStrengthFallback;

  return {
    score,
    label: labels[score],
    feedback,
    loading: false,
  };
}
