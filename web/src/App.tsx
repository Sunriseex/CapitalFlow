import { lazy, Suspense, useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Box, Grid, HStack } from "@chakra-ui/react";
import { PanelRightClose, PanelRightOpen } from "lucide-react";
import { ApiClientError, api, clearStoredSession, getStoredToken } from "./api/client";
import { AccountsView } from "./features/accounts/AccountsView";
import { CreateAccountForm } from "./features/accounts/CreateAccountForm";
import { SettingsView } from "./features/settings/SettingsView";
import { TransactionForm } from "./features/transactions/TransactionForm";
import { TransactionsView } from "./features/transactions/TransactionsView";
import { TransferForm } from "./features/transactions/TransferForm";
import { AuthController } from "./features/auth/AuthController";
import { BrandBlock, CommandMenu, CommandTrigger, ImportPlaceholder, Nav, SidebarFooter } from "./features/shell/AppShell";
import type { QuickAction, View } from "./shared/constants";
import { errorMessage } from "./shared/api/query";
import { Dialog, Empty, PageTransition } from "./shared/ui";
import { toaster } from "./components/ui/toaster-store";

const AccountDetails = lazy(() =>
  import("./features/accounts/AccountDetails").then((module) => ({ default: module.AccountDetails })),
);
const DashboardView = lazy(() =>
  import("./features/dashboard/DashboardView").then((module) => ({ default: module.DashboardView })),
);

