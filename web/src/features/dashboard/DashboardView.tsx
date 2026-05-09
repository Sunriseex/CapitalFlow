import { useQuery } from "@tanstack/react-query";
import {
  Area,
  Bar,
  CartesianGrid,
  ComposedChart,
  Legend,
  Line,
  LineChart,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { api } from "../../api/client";
import { amountFor, formatMoney } from "../../api/money";
import { errorMessage } from "../../shared/api/query";
import { ChartShell, Empty, Panel } from "../../shared/ui";
import { TransactionsTable } from "../transactions/TransactionsTable";

export function DashboardView({ onOpenAccount }: { onOpenAccount: (id: string) => void }) {
  const summary = useQuery({ queryKey: ["dashboard", "summary"], queryFn: api.dashboardSummary });
  const cashflow = useQuery({ queryKey: ["dashboard", "cashflow"], queryFn: api.dashboardCashflow });
  const interest = useQuery({ queryKey: ["dashboard", "interest"], queryFn: api.dashboardInterestIncome });
  const data = summary.data;

  const chartData = (cashflow.data?.buckets ?? []).map((bucket) => ({
    period: bucket.period,
    income: amountFor(bucket.income),
    expense: amountFor(bucket.expense),
    net: amountFor(bucket.net_cashflow),
  }));
  const interestData = (interest.data?.buckets ?? []).map((bucket) => ({
    period: bucket.period,
    interest: amountFor(bucket.interest_income),
  }));
  const balances = data?.account_balances ?? [];
  const primaryCurrency = balances[0]?.currency ?? "RUB";
  const portfolioValue = balances.reduce((sum, account) => sum + Math.max(account.balance_minor, 0), 0);
  const allocation = balances
    .filter((account) => account.balance_minor > 0)
    .sort((a, b) => b.balance_minor - a.balance_minor)
    .slice(0, 6)
    .map((account) => ({
      ...account,
      share: portfolioValue > 0 ? Math.round((account.balance_minor / portfolioValue) * 100) : 0,
    }));
  const monthlyNet = amountFor(data?.monthly_income, primaryCurrency) - amountFor(data?.monthly_expense, primaryCurrency);

  if (summary.isLoading) {
    return <Empty>Loading dashboard</Empty>;
  }
  if (summary.error) {
    return <Empty>{errorMessage(summary.error)}</Empty>;
  }

  return (
    <div className="grid">
      <section className="portfolio-hero">
        <div>
          <p className="eyebrow">Portfolio value</p>
          <strong>{formatMoney(portfolioValue, primaryCurrency)}</strong>
          <span>{data?.active_accounts_count ?? 0} active accounts across {(data?.balances ?? []).length || 1} currency</span>
        </div>
        <div className={monthlyNet < 0 ? "hero-delta negative" : "hero-delta"}>
          <span>Net this month</span>
          <strong>{formatMoney(monthlyNet, primaryCurrency)}</strong>
        </div>
      </section>

      <div className="metric-strip">
        {(data?.balances ?? []).map((amount) => (
          <div className="metric" key={amount.currency}>
            <span>Total {amount.currency}</span>
            <strong>{formatMoney(amount.amount_minor, amount.currency)}</strong>
          </div>
        ))}
        <div className="metric">
          <span>Accounts</span>
          <strong>{data?.active_accounts_count ?? 0}/{data?.accounts_count ?? 0}</strong>
        </div>
        <div className="metric">
          <span>Income this month</span>
          <strong>{formatMoney(amountFor(data?.monthly_income))}</strong>
        </div>
        <div className="metric">
          <span>Expense this month</span>
          <strong>{formatMoney(amountFor(data?.monthly_expense))}</strong>
        </div>
        <div className="metric">
          <span>Interest this month</span>
          <strong>{formatMoney(amountFor(data?.monthly_interest_income))}</strong>
        </div>
      </div>

      <div className="dashboard-main">
        <Panel title="Cashflow trend">
          <ChartShell size="large">
            <ComposedChart data={chartData} margin={{ top: 8, right: 18, bottom: 0, left: 0 }}>
              <defs>
                <linearGradient id="netFlow" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#315f8d" stopOpacity={0.22} />
                  <stop offset="95%" stopColor="#315f8d" stopOpacity={0.02} />
                </linearGradient>
              </defs>
              <CartesianGrid stroke="var(--chart-grid)" vertical={false} />
              <XAxis dataKey="period" axisLine={false} tickLine={false} />
              <YAxis axisLine={false} tickLine={false} width={70} tickFormatter={(value) => formatCompactMoney(Number(value))} />
              <Tooltip formatter={(value) => formatMoney(Number(value), primaryCurrency)} />
              <Legend />
              <Area type="monotone" dataKey="net" name="Net" stroke="#315f8d" fill="url(#netFlow)" strokeWidth={2} />
              <Bar dataKey="income" name="Income" fill="#24735a" radius={[4, 4, 0, 0]} />
              <Bar dataKey="expense" name="Expense" fill="#a23b3b" radius={[4, 4, 0, 0]} />
              <Line type="monotone" dataKey="net" name="Net line" stroke="#1f2937" strokeWidth={2} dot={false} />
            </ComposedChart>
          </ChartShell>
        </Panel>

        <Panel title="Allocation">
          <div className="allocation-list">
            {allocation.map((account) => (
              <button className="allocation-row" key={account.account_id} onClick={() => onOpenAccount(account.account_id)}>
                <span>
                  <strong>{account.name}</strong>
                  <small>{account.bank || account.type}</small>
                </span>
                <span className="allocation-value">{formatMoney(account.balance_minor, account.currency)}</span>
                <span className="allocation-bar"><i style={{ width: `${account.share}%` }} /></span>
                <em>{account.share}%</em>
              </button>
            ))}
            {!allocation.length ? <Empty>No positive balances</Empty> : null}
          </div>
        </Panel>
      </div>

      <Panel title="Cashflow">
        <ChartShell>
          <ComposedChart data={chartData} margin={{ top: 8, right: 14, bottom: 0, left: 0 }}>
            <CartesianGrid stroke="var(--chart-grid)" vertical={false} />
            <XAxis dataKey="period" axisLine={false} tickLine={false} />
            <YAxis axisLine={false} tickLine={false} width={70} tickFormatter={(value) => formatCompactMoney(Number(value))} />
            <Tooltip formatter={(value) => formatMoney(Number(value), primaryCurrency)} />
            <Bar dataKey="income" fill="#24735a" radius={[4, 4, 0, 0]} />
            <Bar dataKey="expense" fill="#a23b3b" radius={[4, 4, 0, 0]} />
          </ComposedChart>
        </ChartShell>
      </Panel>

      <Panel title="Interest income">
        <ChartShell>
          <LineChart data={interestData} margin={{ top: 8, right: 14, bottom: 0, left: 0 }}>
            <CartesianGrid stroke="var(--chart-grid)" vertical={false} />
            <XAxis dataKey="period" axisLine={false} tickLine={false} />
            <YAxis axisLine={false} tickLine={false} width={70} tickFormatter={(value) => formatCompactMoney(Number(value))} />
            <Tooltip formatter={(value) => formatMoney(Number(value), primaryCurrency)} />
            <Line type="monotone" dataKey="interest" stroke="#8a6f2a" strokeWidth={3} dot={{ r: 3 }} activeDot={{ r: 5 }} />
          </LineChart>
        </ChartShell>
      </Panel>

      <Panel title="Account balances">
        <div className="table-wrap">
          <table>
            <tbody>
              {(data?.account_balances ?? []).map((account) => (
                <tr key={account.account_id} onClick={() => onOpenAccount(account.account_id)}>
                  <td>{account.name}</td>
                  <td>{account.bank || "-"}</td>
                  <td>{account.type}</td>
                  <td className="amount">{formatMoney(account.balance_minor, account.currency)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>

      <Panel title="Recent transactions">
        <TransactionsTable transactions={data?.recent_transactions ?? []} accounts={[]} categories={[]} compact />
      </Panel>
    </div>
  );
}

function formatCompactMoney(value: number) {
  const abs = Math.abs(value);
  if (abs >= 1_000_000) {
    return `${Math.round(value / 1_000_000)}M`;
  }
  if (abs >= 1_000) {
    return `${Math.round(value / 1_000)}K`;
  }
  return `${value}`;
}

