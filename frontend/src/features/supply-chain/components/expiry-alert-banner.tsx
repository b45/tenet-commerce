"use client";

import * as React from "react";
import { AlertOctagon, ArrowRight } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import type { ExpirySummary } from "../types";

export interface ExpiryAlertBannerProps {
  summary: ExpirySummary;
  onFilterExpiring: () => void;
}

export function ExpiryAlertBanner({ summary, onFilterExpiring }: ExpiryAlertBannerProps) {
  const { t } = useTranslation();
  const alertCount = summary.expiring_soon_count + summary.expired_count;

  if (alertCount === 0) {
    return null;
  }

  const isCritical = summary.expired_count > 0;

  return (
    <div
      className={`rounded-xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 border transition-all ${
        isCritical
          ? "bg-[var(--color-status-error-bg)] border-[var(--color-status-error-border)] text-[var(--color-status-error-text)]"
          : "bg-[var(--color-status-warning-bg)] border-[var(--color-status-warning-border)] text-[var(--color-status-warning-text)]"
      }`}
    >
      <div className="flex items-start gap-3">
        <AlertOctagon className="h-5 w-5 shrink-0 mt-0.5" />
        <div className="space-y-0.5">
          <h4 className="text-sm font-bold tracking-tight">
            {t("supplyChain.alertBanner.title")} ({alertCount})
          </h4>
          <p className="text-xs leading-relaxed opacity-90">
            {t("supplyChain.alertBanner.description").replace("{count}", String(alertCount))}
          </p>
        </div>
      </div>

      <button
        type="button"
        onClick={onFilterExpiring}
        className="inline-flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold bg-white text-gray-900 shadow-2xs hover:bg-gray-50 dark:bg-gray-900 dark:text-white dark:hover:bg-gray-800 shrink-0 border border-black/10 dark:border-white/10"
      >
        <span>{t("supplyChain.alertBanner.viewExpiring")}</span>
        <ArrowRight className="h-3.5 w-3.5 rtl:rotate-180" />
      </button>
    </div>
  );
}
