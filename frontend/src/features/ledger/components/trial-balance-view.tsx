"use client";

import * as React from "react";
import { formatIDR } from "@/lib/money";
import { Badge } from "@/components/ui/badge";
import { useTranslation } from "@/lib/i18n";
import { CheckCircle2, AlertTriangle, AlertCircle } from "lucide-react";
import type { TrialBalanceSummary } from "../types";

interface TrialBalanceViewProps {
  trialBalance: TrialBalanceSummary | null;
  isLoading: boolean;
}

export function TrialBalanceView({ trialBalance, isLoading }: TrialBalanceViewProps) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <div className="rounded-[18px] border border-black/[0.06] bg-white p-8 text-center text-xs text-[#8B95A5] animate-pulse">
        {t("common.actions.loading")}
      </div>
    );
  }

  if (!trialBalance || !trialBalance.rows || trialBalance.rows.length === 0) {
    return (
      <div className="rounded-[18px] border border-black/[0.06] bg-white p-12 text-center text-xs text-[#8B95A5]">
        <AlertCircle className="h-8 w-8 text-[#8B95A5]/60 mx-auto mb-2" />
        <p>{t("ledger.trialBalance.emptyState")}</p>
      </div>
    );
  }

  const isBalanced = trialBalance.is_balanced && Math.abs(trialBalance.total_debits - trialBalance.total_credits) < 0.01;

  return (
    <div className="space-y-4">
      {/* Integrity Status Header */}
      <div className="rounded-[18px] border border-black/[0.06] bg-white p-4 sm:p-5 flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-[0_1px_3px_rgba(0,0,0,0.02)]">
        <div>
          <h3 className="text-sm font-bold text-[#0B0F19]">
            {t("ledger.trialBalance.title")}
          </h3>
          <p className="text-xs text-[#555D6E] mt-0.5">
            {t("ledger.trialBalance.asOfDate")}: <span className="font-mono font-medium">{trialBalance.as_of_date}</span>
          </p>
        </div>

        <div>
          {isBalanced ? (
            <Badge variant="success" dot className="text-xs py-1 px-3">
              <CheckCircle2 className="h-3.5 w-3.5 mr-1.5 inline" />
              <span>{t("ledger.trialBalance.isBalanced")}</span>
            </Badge>
          ) : (
            <Badge variant="danger" dot className="text-xs py-1 px-3">
              <AlertTriangle className="h-3.5 w-3.5 mr-1.5 inline" />
              <span>{t("ledger.trialBalance.needsReconciliation")}</span>
            </Badge>
          )}
        </div>
      </div>

      {/* Trial Balance Table */}
      <div className="rounded-[18px] border border-black/[0.06] bg-white overflow-hidden shadow-[0_1px_3px_rgba(0,0,0,0.02)]">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs border-collapse">
            <thead>
              <tr className="border-b border-black/[0.06] bg-[#F8F8FA] text-[#555D6E]">
                <th className="py-3 px-4 font-semibold">{t("ledger.trialBalance.accountCode")}</th>
                <th className="py-3 px-4 font-semibold">{t("ledger.trialBalance.accountName")}</th>
                <th className="py-3 px-4 font-semibold">{t("ledger.trialBalance.accountType")}</th>
                <th className="py-3 px-4 font-semibold text-right">{t("ledger.trialBalance.totalDebit")}</th>
                <th className="py-3 px-4 font-semibold text-right">{t("ledger.trialBalance.totalCredit")}</th>
                <th className="py-3 px-4 font-semibold text-right">{t("ledger.trialBalance.netBalance")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-black/[0.04]">
              {trialBalance.rows.map((row) => (
                <tr key={row.account_code} className="hover:bg-slate-50/60 transition-colors">
                  <td className="py-3 px-4 font-mono font-medium text-[#0B0F19]">
                    {row.account_code}
                  </td>
                  <td className="py-3 px-4 font-medium text-[#0B0F19]">
                    {row.account_name}
                  </td>
                  <td className="py-3 px-4">
                    <Badge variant="default" className="text-[10px] font-mono">
                      {row.account_type}
                    </Badge>
                  </td>
                  <td className="py-3 px-4 font-mono text-right text-[#0B0F19]">
                    {formatIDR(row.total_debit)}
                  </td>
                  <td className="py-3 px-4 font-mono text-right text-[#0B0F19]">
                    {formatIDR(row.total_credit)}
                  </td>
                  <td className="py-3 px-4 font-mono text-right font-semibold text-[#0B0F19]">
                    {formatIDR(row.balance)}
                  </td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr className="border-t-2 border-black/[0.12] bg-[#F8F8FA] font-bold">
                <td colSpan={3} className="py-3.5 px-4 text-right text-[#0B0F19] uppercase tracking-wider text-xs">
                  {t("ledger.trialBalance.total")}:
                </td>
                <td className="py-3.5 px-4 font-mono text-right text-base text-emerald-700">
                  {formatIDR(trialBalance.total_debits)}
                </td>
                <td className="py-3.5 px-4 font-mono text-right text-base text-emerald-700">
                  {formatIDR(trialBalance.total_credits)}
                </td>
                <td className="py-3.5 px-4 font-mono text-right text-xs text-[#555D6E]">
                  Δ {formatIDR(Math.abs(trialBalance.total_debits - trialBalance.total_credits))}
                </td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    </div>
  );
}
