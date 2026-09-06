"use client";

import * as React from "react";
import { formatIDR } from "@/lib/money";
import { formatDateTime } from "@/lib/date";
import { useTranslation } from "@/lib/i18n";
import type { DailySummaryResponse } from "../types";

export interface ThermalDailySummaryProps {
  summary: DailySummaryResponse;
  merchantName?: string;
  merchantAddress?: string;
}

export function ThermalDailySummary({
  summary,
  merchantName = "AL-BARAKAH MART",
  merchantAddress = "Jl. Terusan Buah Batu No. 45, Bandung",
}: ThermalDailySummaryProps) {
  const { t } = useTranslation();
  const marginPercent =
    summary.net_sales > 0
      ? ((summary.gross_profit / summary.net_sales) * 100).toFixed(1)
      : "0.0";

  return (
    <div
      id="thermal-daily-summary-print"
      className="print-only hidden font-mono text-[11px] leading-tight text-black bg-white p-4 max-w-[80mm] mx-auto select-none"
    >
      {/* Merchant Header */}
      <div className="text-center pb-2 border-b border-dashed border-black space-y-0.5">
        <h2 className="text-sm font-bold tracking-wider">{merchantName}</h2>
        <p className="text-[10px] text-gray-700">{merchantAddress}</p>
        <p className="text-[11px] font-bold mt-1 uppercase">
          {t("dailySummary.thermalHeader")}
        </p>
      </div>

      {/* Report Meta */}
      <div className="py-2 border-b border-dashed border-black space-y-1 text-[10px]">
        <div className="flex justify-between">
          <span>{t("dailySummary.dateLabel")}:</span>
          <span className="font-bold">{summary.date}</span>
        </div>
        <div className="flex justify-between">
          <span>{t("dailySummary.printTime")}:</span>
          <span>{formatDateTime(new Date().toISOString())}</span>
        </div>
        {summary.cashier_id && (
          <div className="flex justify-between">
            <span>{t("receipt.cashier")}:</span>
            <span>{summary.cashier_id.substring(0, 8)}...</span>
          </div>
        )}
      </div>

      {/* Transaction Volumes */}
      <div className="py-2 border-b border-dashed border-black space-y-1 text-[10px]">
        <div className="font-bold pb-0.5">{t("dailySummary.orders.title")}:</div>
        <div className="flex justify-between">
          <span>{t("dailySummary.orders.total")}:</span>
          <span className="font-bold">{summary.total_orders}</span>
        </div>
        <div className="flex justify-between">
          <span>{t("dailySummary.orders.completed")}:</span>
          <span className="font-semibold">{summary.completed_orders}</span>
        </div>
        <div className="flex justify-between">
          <span>{t("dailySummary.orders.voided")}:</span>
          <span>{summary.voided_orders}</span>
        </div>
      </div>

      {/* Financial Aggregates */}
      <div className="py-2 border-b border-dashed border-black space-y-1 text-[10px]">
        <div className="flex justify-between">
          <span>{t("dailySummary.metrics.grossSales")}:</span>
          <span>{formatIDR(summary.gross_sales)}</span>
        </div>
        <div className="flex justify-between">
          <span>{t("dailySummary.metrics.discounts")}:</span>
          <span>-{formatIDR(summary.discounts)}</span>
        </div>
        <div className="flex justify-between text-xs font-bold pt-1 border-t border-black">
          <span>{t("dailySummary.metrics.netSales")}:</span>
          <span>{formatIDR(summary.net_sales)}</span>
        </div>
        <div className="flex justify-between pt-0.5">
          <span>{t("dailySummary.metrics.cogs")}:</span>
          <span>{formatIDR(summary.total_cogs)}</span>
        </div>
        <div className="flex justify-between font-bold pt-1 border-t border-dotted border-black">
          <span>{t("dailySummary.metrics.grossProfit")}:</span>
          <span>{formatIDR(summary.gross_profit)} ({marginPercent}%)</span>
        </div>
      </div>

      {/* Payment Method Breakdown */}
      <div className="py-2 border-b border-dashed border-black space-y-1 text-[10px]">
        <div className="font-bold pb-0.5">{t("dailySummary.payments.title")}:</div>
        {Object.entries(summary.payment_breakdown || {}).map(([method, data]) => (
          <div key={method} className="flex justify-between">
            <span>
              {method === "CASH" ? t("dailySummary.payments.cash") : method === "QRIS" ? t("dailySummary.payments.qris") : method} ({data.count}x):
            </span>
            <span className="font-semibold">{formatIDR(data.total_amount)}</span>
          </div>
        ))}
      </div>

      {/* Thermal Footer */}
      <div className="pt-3 text-center text-[9px] text-gray-600 space-y-1">
        <p className="font-semibold text-black">
          {t("dailySummary.thermalFooter")}
        </p>
        <p className="font-mono font-bold text-black">{t("dailySummary.endOfReport")}</p>
      </div>
    </div>
  );
}
