import { useEffect, useId, useRef } from "react";
import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  KeyboardEvent,
  ReactNode,
  SelectHTMLAttributes,
} from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { markPerformance } from "../performance";
import { PageTransition } from "./PageTransition";
import { useI18n } from "../i18n/useI18n";

export { PageTransition };

export function Panel({
  title,
  action,
  children,
  className = "",
}: {
  title: string;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={`panel ${className}`.trim()}>
      <div className="panel-header">
        <h2>{title}</h2>
        {action}
      </div>
      {children}
    </section>
  );
}

export function Button({
  className = "",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement>) {
  return <button className={`button ${className}`} {...props} />;
}

export function IconButton({
  className = "",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement>) {
  return <button className={`icon-button ${className}`} {...props} />;
}

export function Field({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <label className="field">
      <span>{label}</span>
      {children}
    </label>
  );
}

export function Input(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input className="input" {...props} />;
}

export function Select(props: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select className="input" {...props} />;
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="empty">{children}</div>;
}

export function EmptyState({
  icon,
  title,
  description,
  primaryAction,
  secondaryAction,
}: {
  icon?: ReactNode;
  title: string;
  description: string;
  primaryAction?: {
    label: string;
    onClick: () => void;
    disabled?: boolean;
  };
  secondaryAction?: {
    label: string;
    onClick: () => void;
    disabled?: boolean;
  };
}) {
  return (
    <div className="empty-state-panel">
      {icon ? <span className="empty-state-icon">{icon}</span> : null}
      <div>
        <strong>{title}</strong>
        <p>{description}</p>
      </div>
      {primaryAction || secondaryAction ? (
        <div className="empty-state-actions">
          {primaryAction ? (
            <Button
              type="button"
              disabled={primaryAction.disabled}
              onClick={primaryAction.onClick}
            >
              {primaryAction.label}
            </Button>
          ) : null}
          {secondaryAction ? (
            <Button
              className="secondary"
              type="button"
              disabled={secondaryAction.disabled}
              onClick={secondaryAction.onClick}
            >
              {secondaryAction.label}
            </Button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

export function FormShell({
  title,
  error,
  onSubmit,
  children,
  showTitle = true,
}: {
  title: string;
  error: string;
  onSubmit: () => void;
  children: ReactNode;
  showTitle?: boolean;
}) {
  return (
    <form
      className="form form-shell dialog-form"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
    >
      {showTitle ? <h2>{title}</h2> : null}
      {error ? <div className="error">{error}</div> : null}
      {children}
    </form>
  );
}

export function Dialog({
  title,
  onClose,
  children,
  variant = "default",
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
  variant?: "default" | "narrow" | "wide";
}) {
  const { t } = useI18n();

  const titleID = useId();
  const dialogRef = useRef<HTMLDivElement>(null);
  const restoreFocusRef = useRef<HTMLElement | null>(null);
  const focusableRef = useRef<HTMLElement[]>([]);

  useEffect(() => {
    const endMeasure = markPerformance(`dialog-open:${title}`);
    restoreFocusRef.current =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;
    const dialog = dialogRef.current;
    focusableRef.current = [
      ...(dialog?.querySelectorAll<HTMLElement>(focusableSelector) ?? []),
    ].filter((element) => !element.hasAttribute("disabled"));
    const firstFocusable = focusableRef.current[0];
    (firstFocusable ?? dialog)?.focus();
    if (typeof window.requestAnimationFrame !== "function") {
      const timeout = window.setTimeout(endMeasure, 0);
      return () => {
        window.clearTimeout(timeout);
        restoreFocusRef.current?.focus();
      };
    }

    const frame = window.requestAnimationFrame(() => {
      window.requestAnimationFrame(endMeasure);
    });

    return () => {
      window.cancelAnimationFrame(frame);
      restoreFocusRef.current?.focus();
    };
  }, [title]);

  function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key === "Escape") {
      event.preventDefault();
      onClose();
      return;
    }

    if (event.key !== "Tab") {
      return;
    }

    const focusable = focusableRef.current;
    if (!focusable.length) {
      event.preventDefault();
      dialogRef.current?.focus();
      return;
    }

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
      className="modal-backdrop"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <div
        ref={dialogRef}
        className={[
          "modal dialog-panel",
          variant === "narrow" ? "dialog-panel-narrow" : "",
          variant === "wide" ? "dialog-panel-wide" : "",
        ]
          .filter(Boolean)
          .join(" ")}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleID}
        tabIndex={-1}
        onKeyDown={handleKeyDown}
      >
        <div className="modal-header dialog-header">
          <div className="dialog-title-stack">
            <h2 className="dialog-title" id={titleID}>
              {title}
            </h2>
          </div>
          <IconButton
            className="dialog-close"
            type="button"
            title={t.common.closeDialog}
            aria-label={t.common.closeDialog}
            onClick={onClose}
          >
            <X size={16} />
          </IconButton>
        </div>
        {children}
      </div>
    </div>,
    document.body,
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
