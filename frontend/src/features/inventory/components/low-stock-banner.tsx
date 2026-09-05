"use client";

import * as React from "react";
import { AlertTriangle, ArrowRight, X } from "lucide-react";
import { useTranslation } from "@/lib/i18n";

interface LowStockBannerProps {
  count: number;
  onFilterLowStock: () => void;
  isFilterActive: boolean;
}

export function LowStockBanner({
  count,
  onFilterLowStock,
  isFilterActive,
}: LowStockBannerProps) {
  const { t } = useTranslation();
  const [dismissed, setDismissed] = React.useState(false);

  if (count <= 0 || dismissed) {
    return null;
  }

  return (
    <aside
      aria-label={t("inventory.lowStockBanner.alertTitle")}
      className="mb-6 flex flex-wrap items-center justify-between gap-4 rounded-xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-amber-900 shadow-sm backdrop-blur-sm dark:text-amber-200"
    >
      <div className="flex items-center gap-3">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-amber-500/20 text-amber-600 dark:text-amber-400">
          <AlertTriangle className="h-5 w-5" aria-hidden="true" />
        </div>
        <div>
          <h2 className="text-sm font-semibold">
            {t("inventory.lowStockBanner.alertTitle")}
          </h2>
          <p className="text-xs text-amber-800/80 dark:text-amber-300/80">
            {t("inventory.lowStockBanner.alertMessage", { count })}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onFilterLowStock}
          className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
            isFilterActive
              ? "bg-amber-600 text-white shadow-sm"
              : "bg-amber-500/20 text-amber-900 hover:bg-amber-500/30 dark:text-amber-200"
          }`}
        >
          <span>{t("inventory.lowStockBanner.viewItems")}</span>
          <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
        </button>

        <button
          type="button"
          onClick={() => setDismissed(true)}
          className="rounded-lg p-1.5 text-amber-700/70 hover:bg-amber-500/20 hover:text-amber-900 dark:text-amber-400 dark:hover:text-amber-200"
          aria-label="Tutup pemberitahuan"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>
      </div>
    </aside>
  );
}
