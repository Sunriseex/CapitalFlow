import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { KeyRound, Pencil, Plus, Trash2 } from "lucide-react";
import { api } from "../../api/client";
import {
  registerPasskey,
  browserSupportsPasskeys,
  passkeyErrorMessage,
} from "../auth/passkeys";
import { apiErrorMessages, errorMessage } from "../../shared/api/query";
import {
  Button,
  Empty,
  Field,
  IconButton,
  Input,
  Panel,
} from "../../shared/ui";
import { useI18n } from "../../shared/i18n/useI18n";
import { dateTimeLabel } from "../../shared/date";

export function PasskeysPanel() {
  const { t, locale } = useI18n();
  const errorMessages = apiErrorMessages(t);
  const passkeyErrorMessages = {
    operationCancelled: t.settings.passkeyOperationCancelled,
    operationFailed: t.settings.passkeyOperationFailed,
  };

  const queryClient = useQueryClient();
  const passkeys = useQuery({
    queryKey: ["passkeys"],
    queryFn: () => api.passkeys?.() ?? Promise.resolve([]),
  });
  const [password, setPassword] = useState("");
  const [editingID, setEditingID] = useState("");
  const [editingName, setEditingName] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const supported = browserSupportsPasskeys();

  async function addPasskey() {
    if (!password) {
      setError(t.settings.passwordConfirmationRequired);
      return;
    }
    setBusy(true);
    setError("");
    try {
      await registerPasskey(password);
      setPassword("");
      await queryClient.invalidateQueries({ queryKey: ["passkeys"] });
    } catch (err) {
      setError(passkeyErrorMessage(err, passkeyErrorMessages));
    } finally {
      setBusy(false);
    }
  }

  async function renamePasskey(id: string) {
    setBusy(true);
    setError("");
    try {
      await api.renamePasskey(id, { name: editingName });
      setEditingID("");
      setEditingName("");
      await queryClient.invalidateQueries({ queryKey: ["passkeys"] });
    } catch (err) {
      setError(errorMessage(err, errorMessages));
    } finally {
      setBusy(false);
    }
  }

  async function deletePasskey(id: string) {
    setBusy(true);
    setError("");
    try {
      await api.deletePasskey(id);
      await queryClient.invalidateQueries({ queryKey: ["passkeys"] });
    } catch (err) {
      setError(errorMessage(err, errorMessages));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Panel
      className="workspace-panel settings-panel security-settings-panel"
      title={t.settings.security}
    >
      <div className="form compact-form">
        <div>
          <strong>{t.settings.passkeys}</strong>
          <p className="muted-text">{t.settings.passkeysDescription}</p>
        </div>

        {!supported ? (
          <div className="error">{t.settings.passkeysUnsupported}</div>
        ) : null}
        {error ? <div className="error">{error}</div> : null}
        {passkeys.error ? (
          <div className="error">
            {errorMessage(passkeys.error, errorMessages)}
          </div>
        ) : null}

        <div className="inline-form">
          <Field label={t.settings.passwordConfirmation}>
            {" "}
            <Input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </Field>
          <Button
            className="primary-button"
            disabled={!supported || busy || !password}
            onClick={() => {
              void addPasskey();
            }}
          >
            <Plus size={16} /> {t.settings.addPasskey}{" "}
          </Button>
        </div>

        {passkeys.isLoading ? (
          <Empty>{t.settings.loadingPasskeys}</Empty>
        ) : null}
        {!passkeys.isLoading && (passkeys.data?.length ?? 0) === 0 ? (
          <Empty>{t.settings.noPasskeysYet}</Empty>
        ) : null}

        <div className="passkey-list">
          {passkeys.data?.map((passkey) => (
            <div key={passkey.id} className="passkey-row">
              <div className="passkey-main">
                <KeyRound size={18} />
                <div>
                  {editingID === passkey.id ? (
                    <Input
                      value={editingName}
                      onChange={(event) => setEditingName(event.target.value)}
                      aria-label={t.settings.passkeyName}
                    />
                  ) : (
                    <strong>{passkey.name}</strong>
                  )}
                  <p className="muted-text">
                    {t.settings.lastUsed}{" "}
                    {passkey.last_used_at
                      ? dateTimeLabel(passkey.last_used_at, locale)
                      : t.settings.never}
                  </p>
                </div>
              </div>

              <div className="passkey-actions">
                {editingID === passkey.id ? (
                  <Button
                    disabled={busy}
                    onClick={() => {
                      void renamePasskey(passkey.id);
                    }}
                  >
                    {t.common.save}{" "}
                  </Button>
                ) : (
                  <IconButton
                    type="button"
                    title={t.settings.renamePasskey}
                    aria-label={t.settings.renamePasskey}
                    onClick={() => {
                      setEditingID(passkey.id);
                      setEditingName(passkey.name);
                    }}
                  >
                    <Pencil size={16} />
                  </IconButton>
                )}
                <IconButton
                  type="button"
                  title={t.settings.deletePasskey}
                  aria-label={t.settings.deletePasskey}
                  disabled={busy}
                  onClick={() => {
                    void deletePasskey(passkey.id);
                  }}
                >
                  <Trash2 size={16} />
                </IconButton>
              </div>
            </div>
          ))}
        </div>
      </div>
    </Panel>
  );
}
