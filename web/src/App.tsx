import { lazy, Suspense, useEffect, useRef, useState } from "react";
import type { KeyboardEvent as ReactKeyboardEvent, ReactNode } from "react";
import { createPortal } from "react-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Box, Flex, Grid, HStack, Stack, Text } from "@chakra-ui/react";
import {
  Command,
  Download,
  LogOut,
  Moon,
  Sun,
} from "lucide-react";
import { useTheme } from "next-themes";
import { ApiClientError, api, clearStoredSession, getStoredToken } from "./api/client";
import { AccountsView } from "./features/accounts/AccountsView";
import { CreateAccountForm } from "./features/accounts/CreateAccountForm";
import { SettingsView } from "./features/settings/SettingsView";
import { TransactionForm } from "./features/transactions/TransactionForm";
import { TransactionsView } from "./features/transactions/TransactionsView";
import { TransferForm } from "./features/transactions/TransferForm";
import { browserSupportsPasskeys, passkeyErrorMessage, signInWithPasskey } from "./features/auth/passkeys";
import { InitialSetupScreen, LoginScreen } from "./features/auth/AuthScreens";
import type { QuickAction, View } from "./shared/constants";
import { currencyOptions } from "./shared/currencies";
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

  function completeQuickAction(message: string) {
    queryClient.invalidateQueries();
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
    return <AuthScreen onAuthenticated={handleAuthenticated} />;
  }

  return (
    <Grid className="app" minH="100vh" templateColumns={{ base: "1fr", lg: "244px minmax(0, 1fr)" }}>
      <Box as="aside" className="sidebar">
        <BrandBlock
          version={serviceStatus.data?.version}
          status={serviceStatus.error ? "Degraded" : serviceStatus.isFetching ? "Checking" : "Healthy"}
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
        <Flex as="header" className="page-head" align="center" justify="space-between" gap={4}>
          <Box minW={0}>
            <Box as="h1" id="pageTitle">{view === "dashboard" ? "Overview" : pageTitle}</Box>
            <Text id="pageSubtitle">
              {view === "dashboard" ? "Daily financial overview without operational noise." : "CapitalFlow workspace"}
            </Text>
            <HStack className="page-title" gap={3} flexWrap="wrap">
              {view === "dashboard" && serviceStatus.data?.version ? (
                <Box as="span" className="version-badge" aria-label={`Service version ${serviceStatus.data.version}`}>
                  {serviceStatus.data.version}
                </Box>
              ) : null}
            </HStack>
          </Box>

          <Box className="head-tools">
            <button className="command-trigger" type="button" aria-label="Open command palette" onClick={() => setCommandOpen(true)}>
              <Command size={16} aria-hidden="true" />
              <span>Search transactions, accounts, commands...</span>
              <span className="kbd">Ctrl K</span>
            </button>
            <span className="chip">{primaryCurrency} · from Settings</span>
          </Box>
        </Flex>

        <PageTransition>
          <Suspense fallback={<Empty>Loading view</Empty>}>
            {view === "dashboard" ? (
              <DashboardView
                key={primaryCurrency}
                primaryCurrency={primaryCurrency}
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
          {quickAction === "account" ? <CreateAccountForm onDone={() => completeQuickAction("Account created")} /> : null}

          {quickAction === "transfer" ? (
            <TransferForm accounts={accounts.data ?? []} onDone={() => completeQuickAction("Transfer created")} />
          ) : null}

          {quickAction === "transaction" ? (
            <TransactionForm
              accounts={accounts.data ?? []}
              categories={categories.data ?? []}
              onDone={() => completeQuickAction("Transaction created")}
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

function BrandBlock({
  version,
  status,
  onCheck,
}: {
  version?: string;
  status: "Healthy" | "Degraded" | "Checking";
  onCheck: () => void;
}) {
  const [healthOpen, setHealthOpen] = useState(false);

  return (
    <Box className="brand">
      <img className="brand-mark" src="/app-icon.png" alt="" aria-hidden="true" />
      <Box className="brand-copy">
        <strong>CapitalFlow</strong>
        <Box className="brand-meta" aria-label="Version, sync and health">
          <span className="version-pill" title="Release tag">{version ?? "dev"}</span>
          <span className="sync-pill" title="Last sync">sync · now</span>
          <button
            className="health-trigger"
            type="button"
            aria-label="Check system health"
            aria-expanded={healthOpen}
            onClick={() => {
              setHealthOpen(true);
              onCheck();
            }}
          >
            {status}
          </button>
        </Box>
        {healthOpen ? <HealthPopover version={version} status={status} onClose={() => setHealthOpen(false)} /> : null}
      </Box>
    </Box>
  );
}

function HealthPopover({
  version,
  status,
  onClose,
}: {
  version?: string;
  status: "Healthy" | "Degraded" | "Checking";
  onClose: () => void;
}) {
  const popoverRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    popoverRef.current?.focus();
  }, []);

  return createPortal(
    <div
      className="popover-layer"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <div
        ref={popoverRef}
        className="health-popover is-open"
        role="dialog"
        aria-modal="false"
        aria-labelledby="healthTitle"
        tabIndex={-1}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            event.preventDefault();
            onClose();
          }
        }}
      >
        <h3 id="healthTitle">System health</h3>
        <div className="health-row"><span>API</span><span className={status === "Healthy" ? "tag good" : "tag"}>{status === "Healthy" ? "OK" : status}</span></div>
        <div className="health-row"><span>Version</span><span className="tag info">{version ?? "unknown"}</span></div>
        <div className="health-row"><span>Rates</span><span className="tag info">On demand</span></div>
      </div>
    </div>,
    document.body,
  );
}

