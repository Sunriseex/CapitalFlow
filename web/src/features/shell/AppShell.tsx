import { useEffect, useRef, useState } from "react";
import type { KeyboardEvent as ReactKeyboardEvent, ReactNode } from "react";
import { createPortal } from "react-dom";
import { Box, Stack } from "@chakra-ui/react";
import { Command, Download, LogOut, Moon, Sun, X } from "lucide-react";
import { useTheme } from "next-themes";
import { api } from "../../api/client";
import { errorMessage, apiErrorMessages } from "../../shared/api/query";
import { useI18n } from "../../shared/i18n/useI18n";
import type { QuickAction, View } from "../../shared/constants";
import { markPerformance } from "../../shared/performance";
import { toaster } from "../../components/ui/toaster-store";

export function BrandBlock({
  version,
  status,
  onCheck,
}: {
  version?: string;
  status: "Healthy" | "Unavailable" | "Checking";
  onCheck: () => void;
}) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [healthOpen, setHealthOpen] = useState(false);
  const { t } = useI18n();

  return (
    <Box className="brand">
      <img
        className="brand-mark"
        src="/app-icon.png"
        alt=""
        aria-hidden="true"
      />
      <Box className="brand-copy">
        <strong>CapitalFlow</strong>
        <Box className="brand-meta" aria-label={t.shell.versionAndHealth}>
          {" "}
          <span className="version-pill" title={t.shell.version}>
            {version ?? "dev"}
          </span>
          <button
            ref={triggerRef}
            className="health-trigger"
            type="button"
            aria-label={t.shell.checkSystemHealth}
            aria-expanded={healthOpen}
            onClick={() => {
              setHealthOpen(true);
              onCheck();
            }}
          >
            {statusLabel(status, t)}
          </button>
        </Box>
        {healthOpen ? (
          <HealthPopover
            version={version}
            status={status}
            onClose={() => {
              setHealthOpen(false);
              triggerRef.current?.focus();
            }}
          />
        ) : null}
      </Box>
    </Box>
  );
}

export function Nav({
  view,
  accountCount,
  navigateTo,
}: {
  view: View;
  accountCount: number;
  navigateTo: (view: View) => void;
}) {
  const { t } = useI18n();

  return (
    <Stack as="nav" className="nav" aria-label={t.nav.workspace}>
      <section className="nav-section">
        <div className="nav-label">{t.nav.workspace}</div>
        <NavButton
          active={view === "dashboard"}
          label={t.nav.overview}
          onClick={() => navigateTo("dashboard")}
        />
        <NavButton
          active={view === "transactions"}
          label={t.nav.transactions}
          onClick={() => navigateTo("transactions")}
        />
        <NavButton
          active={view === "accounts"}
          label={t.nav.accounts}
          count={String(accountCount)}
          onClick={() => navigateTo("accounts")}
        />
        <NavButton
          active={view === "settings"}
          label={t.nav.settings}
          onClick={() => navigateTo("settings")}
        />
      </section>
    </Stack>
  );
}

export function SidebarFooter({ onLogout }: { onLogout: () => void }) {
  const { theme = "light", setTheme } = useTheme();
  const { locale, toggleLocale, t } = useI18n();
  const errorMessages = apiErrorMessages(t);

  const activeTheme = theme === "dark" ? "dark" : "light";
  const nextLocaleLabel = locale === "ru" ? "EN" : "RU";
  const currentLocaleLabel = locale.toUpperCase();

  return (
    <div className="sidebar-footer">
      <button
        className="theme-switch language-switch"
        type="button"
        aria-label={
          locale === "ru" ? t.shell.switchToEnglish : t.shell.switchToRussian
        }
        onClick={() => {
          toggleLocale();
          toaster.create({ type: "info", title: t.shell.languageChanged });
        }}
      >
        <span className="language-badge" aria-hidden="true">
          {currentLocaleLabel}
        </span>
        <span>{t.shell.language}</span>
        <span className="kbd" aria-hidden="true">
          {nextLocaleLabel}
        </span>
      </button>

      <button
        className="theme-switch"
        type="button"
        aria-label={
          activeTheme === "dark"
            ? t.shell.switchToLightTheme
            : t.shell.switchToDarkTheme
        }
        aria-pressed={activeTheme === "dark"}
        onClick={() => {
          const next = activeTheme === "dark" ? "light" : "dark";
          setTheme(next);
          toaster.create({
            type: "info",
            title:
              next === "dark"
                ? t.shell.darkThemeEnabled
                : t.shell.lightThemeEnabled,
          });
        }}
      >
        <span className="theme-switch-track" aria-hidden="true">
          <span className="theme-switch-thumb">
            {activeTheme === "dark" ? <Moon size={14} /> : <Sun size={14} />}
          </span>
        </span>
        <span>
          {activeTheme === "dark" ? t.shell.darkMode : t.shell.lightMode}
        </span>
      </button>

      <button
        className="logout-button"
        type="button"
        onClick={() => {
          void api
            .logout()
            .then(() =>
              toaster.create({ type: "success", title: t.shell.logoutSuccess }),
            )
            .catch((err) =>
              toaster.create({
                type: "error",
                title: t.shell.logoutFailed,
                description: errorMessage(err, errorMessages),
              }),
            )
            .finally(() => {
              onLogout();
            });
        }}
      >
        <LogOut size={16} /> {t.shell.logout}
      </button>
    </div>
  );
}

export function CommandTrigger({ onOpen }: { onOpen: () => void }) {
  const { t } = useI18n();

  return (
    <button
      className="command-trigger"
      type="button"
      aria-label={t.shell.openCommandMenu}
      onClick={onOpen}
    >
      <Command size={16} aria-hidden="true" />
      <span>{t.shell.openCommandMenu}</span>
      <span className="kbd">{t.shell.commandShortcut}</span>
    </button>
  );
}

