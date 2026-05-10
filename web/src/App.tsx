import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowDownLeft, ArrowRightLeft, ArrowUpRight, Landmark, Moon, Plus, Settings, Sun, Wallet } from "lucide-react";
import { api, getStoredApiBase, getStoredToken, setStoredApiBase, setStoredToken } from "./api/client";
import { AccountDetails } from "./features/accounts/AccountDetails";
import { AccountsView } from "./features/accounts/AccountsView";
import { CreateAccountForm } from "./features/accounts/CreateAccountForm";
import { DashboardView } from "./features/dashboard/DashboardView";
import { TransactionForm } from "./features/transactions/TransactionForm";
import { TransactionsView } from "./features/transactions/TransactionsView";
import { TransferForm } from "./features/transactions/TransferForm";
import type { QuickAction, Theme, View } from "./shared/constants";
import { themeStorageKey } from "./shared/constants";
import { Button, Field, IconButton, Input } from "./shared/ui";

export function App() {
  const [view, setView] = useState<View>("dashboard");
  const [selectedAccountId, setSelectedAccountId] = useState("");
  const [quickAction, setQuickAction] = useState<QuickAction>(null);
  const [authOpen, setAuthOpen] = useState(false);
  const [theme, setTheme] = useState<Theme>(() => storedTheme());
  const accounts = useQuery({ queryKey: ["accounts"], queryFn: api.accounts });
  const categories = useQuery({ queryKey: ["categories"], queryFn: api.categories });

  const selectedAccount = accounts.data?.find((account) => account.id === selectedAccountId);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem(themeStorageKey, theme);
  }, [theme]);

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="brand">
          <Wallet size={22} />
          <span>CapitalFlow</span>
        </div>
        <nav>
          <button className={view === "dashboard" ? "active" : ""} onClick={() => setView("dashboard")}>
            <Landmark size={16} /> Dashboard
          </button>
          <button className={view === "accounts" ? "active" : ""} onClick={() => setView("accounts")}>
            <Wallet size={16} /> Accounts
          </button>
          <button className={view === "transactions" ? "active" : ""} onClick={() => setView("transactions")}>
            <ArrowRightLeft size={16} /> Transactions
          </button>
        </nav>
        <Button className="muted-button" onClick={() => setAuthOpen((open) => !open)}>
          <Settings size={16} /> API
        </Button>
        {authOpen ? <AuthPanel /> : null}
      </aside>

      <main>
        <header className="topbar">
          <div>
            <p className="eyebrow">v0.5 MVP</p>
            <h1>{selectedAccount ? selectedAccount.name : titleForView(view)}</h1>
          </div>
          <div className="quick-actions">
            <IconButton
              title={theme === "dark" ? "Light theme" : "Dark theme"}
              onClick={() => setTheme((current) => current === "dark" ? "light" : "dark")}
            >
              {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
            </IconButton>
            <IconButton title="Income" onClick={() => setQuickAction("income")}>
              <ArrowDownLeft size={18} />
            </IconButton>
            <IconButton title="Expense" onClick={() => setQuickAction("expense")}>
              <ArrowUpRight size={18} />
            </IconButton>
            <IconButton title="Transfer" onClick={() => setQuickAction("transfer")}>
              <ArrowRightLeft size={18} />
            </IconButton>
            <IconButton title="Create account" onClick={() => setQuickAction("account")}>
              <Plus size={18} />
            </IconButton>
          </div>
        </header>

        {view === "dashboard" ? <DashboardView onOpenAccount={(id) => { setSelectedAccountId(id); setView("accounts"); }} /> : null}
        {view === "accounts" ? (
          selectedAccount ? (
            <AccountDetails account={selectedAccount} onBack={() => setSelectedAccountId("")} />
          ) : (
            <AccountsView accounts={accounts.data ?? []} onSelect={setSelectedAccountId} />
          )
        ) : null}
        {view === "transactions" ? (
          <TransactionsView accounts={accounts.data ?? []} categories={categories.data ?? []} />
        ) : null}
      </main>

      {quickAction ? (
        <div className="modal-backdrop" onClick={() => setQuickAction(null)}>
          <div className="modal" onClick={(event) => event.stopPropagation()}>
            {quickAction === "account" ? <CreateAccountForm onDone={() => setQuickAction(null)} /> : null}
            {quickAction === "transfer" ? (
              <TransferForm accounts={accounts.data ?? []} onDone={() => setQuickAction(null)} />
            ) : null}
            {quickAction === "income" || quickAction === "expense" ? (
              <TransactionForm
                accounts={accounts.data ?? []}
                categories={categories.data ?? []}
                fixedType={quickAction}
                onDone={() => setQuickAction(null)}
              />
            ) : null}
          </div>
        </div>
      ) : null}
    </div>
  );
}

function AuthPanel() {
  const [token, setToken] = useState(getStoredToken());
  const [apiBase, setApiBase] = useState(getStoredApiBase());

  return (
    <form className="auth-panel" onSubmit={(event) => { event.preventDefault(); setStoredToken(token); setStoredApiBase(apiBase); location.reload(); }}>
      <Field label="API base">
        <Input value={apiBase} onChange={(event) => setApiBase(event.target.value)} />
      </Field>
      <Field label="Bearer token">
        <Input type="password" value={token} onChange={(event) => setToken(event.target.value)} />
      </Field>
      <Button>Save</Button>
    </form>
  );
}

function titleForView(view: View) {
  return {
    dashboard: "Dashboard",
    accounts: "Accounts",
    transactions: "Transactions",
  }[view];
}

function storedTheme(): Theme {
  const stored = localStorage.getItem(themeStorageKey);
  if (stored === "dark" || stored === "light") {
    return stored;
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}
