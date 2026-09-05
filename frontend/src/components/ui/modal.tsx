"use client";

import * as React from "react";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: React.ReactNode;
  description?: React.ReactNode;
  children: React.ReactNode;
  maxWidth?: "sm" | "md" | "lg" | "xl" | "2xl";
  className?: string;
  hideCloseButton?: boolean;
}

const maxWidthMap = {
  sm: "max-w-sm",
  md: "max-w-md",
  lg: "max-w-lg",
  xl: "max-w-xl",
  "2xl": "max-w-2xl",
};

export function Modal({
  isOpen,
  onClose,
  title,
  description,
  children,
  maxWidth = "md",
  className,
  hideCloseButton = false,
}: ModalProps) {
  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen) {
        onClose();
      }
    };
    if (isOpen) {
      document.body.style.overflow = "hidden";
      window.addEventListener("keydown", handleKeyDown);
    }
    return () => {
      document.body.style.overflow = "unset";
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6 overflow-y-auto">
      {/* Backdrop with frosted blur */}
      <div
        className="fixed inset-0 bg-black/35 backdrop-blur-[6px] transition-opacity animate-in fade-in duration-200"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Cupertino Tactile Modal Card */}
      <div
        role="dialog"
        aria-modal="true"
        className={cn(
          "relative w-full bg-[var(--color-surface-base)] rounded-[22px]",
          "border border-[var(--color-border-hairline)] shadow-[var(--shadow-elevated)]",
          "p-6 sm:p-7 overflow-hidden z-10",
          "animate-in fade-in zoom-in-95 duration-200",
          maxWidthMap[maxWidth],
          className
        )}
      >
        {/* Header */}
        {(title || !hideCloseButton) && (
          <div className="flex items-start justify-between pb-4 border-b border-[var(--color-border-hairline)]">
            <div>
              {title && (
                <h3 className="text-lg font-semibold tracking-tight text-[var(--color-text-primary)]">
                  {title}
                </h3>
              )}
              {description && (
                <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">
                  {description}
                </p>
              )}
            </div>

            {!hideCloseButton && (
              <button
                type="button"
                onClick={onClose}
                aria-label="Tutup dialog"
                className="w-8 h-8 rounded-full flex items-center justify-center text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-surface-muted)] transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-action-focus-ring)]"
              >
                <X className="w-4 h-4" />
              </button>
            )}
          </div>
        )}

        {/* Content */}
        <div className="mt-4">{children}</div>
      </div>
    </div>
  );
}
