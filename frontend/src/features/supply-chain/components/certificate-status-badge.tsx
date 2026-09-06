"use client";

import * as React from "react";
import { ShieldCheck, AlertTriangle, XCircle, Ban, HelpCircle } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import type { CertificateStatus } from "../types";

export interface CertificateStatusBadgeProps {
  status?: CertificateStatus | null;
  className?: string;
}

export function CertificateStatusBadge({ status, className = "" }: CertificateStatusBadgeProps) {
  const { t } = useTranslation();

  if (!status) {
    return (
      <span
        className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300 border border-gray-200 dark:border-gray-700 ${className}`}
      >
        <HelpCircle className="h-3.5 w-3.5" />
        <span>{t("supplyChain.status.none")}</span>
      </span>
    );
  }

  switch (status) {
    case "VALID":
      return (
        <span
          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-[var(--color-status-success-bg)] text-[var(--color-status-success-text)] border border-[var(--color-status-success-border)] ${className}`}
        >
          <ShieldCheck className="h-3.5 w-3.5 text-[var(--color-status-success-text)]" />
          <span>{t("supplyChain.status.valid")}</span>
        </span>
      );
    case "EXPIRING_SOON":
      return (
        <span
          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-[var(--color-status-warning-bg)] text-[var(--color-status-warning-text)] border border-[var(--color-status-warning-border)] animate-pulse ${className}`}
        >
          <AlertTriangle className="h-3.5 w-3.5 text-[var(--color-status-warning-text)]" />
          <span>{t("supplyChain.status.expiringSoon")}</span>
        </span>
      );
    case "EXPIRED":
      return (
        <span
          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-[var(--color-status-error-bg)] text-[var(--color-status-error-text)] border border-[var(--color-status-error-border)] ${className}`}
        >
          <XCircle className="h-3.5 w-3.5 text-[var(--color-status-error-text)]" />
          <span>{t("supplyChain.status.expired")}</span>
        </span>
      );
    case "REVOKED":
      return (
        <span
          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-400 border border-red-200 dark:border-red-800 ${className}`}
        >
          <Ban className="h-3.5 w-3.5" />
          <span>{t("supplyChain.status.revoked")}</span>
        </span>
      );
    default:
      return (
        <span
          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-gray-100 text-gray-700 ${className}`}
        >
          <span>{status}</span>
        </span>
      );
  }
}
