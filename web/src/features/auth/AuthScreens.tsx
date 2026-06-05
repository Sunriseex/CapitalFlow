import { useMemo, useState } from "react";
import type { FormEvent } from "react";
import zxcvbn from "zxcvbn";
import type { CurrencyOption } from "../../shared/currencies";
import { PageTransition } from "../../shared/ui";

type LoginScreenProps = {
  email: string;
  password: string;
  error: string;
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
  error: string;
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
  const emailError = error ? "Check the email address for this sign-in." : "";
  const passwordError = error ? "Check the password for this sign-in." : "";

  return (
    <main className="auth-page auth-reference-page">
      <PageTransition>
        <section className="auth-card" aria-labelledby="login-title">
          <a className="brand" href="/" aria-label="CapitalFlow home">
            <img className="brand-mark" src="/app-icon.png" alt="" aria-hidden="true" />
            <span className="brand-text">
              <span className="brand-name">CapitalFlow</span>
              <span className="brand-note">Personal finance dashboard</span>
            </span>
          </a>

          <header className="auth-header">
            <h1 className="auth-title" id="login-title">Sign in</h1>
            <p className="auth-description">Use a passkey or sign in with your email and password.</p>
          </header>

          <form className="form" action="/login" method="post" noValidate onSubmit={submit(onSubmit)} aria-label="Login form">
            <button
              className="button button-primary"
              id="passkey-button"
              type="button"
              aria-describedby="passkey-hint"
              disabled={!passkeysSupported || passkeyLoading}
              onClick={onPasskeySubmit}
            >
              {passkeyLoading ? "Checking passkey" : "Sign in with passkey"}
            </button>
            <p className="form-hint" id="passkey-hint">
              Supports Face ID, Touch ID, Windows Hello, and hardware security keys.
            </p>
            {passkeyError ? <p className="form-status" role="status">{passkeyError}</p> : null}
            {!passkeysSupported ? <p className="form-status" role="status">This browser does not support passkeys</p> : null}

            <div className="divider" aria-hidden="true">or use email</div>

            <div className="field">
              <label htmlFor="email">Email</label>
              <input
                className="input"
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                placeholder="you@example.com"
                required
                aria-invalid={Boolean(emailError)}
                aria-errormessage={emailError ? "email-error" : undefined}
                value={email}
                onChange={(event) => onEmailChange(event.target.value)}
              />
              {emailError ? <p className="field-error" id="email-error" aria-live="polite">{emailError}</p> : null}
            </div>

            <div className="field">
              <div className="label-row">
                <label htmlFor="password">Password</label>
                <a className="helper-link" href="/forgot-password">Forgot password?</a>
              </div>
              <div className="password-control">
                <input
                  className="input"
                  id="password"
                  name="password"
                  type={passwordVisible ? "text" : "password"}
                  autoComplete="current-password"
                  placeholder="Enter your password"
                  required
                  aria-invalid={Boolean(passwordError)}
                  aria-errormessage={passwordError ? "password-error" : undefined}
                  value={password}
                  onChange={(event) => onPasswordChange(event.target.value)}
                />
                <button
                  className="password-toggle"
                  type="button"
                  aria-label={passwordVisible ? "Hide password" : "Show password"}
                  aria-controls="password"
                  aria-pressed={passwordVisible}
                  onClick={() => setPasswordVisible((visible) => !visible)}
                >
                  <span aria-hidden="true">{passwordVisible ? "Hide" : "Show"}</span>
                </button>
              </div>
              {passwordError ? <p className="field-error" id="password-error" aria-live="polite">{passwordError}</p> : null}
            </div>

            <label className="checkbox-label" htmlFor="remember">
              <input className="checkbox" id="remember" name="remember" type="checkbox" />
              Remember this device
            </label>

            {error ? <p className="form-status" id="form-status" role="status" aria-live="polite">{error}</p> : null}
            <button className="button button-secondary" type="submit" disabled={statusLoading}>
              Sign in with email
            </button>
          </form>

          <p className="footer-text">First deployment? <a href="/setup">Run initial setup</a></p>
        </section>
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
  const [submitError, setSubmitError] = useState("");
  const strength = useMemo(() => passwordStrength(password), [password]);
  const emailError = error ? "Check the owner email and try setup again." : "";
  const passwordError = passwordTouched && strength.score < 3 ? "Use a stronger password. Password score must be at least 3 of 4." : "";
  const confirmError = (confirmTouched || confirmPassword) && confirmPassword !== password ? "Passwords do not match." : "";
  const setupConfirmError = submitError === "Please confirm the owner account requirement." ? submitError : "";

  function submitSetup(event: FormEvent) {
    event.preventDefault();
    setPasswordTouched(true);
    setConfirmTouched(true);

    if (strength.score < 3) {
      setSubmitError("Use a stronger password. Password score must be at least 3 of 4.");
      return;
    }

    if (confirmPassword !== password) {
      setSubmitError("Password confirmation does not match.");
      return;
    }

    if (!setupConfirmed) {
      setSubmitError("Please confirm the owner account requirement.");
      return;
    }

    setSubmitError("");
    onSubmit();
  }

  return (
    <main className="setup-page auth-reference-page">
      <PageTransition>
        <section className="setup-card" aria-labelledby="setup-title">
          <a className="brand" href="/" aria-label="CapitalFlow home">
            <img className="brand-mark" src="/app-icon.png" alt="" aria-hidden="true" />
            <span className="brand-text">
              <span className="brand-name">CapitalFlow</span>
              <span className="brand-note">Initial service setup</span>
            </span>
          </a>

          <header className="setup-header">
            <p className="setup-kicker">One-time setup</p>
            <h1 className="setup-title" id="setup-title">Create the owner account</h1>
            <p className="setup-description">This page is available only on the first deployment, before an owner exists.</p>
          </header>

          <aside className="warning-box" aria-labelledby="setup-warning-title">
            <h2 className="warning-title" id="setup-warning-title">
              <span className="warning-icon" aria-hidden="true">!</span>
              Important before continuing
            </h2>
            <p className="warning-text">
              The first account becomes the service owner. After setup succeeds, the backend must disable public
              registration and block this setup page.
            </p>
          </aside>

          <form className="form" action="/setup" method="post" noValidate onSubmit={submitSetup} aria-label="Initial setup form">
            <div className="field">
              <label htmlFor="owner-name">Owner name</label>
              <input className="input" id="owner-name" name="ownerName" type="text" autoComplete="name" placeholder="Denis" />
            </div>

            <div className="field">
              <label htmlFor="owner-email">Owner email</label>
              <input
                className="input"
                id="owner-email"
                name="email"
                type="email"
                autoComplete="email"
                placeholder="you@example.com"
                required
                aria-invalid={Boolean(emailError)}
                aria-errormessage={emailError ? "owner-email-error" : undefined}
                value={email}
                onChange={(event) => onEmailChange(event.target.value)}
              />
              {emailError ? <p className="field-error" id="owner-email-error" aria-live="polite">{emailError}</p> : null}
            </div>

            <div className="field">
              <label htmlFor="owner-password">Password</label>
              <div className="password-control">
                <input
                  className="input"
                  id="owner-password"
                  name="password"
                  type={passwordVisible ? "text" : "password"}
                  autoComplete="new-password"
                  placeholder="Use a strong passphrase"
                  required
                  aria-invalid={Boolean(passwordError)}
                  aria-describedby="password-strength-feedback"
                  aria-errormessage={passwordError ? "owner-password-error" : undefined}
                  value={password}
                  onBlur={() => setPasswordTouched(true)}
                  onChange={(event) => {
                    setSubmitError("");
                    onPasswordChange(event.target.value);
                  }}
                />
                <button
                  className="password-toggle"
                  type="button"
                  data-target="owner-password"
                  aria-label={passwordVisible ? "Hide password" : "Show password"}
                  aria-controls="owner-password"
                  aria-pressed={passwordVisible}
                  onClick={() => setPasswordVisible((visible) => !visible)}
                >
                  <span aria-hidden="true">{passwordVisible ? "Hide" : "Show"}</span>
                </button>
              </div>

              <div className="password-strength" aria-label="Password strength">
                <div className="strength-row">
                  <span>Password strength</span>
                  <strong id="password-strength-label">{strength.label}</strong>
                </div>
                <div
                  className="strength-track"
                  role="meter"
                  aria-label="Password strength score"
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
                <p className="strength-feedback" id="password-strength-feedback" aria-live="polite">
                  {strength.feedback}
                </p>
              </div>

              {passwordError ? <p className="field-error" id="owner-password-error" aria-live="polite">{passwordError}</p> : null}
            </div>

            <div className="field">
              <label htmlFor="owner-password-confirm">Confirm password</label>
              <div className="password-control">
                <input
                  className="input"
                  id="owner-password-confirm"
                  name="passwordConfirm"
                  type={confirmPasswordVisible ? "text" : "password"}
                  autoComplete="new-password"
                  placeholder="Repeat the same password"
                  aria-invalid={Boolean(confirmError)}
                  aria-errormessage={confirmError ? "owner-password-confirm-error" : undefined}
                  value={confirmPassword}
                  onBlur={() => setConfirmTouched(true)}
                  onChange={(event) => {
                    setSubmitError("");
                    setConfirmPassword(event.target.value);
                  }}
                />
                <button
                  className="password-toggle"
                  type="button"
                  data-target="owner-password-confirm"
                  aria-label={confirmPasswordVisible ? "Hide password confirmation" : "Show password confirmation"}
                  aria-controls="owner-password-confirm"
                  aria-pressed={confirmPasswordVisible}
                  onClick={() => setConfirmPasswordVisible((visible) => !visible)}
                >
                  <span aria-hidden="true">{confirmPasswordVisible ? "Hide" : "Show"}</span>
                </button>
              </div>
              {confirmError ? <p className="field-error" id="owner-password-confirm-error" aria-live="polite">{confirmError}</p> : null}
            </div>

            <div className="field">
              <label htmlFor="primary-currency">Primary currency</label>
              <select
                className="input"
                id="primary-currency"
                name="primaryCurrency"
                value={primaryCurrency}
                onChange={(event) => onPrimaryCurrencyChange(event.target.value)}
              >
                {currencyOptions.map((currency) => (
                  <option key={currency.code} value={currency.code}>
                    {currency.label}
                  </option>
                ))}
              </select>
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
                aria-errormessage={setupConfirmError ? "setup-confirm-error" : undefined}
                onChange={(event) => {
                  setSubmitError("");
                  setSetupConfirmed(event.target.checked);
                }}
              />
              I understand that this account becomes the service owner and that registration must be closed after setup.
            </label>
            {setupConfirmError ? <p className="field-error" id="setup-confirm-error" aria-live="polite">{setupConfirmError}</p> : null}

            {submitError ? <p className="form-status" id="form-status" role="status" aria-live="polite">{submitError}</p> : null}
            {error ? <p className="form-status" id="form-status-api" role="status" aria-live="polite">{error}</p> : null}
            <button className="button button-primary" id="setup-submit" type="submit" disabled={statusLoading}>
              Create owner account
            </button>
          </form>

          <p className="footer-text">Setup already completed? <a href="/login">Go to sign in</a></p>
        </section>
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

function passwordStrength(password: string) {
  if (!password) {
    return { score: 0, label: "Not checked", feedback: "Use a memorable passphrase. Score 3 of 4 is required." };
  }

  const result = zxcvbn(password);
  const score = result.score;
  const labels = ["Weak", "Weak", "Fair", "Good", "Strong"];
  const suggestion = result.feedback.suggestions[0] || result.feedback.warning;
  const feedback = score >= 3
    ? "Acceptable for setup."
    : suggestion || "Use a longer, less common passphrase. Score 3 of 4 is required.";

  return {
    score,
    label: labels[score],
    feedback,
  };
}
