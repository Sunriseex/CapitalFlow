import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { CreditCard, Repeat, Target, Zap } from "lucide-react";
import { api } from "../../api/client";
import {
  addMoney,
  compareMoney,
  convertMinor,
  formatMoney,
  moneyToNumber,
  sumConverted,
} from "../../api/money";
import type { Account, Transaction } from "../../api/types";
import { apiErrorMessages, errorMessage } from "../../shared/api/query";
import type { QuickAction, View } from "../../shared/constants";
import { Button, Dialog, Empty } from "../../shared/ui";
import { TransactionDetails } from "../transactions/components/TransactionDetails";
import { CashflowChart } from "./components/CashflowChart";
import { RecentTransactionsTable } from "./components/RecentTransactionsTable";
import { useI18n } from "../../shared/i18n/useI18n";
import {
  cashflowBucketsToChart,
  cashflowPeriods,
  formatChartMoney,
  groupCashflow,
  type CashflowPeriod,
} from "./lib/cashflow";

export function DashboardView({
  primaryCurrency,
  rightRailHidden,
  onOpenAccount,
  onQuickAction,
  onNavigate,
  quickActionsDisabled = false,
}: {
  primaryCurrency: string;
  rightRailHidden: boolean;
  onOpenAccount: (id: string) => void;
  onQuickAction?: (action: NonNullable<QuickAction>) => void;
  onNavigate?: (view: View) => void;
  quickActionsDisabled?: boolean;
}) {
  const { t, locale } = useI18n();
  const errorMessages = apiErrorMessages(t);

  const summary = useQuery({
    queryKey: ["dashboard", "summary"],
    queryFn: api.dashboardSummary,
  });
  const cashflow = useQuery({
    queryKey: ["dashboard", "cashflow"],
    queryFn: api.dashboardCashflow,
  });
  const interestIncome = useQuery({
    queryKey: ["dashboard", "interest-income"],
    queryFn: api.dashboardInterestIncome,
  });
  const [selectedCurrency, setSelectedCurrency] = useState(primaryCurrency);
  const [cashflowPeriod, setCashflowPeriod] = useState<CashflowPeriod>("month");
  const [selectedTransaction, setSelectedTransaction] =
    useState<Transaction | null>(null);
  const data = summary.data;

  const balances = useMemo(
    () => data?.account_balances ?? [],
    [data?.account_balances],
  );
  const currencyTotals = useMemo(() => data?.balances ?? [], [data?.balances]);
  const seenCurrencies = new Set<string>([selectedCurrency]);
  for (const amount of currencyTotals) {
    seenCurrencies.add(amount.currency);
  }
  for (const account of balances) {
    seenCurrencies.add(account.currency);
  }
  const currencies = [...seenCurrencies].sort();
  const rates = useQuery({
    queryKey: ["currency-rates", selectedCurrency],
    queryFn: () => api.currencyRates(selectedCurrency),
    enabled: Boolean(selectedCurrency),
    staleTime: 1000 * 60 * 60,
  });
  const rateTable =
    rates.data?.base === selectedCurrency ? rates.data : undefined;
  const portfolioValue = sumConverted(
    currencyTotals,
    selectedCurrency,
    rateTable,
  );
  const portfolioValueNumber = useMemo(
    () => moneyToNumber(portfolioValue),
    [portfolioValue],
  );
  const allocation = useMemo(
    () =>
      balances
        .filter((account) => compareMoney(account.balance, "0") > 0)
        .map((account) => ({
          ...account,
          converted_balance: convertMinor(
            account.balance,
            account.currency,
            selectedCurrency,
            rateTable,
          ),
        }))
        .sort((a, b) => compareMoney(b.converted_balance, a.converted_balance))
        .slice(0, 6)
        .map((account) => ({
          ...account,
          share:
            portfolioValueNumber > 0
              ? Math.round(
                  (moneyToNumber(account.converted_balance) /
                    portfolioValueNumber) *
                    100,
                )
              : 0,
        })),
    [balances, portfolioValueNumber, rateTable, selectedCurrency],
  );

  const monthlyNet = addMoney(
    sumConverted(data?.monthly_income, selectedCurrency, rateTable),
    `-${sumConverted(data?.monthly_expense, selectedCurrency, rateTable)}`,
  );
  const recentAccounts = useMemo(
    () =>
      balances.map(
        (account): Account => ({
          id: account.account_id,
          name: account.name,
          bank: account.bank,
          type: account.type,
          currency: account.currency,
          is_active: account.is_active,
          opened_at: "",
          created_at: "",
          updated_at: "",
        }),
      ),
    [balances],
  );
  const cashflowBuckets = useMemo(
    () =>
      cashflowBucketsToChart(
        cashflow.data?.buckets ?? [],
        selectedCurrency,
        rateTable,
      ),
    [cashflow.data?.buckets, rateTable, selectedCurrency],
  );
  const cashflowChart = useMemo(
    () => groupCashflow(cashflowBuckets, cashflowPeriod),
    [cashflowBuckets, cashflowPeriod],
  );
  const cashflowEmpty =
    cashflowPeriod === "week" && cashflowBuckets.length > 0
      ? {
          title: t.dashboard.weeklyCashflowUnavailable,
          description: t.dashboard.backendReturnsMonthlyCashflow,
        }
      : {
          title: t.dashboard.noCashflowYet,
          description: t.dashboard.addIncomeOrExpensesToBuildChart,
        };
  const totalInterest = sumConverted(
    interestIncome.data?.total,
    selectedCurrency,
    rateTable,
  );
  const chartSummary = useMemo(() => {
    if (!cashflowChart.length) {
      return t.dashboard.cashflowChartHasNoPeriods;
    }

    const totalIncome = cashflowChart.reduce(
      (sum, bucket) => sum + bucket.income,
      0,
    );
    const totalExpense = cashflowChart.reduce(
      (sum, bucket) => sum + bucket.expense,
      0,
    );
    const totalNet = cashflowChart.reduce((sum, bucket) => sum + bucket.net, 0);

    return t.dashboard.cashflowChartSummary
      .replace("{count}", String(cashflowChart.length))
      .replace(
        "{income}",
        formatChartMoney(totalIncome, selectedCurrency, locale),
      )
      .replace(
        "{expenses}",
        formatChartMoney(totalExpense, selectedCurrency, locale),
      )
      .replace("{net}", formatChartMoney(totalNet, selectedCurrency, locale));
  }, [cashflowChart, selectedCurrency, t, locale]);

  if (summary.isLoading) {
    return <Empty>{t.dashboard.loadingDashboard}</Empty>;
  }

  if (summary.error) {
    return <Empty>{errorMessage(summary.error, errorMessages)}</Empty>;
  }

  return (
    <div className="ref-dashboard">
      <section className="tab-panel" id="overview" aria-labelledby="pageTitle">
            <section className="reference-note" aria-label={t.dashboard.layoutNoteTitle}>
              <strong>{t.dashboard.layoutNoteTitle}</strong>
              <p>{t.dashboard.layoutNoteDescription}</p>
            </section>

            <section className="reference-alert" aria-label={t.dashboard.subscriptionAlertTitle}>
              <strong>{t.dashboard.subscriptionAlertTitle}</strong>
              <p>{t.dashboard.subscriptionAlertDescription}</p>
              <Button type="button" onClick={() => onNavigate?.("transactions")}>
                {t.nav.transactions}
              </Button>
            </section>

            <section className="metrics-grid" aria-label={t.dashboard.overview}>
              <article className="card balance-card metric-card">
                <div className="metric-card-head">
                  <div className="balance-title">
                    <span>{t.dashboard.totalCapital}</span>
                    <small>{t.dashboard.allActiveAccounts}</small>
                  </div>
                  <span className="pill">{t.dashboard.liveLedger}</span>
                </div>
                <div className="metric-value">
                  {formatMoney(portfolioValue, selectedCurrency, locale)}
                </div>
                <span
                  className={
                    compareMoney(monthlyNet, "0") < 0
                      ? "delta-down"
                      : "delta-up"
                  }
                >
                  {formatMoney(monthlyNet, selectedCurrency, locale)}{" "}
                  {t.dashboard.thisMonth}
                </span>
              </article>

              <article className="card metric-card">
                <div className="metric-card-head">
                  <div className="balance-title">
                    <span>{t.dashboard.expenses}</span>
                    <small>{t.dashboard.thisMonth}</small>
                  </div>
                  <span className="pill">
                    {data?.recent_transactions_returned ?? 0}
                  </span>
                </div>
                <div className="metric-value">
                  {formatMoney(
                    sumConverted(
                      data?.monthly_expense,
                      selectedCurrency,
                      rateTable,
                    ),
                    selectedCurrency,
                    locale,
                  )}
                </div>
                <span>{t.dashboard.real}</span>
              </article>

              <article className="card metric-card">
                <div className="metric-card-head">
                  <div className="balance-title">
                    <span>{t.dashboard.reserveFund}</span>
                    <small>{t.dashboard.topPositiveBalances}</small>
                  </div>
                  <span className="pill">{allocation[0]?.share ?? 0}%</span>
                </div>
                <div className="metric-value">
                  {allocation[0]
                    ? formatMoney(
                        allocation[0].balance,
                        allocation[0].currency,
                        locale,
                      )
                    : formatMoney("0", selectedCurrency, locale)}
                </div>
                <span>{allocation[0]?.name ?? t.dashboard.noPositiveBalances}</span>
              </article>

              <article className="card metric-card">
                <div className="metric-card-head">
                  <div className="balance-title">
                    <span>{t.dashboard.subscriptions}</span>
                    <small>{t.dashboard.subscriptionsDescription}</small>
                  </div>
                  <span className="pill">{t.common.notAvailable}</span>
                </div>
                <div className="metric-value">
                  {formatMoney("0", selectedCurrency, locale)}
                </div>
                <span>{t.dashboard.emptySubscriptionsTitle}</span>
              </article>
            </section>

            <div
              className="currency-switcher"
              role="group"
              aria-label={t.dashboard.portfolioCurrency}
            >
              {currencies.map((currency) => (
                <button
                  key={currency}
                  className={
                    currency === selectedCurrency
                      ? "period-btn is-active"
                      : "period-btn"
                  }
                  type="button"
                  aria-pressed={currency === selectedCurrency}
                  onClick={() => setSelectedCurrency(currency)}
                >
                  {currency}
                </button>
              ))}
            </div>

        <div className={rightRailHidden ? "layout is-rail-collapsed" : "layout"}>
          <div className="content">
            <article className="card chart-card">
              <div className="card-head">
                <div className="card-title">
                  <h2>
                    {t.dashboard.cashflow} ({selectedCurrency})
                  </h2>{" "}
                  <p>
                    {cashflow.isLoading
                      ? t.dashboard.loadingLedgerBuckets
                      : cashflowBucketsLabel(
                          cashflowChart.length,
                          cashflowPeriod,
                          t,
                          locale,
                        )}
                  </p>
                </div>
                <div>
                  <div
                    className="period-switcher"
                    role="group"
                    aria-label={t.dashboard.cashflowPeriod}
                  >
                    {cashflowPeriods.map((period) => (
                      <button
                        key={period.value}
                        className={
                          period.value === cashflowPeriod
                            ? "period-btn is-active"
                            : "period-btn"
                        }
                        type="button"
                        aria-pressed={period.value === cashflowPeriod}
                        onClick={() => setCashflowPeriod(period.value)}
                      >
                        {t.dashboard.periods[period.value]}{" "}
                      </button>
                    ))}
                  </div>
                </div>
              </div>

              {cashflow.error ? (
                <div className="empty-state">
                  <strong>{errorMessage(cashflow.error, errorMessages)}</strong>
                  <span>{t.dashboard.cashflowChartCouldNotBeLoaded}</span>{" "}
                </div>
              ) : null}
              {!cashflow.error &&
              !cashflow.isLoading &&
              !cashflowChart.length ? (
                <div className="empty-state">
                  <strong>{cashflowEmpty.title}</strong>
                  <span>{cashflowEmpty.description}</span>
                </div>
              ) : null}
              {!cashflow.error && cashflowChart.length ? (
                <div
                  className="chart-wrap"
                  aria-label={t.dashboard.incomeAndExpenseChart}
                >
                  <CashflowChart
                    data={cashflowChart}
                    currency={selectedCurrency}
                    summary={chartSummary}
                  />
                  <table className="sr-only-table">
                    <caption>{t.dashboard.cashflowData}</caption>
                    <thead>
                      <tr>
                        <th scope="col">{t.dashboard.period}</th>
                        <th scope="col">{t.dashboard.income}</th>
                        <th scope="col">{t.dashboard.expense}</th>
                        <th scope="col">{t.dashboard.net}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {cashflowChart.map((bucket) => (
                        <tr key={bucket.period}>
                          <td>{bucket.period}</td>
                          <td>
                            {formatChartMoney(
                              bucket.income,
                              selectedCurrency,
                              locale,
                            )}
                          </td>
                          <td>
                            {formatChartMoney(
                              bucket.expense,
                              selectedCurrency,
                              locale,
                            )}
                          </td>
                          <td>
                            {formatChartMoney(
                              bucket.net,
                              selectedCurrency,
                              locale,
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : null}

              <div className="legend">
                <span className="legend-item">
                  <span className="legend-mark income"></span>
                  {t.dashboard.income} ·{" "}
                  {formatMoney(
                    sumConverted(
                      data?.monthly_income,
                      selectedCurrency,
                      rateTable,
                    ),
                    selectedCurrency,
                    locale,
                  )}
                </span>
                <span className="legend-item">
                  <span className="legend-mark expense"></span>
                  {t.dashboard.expenses} ·{" "}
                  {formatMoney(
                    sumConverted(
                      data?.monthly_expense,
                      selectedCurrency,
                      rateTable,
                    ),
                    selectedCurrency,
                    locale,
                  )}
                </span>
                <span className="legend-item">
                  <span className="legend-mark net"></span>
                  {t.dashboard.net}{" "}
                </span>
                <span className="legend-item">
                  {t.dashboard.interest} ·{" "}
                  {formatMoney(totalInterest, selectedCurrency, locale)}
                </span>
              </div>
            </article>

            <article className="card">
              <div className="card-head">
                <div className="card-title">
                  <h2>{t.dashboard.recentTransactions}</h2>{" "}
                  <p>{t.dashboard.recentTransactionsDescription}</p>{" "}
                </div>
                <button
                  className="btn"
                  type="button"
                  onClick={() => onNavigate?.("transactions")}
                >
                  {t.dashboard.allTransactions}{" "}
                </button>
              </div>
              <RecentTransactionsTable
                accounts={recentAccounts}
                transactions={data?.recent_transactions ?? []}
                selectedCurrency={selectedCurrency}
                onOpenTransaction={setSelectedTransaction}
              />
            </article>
          </div>

          <aside
          id="dashboard-right-rail"
          className="right-rail"
          aria-label={t.dashboard.rightRailSummary}
          aria-hidden={rightRailHidden}
        >
          <article className="card rail-card">
            <div className="card-head">
              <div className="card-title">
                <h2>{t.dashboard.quickActions}</h2>
                <p>{t.dashboard.quickActionsDescription}</p>
              </div>
              <Zap aria-hidden="true" />
            </div>
            <div className="rail-actions" role="group" aria-label={t.dashboard.quickActions}>
              <button
                className="btn primary"
                type="button"
                disabled={quickActionsDisabled}
                onClick={() => onQuickAction?.("transaction")}
              >
                {t.dashboard.addTransaction}
              </button>
              <button
                className="btn"
                type="button"
                onClick={() => onQuickAction?.("account")}
              >
                {t.accounts.createAccount}
              </button>
              <button
                className="btn"
                type="button"
                disabled={quickActionsDisabled}
                onClick={() => onQuickAction?.("transfer")}
              >
                {t.dashboard.createTransfer}
              </button>
              <button
                className="btn"
                type="button"
                onClick={() => onQuickAction?.("import")}
              >
                {t.dashboard.importTransactions}
              </button>
            </div>
          </article>

          <article className="card rail-card">
            <div className="card-head">
              <div className="card-title">
                <h2>{t.dashboard.accountsSummary}</h2>
                <p>{t.dashboard.accountsSummaryDescription}</p>
              </div>
              <CreditCard aria-hidden="true" />
              <span className="pill">{allocation.length}</span>
            </div>
            <div className="list">
              {allocation.map((account) => (
                <button
                  className="account-summary-row"
                  type="button"
                  key={account.account_id}
                  onClick={() => onOpenAccount(account.account_id)}
                >
                  <div>
                    <strong>{account.name}</strong>
                    <span>
                      {formatMoney(account.balance, account.currency, locale)}
                    </span>
                  </div>
                  <span className="account-summary-side">
                    <strong>
                      {formatMoney(
                        account.converted_balance,
                        selectedCurrency,
                        locale,
                      )}
                    </strong>
                    <span>{account.share}%</span>
                  </span>
                </button>
              ))}
              {!allocation.length ? (
                <div className="empty-state">
                  <strong>{t.dashboard.noPositiveBalances}</strong>
                  <span>{t.dashboard.addAccountsToSeeAllocation}</span>
                </div>
              ) : null}
            </div>
          </article>

          <article className="card rail-card">
            <div className="card-head">
              <div className="card-title">
                <h2>{t.dashboard.goalsAndLimits}</h2>
                <p>{t.dashboard.goalsAndLimitsDescription}</p>
              </div>
              <Target aria-hidden="true" />
            </div>
            <div className="review-placeholder">
              <strong>{t.dashboard.goalsAndLimitsUnavailableTitle}</strong>
              <span>{t.dashboard.goalsAndLimitsUnavailableDescription}</span>
            </div>
          </article>

          <article className="card rail-card">
            <div className="card-head">
              <div className="card-title">
                <h2>{t.dashboard.subscriptions}</h2>
                <p>{t.dashboard.subscriptionsDescription}</p>
              </div>
              <Repeat aria-hidden="true" />
            </div>
            <div className="review-placeholder">
              <strong>{t.dashboard.emptySubscriptionsTitle}</strong>
              <span>{t.dashboard.emptySubscriptionsDescription}</span>
              <Button type="button" onClick={() => onNavigate?.("transactions")}>
                {t.nav.transactions}
              </Button>
            </div>
          </article>
          </aside>
        </div>
      </section>
      {selectedTransaction ? (
        <Dialog
          title={t.transactions.transactionDetails}
          onClose={() => setSelectedTransaction(null)}
        >
          <TransactionDetails
            transaction={selectedTransaction}
            accounts={recentAccounts}
          />
        </Dialog>
      ) : null}
    </div>
  );
}

function cashflowBucketsLabel(
  count: number,
  period: CashflowPeriod,
  t: ReturnType<typeof useI18n>["t"],
  locale: ReturnType<typeof useI18n>["locale"],
) {
  const periodLabel = t.dashboard.periods[period];

  if (locale === "ru") {
    return `${count} ${pluralRu(count, "период", "периода", "периодов")} · ${periodLabel}`;
  }

  return `${count} ${count === 1 ? "period" : "periods"} · ${periodLabel}`;
}

function pluralRu(count: number, one: string, few: string, many: string) {
  const mod10 = count % 10;
  const mod100 = count % 100;

  if (mod10 === 1 && mod100 !== 11) return one;
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return few;
  return many;
}
