import { memo, useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Archive, BadgePercent, Pencil } from "lucide-react";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";
import { api } from "../../api/client";
import { addMoney, formatMoney, moneyToNumber, signedAmount } from "../../api/money";
import type { Account, InterestRule, Transaction } from "../../api/types";
import { errorMessage, invalidateMoney } from "../../shared/api/query";
import { today } from "../../shared/constants";
import { dateLabel } from "../../shared/date";
import { Button, Dialog, Empty, Panel } from "../../shared/ui";
import { ChartShell } from "../../shared/ui/ChartShell";
import { chartAxisProps, chartGridProps } from "../../shared/ui/chartTokens";
import { markPerformance } from "../../shared/performance";
import { useAfterPaint } from "../../shared/ui/useAfterPaint";
import { TransactionsTable } from "../transactions/TransactionsTable";
import { EditAccountForm } from "./EditAccountForm";

const emptyTransactions: Transaction[] = [];

export function AccountDetails({ account, onBack }: { account: Account; onBack: () => void }) {
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);
  const [actionError, setActionError] = useState("");
  const [chartState, setChartState] = useState<RunningBalanceState>(() => emptyRunningBalanceState(account.id, account.currency));
  const afterPaint = useAfterPaint();
  const transactions = useQuery({ queryKey: ["transactions", account.id], queryFn: () => api.transactions(account.id) });
  const balance = useQuery({ queryKey: ["balance", account.id], queryFn: () => api.accountBalance(account.id) });
  const rules = useQuery({ queryKey: ["interest-rules", account.id], queryFn: () => api.interestRules(account.id) });
  const accrue = useMutation({
    mutationFn: () => api.accrueInterest(account.id, today),
    onSuccess: () => invalidateMoney(queryClient),
  });
  const archive = useMutation({
    mutationFn: () => api.archiveAccount(account.id),
    onSuccess: () => {
      setActionError("");
      invalidateMoney(queryClient);
    },
    onError: (err) => setActionError(errorMessage(err)),
  });
  useEffect(() => {
    const endMeasure = markPerformance("account-details-open");
    if (typeof window.requestAnimationFrame !== "function") {
      const timeout = window.setTimeout(endMeasure, 0);
      return () => window.clearTimeout(timeout);
    }

    const frame = window.requestAnimationFrame(() => {
      window.requestAnimationFrame(endMeasure);
    });
    return () => window.cancelAnimationFrame(frame);
  }, [account.id]);

  const accountList = useMemo(() => [account], [account]);
  const transactionsData = transactions.data ?? emptyTransactions;

  useEffect(() => {
    if (!afterPaint) {
      return;
    }

    const run = () => setChartState(buildRunningBalanceState(account.id, transactionsData, account.currency, 240));
    const idleWindow = window as Window & {
      requestIdleCallback?: (callback: () => void, options?: { timeout: number }) => number;
      cancelIdleCallback?: (handle: number) => void;
    };

    if (typeof idleWindow.requestIdleCallback === "function") {
      const idle = idleWindow.requestIdleCallback(run, { timeout: 240 });
      return () => idleWindow.cancelIdleCallback?.(idle);
    }

    const timeout = window.setTimeout(run, 16);
    return () => window.clearTimeout(timeout);
  }, [account.currency, account.id, afterPaint, transactionsData]);

  const chartReady = afterPaint && chartState.accountId === account.id;

  return (
    <div className="grid">
      <Panel
        title="Account summary"
        action={
          <div className="panel-actions">
            <Button onClick={() => setEditOpen(true)}><Pencil size={16} /> Edit</Button>
            <Button onClick={() => archive.mutate()} disabled={archive.isPending || !account.is_active}><Archive size={16} /> Archive</Button>
            <Button onClick={onBack}>Back</Button>
          </div>
        }
      >
        {actionError ? <div className="error inline-error">{actionError}</div> : null}
        <div className="summary-grid">
          <div><span>Balance</span><strong>{formatMoney(balance.data?.balance ?? "0", account.currency)}</strong></div>
          <div><span>Bank</span><strong>{account.bank || "-"}</strong></div>
          <div><span>Status</span><strong>{account.is_active ? "active" : "archived"}</strong></div>
          <div><span>Opened</span><strong>{dateLabel(account.opened_at)}</strong></div>
        </div>
      </Panel>

      <Panel title="Running balance">
        {chartReady ? (
          <RunningBalanceChart data={chartState.data} currency={account.currency} summary={chartState.summary} />
        ) : <Empty>Preparing chart</Empty>}
      </Panel>

      <Panel
        title="Interest rules"
        action={<Button onClick={() => accrue.mutate()} disabled={accrue.isPending}><BadgePercent size={16} /> Accrue</Button>}
      >
        <div className="rule-list">
          {(rules.data ?? []).map((rule) => <RuleRow key={rule.id} rule={rule} />)}
          {!rules.data?.length ? <Empty>No interest rules</Empty> : null}
        </div>
      </Panel>

      <Panel title="Transactions">
        {afterPaint ? (
          <TransactionsTable transactions={transactionsData} accounts={accountList} categories={[]} chunked />
        ) : <Empty>Preparing transactions</Empty>}
      </Panel>

      {editOpen ? (
        <Dialog title="Edit account" onClose={() => setEditOpen(false)}>
          <EditAccountForm account={account} onDone={() => setEditOpen(false)} />
        </Dialog>
      ) : null}
    </div>
  );
}

