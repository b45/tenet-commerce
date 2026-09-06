"use client";

import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { FeatureGate } from "@/components/auth/feature-gate";
import { useDailySummary } from "../hooks/use-daily-summary";
import { ThermalDailySummary } from "./thermal-daily-summary";
import { formatIDR } from "@/lib/money";
import { useTranslation } from "@/lib/i18n";
import {
  Printer,
  RotateCw,
  TrendingUp,
  CreditCard,
  Banknote,
  CheckCircle2,
  XCircle,
  Calendar,
  Layers,
} from "lucide-react";

export interface DailySummaryModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function DailySummaryModal({ isOpen, onClose }: DailySummaryModalProps) {
  const { t } = useTranslation();
  const { summary, isLoading, error, selectedDate, setSelectedDate, refetch } =
    useDailySummary(undefined, isOpen);

  const handlePrint = () => {
    window.print();
  };

  const marginPercent =
    summary && summary.net_sales > 0
      ? ((summary.gross_profit / summary.net_sales) * 100).toFixed(1)
      : "0.0";

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t("dailySummary.modalTitle")}
      description={t("dailySummary.modalDescription")}
      maxWidth="xl"
    >
      <FeatureGate featureKey="pos.daily_summary">
        <div className="space-y-5">
          {/* Filter Bar: Date Selection & Refresh */}
          <div className="flex flex-wrap items-center justify-between gap-2.5 pb-3 border-b border-[var(--color-border-hairline)]">
            <div className="flex items-center gap-2">
              <Calendar className="h-4 w-4 text-[var(--color-text-secondary)] shrink-0" />
              <label htmlFor="daily-summary-date" className="text-xs font-semibold text-[var(--color-text-secondary)]">
                {t("dailySummary.dateLabel")}:
              </label>
              <input
                id="daily-summary-date"
                type="date"
                value={selectedDate}
                onChange={(e) => setSelectedDate(e.target.value)}
                className="text-xs font-medium px-2.5 py-1.5 rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-action-primary)]"
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => {
                  const now = new Date();
                  const year = now.getFullYear();
                  const month = String(now.getMonth() + 1).padStart(2, "0");
                  const day = String(now.getDate()).padStart(2, "0");
                  setSelectedDate(`${year}-${month}-${day}`);
                }}
                className="text-xs h-8 px-2"
              >
                {t("dailySummary.filterToday")}
              </Button>
            </div>

            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => refetch()}
              isLoading={isLoading}
              className="text-xs h-8 gap-1.5"
            >
              <RotateCw className="h-3.5 w-3.5" />
              <span>{t("dailySummary.actions.refresh")}</span>
            </Button>
          </div>

          {/* Body Content */}
          {error ? (
            <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-center text-sm text-rose-800">
              <p className="font-semibold">{error}</p>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => refetch()}
                className="mt-3 text-xs"
              >
                {t("dailySummary.actions.refresh")}
              </Button>
            </div>
          ) : !summary && isLoading ? (
            <div className="py-12 text-center text-sm text-[var(--color-text-secondary)] animate-pulse">
              {t("dailySummary.loading")}
            </div>
          ) : summary ? (
            <div className="space-y-4">
              {/* Metric Cards Grid */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                {/* Net Sales (Primary) */}
                <div className="rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] p-3.5 shadow-xs">
                  <span className="text-[11px] font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider block">
                    {t("dailySummary.metrics.netSales")}
                  </span>
                  <div className="text-xl font-bold font-mono text-[var(--color-text-primary)] mt-1">
                    {formatIDR(summary.net_sales)}
                  </div>
                  <div className="text-[11px] text-[var(--color-text-secondary)] mt-1">
                    {t("dailySummary.metrics.grossSales")}: {formatIDR(summary.gross_sales)}
                  </div>
                </div>

                {/* Gross Profit & Margin */}
                <div className="rounded-xl border border-emerald-600/20 bg-emerald-50/40 p-3.5 shadow-xs">
                  <div className="flex items-center justify-between">
                    <span className="text-[11px] font-semibold text-emerald-800 uppercase tracking-wider">
                      {t("dailySummary.metrics.grossProfit")}
                    </span>
                    <TrendingUp className="h-3.5 w-3.5 text-emerald-600" />
                  </div>
                  <div className="text-xl font-bold font-mono text-emerald-900 mt-1">
                    {formatIDR(summary.gross_profit)}
                  </div>
                  <div className="flex items-center gap-1 mt-1 text-[11px] font-medium text-emerald-700">
                    <span>{t("dailySummary.metrics.margin")}:</span>
                    <Badge variant="success" className="text-[10px] py-0 px-1.5 h-4">
                      {marginPercent}%
                    </Badge>
                  </div>
                </div>

                {/* COGS & Discounts */}
                <div className="rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] p-3.5 shadow-xs sm:col-span-2 lg:col-span-1">
                  <div className="flex items-center justify-between">
                    <span className="text-[11px] font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider">
                      {t("dailySummary.metrics.cogs")}
                    </span>
                    <Layers className="h-3.5 w-3.5 text-[var(--color-text-tertiary)]" />
                  </div>
                  <div className="text-xl font-bold font-mono text-[var(--color-text-primary)] mt-1">
                    {formatIDR(summary.total_cogs)}
                  </div>
                  <div className="text-[11px] text-amber-700 mt-1">
                    {t("dailySummary.metrics.discounts")}: -{formatIDR(summary.discounts)}
                  </div>
                </div>
              </div>

              {/* Order Volumes Strip */}
              <div className="rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-muted)] p-3 flex flex-wrap items-center justify-around gap-2 text-center">
                <div>
                  <span className="text-[11px] text-[var(--color-text-secondary)] block">
                    {t("dailySummary.orders.total")}
                  </span>
                  <span className="text-base font-bold font-mono text-[var(--color-text-primary)]">
                    {summary.total_orders}
                  </span>
                </div>
                <div className="h-8 w-px bg-[var(--color-border-hairline)]" />
                <div>
                  <span className="text-[11px] text-emerald-700 flex items-center justify-center gap-1">
                    <CheckCircle2 className="h-3 w-3" />
                    {t("dailySummary.orders.completed")}
                  </span>
                  <span className="text-base font-bold font-mono text-emerald-800">
                    {summary.completed_orders}
                  </span>
                </div>
                <div className="h-8 w-px bg-[var(--color-border-hairline)]" />
                <div>
                  <span className="text-[11px] text-rose-700 flex items-center justify-center gap-1">
                    <XCircle className="h-3 w-3" />
                    {t("dailySummary.orders.voided")}
                  </span>
                  <span className="text-base font-bold font-mono text-rose-800">
                    {summary.voided_orders}
                  </span>
                </div>
              </div>

              {/* Payment Methods Breakdown */}
              <div className="rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] overflow-hidden">
                <div className="px-4 py-2.5 border-b border-[var(--color-border-hairline)] bg-[var(--color-surface-muted)]/50">
                  <h3 className="text-xs font-semibold text-[var(--color-text-primary)] uppercase tracking-wider">
                    {t("dailySummary.payments.title")}
                  </h3>
                </div>
                <div className="divide-y divide-[var(--color-border-hairline)] text-xs">
                  {Object.keys(summary.payment_breakdown || {}).length === 0 ? (
                    <div className="p-4 text-center text-[var(--color-text-secondary)]">
                      {t("dailySummary.emptyState")}
                    </div>
                  ) : (
                    Object.entries(summary.payment_breakdown).map(([method, data]) => (
                      <div
                        key={method}
                        className="px-4 py-2.5 flex items-center justify-between"
                      >
                        <div className="flex items-center gap-2">
                          {method === "CASH" ? (
                            <Banknote className="h-4 w-4 text-emerald-600" />
                          ) : (
                            <CreditCard className="h-4 w-4 text-[#0066CC]" />
                          )}
                          <span className="font-semibold text-[var(--color-text-primary)]">
                            {method === "CASH"
                              ? t("dailySummary.payments.cash")
                              : method === "QRIS"
                              ? t("dailySummary.payments.qris")
                              : method}
                          </span>
                          <span className="text-[var(--color-text-secondary)]">
                            ({data.count} {t("dailySummary.orders.total").toLowerCase()})
                          </span>
                        </div>
                        <span className="font-mono font-bold text-[var(--color-text-primary)]">
                          {formatIDR(data.total_amount)}
                        </span>
                      </div>
                    ))
                  )}
                </div>
              </div>

              {/* Actions */}
              <div className="flex flex-col sm:flex-row gap-2.5 pt-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={handlePrint}
                  className="flex-1 gap-2 h-11 text-xs font-medium"
                >
                  <Printer className="h-4 w-4" />
                  <span>{t("dailySummary.actions.print")}</span>
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  onClick={onClose}
                  className="flex-1 h-11 text-xs font-medium"
                >
                  {t("dailySummary.actions.close")}
                </Button>
              </div>

              {/* Hidden Thermal Print Component */}
              <ThermalDailySummary summary={summary} />
            </div>
          ) : (
            <div className="py-8 text-center text-sm text-[var(--color-text-secondary)]">
              {t("dailySummary.emptyState")}
            </div>
          )}
        </div>
      </FeatureGate>
    </Modal>
  );
}
