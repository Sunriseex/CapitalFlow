import { useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { api } from "../../api/client";
import type { Profile } from "../../api/types";
import { errorMessage } from "../../shared/api/query";
import { currencyOptions } from "../../shared/currencies";
import { Button, Field, Input, Panel, Select } from "../../shared/ui";
import { PasskeysPanel } from "./PasskeysPanel";
import { useI18n } from "../../shared/i18n/useI18n";

export function SettingsView({ profile }: { profile?: Profile }) {
  const { t } = useI18n();

  const queryClient = useQueryClient();

  const profileCurrency = profile?.user.primary_currency ?? "RUB";
  const [draftCurrency, setDraftCurrency] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  const primaryCurrency = draftCurrency ?? profileCurrency;
  const currencies = useMemo(() => currencyOptions(), []);

  async function save() {
    if (!profile) {
      return;
    }

    setError("");
    setSaved(false);

    try {
      const updatedProfile = await api.updateProfile({
        primary_currency: primaryCurrency,
      });

      setDraftCurrency(updatedProfile.user.primary_currency);
      await queryClient.invalidateQueries({ queryKey: ["profile"] });
      await queryClient.invalidateQueries({ queryKey: ["dashboard"] });

      setSaved(true);
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  return (
    <div className="grid settings-grid workspace-settings">
      <Panel
        className="workspace-panel settings-panel profile-settings-panel"
        title={t.settings.profile}
      >
        <form
          className="form compact-form"
          onSubmit={(event) => {
            event.preventDefault();
            void save();
          }}
        >
          <Field label={t.settings.email}>
            {" "}
            <Input value={profile?.user.email ?? ""} readOnly />
          </Field>
          <Field label={t.settings.primaryCurrency}>
            {" "}
            <Select
              value={primaryCurrency}
              disabled={!profile}
              onChange={(event) => {
                setDraftCurrency(event.target.value);
                setSaved(false);
              }}
            >
              {currencies.map((currency) => (
                <option key={currency.code} value={currency.code}>
                  {currency.label}
                </option>
              ))}
            </Select>
          </Field>
          {error ? <div className="error">{error}</div> : null}
          {saved ? <div className="success">{t.settings.saved}</div> : null}
          <Button disabled={!profile}>{t.settings.saveSettings}</Button>{" "}
        </form>
      </Panel>

      <PasskeysPanel />
    </div>
  );
}
