import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { CartesianGrid, Line, LineChart, Tooltip, XAxis, YAxis } from "recharts";
import { api } from "../../api/client";
import { addMoney, compareMoney, convertMinor, formatMoney, moneyToNumber, sumConverted, transactionTypeLabel } from "../../api/money";
import type { Account, Transaction } from "../../api/types";
import { errorMessage } from "../../shared/api/query";
import type { QuickAction, View } from "../../shared/constants";
import { Empty } from "../../shared/ui";
import { ChartShell } from "../../shared/ui/ChartShell";

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
  const balanceCurrencies = useMemo(() => [...new Set(balances.map((account) => account.currency))], [balances]);
  const rateEntries = useMemo(() => {
    if (!rateTable) {
      return [];
    }

    const priority = new Map(balanceCurrencies.map((currency, index) => [currency, index]));
    return Object.entries(rateTable.rates)
      .sort(([left], [right]) => {
        const leftPriority = priority.get(left);
        const rightPriority = priority.get(right);
        if (leftPriority != null || rightPriority != null) {
          return (leftPriority ?? Number.MAX_SAFE_INTEGER) - (rightPriority ?? Number.MAX_SAFE_INTEGER);
        }
        return left.localeCompare(right);
      })
      .slice(0, 5);
  }, [balanceCurrencies, rateTable]);
  const portfolioValue = sumConverted(currencyTotals, selectedCurrency, rateTable);
  const portfolioValueNumber = useMemo(() => moneyToNumber(portfolioValue), [portfolioValue]);
  const conversionStatus = rates.error
    ? errorMessage(rates.error)
    : rateTable
      ? `${rateTable.provider}, ${rateTable.date}`
      : "Loading rates";

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
  const cashflowChart = useMemo(() => (cashflow.data?.buckets ?? []).map((bucket) => ({
    period: shortPeriod(bucket.period),
    income: moneyToNumber(sumConverted(bucket.income, selectedCurrency, rateTable)),
    expense: moneyToNumber(sumConverted(bucket.expense, selectedCurrency, rateTable)),
    net: moneyToNumber(sumConverted(bucket.net_cashflow, selectedCurrency, rateTable)),
    transactions: bucket.transaction_count,
  })), [cashflow.data?.buckets, rateTable, selectedCurrency]);
  const totalInterest = sumConverted(interestIncome.data?.total, selectedCurrency, rateTable);
  const chartSummary = describeCashflow(cashflowChart, selectedCurrency);
  const currencyCountLabel = `${currencies.length || 1} ${(currencies.length || 1) === 1 ? "currency" : "currencies"}`;

  if (summary.isLoading) {
    return <Empty>Loading dashboard</Empty>;
  }

  if (summary.error) {
    return <Empty>{errorMessage(summary.error)}</Empty>;
  }

  return (
    <div className="ref-dashboard">
      <div className="layout">
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
                  <span>{data?.active_accounts_count ?? "0"} active accounts across {currencyCountLabel}</span>
                  <span>{conversionStatus}</span>
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
                <div className="stat"><span>Main currency</span><strong>{selectedCurrency}</strong></div>
                <div className="stat"><span>Income</span><strong>{formatMoney(sumConverted(data?.monthly_income, selectedCurrency, rateTable), selectedCurrency)}</strong></div>
                <div className="stat"><span>Expenses</span><strong>{formatMoney(sumConverted(data?.monthly_expense, selectedCurrency, rateTable), selectedCurrency)}</strong></div>
              </div>
            </article>

            <article className="card chart-card">
              <div className="card-head">
                <div className="card-title">
                  <h2>Cashflow ({selectedCurrency})</h2>
                  <p>{cashflow.isLoading ? "Loading ledger buckets" : `${cashflowChart.length} monthly buckets from real transactions`}</p>
                </div>
                <div>
                  <div className="period-switcher" aria-label="Dashboard currency">
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
                  <div className="period-label">Month · current ledger</div>
                </div>
              </div>

              {cashflow.error ? <div className="empty-state"><strong>{errorMessage(cashflow.error)}</strong><span>Cashflow chart could not be loaded.</span></div> : null}
              {!cashflow.error && !cashflow.isLoading && !cashflowChart.length ? (
                <div className="empty-state"><strong>No cashflow yet</strong><span>Add income or expenses to build this chart.</span></div>
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

        <aside className="right-rail" aria-label="Right rail summary">
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
                <p>{rateTable ? `${rateTable.provider} · ${rateTable.date}` : "Rates unavailable"}</p>
              </div>
              <button className="btn" type="button" onClick={() => onNavigate?.("settings")}>Settings</button>
            </div>
            <div className="list">
              {rateEntries.length ? rateEntries.map(([currency, rate]) => (
                <div className="row" key={currency}>
                  <div className="row-main"><strong>{selectedCurrency}/{currency}</strong><span>Provider rate</span></div>
                  <span className="row-side">{rate.toLocaleString(undefined, { maximumFractionDigits: 4 })}</span>
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
  if (!transactions.length) {
    return <div className="empty-state"><strong>No transactions</strong><span>Add the first transaction or import a bank statement.</span></div>;
  }

  const accountNames = new Map(accounts.map((account) => [account.id, account.name]));
  const accountCurrencies = new Map(accounts.map((account) => [account.id, account.currency]));

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
          {transactions.slice(0, 5).map((transaction) => {
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
