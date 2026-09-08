"use client";

import * as React from "react";
import { formatIDR } from "@/lib/money";
import { formatDateTime } from "@/lib/date";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import { Eye, AlertCircle } from "lucide-react";
import type { LedgerEntry } from "../types";
import { EntryDetailModal } from "./entry-detail-modal";

interface JournalEntriesTableProps {
  entries: LedgerEntry[];
  isLoading: boolean;
}

export function JournalEntriesTable({ entries, isLoading }: JournalEntriesTableProps) {
  const { t } = useTranslation();
  const [selectedEntry, setSelectedEntry] = React.useState<LedgerEntry | null>(null);

  if (isLoading) {
    return (
      <div className="rounded-[18px] border border-black/[0.06] bg-white p-8 text-center text-xs text-[#8B95A5] animate-pulse">
        {t("common.actions.loading")}
      </div>
    );
  }

  if (!entries || entries.length === 0) {
    return (
      <div className="rounded-[18px] border border-black/[0.06] bg-white p-12 text-center text-xs text-[#8B95A5]">
        <AlertCircle className="h-8 w-8 text-[#8B95A5]/60 mx-auto mb-2" />
        <p>{t("ledger.entries.emptyState")}</p>
      </div>
    );
  }

  return (
    <>
      <div className="rounded-[18px] border border-black/[0.06] bg-white overflow-hidden shadow-[0_1px_3px_rgba(0,0,0,0.02)]">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs border-collapse">
            <thead>
              <tr className="border-b border-black/[0.06] bg-[#F8F8FA] text-[#555D6E]">
                <th className="py-3 px-4 font-semibold">{t("ledger.entries.entryNumber")}</th>
                <th className="py-3 px-4 font-semibold">{t("ledger.entries.date")}</th>
                <th className="py-3 px-4 font-semibold">{t("ledger.entries.sourceDoc")}</th>
                <th className="py-3 px-4 font-semibold">{t("ledger.entries.memo")}</th>
                <th className="py-3 px-4 font-semibold text-right">{t("ledger.entries.debit")}</th>
                <th className="py-3 px-4 font-semibold text-right">{t("ledger.entries.credit")}</th>
                <th className="py-3 px-4 font-semibold text-center">{t("ledger.entries.status")}</th>
                <th className="py-3 px-4 font-semibold text-right">{t("ledger.entries.actions")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-black/[0.04]">
              {entries.map((entry) => {
                return (
                  <tr key={entry.id} className="hover:bg-slate-50/60 transition-colors">
                    <td className="py-3 px-4 font-mono font-medium text-[#0B0F19]">
                      {entry.entry_number}
                    </td>
                    <td className="py-3 px-4 text-[#555D6E]">
                      {formatDateTime(entry.entry_date || entry.created_at)}
                    </td>
                    <td className="py-3 px-4">
                      <Badge variant="default" className="text-[10px] font-mono">
                        {entry.source_document_type}
                      </Badge>
                    </td>
                    <td className="py-3 px-4 text-[#0B0F19] max-w-xs truncate" title={entry.memo}>
                      {entry.memo}
                    </td>
                    <td className="py-3 px-4 font-mono text-right font-medium text-[#0B0F19]">
                      {formatIDR(entry.total_debit)}
                    </td>
                    <td className="py-3 px-4 font-mono text-right font-medium text-[#0B0F19]">
                      {formatIDR(entry.total_credit)}
                    </td>
                    <td className="py-3 px-4 text-center">
                      <Badge variant={entry.status === "POSTED" ? "success" : "warning"} dot className="text-[10px]">
                        {entry.status === "POSTED" ? t("ledger.entries.postedStatus") : t("ledger.entries.reversedStatus")}
                      </Badge>
                    </td>
                    <td className="py-3 px-4 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setSelectedEntry(entry)}
                        className="h-7 px-2.5 text-xs text-[#0066CC] hover:bg-blue-50"
                      >
                        <Eye className="h-3.5 w-3.5 mr-1" />
                        <span>{t("ledger.entries.viewDetail")}</span>
                      </Button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      <EntryDetailModal
        isOpen={!!selectedEntry}
        onClose={() => setSelectedEntry(null)}
        entry={selectedEntry}
      />
    </>
  );
}
