import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import type { CurrencyOption } from "../../shared/currencies";
import { Button, Input, PageTransition, Select } from "../../shared/ui";
import { Button as ShadcnButton } from "../../components/ui/button";
import { useI18n } from "../../shared/i18n/useI18n";

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
  email: string;
  password: string;
  error: AuthScreenError;
  passkeyError: string;
  passkeysSupported: boolean;
  passkeyLoading: boolean;
  statusLoading: boolean;
  onEmailChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onSubmit: () => void;
  onPasskeySubmit: () => void;
};

type InitialSetupScreenProps = {
  email: string;
  password: string;
  primaryCurrency: string;
  currencyOptions: CurrencyOption[];
  error: AuthScreenError;
  statusLoading: boolean;
  onEmailChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onPrimaryCurrencyChange: (value: string) => void;
  onSubmit: () => void;
};

export function LoginScreen({
  email,
  password,
  error,
  passkeyError,
  passkeysSupported,
  passkeyLoading,
  statusLoading,
  onEmailChange,
  onPasswordChange,
  onSubmit,
  onPasskeySubmit,
}: LoginScreenProps) {
  const [passwordVisible, setPasswordVisible] = useState(false);
  const { t } = useI18n();
  const fieldError = error?.kind === "field";
  const emailError = fieldError ? t.auth.emailSignInError : "";
  const passwordError = fieldError ? t.auth.passwordSignInError : "";
  const globalError = error?.kind === "global" ? error.message : "";

  return (
    <main className="auth-page auth-reference-page">
      <PageTransition>
        <div className="auth-stack">
          <a className="brand" href="/" aria-label={t.auth.capitalFlowHome}>
            <span className="brand-mark" aria-hidden="true">
              CF
            </span>
            <span className="brand-name">CapitalFlow</span>
          </a>

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
                onSubmit={submit(onSubmit)}
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
                    name="email"
                    type="email"
                    autoComplete="email"
                    placeholder={t.auth.emailPlaceholder}
                    required
                    aria-invalid={Boolean(emailError)}
                    aria-errormessage={emailError ? "email-error" : undefined}
                    value={email}
                    onChange={(event) => onEmailChange(event.target.value)}
                  />
                  {emailError ? (
                    <p
                      className="field-error"
                      id="email-error"
                      aria-live="polite"
                    >
                      {emailError}
                    </p>
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
                      name="password"
                      type={passwordVisible ? "text" : "password"}
                      autoComplete="current-password"
                      placeholder={t.auth.passwordPlaceholder}
                      required
                      aria-invalid={Boolean(passwordError)}
                      aria-errormessage={
                        passwordError ? "password-error" : undefined
                      }
                      value={password}
                      onChange={(event) => onPasswordChange(event.target.value)}
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
                    <p
                      className="field-error"
                      id="password-error"
                      aria-live="polite"
                    >
                      {passwordError}
                    </p>
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
  email,
  password,
  primaryCurrency,
  currencyOptions,
  error,
  statusLoading,
  onEmailChange,
  onPasswordChange,
  onPrimaryCurrencyChange,
  onSubmit,
}: InitialSetupScreenProps) {
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [confirmPassword, setConfirmPassword] = useState("");
  const [confirmPasswordVisible, setConfirmPasswordVisible] = useState(false);
  const [passwordTouched, setPasswordTouched] = useState(false);
  const [confirmTouched, setConfirmTouched] = useState(false);
  const [setupConfirmed, setSetupConfirmed] = useState(false);
  const { t } = useI18n();
  const [submitError, setSubmitError] = useState<{
    target: "password" | "confirm" | "setup-confirm";
    message: string;
  } | null>(null);
  const strength = usePasswordStrength(password, t);
  const apiFieldError = error?.kind === "field";
  const emailError = apiFieldError ? t.auth.ownerEmailError : "";
  const passwordError =
    submitError?.target === "password"
      ? submitError.message
      : passwordTouched && password && strength.score < 3
        ? t.auth.passwordScoreRequirement
        : "";
  const confirmError =
    submitError?.target === "confirm"
      ? submitError.message
      : (confirmTouched || confirmPassword) && confirmPassword !== password
        ? t.auth.passwordsDoNotMatch
        : "";
  const setupConfirmError =
    submitError?.target === "setup-confirm" ? submitError.message : "";
  const globalError = error?.kind === "global" ? error.message : "";

  function submitSetup(event: FormEvent) {
    event.preventDefault();
    setPasswordTouched(true);
    setConfirmTouched(true);

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

    if (confirmPassword !== password) {
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
    onSubmit();
  }

  return (
    <main className="setup-page auth-reference-page">
      <PageTransition>
        <div className="auth-stack auth-stack-wide">
          <a className="brand" href="/" aria-label={t.auth.capitalFlowHome}>
            <span className="brand-mark" aria-hidden="true">
              CF
            </span>
            <span className="brand-name">CapitalFlow</span>
          </a>

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
                    name="ownerName"
                    type="text"
                    autoComplete="name"
                    placeholder={t.auth.ownerNamePlaceholder}
                  />
                </div>

                <div className="field">
                  <label htmlFor="owner-email">{t.auth.ownerEmail}</label>
                  <Input
                    id="owner-email"
                    name="email"
                    type="email"
                    autoComplete="email"
                    placeholder={t.auth.emailPlaceholder}
                    required
                    aria-invalid={Boolean(emailError)}
                    aria-errormessage={
                      emailError ? "owner-email-error" : undefined
                    }
                    value={email}
                    onChange={(event) => onEmailChange(event.target.value)}
                  />
                  {emailError ? (
                    <p
                      className="field-error"
                      id="owner-email-error"
                      aria-live="polite"
                    >
                      {emailError}
                    </p>
                  ) : null}
                </div>

                <div className="field">
                  <label htmlFor="owner-password">{t.auth.password}</label>{" "}
                  <div className="password-control">
                    <Input
                      id="owner-password"
                      name="password"
                      type={passwordVisible ? "text" : "password"}
                      autoComplete="new-password"
                      placeholder={t.auth.useStrongPassphrase}
                      required
                      aria-invalid={Boolean(passwordError)}
                      aria-describedby="password-strength-feedback"
                      aria-errormessage={
                        passwordError ? "owner-password-error" : undefined
                      }
                      value={password}
                      onFocus={preloadPasswordStrength}
                      onBlur={() => setPasswordTouched(true)}
                      onChange={(event) => {
                        setSubmitError(null);
                        onPasswordChange(event.target.value);
                      }}
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
                    <p
                      className="field-error"
                      id="owner-password-error"
                      aria-live="polite"
                    >
                      {passwordError}
                    </p>
                  ) : null}
                </div>

                <div className="field">
                  <label htmlFor="owner-password-confirm">
                    {t.auth.confirmPassword}
                  </label>{" "}
                  <div className="password-control">
                    <Input
                      id="owner-password-confirm"
                      name="passwordConfirm"
                      type={confirmPasswordVisible ? "text" : "password"}
                      autoComplete="new-password"
                      placeholder={t.auth.confirmPasswordPlaceholder}
                      aria-invalid={Boolean(confirmError)}
                      aria-errormessage={
                        confirmError
                          ? "owner-password-confirm-error"
                          : undefined
                      }
                      value={confirmPassword}
                      onBlur={() => setConfirmTouched(true)}
                      onChange={(event) => {
                        setSubmitError(null);
                        setConfirmPassword(event.target.value);
                      }}
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
                    <p
                      className="field-error"
                      id="owner-password-confirm-error"
                      aria-live="polite"
                    >
                      {confirmError}
                    </p>
                  ) : null}
                </div>

                <div className="field">
                  <label htmlFor="primary-currency">
                    {t.auth.primaryCurrency}
                  </label>{" "}
                  <Select
                    id="primary-currency"
                    name="primaryCurrency"
                    value={primaryCurrency}
                    onChange={(event) =>
                      onPrimaryCurrencyChange(event.target.value)
                    }
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
                    name="setupConfirm"
                    type="checkbox"
                    required
                    checked={setupConfirmed}
                    aria-invalid={Boolean(setupConfirmError)}
                    aria-errormessage={
                      setupConfirmError ? "setup-confirm-error" : undefined
                    }
                    onChange={(event) => {
                      setSubmitError(null);
                      setSetupConfirmed(event.target.checked);
                    }}
                  />
                  {t.auth.ownerAccountRequirement}
                </label>
                {setupConfirmError ? (
                  <p
                    className="field-error"
                    id="setup-confirm-error"
                    aria-live="polite"
                  >
                    {setupConfirmError}
                  </p>
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
                <Button
                  type="button"
                  onClick={action.onClick}
                >
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

function submit(onSubmit: () => void) {
  return (event: FormEvent) => {
    event.preventDefault();
    onSubmit();
  };
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