export function App() {
  const queryClient = useQueryClient();

  const [hasSession, setHasSession] = useState(() => Boolean(getStoredToken()));
  const [sessionNonce, setSessionNonce] = useState(0);
  const initialRoute = currentRoute();
  const [view, setView] = useState<View>(initialRoute.view);
  const [selectedAccountId, setSelectedAccountId] = useState(initialRoute.accountId);
  const [quickAction, setQuickAction] = useState<QuickAction>(null);
  const [commandOpen, setCommandOpen] = useState(false);
  const [rightRailHidden, setRightRailHidden] = useState(false);

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

  const selectedAccount = accounts.data?.find((account) => account.id === selectedAccountId);
  const pageTitle = selectedAccount ? selectedAccount.name : titleForView(view);
  const primaryCurrency = profile.data?.user.primary_currency ?? "RUB";
  const sessionInvalid = profile.error instanceof ApiClientError && profile.error.status === 401;
  const accountsReady = accounts.isSuccess && (accounts.data?.length ?? "0") > 0;
  const transactionActionsDisabled = accounts.isLoading || Boolean(accounts.error) || !accountsReady;

  useEffect(() => {
    if (sessionInvalid) {
      clearStoredSession();
    }
  }, [sessionInvalid]);

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
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setCommandOpen(true);
      }
    };

    window.addEventListener("keydown", handleCommandShortcut);
    return () => window.removeEventListener("keydown", handleCommandShortcut);
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

  function completeQuickAction(action: Exclude<NonNullable<QuickAction>, "import">, message: string) {
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
      toaster.create({ type: "info", title: "Import preview", description: "Backend import is not available yet." });
    }
  }

  if (!hasSession || sessionInvalid) {
    return <AuthController onAuthenticated={handleAuthenticated} />;
  }

  return (
    <Grid className="app" minH="100vh" templateColumns={{ base: "1fr", lg: "244px minmax(0, 1fr)" }}>
      <Box as="aside" className="sidebar">
        <BrandBlock
          version={serviceStatus.data?.version}
          status={serviceStatus.error ? "Unavailable" : serviceStatus.isFetching ? "Checking" : "Healthy"}
          onCheck={() => {
            void serviceStatus.refetch().then((result) => {
              toaster.create({
                type: result.error ? "error" : "success",
                title: result.error ? "Status check failed" : "System healthy",
                description: result.error ? errorMessage(result.error) : result.data?.version,
              });
            });
          }}
        />
        <Nav view={view} accountCount={accounts.data?.length ?? 0} navigateTo={navigateTo} />
        <SidebarFooter
          onLogout={handleLogout}
        />
      </Box>

      <Box as="main" pb={{ base: 24, lg: 8 }}>
        <Box as="header" className="page-head">
          <Box minW={0}>
            <Box as="h1" id="pageTitle">{view === "dashboard" ? "Overview" : pageTitle}</Box>
            <HStack className="page-title" gap={3} flexWrap="wrap">
              {view === "dashboard" && serviceStatus.data?.version ? (
                <Box as="span" className="version-badge" aria-label={`Service version ${serviceStatus.data.version}`}>
                  {serviceStatus.data.version}
                </Box>
              ) : null}
            </HStack>
          </Box>

          <Box className="head-tools">
            <CommandTrigger onOpen={() => setCommandOpen(true)} />
            {view === "dashboard" ? (
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
            ) : null}
          </Box>
        </Box>

        <PageTransition>
          <Suspense fallback={<Empty>Loading view</Empty>}>
            {view === "dashboard" ? (
              <DashboardView
                key={primaryCurrency}
                primaryCurrency={primaryCurrency}
                rightRailHidden={rightRailHidden}
                quickActionsDisabled={transactionActionsDisabled}
                onToggleRightRail={() => setRightRailHidden((hidden) => !hidden)}
                onQuickAction={openQuickAction}
                onNavigate={navigateTo}
                onOpenAccount={(id) => {
                  navigateTo("accounts", id);
                }}
              />
            ) : null}

            {view === "accounts" ? (
              selectedAccount ? (
                <AccountDetails account={selectedAccount} onBack={() => navigateTo("accounts")} />
              ) : (
                <AccountsView
                  accounts={accounts.data ?? []}
                  isLoading={accounts.isLoading}
                  error={accounts.error}
                  onSelect={(id) => navigateTo("accounts", id)}
                />
              )
            ) : null}
          </Suspense>

          {view === "transactions" ? (
            <TransactionsView
              accounts={accounts.data ?? []}
              categories={categories.data ?? []}
              accountsLoading={accounts.isLoading}
              accountsError={accounts.error}
              categoriesLoading={categories.isLoading}
              categoriesError={categories.error}
            />
          ) : null}

          {view === "settings" ? (
            profile.isLoading ? (
              <Empty>Loading profile</Empty>
            ) : profile.error ? (
              <Box className="error inline-error">{errorMessage(profile.error)}</Box>
            ) : (
              <SettingsView profile={profile.data} />
            )
          ) : null}
        </PageTransition>
      </Box>

      {quickAction ? (
        <Dialog title={quickActionTitle(quickAction)} onClose={() => setQuickAction(null)}>
          {quickAction === "account" ? <CreateAccountForm onDone={() => completeQuickAction("account", "Account created")} /> : null}

          {quickAction === "transfer" ? (
            <TransferForm accounts={accounts.data ?? []} onDone={() => completeQuickAction("transfer", "Transfer created")} />
          ) : null}

          {quickAction === "transaction" ? (
            <TransactionForm
              accounts={accounts.data ?? []}
              categories={categories.data ?? []}
              onDone={() => completeQuickAction("transaction", "Transaction created")}
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
    </Grid>
  );
}

function titleForView(view: View) {
  return {
    dashboard: "Dashboard",
    accounts: "Accounts",
    transactions: "Transactions",
    settings: "Settings",
  }[view];
}

function currentRoute(): { view: View; accountId: string } {
  const segments = window.location.pathname.split("/").filter(Boolean);
  const view = segments[0];

  if (view === "accounts") {
    return { view: "accounts", accountId: segments[1] ? safeDecodePathSegment(segments[1]) : "" };
  }

  if (view === "transactions" || view === "settings" || view === "dashboard") {
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

function quickActionTitle(action: NonNullable<QuickAction>) {
  return {
    transaction: "Create transaction",
    transfer: "Create transfer",
    account: "Create account",
    import: "Import transactions",
  }[action];
}
