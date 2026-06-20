import { useEffect, useMemo, useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { api } from "../../api/client";
import type { Profile } from "../../api/types";
import { apiErrorMessages, errorMessage } from "../../shared/api/query";
import { currencyOptions } from "../../shared/currencies";
import { Button, Field, Input, Panel, Select } from "../../shared/ui";
import { PasskeysPanel } from "./PasskeysPanel";
import { useI18n } from "../../shared/i18n/useI18n";

type SettingsFormValues = {
  primary_currency: string;
};

const settingsFormSchema = z.object({
  primary_currency: z.string().min(1),
});

export function SettingsView({ profile }: { profile?: Profile }) {
  const { t, locale } = useI18n();
  const errorMessages = apiErrorMessages(t);
  const queryClient = useQueryClient();

  const profileCurrency = profile?.user.primary_currency ?? "RUB";
  const [error, setError] = useState("");
  const [savedKey, setSavedKey] = useState<string | null>(null);

  const currencies = useMemo(() => currencyOptions(locale), [locale]);
  const { control, handleSubmit, register, reset } =
    useForm<SettingsFormValues>({
      defaultValues: { primary_currency: profileCurrency },
      mode: "onBlur",
      resolver: zodResolver(settingsFormSchema),
    });
  const primaryCurrency =
    useWatch({ control, name: "primary_currency" }) ?? profileCurrency;
  const currentSavedKey = profile
    ? `${profile.user.id}:${primaryCurrency}`
    : null;
  const saved = Boolean(savedKey && savedKey === currentSavedKey);

  useEffect(() => {
    reset({ primary_currency: profileCurrency });
  }, [profileCurrency, reset]);

  async function save(values: SettingsFormValues) {
    if (!profile) {
      return;
    }

    setError("");
    setSavedKey(null);

    try {
      const updatedProfile = await api.updateProfile({
        primary_currency: values.primary_currency,
      });

      reset({ primary_currency: updatedProfile.user.primary_currency });
      await queryClient.invalidateQueries({ queryKey: ["profile"] });
      await queryClient.invalidateQueries({ queryKey: ["dashboard"] });

      setSavedKey(
        `${updatedProfile.user.id}:${updatedProfile.user.primary_currency}`,
      );
    } catch (err) {
      setError(errorMessage(err, errorMessages));
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
          onSubmit={handleSubmit((values) => {
            void save(values);
          })}
        >
          <Field label={t.settings.email}>
            <Input value={profile?.user.email ?? ""} readOnly />
          </Field>
          <Field label={t.settings.primaryCurrency}>
            <Select
              disabled={!profile}
              {...register("primary_currency", {
                onChange: () => {
                  setError("");
                  setSavedKey(null);
                },
              })}
              value={primaryCurrency}
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
          <Button disabled={!profile}>{t.settings.saveSettings}</Button>
        </form>
      </Panel>

      <PasskeysPanel />
    </div>
  );
}
