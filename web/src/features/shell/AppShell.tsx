import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import {
  Command as CommandIcon,
  CreditCard,
  Download,
  type LucideIcon,
  LayoutDashboard,
  List,
  LogOut,
  Moon,
  Plus,
  Repeat,
  Settings,
  Sun,
  X,
} from "lucide-react";
import { useTheme } from "next-themes";
import { api } from "../../api/client";
import { errorMessage, apiErrorMessages } from "../../shared/api/query";
import { useI18n } from "../../shared/i18n/useI18n";
import type { Locale } from "../../shared/i18n/i18n";
import type { QuickAction, View } from "../../shared/constants";
import { toaster } from "../../components/ui/toaster-store";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandShortcut,
} from "../../components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../components/ui/popover";
import { Button } from "../../components/ui/button";

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
    <>
      <div className="brand">
        <span className="brand-mark" aria-hidden="true">
          CF
        </span>
        <div className="brand-copy">
          <strong>CapitalFlow</strong>
          <span>{t.nav.workspace}</span>
        </div>
      </div>
      <section
        className="sidebar-status-card"
        aria-label={t.shell.versionAndHealth}
      >
        <div>
          <span>{t.shell.systemHealth}</span>
          <strong>{statusLabel(status, t)}</strong>
        </div>
        <div className="brand-meta">
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
        </div>
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
      </section>
    </>
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
    <nav className="nav" aria-label={t.nav.workspace}>
      <section className="nav-section">
        <div className="nav-label">{t.nav.workspace}</div>
        <NavButton
          active={view === "dashboard"}
          icon={LayoutDashboard}
          label={t.nav.overview}
          onClick={() => navigateTo("dashboard")}
        />
        <NavButton
          active={view === "transactions"}
          icon={List}
          label={t.nav.transactions}
          onClick={() => navigateTo("transactions")}
        />
        <NavButton
          active={view === "accounts"}
          icon={CreditCard}
          label={t.nav.accounts}
          count={String(accountCount)}
          onClick={() => navigateTo("accounts")}
        />
        <NavButton
          active={view === "settings"}
          icon={Settings}
          label={t.nav.settings}
          onClick={() => navigateTo("settings")}
        />
      </section>
    </nav>
  );
}

export function SidebarFooter({
  collapsed = false,
  onLogout,
}: {
  collapsed?: boolean;
  onLogout: () => void;
}) {
  const { theme = "light", setTheme } = useTheme();
  const { locale, setLocale, t } = useI18n();
  const errorMessages = apiErrorMessages(t);

  const activeTheme = theme === "dark" ? "dark" : "light";
  const currentLocaleFlag = locale === "ru" ? "🇷🇺" : "🇬🇧";

  return (
    <div className="sidebar-footer">
      {!collapsed ? (
        <>
          <button
            className="sidebar-icon-button"
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
            {activeTheme === "dark" ? (
              <Moon aria-hidden="true" />
            ) : (
              <Sun aria-hidden="true" />
            )}
            <span className="sr-only">
              {activeTheme === "dark" ? t.shell.darkMode : t.shell.lightMode}
            </span>
          </button>

          <Popover>
            <PopoverTrigger asChild>
              <button
                className="sidebar-icon-button"
                type="button"
                aria-label={t.shell.chooseLanguage}
                title={t.shell.chooseLanguage}
              >
                <span className="language-trigger-flag" aria-hidden="true">
                  {currentLocaleFlag}
                </span>
                <span className="sr-only">{t.shell.language}</span>
              </button>
            </PopoverTrigger>
            <PopoverContent
              className="language-popover"
              align="start"
              role="menu"
            >
              <p className="language-popover-title">{t.shell.language}</p>
              <LanguageChoice
                locale="ru"
                active={locale === "ru"}
                flag="🇷🇺"
                label="Русский"
                onSelect={setLocale}
              />
              <LanguageChoice
                locale="en"
                active={locale === "en"}
                flag="🇬🇧"
                label="English"
                onSelect={setLocale}
              />
            </PopoverContent>
          </Popover>
        </>
      ) : null}

      <button
        className="logout-button"
        type="button"
        aria-label={t.shell.logout}
        title={t.shell.logout}
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
        <LogOut aria-hidden="true" />
        <span className="sr-only">{t.shell.logout}</span>
      </button>
    </div>
  );
}

function LanguageChoice({
  locale,
  active,
  flag,
  label,
  onSelect,
}: {
  locale: Locale;
  active: boolean;
  flag: string;
  label: string;
  onSelect: (locale: Locale) => void;
}) {
  const { t } = useI18n();
  return (
    <button
      className="language-choice"
      type="button"
      role="menuitemradio"
      aria-checked={active}
      onClick={() => {
        onSelect(locale);
        toaster.create({ type: "info", title: t.shell.languageChanged });
      }}
    >
      <span className="language-choice-copy">
        <span aria-hidden="true">{flag}</span>
        <span>{label}</span>
      </span>
      <span className="language-choice-check" aria-hidden="true">
        {active ? "✓" : ""}
      </span>
    </button>
  );
}