export function CommandMenu({
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
  const restoreFocusRef = useRef<HTMLElement | null>(null);
  const focusableRef = useRef<HTMLElement[]>([]);
  const { t } = useI18n();

  useEffect(() => {
    const endMeasure = markPerformance("command-menu-open");
    restoreFocusRef.current =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;
    focusableRef.current = [
      ...(dialogRef.current?.querySelectorAll<HTMLElement>(focusableSelector) ??
        []),
    ].filter((element) => !element.hasAttribute("disabled"));
    const first = focusableRef.current[0];
    first?.focus();
    endMeasure();

    return () => {
      restoreFocusRef.current?.focus();
    };
  }, []);

  function handleKeyDown(event: ReactKeyboardEvent<HTMLDivElement>) {
    if (event.key === "Escape") {
      event.preventDefault();
      onClose();
      return;
    }

    if (event.key !== "Tab") return;

    const focusable = focusableRef.current;
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
          <h2 id="command-menu-title">{t.shell.commandMenu}</h2>{" "}
          <span className="kbd">Esc</span>
        </div>
        <div className="command-menu-grid">
          <CommandMenuSection title={t.shell.navigate}>
            {" "}
            <CommandItem onClick={() => onNavigate("dashboard")}>
              {t.nav.overview}
            </CommandItem>
            <CommandItem onClick={() => onNavigate("accounts")}>
              {t.nav.accounts}
            </CommandItem>
            <CommandItem onClick={() => onNavigate("transactions")}>
              {t.nav.transactions}
            </CommandItem>
            <CommandItem onClick={() => onNavigate("settings")}>
              {t.nav.settings}
            </CommandItem>
          </CommandMenuSection>
          <CommandMenuSection title={t.shell.actions}>
            {" "}
            <CommandItem
              disabled={transactionActionsDisabled}
              onClick={() => onQuickAction("transaction")}
            >
              {t.dashboard.addTransaction}
            </CommandItem>
            <CommandItem
              disabled={transactionActionsDisabled}
              onClick={() => onQuickAction("transfer")}
            >
              {t.dashboard.createTransfer}
            </CommandItem>
            <CommandItem onClick={() => onQuickAction("import")}>
              {t.dashboard.importTransactions}
            </CommandItem>
            <CommandItem onClick={() => onQuickAction("account")}>
              {t.accounts.createAccount}
            </CommandItem>
          </CommandMenuSection>
        </div>
      </div>
    </div>,
    document.body,
  );
}

export function ImportPlaceholder({
  onOpenTransactions,
}: {
  onOpenTransactions: () => void;
}) {
  const { t } = useI18n();
  return (
    <div className="import-placeholder">
      <div className="import-drop" aria-disabled="true">
        <Download size={22} aria-hidden="true" />
        <strong>{t.shell.bankImportNotConnected}</strong>
        <span>{t.shell.backendImportUnavailable}</span>
      </div>
      <div className="form-actions">
        <button
          className="btn primary"
          type="button"
          onClick={onOpenTransactions}
        >
          {t.dashboard.allTransactions}
        </button>
      </div>
    </div>
  );
}

function HealthPopover({
  version,
  status,
  onClose,
}: {
  version?: string;
  status: "Healthy" | "Unavailable" | "Checking";
  onClose: () => void;
}) {
  const { t } = useI18n();
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
        <div className="health-popover-head">
          <h3 id="healthTitle">{t.shell.systemHealth}</h3>{" "}
          <button
            className="health-close"
            type="button"
            aria-label={t.shell.closeSystemHealth}
            onClick={onClose}
          >
            <X size={14} aria-hidden="true" />
          </button>
        </div>
        <div className="health-row">
          <span>{t.shell.api}</span>
          <span className={status === "Healthy" ? "tag good" : "tag info"}>
            {status === "Healthy" ? t.shell.status.ok : statusLabel(status, t)}
          </span>
        </div>
        <div className="health-row">
          <span>{t.shell.version}</span>
          <span className="tag info">{version ?? t.common.unknown}</span>
        </div>
        <div className="health-row">
          <span>{t.shell.rates}</span>
          <span className="tag info">{t.shell.onDemand}</span>
        </div>
      </div>
    </div>,
    document.body,
  );
}

function NavButton({
  active,
  label,
  count,
  onClick,
}: {
  active: boolean;
  label: string;
  count?: string;
  onClick: () => void;
}) {
  return (
    <button
      className="nav-btn"
      type="button"
      aria-current={active ? "page" : undefined}
      onClick={onClick}
    >
      <span className="nav-name">
        <span className="nav-dot"></span>
        <span>{label}</span>
      </span>
      {count ? <span className="nav-count">{count}</span> : null}
    </button>
  );
}

function CommandMenuSection({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="command-section" aria-label={title}>
      <h3>{title}</h3>
      <div>{children}</div>
    </section>
  );
}

function CommandItem({
  disabled,
  onClick,
  children,
}: {
  disabled?: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      className="command-item"
      type="button"
      disabled={disabled}
      onClick={onClick}
    >
      {children}
    </button>
  );
}

function statusLabel(
  status: "Healthy" | "Unavailable" | "Checking",
  t: ReturnType<typeof useI18n>["t"],
) {
  return {
    Healthy: t.shell.status.healthy,
    Unavailable: t.shell.status.unavailable,
    Checking: t.shell.status.checking,
  }[status];
}

const focusableSelector = [
  "button",
  "[href]",
  "input",
  "select",
  "textarea",
  '[tabindex]:not([tabindex="-1"])',
].join(",");
