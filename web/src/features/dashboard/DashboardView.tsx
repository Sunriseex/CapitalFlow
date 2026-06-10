import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
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
import { errorMessage } from "../../shared/api/query";
import type { QuickAction, View } from "../../shared/constants";
import { Dialog, Empty } from "../../shared/ui";
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

const fallbackRateTargets = ["USD", "EUR", "BTC"];

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
  const { t } = useI18n();
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
  const rateTargets = useMemo(
    () => selectRateTargets(currencies, selectedCurrency),
    [currencies, selectedCurrency],
  );
  const rateEntries = useMemo(() => {
    return rateTargets.map(
      (currency) => [currency, rateTable?.rates[currency]] as const,
    );
  }, [rateTable, rateTargets]);
  const portfolioValue = sumConverted(
    currencyTotals,
    selectedCurrency,
    rateTable,
  );
  const portfolioValueNumber = useMemo(
    () => moneyToNumber(portfolioValue),
    [portfolioValue],
  );
  const ratesSyncLabel = rateTable
    ? formatRateSync(
        rateTable.fetched_at || rateTable.date,
        t.dashboard.ratesUnavailable,
      )
    : t.dashboard.ratesUnavailable;

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
      .replace("{income}", formatChartMoney(totalIncome, selectedCurrency))
      .replace("{expenses}", formatChartMoney(totalExpense, selectedCurrency))
      .replace("{net}", formatChartMoney(totalNet, selectedCurrency));
  }, [cashflowChart, selectedCurrency, t]);

  if (summary.isLoading) {
    return <Empty>{t.dashboard.loadingDashboard}</Empty>;
  }

  if (summary.error) {
    return <Empty>{errorMessage(summary.error)}</Empty>;
  }

  return (
    <div className="ref-dashboard">
      <div className={rightRailHidden ? "layout is-rail-collapsed" : "layout"}>
        <div className="content">
          <section
            className="tab-panel"
            id="overview"
            aria-labelledby="pageTitle"
          >
            <article className="card balance-card">
              <div className="balance-top">
                <div className="balance-title">
                  <span>{t.dashboard.totalCapital}</span>{" "}
                  <div className="balance-value">
                    {formatMoney(portfolioValue, selectedCurrency)}
                  </div>
                </div>
                <span className="pill">{t.dashboard.liveLedger}</span>{" "}
              </div>

              <div className="balance-meta">
                <span
                  className={
                    compareMoney(monthlyNet, "0") < 0
                      ? "delta-down"
                      : "delta-up"
                  }
                >
                  {formatMoney(monthlyNet, selectedCurrency)}{" "}
                  {t.dashboard.thisMonth}{" "}
                </span>
              </div>

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

              <div
                className="balance-actions"
                role="group"
                aria-label={t.dashboard.quickActions}
              >
                <button
                  className="btn primary"
                  type="button"
                  disabled={quickActionsDisabled}
                  onClick={() => onQuickAction?.("transaction")}
                >
                  {t.dashboard.addTransaction}{" "}
                </button>
                <button
                  className="btn"
                  type="button"
                  disabled={quickActionsDisabled}
                  onClick={() => onQuickAction?.("transfer")}
                >
                  {t.dashboard.createTransfer}{" "}
                </button>
                <button
                  className="btn"
                  type="button"
                  onClick={() => onQuickAction?.("import")}
                >
                  {t.dashboard.importTransactions}{" "}
                </button>
              </div>

              <div className="stat-grid">
                <div className="stat">
                  <span>{t.dashboard.income}</span>{" "}
                  <strong>
                    {formatMoney(
                      sumConverted(
                        data?.monthly_income,
                        selectedCurrency,
                        rateTable,
                      ),
                      selectedCurrency,
                    )}
                  </strong>
                </div>
                <div className="stat">
                  <span>{t.dashboard.expenses}</span>{" "}
                  <strong>
                    {formatMoney(
                      sumConverted(
                        data?.monthly_expense,
                        selectedCurrency,
                        rateTable,
                      ),
                      selectedCurrency,
                    )}
                  </strong>
                </div>
              </div>
            </article>

            <article className="card chart-card">
              <div className="card-head">
                <div className="card-title">
                  <h2>
                    {t.dashboard.cashflow} ({selectedCurrency})
                  </h2>{" "}
                  <p>
                    {cashflow.isLoading
                      ? t.dashboard.loadingLedgerBuckets
                      : `${cashflowChart.length} ${cashflowPeriod} ${t.dashboard.buckets}`}
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
                  <strong>{errorMessage(cashflow.error)}</strong>
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
                            {formatChartMoney(bucket.income, selectedCurrency)}
                          </td>
                          <td>
                            {formatChartMoney(bucket.expense, selectedCurrency)}
                          </td>
                          <td>
                            {formatChartMoney(bucket.net, selectedCurrency)}
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
                  )}
                </span>
                <span className="legend-item">
                  <span className="legend-mark net"></span>
                  {t.dashboard.net}{" "}
                </span>
                <span className="legend-item">
                  {t.dashboard.interest} ·{" "}
                  {formatMoney(totalInterest, selectedCurrency)}
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
          </section>
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
                <h2>{t.dashboard.upcoming}</h2>
                <p>{t.dashboard.currentMonthStatus}</p>{" "}
              </div>
              <button
                className="btn"
                type="button"
                onClick={() => onNavigate?.("transactions")}
              >
                {t.dashboard.openLedger}{" "}
              </button>
            </div>
            <div className="list">
              {(data?.recent_transactions_returned ?? 0) > 0 ||
              (data?.active_accounts_count ?? 0) > 0 ? (
                <>
                  <div className="row">
                    <div className="row-main">
                      <strong>{t.dashboard.monthlyNet}</strong>{" "}
                      <span>{formatMoney(monthlyNet, selectedCurrency)}</span>
                    </div>
                    <span className="tag info">{t.dashboard.real}</span>{" "}
                  </div>
                  <div className="row">
                    <div className="row-main">
                      <strong>{t.dashboard.ledgerEvents}</strong>
                      <span>
                        {data?.recent_transactions_returned ?? 0}{" "}
                        {t.dashboard.recent}
                      </span>
                    </div>
                    <span className="tag good">{t.dashboard.loaded}</span>{" "}
                  </div>
                </>
              ) : (
                <div className="empty-state">
                  <strong>{t.dashboard.noUpcomingData}</strong>
                  <span>{t.dashboard.recurringSchedulesUnavailable}</span>
                </div>
              )}
            </div>
          </article>

          <article className="card rail-card">
            <div className="card-head">
              <div className="card-title">
                <h2>{t.dashboard.rates}</h2> <p>{ratesSyncLabel}</p>
              </div>
              <button
                className="btn"
                type="button"
                onClick={() => onNavigate?.("settings")}
              >
                {t.nav.settings}{" "}
              </button>
            </div>
            <div className="list">
              {rateTable ? (
                rateEntries.map(([currency, rate]) => (
                  <div className="row" key={currency}>
                    <div className="row-main">
                      <strong>
                        {selectedCurrency}/{currency}
                      </strong>
                      <span>{t.dashboard.latestSyncedRate}</span>{" "}
                    </div>
                    <span className="row-side">
                      {typeof rate === "number"
                        ? rate.toLocaleString(undefined, {
                            maximumFractionDigits: 8,
                          })
                        : "—"}
                    </span>
                  </div>
                ))
              ) : (
                <div className="empty-state">
                  <strong>{t.dashboard.ratesUnavailable}</strong>
                  <span>{t.dashboard.openSettingsToCheckCurrency}</span>
                </div>
              )}
            </div>
          </article>

          <article className="card rail-card">
            <div className="card-head">
              <div className="card-title">
                <h2>{t.dashboard.allocation}</h2>
                <p>{t.dashboard.topPositiveBalances}</p>{" "}
              </div>
              <span className="pill">{allocation.length}</span>
            </div>
            <div className="list">
              {allocation.map((account) => (
                <button
                  className="review-action-row"
                  type="button"
                  key={account.account_id}
                  onClick={() => onOpenAccount(account.account_id)}
                >
                  <div>
                    <strong>{account.name}</strong>
                    <span>
                      {formatMoney(account.balance, account.currency)}
                    </span>
                  </div>
                  <span className="tag info">{account.share}%</span>
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
        </aside>
      </div>
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

function selectRateTargets(currencies: string[], selectedCurrency: string) {
  const targets = new Set<string>();
  for (const currency of [...currencies, ...fallbackRateTargets]) {
    if (currency && currency !== selectedCurrency) {
      targets.add(currency);
    }
    if (targets.size >= 5) {
      break;
    }
  }
  return [...targets];
}

function formatRateSync(value: string, fallback: string) {
  if (!value) {
    return fallback;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toUTCString().replace(/ GMT$/, "");
}
