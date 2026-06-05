import { useEffect, useRef, useState } from "react";
import type { KeyboardEvent as ReactKeyboardEvent, ReactNode } from "react";
import { createPortal } from "react-dom";
import { Box, Stack } from "@chakra-ui/react";
import { Command, Download, LogOut, Moon, Sun, X } from "lucide-react";
import { useTheme } from "next-themes";
import { api } from "../../api/client";
import { errorMessage } from "../../shared/api/query";
import type { QuickAction, View } from "../../shared/constants";
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

  return (
    <Box className="brand">
      <img className="brand-mark" src="/app-icon.png" alt="" aria-hidden="true" />
      <Box className="brand-copy">
        <strong>CapitalFlow</strong>
        <Box className="brand-meta" aria-label="Version and health">
          <span className="version-pill" title="Release tag">{version ?? "dev"}</span>
          <button
            ref={triggerRef}
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

export function Nav({ view, accountCount, navigateTo }: { view: View; accountCount: number; navigateTo: (view: View) => void }) {
  return (
    <Stack as="nav" className="nav" aria-label="Main navigation">
      <section className="nav-section">
        <div className="nav-label">Workspace</div>
        <NavButton active={view === "dashboard"} label="Overview" onClick={() => navigateTo("dashboard")} />
        <NavButton active={view === "transactions"} label="Transactions" onClick={() => navigateTo("transactions")} />
        <NavButton active={view === "accounts"} label="Accounts" count={String(accountCount)} onClick={() => navigateTo("accounts")} />
        <NavButton active={view === "settings"} label="Settings" onClick={() => navigateTo("settings")} />
      </section>
    </Stack>
  );
}

export function SidebarFooter({ onLogout }: { onLogout: () => void }) {
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

export function CommandTrigger({ onOpen }: { onOpen: () => void }) {
  return (
    <button className="command-trigger" type="button" aria-label="Open command menu" onClick={onOpen}>
      <Command size={16} aria-hidden="true" />
      <span>Open command menu</span>
      <span className="kbd">Ctrl K</span>
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

  useEffect(() => {
    restoreFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const first = dialogRef.current?.querySelector<HTMLElement>(focusableSelector);
    first?.focus();

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

export function ImportPlaceholder({ onOpenTransactions }: { onOpenTransactions: () => void }) {
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

function HealthPopover({
  version,
  status,
  onClose,
}: {
  version?: string;
  status: "Healthy" | "Unavailable" | "Checking";
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
        <div className="health-popover-head">
          <h3 id="healthTitle">System health</h3>
          <button className="health-close" type="button" aria-label="Close system health" onClick={onClose}>
            <X size={14} aria-hidden="true" />
          </button>
        </div>
        <div className="health-row"><span>API</span><span className={status === "Healthy" ? "tag good" : "tag info"}>{status === "Healthy" ? "OK" : status}</span></div>
        <div className="health-row"><span>Version</span><span className="tag info">{version ?? "unknown"}</span></div>
        <div className="health-row"><span>Rates</span><span className="tag info">On demand</span></div>
      </div>
    </div>,
    document.body,
  );
}

function NavButton({ active, label, count, onClick }: { active: boolean; label: string; count?: string; onClick: () => void }) {
  return (
    <button className="nav-btn" type="button" aria-current={active ? "page" : undefined} onClick={onClick}>
      <span className="nav-name"><span className="nav-dot"></span><span>{label}</span></span>
      {count ? <span className="nav-count">{count}</span> : null}
    </button>
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

const focusableSelector = [
  "button",
  "[href]",
  "input",
  "select",
  "textarea",
  '[tabindex]:not([tabindex="-1"])',
].join(",");