function Nav({ view, accountCount, navigateTo }: { view: View; accountCount: number; navigateTo: (view: View) => void }) {
  return (
    <Stack as="nav" className="nav" aria-label="Main navigation">
      <section className="nav-section">
        <div className="nav-label">Workspace</div>
        <ReferenceNavButton active={view === "dashboard"} label="Overview" onClick={() => navigateTo("dashboard")} />
        <ReferenceNavButton active={view === "transactions"} label="Transactions" onClick={() => navigateTo("transactions")} />
        <ReferenceNavButton active={view === "accounts"} label="Accounts" count={String(accountCount)} onClick={() => navigateTo("accounts")} />
        <ReferenceNavButton active={view === "settings"} label="Settings" onClick={() => navigateTo("settings")} />
      </section>
    </Stack>
  );
}

function ReferenceNavButton({ active, label, count, onClick }: { active: boolean; label: string; count?: string; onClick: () => void }) {
  return (
    <button className="nav-btn" type="button" aria-current={active ? "page" : undefined} onClick={onClick}>
      <span className="nav-name"><span className="nav-dot"></span><span>{label}</span></span>
      {count ? <span className="nav-count">{count}</span> : null}
    </button>
  );
}

function SidebarFooter({ onLogout }: { onLogout: () => void }) {
  const { theme = "light", setTheme } = useTheme();
  const activeTheme = theme === "dark" ? "dark" : "light";

  return (
    <div className="sidebar-footer">
      <button
        className="theme-switch"
        type="button"
        aria-label={activeTheme === "dark" ? "Switch to light theme" : "Switch to dark theme"}
        aria-pressed={activeTheme === "dark"}
        onClick={() => {
          const next = activeTheme === "dark" ? "light" : "dark";
          setTheme(next);
          toaster.create({ type: "info", title: next === "dark" ? "Dark theme enabled" : "Light theme enabled" });
        }}
      >
        <span className="theme-switch-track" aria-hidden="true">
          <span className="theme-switch-thumb">{activeTheme === "dark" ? <Moon size={14} /> : <Sun size={14} />}</span>
        </span>
        <span>{activeTheme === "dark" ? "Dark" : "Light"} mode</span>
      </button>
      <button
        className="logout-button"
        type="button"
        onClick={() => {
          void api.logout()
            .then(() => toaster.create({ type: "success", title: "Logged out" }))
            .catch((err) => toaster.create({ type: "error", title: "Logout failed", description: errorMessage(err) }))
            .finally(() => {
              onLogout();
            });
        }}
      >
        <LogOut size={16} /> Logout
      </button>
    </div>
  );
}

