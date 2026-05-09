import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { api } from "../../api/client";
import type { Account, Category } from "../../api/types";
import { transactionTypes } from "../../shared/constants";
import { Button, Input, Panel, Select } from "../../shared/ui";
import { TransactionForm } from "./TransactionForm";
import { TransactionsTable } from "./TransactionsTable";

export function TransactionsView({ accounts, categories }: { accounts: Account[]; categories: Category[] }) {
  const transactions = useQuery({ queryKey: ["transactions"], queryFn: () => api.transactions() });
  const [createOpen, setCreateOpen] = useState(false);
  const [accountId, setAccountId] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [type, setType] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const filtered = (transactions.data ?? []).filter((transaction) => {
    const day = transaction.occurred_at.slice(0, 10);
    return (!accountId || transaction.account_id === accountId) &&
      (!categoryId || transaction.category_id === categoryId) &&
      (!type || transaction.type === type) &&
      (!from || day >= from) &&
      (!to || day <= to);
  });

  return (
    <Panel title="Transactions" action={<Button onClick={() => setCreateOpen(true)}><Plus size={16} /> Adjustment</Button>}>
      <div className="filters">
        <Select value={accountId} onChange={(event) => setAccountId(event.target.value)}>
          <option value="">All accounts</option>
          {accounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}
        </Select>
        <Select value={categoryId} onChange={(event) => setCategoryId(event.target.value)}>
          <option value="">All categories</option>
          {categories.map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}
        </Select>
        <Select value={type} onChange={(event) => setType(event.target.value)}>
          <option value="">All types</option>
          {transactionTypes.map((transactionType) => <option key={transactionType}>{transactionType}</option>)}
        </Select>
        <Input type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
        <Input type="date" value={to} onChange={(event) => setTo(event.target.value)} />
      </div>
      <TransactionsTable transactions={filtered} accounts={accounts} categories={categories} allowDelete />
      {createOpen ? (
        <div className="modal-backdrop" onClick={() => setCreateOpen(false)}>
          <div className="modal" onClick={(event) => event.stopPropagation()}>
            <TransactionForm accounts={accounts} categories={categories} fixedType="adjustment" onDone={() => setCreateOpen(false)} />
          </div>
        </div>
      ) : null}
    </Panel>
  );
}

