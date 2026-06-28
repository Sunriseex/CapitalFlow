import { lazy, Suspense, useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Bell,
  PanelLeftClose,
  PanelLeftOpen,
  PanelRightClose,
  PanelRightOpen,
  Search,
} from "lucide-react";
import {
  ApiClientError,
  api,
  clearStoredSession,
  getStoredToken,
} from "./api/client";
import {
  BrandBlock,
  CommandMenu,
  CommandTrigger,
  ImportPlaceholder,
  Nav,
  SidebarStatusCard,
  SidebarFooter,
} from "./features/shell/AppShell";
import type { QuickAction, View } from "./shared/constants";
import { apiErrorMessages, errorMessage } from "./shared/api/query";
import { Dialog, Empty, PageTransition } from "./shared/ui";
import { toaster } from "./components/ui/toaster-store";
import { useI18n } from "./shared/i18n/useI18n";

const AuthController = lazy(() =>
  import("./features/auth/AuthController").then((module) => ({
    default: module.AuthController,
  })),
);
const AccountDetails = lazy(() =>
  import("./features/accounts/AccountDetails").then((module) => ({
    default: module.AccountDetails,
  })),
);
const AccountsView = lazy(() =>
  import("./features/accounts/AccountsView").then((module) => ({
    default: module.AccountsView,
  })),
);
const CreateAccountForm = lazy(() =>
  import("./features/accounts/CreateAccountForm").then((module) => ({
    default: module.CreateAccountForm,
  })),
);
const DashboardView = lazy(() =>
  import("./features/dashboard/DashboardView").then((module) => ({
    default: module.DashboardView,
  })),
);
const CategoryManager = lazy(() =>
  import("./features/categories/CategoryManager").then((module) => ({
    default: module.CategoryManager,
  })),
);
const SettingsView = lazy(() =>
  import("./features/settings/SettingsView").then((module) => ({
    default: module.SettingsView,
  })),
);
const GoalsView = lazy(() =>
  import("./features/goals/GoalsView").then((module) => ({
    default: module.GoalsView,
  })),
);
const TransactionForm = lazy(() =>
  import("./features/transactions/TransactionForm").then((module) => ({
    default: module.TransactionForm,
  })),
);
const TransactionSearchDialog = lazy(() =>
  import("./features/transactions/TransactionSearchDialog").then((module) => ({
    default: module.TransactionSearchDialog,
  })),
);
const TransactionsView = lazy(() =>
  import("./features/transactions/TransactionsView").then((module) => ({
    default: module.TransactionsView,
  })),
);
const TransferForm = lazy(() =>
  import("./features/transactions/TransferForm").then((module) => ({
    default: module.TransferForm,
  })),
);

