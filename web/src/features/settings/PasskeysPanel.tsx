import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { KeyRound, Pencil, Plus, Trash2 } from "lucide-react";
import { api } from "../../api/client";
import { registerPasskey, browserSupportsPasskeys, passkeyErrorMessage } from "../auth/passkeys";
import { errorMessage } from "../../shared/api/query";
import { Button, Empty, IconButton, Input, Panel } from "../../shared/ui";

export function PasskeysPanel() {
  const queryClient = useQueryClient();
  const passkeys = useQuery({ queryKey: ["passkeys"], queryFn: () => api.passkeys?.() ?? Promise.resolve([]) });
  const [password, setPassword] = useState("");
  const [editingID, setEditingID] = useState("");
  const [editingName, setEditingName] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const supported = browserSupportsPasskeys();

  async function addPasskey() {
    if (!password) {
      setError("Password confirmation is required");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await registerPasskey(password);
      setPassword("");
      await queryClient.invalidateQueries({ queryKey: ["passkeys"] });
    } catch (err) {
      setError(passkeyErrorMessage(err));
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
      setError(errorMessage(err));
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
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Panel title="Security">
      <div className="form compact-form">
        <div>
          <strong>Passkeys</strong>
          <p className="muted-text">Use a device passkey to sign in without typing your password.</p>
        </div>

        {!supported ? <div className="error">This browser does not support passkeys</div> : null}
        {error ? <div className="error">{error}</div> : null}
        {passkeys.error ? <div className="error">{errorMessage(passkeys.error)}</div> : null}

        <div className="inline-form">
          <Input
            type="password"
            placeholder="Password confirmation"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
          <Button
            className="primary-button"
            disabled={!supported || busy || !password}
            onClick={() => {
              void addPasskey();
            }}
          >
            <Plus size={16} /> Add passkey
          </Button>
        </div>

        {passkeys.isLoading ? <Empty>Loading passkeys</Empty> : null}

        {!passkeys.isLoading && (passkeys.data?.length ?? 0) === 0 ? <Empty>No passkeys yet</Empty> : null}

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
                        aria-label="Passkey name"
                      />
                    ) : (
                      <strong>{passkey.name}</strong>
                    )}
                    <p className="muted-text">
                      Last used {passkey.last_used_at ? new Date(passkey.last_used_at).toLocaleString() : "never"}
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
                      Save
                    </Button>
                  ) : (
                    <IconButton
                      type="button"
                      title="Rename passkey"
                      aria-label="Rename passkey"
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
                    title="Delete passkey"
                    aria-label="Delete passkey"
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
