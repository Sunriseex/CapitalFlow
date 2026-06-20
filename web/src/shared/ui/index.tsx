import { useEffect, useRef } from "react";
import type {
  ButtonHTMLAttributes,
  ComponentProps,
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
} from "react";
import { X } from "lucide-react";
import { Button as ShadcnButton } from "../../components/ui/button";
import {
  Dialog as RadixDialog,
  DialogClose,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "../../components/ui/dialog";
import { Input as ShadcnInput } from "../../components/ui/input";
import { cn } from "../../lib/utils";
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

type SharedButtonProps = ComponentProps<typeof ShadcnButton>;

export function Button({
  className = "",
  variant,
  type,
  ...props
}: SharedButtonProps) {
  const resolvedVariant = variant ?? buttonVariant(className, type);

  return (
    <ShadcnButton
      className={cn(
        "button",
        resolvedVariant === "default" ? "button-primary" : "",
        resolvedVariant === "outline" ? "button-secondary" : "",
        className,
      )}
      type={type}
      variant={resolvedVariant}
      {...props}
    />
  );
}

export function IconButton({
  className = "",
  type = "button",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <ShadcnButton
      className={cn("icon-button", className)}
      size="icon"
      type={type}
      variant="outline"
      {...props}
    />
  );
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

export function ValidatedField({
  children,
  error,
  errorId,
  label,
}: {
  children: ReactNode;
  error?: string;
  errorId: string;
  label: string;
}) {
  return (
    <div className="field">
      <label className="field-control">
        <span>{label}</span>
        {children}
      </label>
      {error ? (
        <span className="field-error" id={errorId}>
          {error}
        </span>
      ) : null}
    </div>
  );
}

export function Input({
  className = "",
  ...props
}: InputHTMLAttributes<HTMLInputElement>) {
  return <ShadcnInput className={cn("input", className)} {...props} />;
}

export function Select({
  className = "",
  ...props
}: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select className={cn("input", className)} {...props} />;
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
  const restoreFocusRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    const endMeasure = markPerformance(`dialog-open:${title}`);
    restoreFocusRef.current =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;
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

  return (
    <RadixDialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        className={[
          "modal dialog-panel",
          variant === "narrow" ? "dialog-panel-narrow" : "",
          variant === "wide" ? "dialog-panel-wide" : "",
        ]
          .filter(Boolean)
          .join(" ")}
        aria-describedby={undefined}
        aria-modal="true"
        showCloseButton={false}
      >
        <DialogHeader className="modal-header dialog-header">
          <div className="dialog-title-stack">
            <DialogTitle className="dialog-title">
              {title}
            </DialogTitle>
          </div>
          <DialogClose asChild>
            <IconButton
              className="dialog-close"
              title={t.common.closeDialog}
              aria-label={t.common.closeDialog}
            >
              <X aria-hidden="true" />
            </IconButton>
          </DialogClose>
        </DialogHeader>
        {children}
      </DialogContent>
    </RadixDialog>
  );
}

function buttonVariant(
  className: string,
  type: SharedButtonProps["type"],
): SharedButtonProps["variant"] {
  if (className.includes("secondary")) {
    return "outline";
  }

  if (className.includes("danger")) {
    return "destructive";
  }

  if (className.includes("button-primary") || type === "submit") {
    return "default";
  }

  return "outline";
}