function CommandMenu({
  transactionActionsDisabled,
  onClose,
  onNavigate,
  onQuickAction,
}: {
  transactionActionsDisabled: boolean;
  onClose: () => void;
  onNavigate: (view: View) => void;
  onQuickAction: (action: NonNullable<QuickAction>) => void;
}) {
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const first = dialogRef.current?.querySelector<HTMLElement>(focusableSelector);
    first?.focus();
  }, []);

  function handleKeyDown(event: ReactKeyboardEvent<HTMLDivElement>) {
    if (event.key === "Escape") {
      event.preventDefault();
      onClose();
      return;
    }

    if (event.key !== "Tab") return;

    const focusable = [...(dialogRef.current?.querySelectorAll<HTMLElement>(focusableSelector) ?? [])]
      .filter((element) => !element.hasAttribute("disabled"));
    if (!focusable.length) return;

    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  return createPortal(
    <div
      className="command-backdrop"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <div
        ref={dialogRef}
        className="command-menu"
        role="dialog"
        aria-modal="true"
        aria-labelledby="command-menu-title"
        tabIndex={-1}
        onKeyDown={handleKeyDown}
      >
        <div className="command-menu-head">
          <Command size={18} aria-hidden="true" />
          <h2 id="command-menu-title">Command menu</h2>
          <span className="kbd">Esc</span>
        </div>
        <div className="command-menu-grid">
          <CommandMenuSection title="Navigate">
            <CommandItem onClick={() => onNavigate("dashboard")}>Overview</CommandItem>
            <CommandItem onClick={() => onNavigate("accounts")}>Accounts</CommandItem>
            <CommandItem onClick={() => onNavigate("transactions")}>Transactions</CommandItem>
            <CommandItem onClick={() => onNavigate("settings")}>Settings</CommandItem>
          </CommandMenuSection>
          <CommandMenuSection title="Actions">
            <CommandItem disabled={transactionActionsDisabled} onClick={() => onQuickAction("transaction")}>+ Transaction</CommandItem>
            <CommandItem disabled={transactionActionsDisabled} onClick={() => onQuickAction("transfer")}>+ Transfer</CommandItem>
            <CommandItem onClick={() => onQuickAction("import")}>Import</CommandItem>
            <CommandItem onClick={() => onQuickAction("account")}>Create account</CommandItem>
          </CommandMenuSection>
        </div>
      </div>
    </div>,
    document.body,
  );
}

function CommandMenuSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="command-section" aria-label={title}>
      <h3>{title}</h3>
      <div>{children}</div>
    </section>
  );
}

function CommandItem({ disabled, onClick, children }: { disabled?: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button className="command-item" type="button" disabled={disabled} onClick={onClick}>
      {children}
    </button>
  );
}

function ImportPlaceholder({ onOpenTransactions }: { onOpenTransactions: () => void }) {
  return (
    <div className="import-placeholder">
      <div className="import-drop" aria-disabled="true">
        <Download size={22} aria-hidden="true" />
        <strong>Bank import is not connected yet</strong>
        <span>Backend import is not available yet. Manual transactions and transfers are ready.</span>
      </div>
      <div className="form-actions">
        <button className="btn primary" type="button" onClick={onOpenTransactions}>Open transactions</button>
      </div>
    </div>
  );
}

function AuthScreen({
  onAuthenticated,
}: {
  onAuthenticated: () => void;
}) {
  const status = useQuery({ queryKey: ["auth-status"], queryFn: api.authStatus, retry: false });
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [primaryCurrency, setPrimaryCurrency] = useState("RUB");
  const [error, setError] = useState("");
  const [passkeyError, setPasskeyError] = useState("");
  const [passkeyLoading, setPasskeyLoading] = useState(false);

  const setupRequired = status.data?.setup_required;
  const isSetup = setupRequired === true;
  const passkeysSupported = browserSupportsPasskeys();

  async function submit() {
    setError("");

    try {
      if (isSetup) {
        await api.setup({ email, password, primary_currency: primaryCurrency });
      } else {
        await api.login({ email, password });
      }

      onAuthenticated();
    } catch (err) {
      setError(errorText(err));
    }
  }

  async function submitPasskey() {
    setError("");
    setPasskeyError("");
    setPasskeyLoading(true);

    try {
      await signInWithPasskey();
      onAuthenticated();
    } catch (err) {
      setPasskeyError(passkeyErrorMessage(err));
    } finally {
      setPasskeyLoading(false);
    }
  }

  return (
    isSetup ? (
      <InitialSetupScreen
        email={email}
        password={password}
        primaryCurrency={primaryCurrency}
        currencyOptions={currencyOptions()}
        error={error}
        statusLoading={status.isLoading}
        onEmailChange={setEmail}
        onPasswordChange={setPassword}
        onPrimaryCurrencyChange={setPrimaryCurrency}
        onSubmit={() => {
          void submit();
        }}
      />
    ) : (
      <LoginScreen
        email={email}
        password={password}
        error={error}
        passkeyError={passkeyError}
        passkeysSupported={passkeysSupported}
        passkeyLoading={passkeyLoading}
        statusLoading={status.isLoading}
        onEmailChange={setEmail}
        onPasswordChange={setPassword}
        onSubmit={() => {
          void submit();
        }}
        onPasskeySubmit={() => {
          void submitPasskey();
        }}
      />
    )
  );
}

function errorText(err: unknown) {
  if (err instanceof ApiClientError) {
    return err.message;
  }

  if (err instanceof Error) {
    return err.message;
  }

  return "Request failed";
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

const focusableSelector = [
  "button",
  "[href]",
  "input",
  "select",
  "textarea",
  '[tabindex]:not([tabindex="-1"])',
].join(",");
