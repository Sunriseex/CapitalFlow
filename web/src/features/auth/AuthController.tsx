import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ApiClientError, api } from "../../api/client";
import { currencyOptions } from "../../shared/currencies";
import { browserSupportsPasskeys, passkeyErrorMessage, signInWithPasskey } from "./passkeys";
import { InitialSetupScreen, LoginScreen } from "./AuthScreens";

export function AuthController({ onAuthenticated }: { onAuthenticated: () => void }) {
  const status = useQuery({ queryKey: ["auth-status"], queryFn: api.authStatus, retry: false });
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [primaryCurrency, setPrimaryCurrency] = useState("RUB");
  const [error, setError] = useState("");
  const [passkeyError, setPasskeyError] = useState("");
  const [passkeyLoading, setPasskeyLoading] = useState(false);

  const isSetup = status.data?.setup_required === true;
  const passkeysSupported = browserSupportsPasskeys();

  async function submit() {
    setError("");

    try {
      if (isSetup) {
        await api.setup({ email, password, primary_currency: primaryCurrency });
      } else {
        await api.login({ email, password });
      }

      onAuthenticated();
    } catch (err) {
      setError(errorText(err));
    }
  }

  async function submitPasskey() {
    setError("");
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

  if (isSetup) {
    return (
      <InitialSetupScreen
        email={email}
        password={password}
        primaryCurrency={primaryCurrency}
        currencyOptions={currencyOptions()}
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

function errorText(err: unknown) {
  if (err instanceof ApiClientError) {
    return err.message;
  }

  if (err instanceof Error) {
    return err.message;
  }

  return "Request failed";
}