const RunningBalanceChart = memo(function RunningBalanceChart({
  data,
  currency,
  summary,
}: {
  data: Array<{ date: string; balance: number }>;
  currency: string;
  summary: string;
}) {
  return (
    <ChartShell summary={summary}>
      <LineChart data={data} margin={{ top: 14, right: 18, bottom: 6, left: 0 }}>
        <defs>
          <linearGradient id="runningBalanceStroke" x1="0" x2="1" y1="0" y2="0">
            <stop offset="0%" stopColor="var(--chart-balance)" stopOpacity={0.72} />
            <stop offset="100%" stopColor="var(--chart-balance-strong)" stopOpacity={1} />
          </linearGradient>
        </defs>
        <CartesianGrid {...chartGridProps} />
        <XAxis {...chartAxisProps} dataKey="date" />
        <YAxis {...chartAxisProps} tickFormatter={(value) => compactChartMoney(Number(value), currency)} width={72} />
        <Line type="monotone" dataKey="balance" stroke="url(#runningBalanceStroke)" strokeWidth={3} dot={false} activeDot={false} isAnimationActive={false} />
      </LineChart>
    </ChartShell>
  );
});

function RuleRow({ rule }: { rule: InterestRule }) {
  const rate = (rule.annual_rate_bps / 100).toFixed(2);
  return (
    <div className="rule-row">
      <strong>{rate}%</strong>
      <span>{rule.accrual_frequency}</span>
      <span>{rule.capitalization_frequency}</span>
      <span>{rule.is_active ? "active" : "inactive"}</span>
    </div>
  );
}

type RunningBalanceState = {
  accountId: string;
  data: Array<{ date: string; balance: number }>;
  summary: string;
};

function emptyRunningBalanceState(accountId: string, currency: string): RunningBalanceState {
  return {
    accountId,
    data: [],
    summary: describeRunningBalance(0, "", "", 0, currency),
  };
}

function buildRunningBalanceState(accountId: string, transactions: Transaction[], currency: string, limit: number): RunningBalanceState {
  if (!transactions.length) {
    return emptyRunningBalanceState(accountId, currency);
  }

  let balance = "0";
  const sorted = [...transactions].sort((a, b) => a.occurred_at.localeCompare(b.occurred_at));
  const lastIndex = sorted.length - 1;
  const sampleIndices = new Set<number>();
  if (sorted.length > limit) {
    const step = lastIndex / (limit - 1);
    for (let index = 0; index < limit; index += 1) {
      sampleIndices.add(Math.round(index * step));
    }
    sampleIndices.add(lastIndex);
  }
  const data: Array<{ date: string; balance: number }> = [];

  for (let index = 0; index < sorted.length; index += 1) {
    const transaction = sorted[index];
    balance = addMoney(balance, signedAmount(transaction));

    if (sorted.length <= limit || sampleIndices.has(index)) {
      data.push({ date: transaction.occurred_at.slice(0, 10), balance: moneyToNumber(balance) });
    }
  }

  return {
    accountId,
    data,
    summary: describeRunningBalance(
      sorted.length,
      sorted[0].occurred_at.slice(0, 10),
      sorted[lastIndex].occurred_at.slice(0, 10),
      moneyToNumber(balance),
      currency,
    ),
  };
}

function compactChartMoney(value: number, currency: string) {
  if (Math.abs(value) >= 1000000) return `${Math.round(value / 1000000)}M ${currency}`;
  if (Math.abs(value) >= 1000) return `${Math.round(value / 1000)}K ${currency}`;
  return `${value} ${currency}`;
}

function describeRunningBalance(count: number, firstDate: string, lastDate: string, finalBalance: number, currency: string) {
  if (!count) {
    return "Running balance chart has no transactions.";
  }

  return `Running balance chart covers ${count} transactions from ${firstDate} to ${lastDate}. Final balance ${formatMoney(String(finalBalance), currency)}.`;
}