export function App() {
  const queryClient = useQueryClient();
  const { t } = useI18n();

  const errorMessages = apiErrorMessages(t);

  const [hasSession, setHasSession] = useState(() => Boolean(getStoredToken()));
  const [sessionNonce, setSessionNonce] = useState(0);
  const initialRoute = currentRoute();
  const [view, setView] = useState<View>(initialRoute.view);
  const [selectedAccountId, setSelectedAccountId] = useState(
    initialRoute.accountId,
  );
  const [quickAction, setQuickAction] = useState<QuickAction>(null);
  const [commandOpen, setCommandOpen] = useState(false);
  const [transactionSearchOpen, setTransactionSearchOpen] = useState(false);
  const [categoryManagerOpen, setCategoryManagerOpen] = useState(false);
  const [rightRailHidden, setRightRailHidden] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() =>
    readStoredBoolean("capitalflow_sidebar_collapsed"),
  );

  const accounts = useQuery({
    queryKey: ["accounts", sessionNonce],
    queryFn: api.accounts,
    enabled: hasSession,
  });

  const categories = useQuery({
    queryKey: ["categories", sessionNonce],
    queryFn: api.categories,
    enabled: hasSession,
  });

  const profile = useQuery({
    queryKey: ["profile", sessionNonce],
    queryFn: api.profile,
    enabled: hasSession,
    retry: false,
  });
  const serviceStatus = useQuery({
    queryKey: ["service-status", sessionNonce],
    queryFn: api.serviceStatus,
    enabled: hasSession,
    staleTime: 1000 * 60 * 5,
  });

  const selectedAccount = accounts.data?.find(
    (account) => account.id === selectedAccountId,
  );
  const pageTitle = selectedAccount
    ? selectedAccount.name
    : titleForView(view, t);
  const primaryCurrency = profile.data?.user.primary_currency ?? "RUB";
  const sessionInvalid =
    profile.error instanceof ApiClientError && profile.error.status === 401;
  const accountsReady =
    accounts.isSuccess && (accounts.data?.length ?? "0") > 0;
  const transactionActionsDisabled =
    accounts.isLoading || Boolean(accounts.error) || !accountsReady;

  useEffect(() => {
    if (sessionInvalid) {
      clearStoredSession();
    }
  }, [sessionInvalid]);

  useEffect(() => {
    if (!hasSession || sessionInvalid) {
      return;
    }

    void queryClient.prefetchQuery({
      queryKey: ["transactions"],
      queryFn: () => api.transactions(),
      staleTime: 30_000,
    });
  }, [hasSession, queryClient, sessionInvalid, sessionNonce]);

  useEffect(() => {
    const handlePopState = () => {
      const route = currentRoute();
      setView(route.view);
      setSelectedAccountId(route.accountId);
    };

    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  useEffect(() => {
    const handleCommandShortcut = (event: globalThis.KeyboardEvent) => {
      const isCommandShortcut =
        (event.ctrlKey || event.metaKey) &&
        !event.shiftKey &&
        !event.altKey &&
        ((event.key || "").toLowerCase() === "k" || event.code === "KeyK");

      if (!isCommandShortcut || isTextEditingTarget(event.target)) {
        return;
      }

      event.preventDefault();
      setCommandOpen((open) => !open);
    };

    window.addEventListener("keydown", handleCommandShortcut);
    return () => window.removeEventListener("keydown", handleCommandShortcut);
  }, []);

  useEffect(() => {
    const handleTransactionSearchShortcut = (
      event: globalThis.KeyboardEvent,
    ) => {
      const isSearchShortcut =
        (event.ctrlKey || event.metaKey) &&
        !event.shiftKey &&
        !event.altKey &&
        ((event.key || "").toLowerCase() === "f" || event.code === "KeyF");

      if (!isSearchShortcut || isTextEditingTarget(event.target)) {
        return;
      }

      event.preventDefault();
      setTransactionSearchOpen(true);
    };

    window.addEventListener("keydown", handleTransactionSearchShortcut);
    return () =>
      window.removeEventListener("keydown", handleTransactionSearchShortcut);
  }, []);

  function navigateTo(nextView: View, accountId = "") {
    setView(nextView);
    setSelectedAccountId(accountId);
    const nextPath = pathForRoute(nextView, accountId);
    if (window.location.pathname !== nextPath) {
      window.history.pushState({}, "", nextPath);
    }
  }

  function handleAuthenticated() {
    queryClient.clear();
    setSessionNonce((nonce) => nonce + 1);
    setHasSession(true);
  }

  function handleLogout() {
    clearStoredSession();
    queryClient.clear();
    setSelectedAccountId("");
    setView("dashboard");
    setQuickAction(null);
    setSessionNonce((nonce) => nonce + 1);
    setHasSession(false);
  }

  function completeQuickAction(
    action: Exclude<NonNullable<QuickAction>, "import">,
    message: string,
  ) {
    if (action === "account") {
      void queryClient.invalidateQueries({ queryKey: ["accounts"] });
      void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    } else {
      void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
      void queryClient.invalidateQueries({ queryKey: ["transactions"] });
      void queryClient.invalidateQueries({ queryKey: ["accounts"] });
    }
    setQuickAction(null);
    toaster.create({ type: "success", title: message });
  }

  function openQuickAction(action: NonNullable<QuickAction>) {
    setQuickAction(action);
    if (action === "import") {
      toaster.create({
        type: "info",
        title: t.dashboard.importTransactions,
        description: t.shell.backendImportUnavailable,
      });
    }
  }

  function toggleSidebar() {
    setSidebarCollapsed((collapsed) => {
      const next = !collapsed;
      writeStoredBoolean("capitalflow_sidebar_collapsed", next);
      return next;
    });
  }

  if (!hasSession || sessionInvalid) {
    return (
      <Suspense fallback={<Empty>{t.common.loadingView}</Empty>}>
        <AuthController onAuthenticated={handleAuthenticated} />
      </Suspense>
    );
  }

  return (
    <div
      className={
        sidebarCollapsed
          ? "app app-shell is-sidebar-collapsed"
          : "app app-shell"
      }
    >
      <aside className="sidebar">
        <BrandBlock />
        <Nav
          view={view}
          accountCount={accounts.data?.length ?? 0}
          navigateTo={navigateTo}
        />
        <SidebarStatusCard
          version={serviceStatus.data?.version}
          status={
            serviceStatus.error
              ? "Unavailable"
              : serviceStatus.isFetching
                ? "Checking"
                : "Healthy"
          }
          onCheck={() => {
            void serviceStatus.refetch().then((result) => {
              toaster.create({
                type: result.error ? "error" : "success",
                title: result.error
                  ? t.shell.statusCheckFailed
                  : t.shell.systemHealthy,
                description: result.error
                  ? errorMessage(result.error, errorMessages)
                  : result.data?.version,
              });
            });
          }}
        />
        <SidebarFooter collapsed={sidebarCollapsed} onLogout={handleLogout} />
      </aside>

      <main>
        <header className="page-head">
          <div className="page-head-title">
            <h1 id="pageTitle">
              {view === "dashboard" ? t.nav.overview : pageTitle}
            </h1>
            <div className="page-title">
              {view === "dashboard" && serviceStatus.data?.version ? (
                <span
                  className="version-badge"
                  aria-label={t.shell.serviceVersion.replace(
                    "{version}",
                    serviceStatus.data.version,
                  )}
                >
                  {serviceStatus.data.version}
                </span>
              ) : null}
            </div>
          </div>

          <div className="head-tools">
            <button
              className="shell-icon-button sidebar-toggle-button"
              type="button"
              aria-label={
                sidebarCollapsed
                  ? t.shell.expandSidebar
                  : t.shell.collapseSidebar
              }
              title={
                sidebarCollapsed
                  ? t.shell.expandSidebar
                  : t.shell.collapseSidebar
              }
              aria-pressed={sidebarCollapsed}
              onClick={toggleSidebar}
            >
              {sidebarCollapsed ? (
                <PanelLeftOpen aria-hidden="true" />
              ) : (
                <PanelLeftClose aria-hidden="true" />
              )}
            </button>
            <CommandTrigger onOpen={() => setCommandOpen(true)} />
            <button
              className="shell-icon-button"
              type="button"
              aria-label={t.shell.searchTransactions}
              title={t.shell.searchTransactions}
              aria-haspopup="dialog"
              aria-keyshortcuts="Control+F Meta+F"
              onClick={() => setTransactionSearchOpen(true)}
            >
              <Search aria-hidden="true" />
            </button>
            <button
              className="topbar-action"
              type="button"
              onClick={() => setCategoryManagerOpen(true)}
            >
              {t.dashboard.categories}
            </button>
            <button
              className="topbar-action"
              type="button"
              onClick={() =>
                toaster.create({
                  type: "info",
                  title: t.dashboard.emptyStartUnavailable,
                })
              }
            >
              {t.dashboard.emptyStart}
            </button>
            <button
              className="topbar-action primary"
              type="button"
              onClick={() => openQuickAction("account")}
            >
              {t.accounts.createAccount}
            </button>
            {view === "dashboard" ? null : (
              <button
                className="shell-icon-button"
                type="button"
                aria-label={t.shell.notifications}
                title={t.shell.notifications}
                onClick={() =>
                  toaster.create({
                    type: "info",
                    title: t.shell.notificationsUnavailable,
                  })
                }
              >
                <Bell aria-hidden="true" />
              </button>
            )}
            {view === "dashboard" ? (
              <button
                className="rail-toggle"
                type="button"
                aria-label={
                  rightRailHidden
                    ? t.dashboard.showInsights
                    : t.dashboard.hideInsights
                }
                title={
                  rightRailHidden
                    ? t.dashboard.showInsights
                    : t.dashboard.hideInsights
                }
                aria-controls="dashboard-right-rail"
                aria-expanded={!rightRailHidden}
                onClick={() => setRightRailHidden((hidden) => !hidden)}
              >
                {rightRailHidden ? (
                  <PanelRightOpen size={17} aria-hidden="true" />
                ) : (
                  <PanelRightClose size={17} aria-hidden="true" />
                )}
              </button>
            ) : null}
          </div>
        </header>

        <PageTransition>
          <Suspense fallback={<Empty>{t.common.loadingView}</Empty>}>
            {view === "dashboard" ? (
              <DashboardView
                key={primaryCurrency}
                primaryCurrency={primaryCurrency}
                categories={categories.data ?? []}
                rightRailHidden={rightRailHidden}
                quickActionsDisabled={transactionActionsDisabled}
                onQuickAction={openQuickAction}
                onNavigate={navigateTo}
                onOpenAccount={(id) => {
                  navigateTo("accounts", id);
                }}
              />
            ) : null}
            {view === "accounts" ? (
              selectedAccount ? (
                <AccountDetails
                  account={selectedAccount}
                  onBack={() => navigateTo("accounts")}
                />
              ) : (
                <AccountsView
                  accounts={accounts.data ?? []}
                  isLoading={accounts.isLoading}
                  error={accounts.error}
                  onSelect={(id) => navigateTo("accounts", id)}
                  onCreateAccount={() => openQuickAction("account")}
                  onImport={() => openQuickAction("import")}
                />
              )
            ) : null}

            {view === "transactions" ? (
              <TransactionsView
                accounts={accounts.data ?? []}
                categories={categories.data ?? []}
                accountsLoading={accounts.isLoading}
                accountsError={accounts.error}
                categoriesLoading={categories.isLoading}
                categoriesError={categories.error}
                onCreateTransaction={() => openQuickAction("transaction")}
                onImport={() => openQuickAction("import")}
              />
            ) : null}

            {view === "goals" ? (
              <GoalsView
                accounts={accounts.data ?? []}
                categories={categories.data ?? []}
                primaryCurrency={primaryCurrency}
              />
            ) : null}

            {view === "settings" ? (
              profile.isLoading ? (
                <Empty>{t.settings.loadingProfile}</Empty>
              ) : profile.error ? (
                <div className="error inline-error">
                  {errorMessage(profile.error, errorMessages)}
                </div>
              ) : (
                <SettingsView profile={profile.data} />
              )
            ) : null}
          </Suspense>
        </PageTransition>
      </main>

      {quickAction ? (
        <Dialog
          title={quickActionTitle(quickAction, t)}
          onClose={() => setQuickAction(null)}
          variant={quickAction === "account" ? "wide" : "default"}
        >
          <Suspense fallback={<Empty>{t.common.loadingView}</Empty>}>
            {quickAction === "account" ? (
              <CreateAccountForm
                onDone={() =>
                  completeQuickAction("account", t.accounts.accountCreated)
                }
              />
            ) : null}

            {quickAction === "transfer" ? (
              <TransferForm
                accounts={accounts.data ?? []}
                onDone={() =>
                  completeQuickAction("transfer", t.dashboard.createTransfer)
                }
              />
            ) : null}

            {quickAction === "transaction" ? (
              <TransactionForm
                accounts={accounts.data ?? []}
                categories={categories.data ?? []}
                onDone={() =>
                  completeQuickAction("transaction", t.dashboard.addTransaction)
                }
              />
            ) : null}

            {quickAction === "import" ? (
              <ImportPlaceholder
                onOpenTransactions={() => {
                  setQuickAction(null);
                  navigateTo("transactions");
                }}
              />
            ) : null}
          </Suspense>
        </Dialog>
      ) : null}

      {commandOpen ? (
        <CommandMenu
          transactionActionsDisabled={transactionActionsDisabled}
          onClose={() => setCommandOpen(false)}
          onNavigate={(nextView) => {
            setCommandOpen(false);
            navigateTo(nextView);
          }}
          onQuickAction={(action) => {
            setCommandOpen(false);
            openQuickAction(action);
          }}
        />
      ) : null}

      {transactionSearchOpen ? (
        <Suspense fallback={null}>
          <TransactionSearchDialog
            accounts={accounts.data ?? []}
            categories={categories.data ?? []}
            onClose={() => setTransactionSearchOpen(false)}
          />
        </Suspense>
      ) : null}

      {categoryManagerOpen ? (
        <Dialog
          title={t.categoriesManagement.title}
          onClose={() => setCategoryManagerOpen(false)}
          variant="wide"
        >
          <Suspense fallback={<Empty>{t.common.loading}</Empty>}>
            {categories.isLoading ? (
              <Empty>{t.common.loading}</Empty>
            ) : categories.error ? (
              <div className="error inline-error">
                {errorMessage(categories.error, errorMessages)}
              </div>
            ) : (
              <CategoryManager categories={categories.data ?? []} />
            )}
          </Suspense>
        </Dialog>
      ) : null}
    </div>
  );
}

function titleForView(view: View, t: ReturnType<typeof useI18n>["t"]) {
  return {
    dashboard: t.dashboard.dashboard,
    accounts: t.accounts.title,
    transactions: t.transactions.title,
    goals: t.goals.title,
    settings: t.settings.title,
  }[view];
}

function currentRoute(): { view: View; accountId: string } {
  const segments = window.location.pathname.split("/").filter(Boolean);
  const view = segments[0];

  if (view === "accounts") {
    return {
      view: "accounts",
      accountId: segments[1] ? safeDecodePathSegment(segments[1]) : "",
    };
  }

  if (view === "transactions" || view === "goals" || view === "settings" || view === "dashboard") {
    return { view, accountId: "" };
  }

  return { view: "dashboard" as const, accountId: "" };
}

function safeDecodePathSegment(segment: string) {
  try {
    return decodeURIComponent(segment);
  } catch {
    return "";
  }
}

function pathForRoute(view: View, accountId = "") {
  if (view === "accounts" && accountId) {
    return `/accounts/${encodeURIComponent(accountId)}`;
  }
  return `/${view}`;
}

function readStoredBoolean(key: string) {
  try {
    return window.localStorage.getItem(key) === "true";
  } catch {
    return false;
  }
}

function writeStoredBoolean(key: string, value: boolean) {
  try {
    window.localStorage.setItem(key, String(value));
  } catch {
    // Non-critical preference; keep the in-memory state.
  }
}

function isTextEditingTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) {
    return false;
  }

  const tag = target.tagName.toLowerCase();
  return (
    tag === "input" ||
    tag === "textarea" ||
    target.isContentEditable ||
    target.closest('[contenteditable="true"]') !== null
  );
}

function quickActionTitle(
  action: NonNullable<QuickAction>,
  t: ReturnType<typeof useI18n>["t"],
) {
  return {
    transaction: t.transactions.createTransaction,
    transfer: t.dashboard.createTransfer,
    account: t.accounts.createAccount,
    import: t.dashboard.importTransactions,
  }[action];
}
