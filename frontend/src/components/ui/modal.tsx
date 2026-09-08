"use client";

import * as React from "react";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTranslation } from "@/lib/i18n";

export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: React.ReactNode;
  description?: React.ReactNode;
  children: React.ReactNode;
  maxWidth?: "sm" | "md" | "lg" | "xl" | "2xl";
  className?: string;
  hideCloseButton?: boolean;
  dismissible?: boolean;
}

const maxWidthMap = {
  sm: "max-w-sm", md: "max-w-md", lg: "max-w-lg", xl: "max-w-xl", "2xl": "max-w-2xl",
};

export function Modal({
  isOpen, onClose, title, description, children, maxWidth = "md", className,
  hideCloseButton = false, dismissible = true,
}: ModalProps) {
  const { t } = useTranslation();
  const id = React.useId();
  const dialogRef = React.useRef<HTMLDialogElement>(null);
  const headingRef = React.useRef<HTMLHeadingElement>(null);

  React.useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog || !isOpen) return;
    const previousOverflow = document.body.style.overflow;
    dialog.showModal();
    dialog.scrollTop = 0;
    document.body.style.overflow = "hidden";
    // Focus non-input content: opening a cash dialog must not summon a touch keyboard.
    headingRef.current?.focus({ preventScroll: true });
    return () => {
      dialog.close();
      document.body.style.overflow = previousOverflow;
    };
  }, [isOpen]);

  const requestClose = () => { if (dismissible) onClose(); };

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby={id + "-title"}
      aria-describedby={description ? id + "-description" : undefined}
      onCancel={event => { event.preventDefault(); requestClose(); }}
      onClick={event => {
        if (event.target !== event.currentTarget) return;
        const rect = event.currentTarget.getBoundingClientRect();
        if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) requestClose();
      }}
      className={cn(
        "m-auto max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] overflow-y-auto overscroll-contain rounded-2xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] p-0 text-[var(--color-text-primary)] shadow-[var(--shadow-elevated)] backdrop:bg-black/40",
        maxWidthMap[maxWidth]
      )}
    >
      <div className={cn("p-4 sm:p-6", className)}>
        <div className="flex items-start justify-between gap-3 border-b border-[var(--color-border-hairline)] pb-4">
          <div className="min-w-0">
            <h2 id={id + "-title"} ref={headingRef} tabIndex={-1}
              className="break-words text-lg font-semibold focus-visible:outline focus-visible:outline-2">
              {title || t("common.actions.confirm")}
            </h2>
            {description && <p id={id + "-description"} className="mt-2 [overflow-wrap:anywhere] text-sm text-[var(--color-text-secondary)]">{description}</p>}
          </div>
          {!hideCloseButton && dismissible && (
            <button type="button" onClick={requestClose} aria-label={t("common.actions.close")}
              className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-action-primary)]">
              <X className="h-5 w-5" aria-hidden="true" />
            </button>
          )}
        </div>
        <div className="mt-4">{children}</div>
      </div>
    </dialog>
  );
}
