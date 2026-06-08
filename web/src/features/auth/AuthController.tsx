import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ApiClientError, api } from "../../api/client";
import { currencyOptions } from "../../shared/currencies";
import {
  browserSupportsPasskeys,
  passkeyErrorMessage,
  signInWithPasskey,
} from "./passkeys";
import {
  AuthStatusScreen,
  InitialSetupScreen,
  LoginScreen,
} from "./AuthScreens";
import type { AuthScreenError } from "./AuthScreens";
import { useI18n } from "../../shared/i18n/useI18n";

export function AuthController({
  onAuthenticated,
}: {
  onAuthenticated: () => void;
}) {
  const { t } = useI18n();
  const status = useQuery({
    queryKey: ["auth-status"],
    queryFn: api.authStatus,
    retry: false,
  });
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [primaryCurrency, setPrimaryCurrency] = useState("RUB");
  const [error, setError] = useState<AuthScreenError>(null);
  const [passkeyError, setPasskeyError] = useState("");
  const [passkeyLoading, setPasskeyLoading] = useState(false);
  const currencies = useMemo(() => currencyOptions(), []);

  const isSetup = status.data?.setup_required === true;
  const passkeysSupported = browserSupportsPasskeys();

  async function submit() {
    setError(null);

    try {
      if (isSetup) {
        await api.setup({ email, password, primary_currency: primaryCurrency });
      } else {
        await api.login({ email, password });
      }

      onAuthenticated();
    } catch (err) {
      setError(authScreenError(err, t));
    }
  }

  async function submitPasskey() {
    setError(null);
    setPasskeyError("");
    setPasskeyLoading(true);

    try {
      await signInWithPasskey();
      onAuthenticated();
    } catch (err) {
      setPasskeyError(passkeyErrorMessage(err));
    } finally {
      setPasskeyLoading(false);
    }
  }

  if (status.isLoading) {
    return (
      <AuthStatusScreen
        title={t.auth.checkingAccess}
        message={t.auth.checkingAccessMessage}
      />
    );
  }

  if (status.error) {
    return (
      <AuthStatusScreen
        title={t.auth.authenticationUnavailable}
        message={errorText(status.error, t)}
        action={{
          label: t.auth.retryStatusCheck,
          onClick: () => {
            void status.refetch();
          },
        }}
      />
    );
  }

  if (isSetup) {
    return (
      <InitialSetupScreen
        email={email}
        password={password}
        primaryCurrency={primaryCurrency}
        currencyOptions={currencies}
        error={error}
        statusLoading={status.isLoading}
        onEmailChange={setEmail}
        onPasswordChange={setPassword}
        onPrimaryCurrencyChange={setPrimaryCurrency}
        onSubmit={() => {
          void submit();
        }}
      />
    );
  }

  return (
    <LoginScreen
      email={email}
      password={password}
      error={error}
      passkeyError={passkeyError}
      passkeysSupported={passkeysSupported}
      passkeyLoading={passkeyLoading}
      statusLoading={status.isLoading}
      onEmailChange={setEmail}
      onPasswordChange={setPassword}
      onSubmit={() => {
        void submit();
      }}
      onPasskeySubmit={() => {
        void submitPasskey();
      }}
    />
  );
}

function authScreenError(
  err: unknown,
  t: ReturnType<typeof useI18n>["t"],
): AuthScreenError {
  const message = errorText(err, t);
  if (
    err instanceof ApiClientError &&
    (err.status === 400 || err.status === 401)
  ) {
    return { kind: "field", message };
  }

  return { kind: "global", message };
}

function errorText(err: unknown, t: ReturnType<typeof useI18n>["t"]) {
  if (err instanceof ApiClientError) {
    return err.message;
  }

  if (err instanceof Error) {
    return err.message;
  }

  return t.auth.requestFailed;
}
