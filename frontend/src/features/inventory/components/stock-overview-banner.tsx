"use client";

import * as React from "react";
import { Package, Layers, AlertCircle, AlertTriangle, MapPin, Clock } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import type { StockOverviewCard } from "../types";

interface StockOverviewBannerProps {
  overview: StockOverviewCard | null;
  isLoading?: boolean;
}

export function StockOverviewBanner({ overview, isLoading }: StockOverviewBannerProps) {
  const { t } = useTranslation();

  if (isLoading || !overview) {
    return null;
  }

  const asOfTime = overview.as_of ? new Date(overview.as_of).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) : "";

  return (
    <div className="rounded-2xl border border-black/[0.08] bg-white p-4 sm:p-5 shadow-[0_1px_3px_rgba(0,0,0,0.02)]">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-black/[0.06] pb-3.5 mb-4">
        <div className="flex items-center gap-2.5">
          <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-blue-50 text-[#0066CC]">
            <Package className="h-4 w-4" aria-hidden="true" />
          </div>
          <div>
            <h2 className="text-sm font-bold tracking-tight text-[var(--color-text-primary)]">
              {t("inventory.overviewCard.title")}
            </h2>
            <div className="flex items-center gap-3 text-[11px] text-[var(--color-text-secondary)] mt-0.5">
              <span className="flex items-center gap-1 font-mono">
                <MapPin className="h-3 w-3 text-[#8B95A5]" aria-hidden="true" />
                <span>{overview.warehouse_location}</span>
              </span>
              {asOfTime && (
                <span className="flex items-center gap-1 font-mono">
                  <Clock className="h-3 w-3 text-[#8B95A5]" aria-hidden="true" />
                  <span>{t("inventory.overviewCard.asOf")} {asOfTime}</span>
                </span>
              )}
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 sm:gap-4">
        {/* Total SKUs */}
        <div className="rounded-xl bg-[#F8F8FA] p-3.5 border border-black/[0.04]">
          <div className="flex items-center gap-1.5 text-[var(--color-text-secondary)] text-xs font-medium">
            <Layers className="h-3.5 w-3.5 text-[#8B95A5]" aria-hidden="true" />
            <span>{t("inventory.overviewCard.totalSkus")}</span>
          </div>
          <div className="mt-1 font-mono text-xl font-bold text-[var(--color-text-primary)]">
            {overview.total_skus}
          </div>
        </div>

        {/* Total Units On Hand */}
        <div className="rounded-xl bg-emerald-50/50 p-3.5 border border-emerald-200/50">
          <div className="flex items-center gap-1.5 text-emerald-800 text-xs font-medium">
            <Package className="h-3.5 w-3.5 text-emerald-600" aria-hidden="true" />
            <span>{t("inventory.overviewCard.totalUnits")}</span>
          </div>
          <div className="mt-1 font-mono text-xl font-bold text-emerald-700">
            {overview.total_units_on_hand.toLocaleString()} <span className="text-xs font-normal text-emerald-600/80">{t("inventory.unit")}</span>
          </div>
        </div>

        {/* Low Stock SKUs */}
        <div className="rounded-xl bg-amber-50/60 p-3.5 border border-amber-200/50">
          <div className="flex items-center gap-1.5 text-amber-800 text-xs font-medium">
            <AlertTriangle className="h-3.5 w-3.5 text-amber-600" aria-hidden="true" />
            <span>{t("inventory.overviewCard.lowStock")}</span>
          </div>
          <div className="mt-1 font-mono text-xl font-bold text-amber-700">
            {overview.low_stock_skus}
          </div>
        </div>

        {/* Out of Stock SKUs */}
        <div className="rounded-xl bg-rose-50/50 p-3.5 border border-rose-200/50">
          <div className="flex items-center gap-1.5 text-rose-800 text-xs font-medium">
            <AlertCircle className="h-3.5 w-3.5 text-rose-600" aria-hidden="true" />
            <span>{t("inventory.overviewCard.outOfStock")}</span>
          </div>
          <div className="mt-1 font-mono text-xl font-bold text-rose-700">
            {overview.out_of_stock_skus}
          </div>
        </div>
      </div>
    </div>
  );
}