export function CommandTrigger({ onOpen }: { onOpen: () => void }) {
  const { t } = useI18n();

  return (
    <button
      className="command-trigger"
      type="button"
      aria-label={t.shell.openCommandMenu}
      aria-haspopup="dialog"
      aria-keyshortcuts="Control+K Meta+K"
      onClick={onOpen}
    >
      <CommandIcon className="command-trigger-icon" aria-hidden="true" />
      <span className="command-trigger-text">{t.shell.openCommandMenu}</span>
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
  const { t } = useI18n();

  return (
    <CommandDialog
      open
      title={t.shell.commandMenu}
      description={t.shell.commandMenuDescription}
      className="command-menu"
      showCloseButton
      onOpenChange={(open) => !open && onClose()}
    >
      <CommandInput placeholder={t.shell.commandMenuPlaceholder} />
      <CommandList>
        <CommandEmpty>{t.shell.noCommandResults}</CommandEmpty>
        <CommandGroup heading={t.shell.actions}>
          <CommandAction
            value="add transaction income expense operation manual"
            disabled={transactionActionsDisabled}
            icon={<Plus aria-hidden="true" />}
            title={t.dashboard.addTransaction}
            description={t.shell.addTransactionCommandDescription}
            onSelect={() => onQuickAction("transaction")}
          />
          <CommandAction
            value="transfer move money between accounts"
            disabled={transactionActionsDisabled}
            icon={<Repeat aria-hidden="true" />}
            title={t.dashboard.createTransfer}
            description={t.shell.createTransferCommandDescription}
            onSelect={() => onQuickAction("transfer")}
          />
          <CommandAction
            value="account create add card cash savings"
            icon={<CreditCard aria-hidden="true" />}
            title={t.accounts.createAccount}
            description={t.shell.addAccountCommandDescription}
            onSelect={() => onQuickAction("account")}
          />
          <CommandAction
            value="import csv bank statement"
            icon={<Download aria-hidden="true" />}
            title={t.dashboard.importTransactions}
            description={t.shell.importCommandDescription}
            onSelect={() => onQuickAction("import")}
          />
        </CommandGroup>
        <CommandGroup heading={t.shell.navigate}>
          <CommandAction
            value="overview dashboard balance home"
            icon={<LayoutDashboard aria-hidden="true" />}
            title={t.nav.overview}
            description={t.shell.openOverviewCommandDescription}
            onSelect={() => onNavigate("dashboard")}
          />
          <CommandAction
            value="transactions ledger operations"
            icon={<List aria-hidden="true" />}
            title={t.nav.transactions}
            description={t.shell.openTransactionsCommandDescription}
            onSelect={() => onNavigate("transactions")}
          />
          <CommandAction
            value="accounts cards cash savings deposits"
            icon={<CreditCard aria-hidden="true" />}
            title={t.nav.accounts}
            description={t.shell.openAccountsCommandDescription}
            onSelect={() => onNavigate("accounts")}
          />
          <CommandAction
            value="settings profile security passkeys currency"
            icon={<Settings aria-hidden="true" />}
            title={t.nav.settings}
            description={t.shell.openSettingsCommandDescription}
            onSelect={() => onNavigate("settings")}
          />
        </CommandGroup>
      </CommandList>
      <div className="command-footer">
        <span>{t.shell.commandMenuHelp}</span>
        <span className="command-help">
          <span className="kbd">Enter</span>
          {t.shell.commandMenuSelectHint}
          <span className="kbd">Esc</span>
        </span>
      </div>
    </CommandDialog>
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
        <Button
          type="button"
          onClick={onOpenTransactions}
        >
          {t.dashboard.allTransactions}
        </Button>
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
  icon: Icon,
  label,
  count,
  onClick,
}: {
  active: boolean;
  icon: LucideIcon;
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
        <span className="nav-icon" aria-hidden="true">
          <Icon />
        </span>
        <span>{label}</span>
      </span>
      {count ? <span className="nav-count">{count}</span> : null}
    </button>
  );
}

function CommandAction({
  disabled,
  icon,
  title,
  description,
  value,
  onSelect,
}: {
  disabled?: boolean;
  icon: ReactNode;
  title: string;
  description: string;
  value: string;
  onSelect: () => void;
}) {
  return (
    <CommandItem
      className="command-item"
      value={`${title} ${description} ${value}`}
      disabled={disabled}
      onSelect={() => {
        if (!disabled) {
          onSelect();
        }
      }}
    >
      <span className="command-item-icon" aria-hidden="true">
        {icon}
      </span>
      <span className="command-action-copy">
        <strong>{title}</strong>
        <small>{description}</small>
      </span>
      <CommandShortcut>↵</CommandShortcut>
    </CommandItem>
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
