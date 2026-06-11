import { formatMoney } from "../../../api/money";
import type {
  Account,
  DashboardAccountBalance,
  InterestRule,
} from "../../../api/types";
import { apiErrorMessages, errorMessage } from "../../../shared/api/query";
import { Button } from "../../../shared/ui";
import { useI18n } from "../../../shared/i18n/useI18n";

export function AccountsTable({
  accounts,
  balances,
  activeRules,
  rulesLoading,
  rulesError,
  onSelect,
}: {
  accounts: Account[];
  balances: Map<string, DashboardAccountBalance>;
  activeRules: Map<string, InterestRule>;
  rulesLoading: boolean;
  rulesError: unknown;
  onSelect: (id: string) => void;
}) {
  const { t, locale } = useI18n();
  return (
    <div className="table-wrap workspace-table-wrap accounts-table-wrap">
      <table className="workspace-table accounts-table">
        <thead>
          <tr>
            <th>{t.accounts.name}</th>
            <th>{t.accounts.bank}</th>
            <th>{t.accounts.type}</th>
            <th>{t.accounts.balance}</th>
            <th>{t.accounts.rate}</th>
            <th>{t.accounts.status}</th>
            <th>{t.accounts.action}</th>
          </tr>
        </thead>
        <tbody>
          {accounts.map((account) => (
            <tr key={account.id}>
              <td data-label={t.accounts.name}>{account.name}</td>
              <td data-label={t.accounts.bank}>{account.bank || "-"}</td>
              <td data-label={t.accounts.type}>
                {t.accounts.types[account.type]}
              </td>
              <td data-label={t.accounts.balance} className="amount">
                {formatMoney(
                  balances.get(account.id)?.balance ?? "0",
                  account.currency,
                  locale,
                )}
              </td>
              <td data-label={t.accounts.rate}>
                <AccountRate
                  rule={activeRules.get(account.id)}
                  isLoading={rulesLoading}
                  error={rulesError}
                  loadingLabel={t.common.loading}
                />
              </td>
              <td data-label={t.accounts.status}>
                {account.is_active ? t.accounts.active : t.accounts.archived}
              </td>
              <td data-label={t.accounts.action}>
                <Button onClick={() => onSelect(account.id)}>
                  {t.common.open}
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function AccountRate({
  rule,
  isLoading,
  error,
  loadingLabel,
}: {
  rule?: InterestRule;
  isLoading: boolean;
  error: unknown;
  loadingLabel: string;
}) {
  const { t } = useI18n();
  const errorMessages = apiErrorMessages(t);
  if (isLoading) {
    return <span>{loadingLabel}</span>;
  }

  if (error) {
    return (
      <span className="error-text">{errorMessage(error, errorMessages)}</span>
    );
  }

  if (!rule) {
    return <span>-</span>;
  }

  return <span>{(rule.annual_rate_bps / 100).toFixed(2)}%</span>;
}
