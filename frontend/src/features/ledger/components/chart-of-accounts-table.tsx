"use client";

import * as React from "react";
import { Badge } from "@/components/ui/badge";
import { useTranslation } from "@/lib/i18n";
import { AlertCircle } from "lucide-react";
import type { LedgerAccount } from "../types";

interface ChartOfAccountsTableProps {
  accounts: LedgerAccount[];
  isLoading: boolean;
}

export function ChartOfAccountsTable({ accounts, isLoading }: ChartOfAccountsTableProps) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <div className="rounded-[18px] border border-black/[0.06] bg-white p-8 text-center text-xs text-[#8B95A5] animate-pulse">
        {t("common.actions.loading")}
      </div>
    );
  }

  if (!accounts || accounts.length === 0) {
    return (
      <div className="rounded-[18px] border border-black/[0.06] bg-white p-12 text-center text-xs text-[#8B95A5]">
        <AlertCircle className="h-8 w-8 text-[#8B95A5]/60 mx-auto mb-2" />
        <p>{t("ledger.accounts.emptyState")}</p>
      </div>
    );
  }

  return (
    <div className="rounded-[18px] border border-black/[0.06] bg-white overflow-hidden shadow-[0_1px_3px_rgba(0,0,0,0.02)]">
      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs border-collapse">
          <thead>
            <tr className="border-b border-black/[0.06] bg-[#F8F8FA] text-[#555D6E]">
              <th className="py-3 px-4 font-semibold">{t("ledger.accounts.code")}</th>
              <th className="py-3 px-4 font-semibold">{t("ledger.accounts.name")}</th>
              <th className="py-3 px-4 font-semibold">{t("ledger.accounts.type")}</th>
              <th className="py-3 px-4 font-semibold text-center">{t("ledger.accounts.zakatEligible")}</th>
              <th className="py-3 px-4 font-semibold text-center">{t("ledger.accounts.status")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-black/[0.04]">
            {accounts.map((acc) => (
              <tr key={acc.id} className="hover:bg-slate-50/60 transition-colors">
                <td className="py-3 px-4 font-mono font-medium text-[#0B0F19]">
                  {acc.code}
                </td>
                <td className="py-3 px-4 font-semibold text-[#0B0F19]">
                  {acc.name}
                </td>
                <td className="py-3 px-4">
                  <Badge variant="default" className="text-[10px] font-mono">
                    {acc.account_type}
                  </Badge>
                </td>
                <td className="py-3 px-4 text-center">
                  {acc.is_zakat_eligible ? (
                    <Badge variant="success" className="text-[10px]">
                      Ya (Zakat Mal)
                    </Badge>
                  ) : (
                    <span className="text-[#8B95A5] text-[11px]">-</span>
                  )}
                </td>
                <td className="py-3 px-4 text-center">
                  <Badge variant={acc.is_active ? "success" : "default"} dot className="text-[10px]">
                    {acc.is_active ? t("ledger.accounts.activeStatus") : t("ledger.accounts.inactiveStatus")}
                  </Badge>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
