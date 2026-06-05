import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { PanelRightClose, PanelRightOpen } from "lucide-react";
import { CartesianGrid, Line, LineChart, Tooltip, XAxis, YAxis } from "recharts";
import { api } from "../../api/client";
import { addMoney, compareMoney, convertMinor, formatMoney, moneyToNumber, sumConverted, transactionTypeLabel } from "../../api/money";
import type { Account, Transaction } from "../../api/types";
import { errorMessage } from "../../shared/api/query";
import type { QuickAction, View } from "../../shared/constants";
import { Empty } from "../../shared/ui";
import { ChartShell } from "../../shared/ui/ChartShell";

type CashflowPeriod = "week" | "month" | "quarter" | "year";

type CashflowChartBucket = {
  period: string;
  sourcePeriod: string;
  income: number;
  expense: number;
  net: number;
  transactions: number;
};

const cashflowPeriods: Array<{ value: CashflowPeriod; label: string }> = [
  { value: "week", label: "Week" },
  { value: "month", label: "Month" },
  { value: "quarter", label: "Quarter" },
  { value: "year", label: "Year" },
];

const rateTargets = ["USD", "EUR", "BTC"];

export function DashboardView({
  primaryCurrency,
  onOpenAccount,
  onQuickAction,
  onNavigate,
  quickActionsDisabled = false,
}: {
  primaryCurrency: string;
  onOpenAccount: (id: string) => void;
  onQuickAction?: (action: NonNullable<QuickAction>) => void;
  onNavigate?: (view: View) => void;
  quickActionsDisabled?: boolean;
}) {
  const summary = useQuery({ queryKey: ["dashboard", "summary"], queryFn: api.dashboardSummary });
  const cashflow = useQuery({ queryKey: ["dashboard", "cashflow"], queryFn: api.dashboardCashflow });
  const interestIncome = useQuery({ queryKey: ["dashboard", "interest-income"], queryFn: api.dashboardInterestIncome });
  const [selectedCurrency, setSelectedCurrency] = useState(primaryCurrency);
  const [cashflowPeriod, setCashflowPeriod] = useState<CashflowPeriod>("month");
  const [rightRailHidden, setRightRailHidden] = useState(false);
  const data = summary.data;

  const balances = useMemo(() => data?.account_balances ?? [], [data?.account_balances]);
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
  const rateTable = rates.data?.base === selectedCurrency ? rates.data : undefined;
  const rateEntries = useMemo(() => {
    return rateTargets.map((currency) => [currency, rateTable?.rates[currency]] as const);
  }, [rateTable]);
  const portfolioValue = sumConverted(currencyTotals, selectedCurrency, rateTable);
  const portfolioValueNumber = useMemo(() => moneyToNumber(portfolioValue), [portfolioValue]);
  const ratesSyncLabel = rateTable ? formatRateSync(rateTable.fetched_at || rateTable.date) : "Rates unavailable";

  const allocation = useMemo(() => (
    balances
      .filter((account) => compareMoney(account.balance, "0") > 0)
      .map((account) => ({
        ...account,
        converted_balance: convertMinor(account.balance, account.currency, selectedCurrency, rateTable),
      }))
      .sort((a, b) => compareMoney(b.converted_balance, a.converted_balance))
      .slice(0, 6)
      .map((account) => ({
        ...account,
        share: portfolioValueNumber > 0 ? Math.round((moneyToNumber(account.converted_balance) / portfolioValueNumber) * 100) : 0,
      }))
  ), [balances, portfolioValueNumber, rateTable, selectedCurrency]);

  const monthlyNet = addMoney(
    sumConverted(data?.monthly_income, selectedCurrency, rateTable),
    `-${sumConverted(data?.monthly_expense, selectedCurrency, rateTable)}`,
  );
  const recentAccounts = useMemo(() => balances.map((account): Account => ({
    id: account.account_id,
    name: account.name,
    bank: account.bank,
    type: account.type,
    currency: account.currency,
    is_active: account.is_active,
    opened_at: "",
    created_at: "",
    updated_at: "",
  })), [balances]);
  const cashflowBuckets = useMemo(() => (cashflow.data?.buckets ?? []).map((bucket) => ({
    period: shortPeriod(bucket.period),
    sourcePeriod: bucket.period,
    income: moneyToNumber(sumConverted(bucket.income, selectedCurrency, rateTable)),
    expense: moneyToNumber(sumConverted(bucket.expense, selectedCurrency, rateTable)),
    net: moneyToNumber(sumConverted(bucket.net_cashflow, selectedCurrency, rateTable)),
    transactions: bucket.transaction_count,
  })), [cashflow.data?.buckets, rateTable, selectedCurrency]);
  const cashflowChart = useMemo(() => groupCashflow(cashflowBuckets, cashflowPeriod), [cashflowBuckets, cashflowPeriod]);
  const cashflowEmpty = cashflowEmptyState(cashflowPeriod, cashflowBuckets.length > 0);
  const totalInterest = sumConverted(interestIncome.data?.total, selectedCurrency, rateTable);
  const chartSummary = describeCashflow(cashflowChart, selectedCurrency);

  if (summary.isLoading) {
    return <Empty>Loading dashboard</Empty>;
  }

  if (summary.error) {
    return <Empty>{errorMessage(summary.error)}</Empty>;
  }

  return (
    <div className="ref-dashboard">
      <div className="dashboard-toolbar">
        <button
          className="rail-toggle"
          type="button"
          aria-label={rightRailHidden ? "Show insights" : "Hide insights"}
          title={rightRailHidden ? "Show insights" : "Hide insights"}
          aria-controls="dashboard-right-rail"
          aria-expanded={!rightRailHidden}
          onClick={() => setRightRailHidden((hidden) => !hidden)}
        >
          {rightRailHidden ? <PanelRightOpen size={17} aria-hidden="true" /> : <PanelRightClose size={17} aria-hidden="true" />}
        </button>
      </div>
      <div className={rightRailHidden ? "layout is-rail-collapsed" : "layout"}>
        <div className="content">
          <section className="tab-panel" id="overview" aria-labelledby="pageTitle">
            <article className="card balance-card">
              <div className="balance-top">
                <div className="balance-title">
                  <span>Total capital</span>
                  <div className="balance-value">{formatMoney(portfolioValue, selectedCurrency)}</div>
                </div>
                <span className="pill">Live ledger</span>
              </div>

              <div className="balance-meta">
                <span className={compareMoney(monthlyNet, "0") < 0 ? "delta-down" : "delta-up"}>
                  {formatMoney(monthlyNet, selectedCurrency)} this month
                </span>
              </div>

              <div className="currency-switcher" aria-label="Portfolio currency">
                {currencies.map((currency) => (
                  <button
                    key={currency}
                    className={currency === selectedCurrency ? "period-btn is-active" : "period-btn"}
                    type="button"
                    aria-pressed={currency === selectedCurrency}
                    onClick={() => setSelectedCurrency(currency)}
                  >
                    {currency}
                  </button>
                ))}
              </div>

              <div className="balance-actions" aria-label="Quick actions">
                <button className="btn primary" type="button" disabled={quickActionsDisabled} onClick={() => onQuickAction?.("transaction")}>
                  + Transaction
                </button>
                <button className="btn" type="button" disabled={quickActionsDisabled} onClick={() => onQuickAction?.("transfer")}>
                  + Transfer
                </button>
                <button className="btn" type="button" onClick={() => onQuickAction?.("import")}>Import</button>
              </div>

              <div className="stat-grid">
                <div className="stat"><span>Income</span><strong>{formatMoney(sumConverted(data?.monthly_income, selectedCurrency, rateTable), selectedCurrency)}</strong></div>
                <div className="stat"><span>Expenses</span><strong>{formatMoney(sumConverted(data?.monthly_expense, selectedCurrency, rateTable), selectedCurrency)}</strong></div>
              </div>
            </article>

            <article className="card chart-card">
              <div className="card-head">
                <div className="card-title">
                  <h2>Cashflow ({selectedCurrency})</h2>
                  <p>{cashflow.isLoading ? "Loading ledger buckets" : `${cashflowChart.length} ${cashflowPeriod} buckets`}</p>
                </div>
                <div>
                  <div className="period-switcher" aria-label="Cashflow period">
                    {cashflowPeriods.map((period) => (
                      <button
                        key={period.value}
                        className={period.value === cashflowPeriod ? "period-btn is-active" : "period-btn"}
                        type="button"
                        aria-pressed={period.value === cashflowPeriod}
                        onClick={() => setCashflowPeriod(period.value)}
                      >
                        {period.label}
                      </button>
                    ))}
                  </div>
                </div>
              </div>

              {cashflow.error ? <div className="empty-state"><strong>{errorMessage(cashflow.error)}</strong><span>Cashflow chart could not be loaded.</span></div> : null}
              {!cashflow.error && !cashflow.isLoading && !cashflowChart.length ? (
                <div className="empty-state"><strong>{cashflowEmpty.title}</strong><span>{cashflowEmpty.description}</span></div>
              ) : null}
              {!cashflow.error && cashflowChart.length ? (
                <div className="chart-wrap" aria-label="Income and expense chart">
                  <p className="sr-only">{chartSummary}</p>
                  <ChartShell>
                    <LineChart data={cashflowChart} margin={{ top: 14, right: 18, bottom: 6, left: 0 }}>
                      <CartesianGrid stroke="rgba(255,255,255,.08)" vertical={false} />
                      <XAxis dataKey="period" tickLine={false} axisLine={false} />
                      <YAxis tickLine={false} axisLine={false} tickFormatter={(value) => compactMoney(Number(value), selectedCurrency)} width={72} />
                      <Tooltip formatter={(value, name) => [formatChartMoney(Number(value), selectedCurrency), labelForSeries(String(name))]} labelFormatter={(label) => `Period ${label}`} />
                      <Line type="monotone" dataKey="income" stroke="var(--green)" strokeWidth={3} dot={false} activeDot={{ r: 4 }} />
                      <Line type="monotone" dataKey="expense" stroke="var(--red)" strokeWidth={3} dot={false} activeDot={{ r: 4 }} />
                      <Line type="monotone" dataKey="net" stroke="var(--blue)" strokeWidth={2} dot={false} strokeDasharray="5 5" />
                    </LineChart>
                  </ChartShell>
                  <table className="sr-only-table">
                    <caption>Cashflow data</caption>
                    <thead>
                      <tr>
                        <th scope="col">Period</th>
                        <th scope="col">Income</th>
                        <th scope="col">Expense</th>
                        <th scope="col">Net</th>
                      </tr>
                    </thead>
                    <tbody>
                      {cashflowChart.map((bucket) => (
                        <tr key={bucket.period}>
                          <td>{bucket.period}</td>
                          <td>{formatChartMoney(bucket.income, selectedCurrency)}</td>
                          <td>{formatChartMoney(bucket.expense, selectedCurrency)}</td>
                          <td>{formatChartMoney(bucket.net, selectedCurrency)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : null}

              <div className="legend">
                <span className="legend-item"><span className="legend-mark income"></span>Income · {formatMoney(sumConverted(data?.monthly_income, selectedCurrency, rateTable), selectedCurrency)}</span>
                <span className="legend-item"><span className="legend-mark expense"></span>Expenses · {formatMoney(sumConverted(data?.monthly_expense, selectedCurrency, rateTable), selectedCurrency)}</span>
                <span className="legend-item"><span className="legend-mark net"></span>Net cashflow</span>
                <span className="legend-item">Interest · {formatMoney(totalInterest, selectedCurrency)}</span>
              </div>
            </article>

            <article className="card">
              <div className="card-head">
                <div className="card-title">
                  <h2>Recent transactions</h2>
                  <p>Last 5 transactions without dashboard overload</p>
                </div>
                <button className="btn" type="button" onClick={() => onNavigate?.("transactions")}>All transactions</button>
              </div>
              <RecentTransactionsTable
                accounts={recentAccounts}
                transactions={data?.recent_transactions ?? []}
                selectedCurrency={selectedCurrency}
                onNavigate={onNavigate}
              />
            </article>

          </section>
        </div>

        <aside id="dashboard-right-rail" className="right-rail" aria-label="Right rail summary" aria-hidden={rightRailHidden}>
          <article className="card rail-card">
            <div className="card-head">
              <div className="card-title">
                <h2>Upcoming</h2>
                <p>Current month status</p>
              </div>
              <button className="btn" type="button" onClick={() => onNavigate?.("transactions")}>Open ledger</button>
            </div>
            <div className="list">
              {(data?.recent_transactions_returned ?? 0) > 0 || (data?.active_accounts_count ?? 0) > 0 ? (
                <>
                  <div className="row"><div className="row-main"><strong>Monthly net</strong><span>{formatMoney(monthlyNet, selectedCurrency)}</span></div><span className="tag info">Real</span></div>
                  <div className="row"><div className="row-main"><strong>Ledger events</strong><span>{data?.recent_transactions_returned ?? 0} recent</span></div><span className="tag good">Loaded</span></div>
                </>
              ) : (
                <div className="empty-state"><strong>No upcoming data</strong><span>Recurring schedules are not available from the backend yet.</span></div>
              )}
            </div>
          </article>

          <article className="card rail-card">
            <div className="card-head">
              <div className="card-title">
                <h2>Rates</h2>
                <p>{ratesSyncLabel}</p>
              </div>
              <button className="btn" type="button" onClick={() => onNavigate?.("settings")}>Settings</button>
            </div>
            <div className="list">
              {rateTable ? rateEntries.map(([currency, rate]) => (
                <div className="row" key={currency}>
                  <div className="row-main"><strong>{selectedCurrency}/{currency}</strong><span>Latest synced rate</span></div>
                  <span className="row-side">{typeof rate === "number" ? rate.toLocaleString(undefined, { maximumFractionDigits: 8 }) : "—"}</span>
                </div>
              )) : <div className="empty-state"><strong>Rates unavailable</strong><span>Open settings to check currency configuration.</span></div>}
            </div>
          </article>

          <article className="card rail-card">
            <div className="card-head">
              <div className="card-title">
                <h2>Allocation</h2>
                <p>Top positive balances</p>
              </div>
              <span className="pill">{allocation.length}</span>
            </div>
            <div className="list">
              {allocation.map((account) => (
                <button className="review-action-row" type="button" key={account.account_id} onClick={() => onOpenAccount(account.account_id)}>
                  <div><strong>{account.name}</strong><span>{formatMoney(account.balance, account.currency)}</span></div>
                  <span className="tag info">{account.share}%</span>
                </button>
              ))}
              {!allocation.length ? <div className="empty-state"><strong>No positive balances</strong><span>Add accounts with positive balances to see allocation.</span></div> : null}
            </div>
          </article>
        </aside>
      </div>
    </div>
  );
}

function RecentTransactionsTable({
  accounts,
  transactions,
  selectedCurrency,
  onNavigate,
}: {
  accounts: Account[];
  transactions: Transaction[];
  selectedCurrency: string;
  onNavigate?: (view: View) => void;
}) {
  const visibleTransactions = useMemo(() => transactions.slice(0, 5), [transactions]);
  const accountNames = useMemo(() => new Map(accounts.map((account) => [account.id, account.name])), [accounts]);
  const accountCurrencies = useMemo(() => new Map(accounts.map((account) => [account.id, account.currency])), [accounts]);

  if (!transactions.length) {
    return <div className="empty-state"><strong>No transactions</strong><span>Add the first transaction or import a bank statement.</span></div>;
  }

  return (
    <div className="table-scroll">
      <table className="tx-table" aria-label="Recent transactions">
        <colgroup>
          <col className="col-operation" />
          <col className="col-account" />
          <col className="col-category" />
          <col className="col-amount" />
          <col className="col-view" />
        </colgroup>
        <thead>
          <tr>
            <th scope="col">Operation</th>
            <th scope="col">Account</th>
            <th scope="col">Category</th>
            <th scope="col">Amount</th>
            <th scope="col">View</th>
          </tr>
        </thead>
        <tbody>
          {visibleTransactions.map((transaction) => {
            const negative = transaction.type === "expense" || transaction.type === "transfer_out";
            const sign = negative ? "-" : "+";
            return (
              <tr className="tx" key={transaction.id}>
                <td data-label="Operation"><strong>{transaction.description || transaction.type}</strong><small>{transactionTypeLabel(transaction.type)} · ledger event</small></td>
                <td data-label="Account">{accountNames.get(transaction.account_id) ?? transaction.account_id}</td>
                <td data-label="Category">{transaction.category_id ?? "—"}</td>
                <td data-label="Amount" className={negative ? "delta-down" : "delta-up"}>
                  {sign}{formatMoney(transaction.amount, accountCurrencies.get(transaction.account_id) ?? selectedCurrency ?? "RUB")}
                </td>
                <td data-label="View">
                  <button className="view-cell" type="button" aria-label="Open transaction details" onClick={() => onNavigate?.("transactions")}>
                    View
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function shortPeriod(period: string) {
  const [, month] = period.split("-");
  return month ? `${month}/${period.slice(2, 4)}` : period;
}

function groupCashflow(buckets: CashflowChartBucket[], period: CashflowPeriod) {
  if (period === "month") {
    return buckets;
  }

  if (period === "week") {
    return [];
  }

  const grouped = new Map<string, CashflowChartBucket>();
  for (const bucket of buckets) {
    const key = period === "quarter" ? quarterLabel(bucket.sourcePeriod) : bucket.sourcePeriod.slice(0, 4);
    const existing = grouped.get(key);
    if (existing) {
      existing.income += bucket.income;
      existing.expense += bucket.expense;
      existing.net += bucket.net;
      existing.transactions += bucket.transactions;
    } else {
      grouped.set(key, { ...bucket, period: key, sourcePeriod: key });
    }
  }

  return [...grouped.values()];
}

function cashflowEmptyState(period: CashflowPeriod, hasMonthlyData: boolean) {
  if (period === "week" && hasMonthlyData) {
    return {
      title: "Weekly cashflow unavailable",
      description: "The backend currently returns monthly cashflow buckets.",
    };
  }

  return {
    title: "No cashflow yet",
    description: "Add income or expenses to build this chart.",
  };
}

function quarterLabel(period: string) {
  const [year, month] = period.split("-");
  const monthNumber = Number(month);
  if (!year || !monthNumber) {
    return period;
  }
  return `${year} Q${Math.ceil(monthNumber / 3)}`;
}

function formatRateSync(value: string) {
  if (!value) {
    return "Rates unavailable";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toUTCString().replace(/ GMT$/, "");
}

function compactMoney(value: number, currency: string) {
  if (Math.abs(value) >= 1000000) return `${Math.round(value / 1000000)}M ${currency}`;
  if (Math.abs(value) >= 1000) return `${Math.round(value / 1000)}K ${currency}`;
  return `${value} ${currency}`;
}

function formatChartMoney(value: number, currency: string) {
  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency,
      currencyDisplay: "code",
      maximumFractionDigits: 2,
    }).format(value);
  } catch {
    return `${value.toLocaleString(undefined, { maximumFractionDigits: 2 })} ${currency}`;
  }
}

function labelForSeries(name: string) {
  return {
    income: "Income",
    expense: "Expenses",
    net: "Net cashflow",
  }[name] ?? name;
}

function describeCashflow(data: Array<{ period: string; income: number; expense: number; net: number }>, currency: string) {
  if (!data.length) {
    return "Cashflow chart has no periods.";
  }

  const totalIncome = data.reduce((sum, bucket) => sum + bucket.income, 0);
  const totalExpense = data.reduce((sum, bucket) => sum + bucket.expense, 0);
  const totalNet = data.reduce((sum, bucket) => sum + bucket.net, 0);
  return `Cashflow chart covers ${data.length} periods. Income ${formatChartMoney(totalIncome, currency)}, expenses ${formatChartMoney(totalExpense, currency)}, net ${formatChartMoney(totalNet, currency)}.`;
}
