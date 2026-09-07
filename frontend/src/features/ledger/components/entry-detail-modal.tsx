"use client";

import * as React from "react";
import Link from "next/link";
import { formatIDR } from "@/lib/money";
import { formatDateTime } from "@/lib/date";
import { Modal } from "@/components/ui/modal";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import { ExternalLink, CheckCircle2 } from "lucide-react";
import type { LedgerEntry } from "../types";

export interface EntryDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  entry: LedgerEntry | null;
}

export function EntryDetailModal({ isOpen, onClose, entry }: EntryDetailModalProps) {
  const { t } = useTranslation();

  if (!entry) return null;

  const getSourceHref = (docType: string, docId?: string) => {
    if (!docId) return null;
    switch (docType) {
      case "POS_SALE":
      case "POS_VOID":
        return `/pos/orders`;
      case "GOODS_RECEIPT":
        return `/supply-chain/po`;
      default:
        return null;
    }
  };

  const sourceHref = getSourceHref(entry.source_document_type, entry.source_document_id);
  const isBalanced = Math.abs(entry.total_debit - entry.total_credit) < 0.01;

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={t("ledger.detail.title")} maxWidth="lg">
      <div className="space-y-5">
        {/* Header Summary Card */}
        <div className="rounded-xl border border-black/[0.06] bg-[#F8F8FA] p-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-3 border-b border-black/[0.04]">
            <div>
              <span className="text-[11px] font-medium uppercase tracking-wider text-[#555D6E]">
                {t("ledger.entries.entryNumber")}
              </span>
              <h3 className="text-base font-bold font-mono text-[#0B0F19]">
                {entry.entry_number}
              </h3>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant={entry.status === "POSTED" ? "success" : "warning"} dot>
                {entry.status === "POSTED" ? t("ledger.entries.postedStatus") : t("ledger.entries.reversedStatus")}
              </Badge>
              {isBalanced ? (
                <Badge variant="outline" className="text-emerald-700 bg-emerald-50/60 border-emerald-200">
                  <CheckCircle2 className="h-3 w-3 mr-1 inline" />
                  {t("ledger.entries.balanced")}
                </Badge>
              ) : (
                <Badge variant="danger">
                  {t("ledger.entries.unbalanced")}
                </Badge>
              )}
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-3 text-xs">
            <div>
              <span className="text-[#555D6E]">{t("ledger.entries.date")}:</span>{" "}
              <span className="font-medium text-[#0B0F19]">{formatDateTime(entry.entry_date || entry.created_at)}</span>
            </div>
            <div>
              <span className="text-[#555D6E]">{t("ledger.entries.sourceDoc")}:</span>{" "}
              <span className="font-mono font-medium text-[#0B0F19]">{entry.source_document_type}</span>
            </div>
            <div className="sm:col-span-2">
              <span className="text-[#555D6E]">{t("ledger.entries.memo")}:</span>{" "}
              <span className="text-[#0B0F19]">{entry.memo}</span>
            </div>
            {entry.reversed_by_entry_id && (
              <div className="sm:col-span-2 text-amber-800 bg-amber-50 p-2 rounded-lg text-[11px]">
                {t("ledger.detail.reversedNotice")}
              </div>
            )}
          </div>
        </div>

        {/* Journal Lines Table */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wider text-[#555D6E] mb-2">
            {t("ledger.detail.linesTitle")}
          </h4>
          <div className="rounded-xl border border-black/[0.06] overflow-hidden">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="border-b border-black/[0.06] bg-[#F5F5F7] text-[#555D6E]">
                  <th className="py-2.5 px-3 font-semibold">{t("ledger.detail.accountCode")}</th>
                  <th className="py-2.5 px-3 font-semibold">{t("ledger.detail.accountName")}</th>
                  <th className="py-2.5 px-3 font-semibold text-right">{t("ledger.detail.debit")}</th>
                  <th className="py-2.5 px-3 font-semibold text-right">{t("ledger.detail.credit")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-black/[0.04] bg-white">
                {entry.lines && entry.lines.length > 0 ? (
                  entry.lines.map((line) => (
                    <tr key={line.id} className="hover:bg-slate-50/50">
                      <td className="py-2.5 px-3 font-mono font-medium text-[#0B0F19]">{line.account_code || "-"}</td>
                      <td className="py-2.5 px-3 text-[#0B0F19]">{line.account_name || "-"}</td>
                      <td className="py-2.5 px-3 font-mono text-right text-[#0B0F19]">
                        {line.debit_amount > 0 ? formatIDR(line.debit_amount) : "-"}
                      </td>
                      <td className="py-2.5 px-3 font-mono text-right text-[#0B0F19]">
                        {line.credit_amount > 0 ? formatIDR(line.credit_amount) : "-"}
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={4} className="py-4 text-center text-[#8B95A5]">
                      -
                    </td>
                  </tr>
                )}
              </tbody>
              <tfoot>
                <tr className="border-t border-black/[0.08] bg-[#F8F8FA] font-bold">
                  <td colSpan={2} className="py-2.5 px-3 text-right text-[#0B0F19]">
                    {t("ledger.detail.total")}:
                  </td>
                  <td className="py-2.5 px-3 font-mono text-right text-emerald-700">
                    {formatIDR(entry.total_debit)}
                  </td>
                  <td className="py-2.5 px-3 font-mono text-right text-emerald-700">
                    {formatIDR(entry.total_credit)}
                  </td>
                </tr>
              </tfoot>
            </table>
          </div>
        </div>

        {/* Footer Actions */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-3 pt-2">
          {sourceHref ? (
            <Link
              href={sourceHref}
              className="inline-flex items-center gap-1.5 text-xs font-medium text-[#0066CC] hover:underline"
            >
              <span>{t("ledger.detail.viewSource")}</span>
              <ExternalLink className="h-3.5 w-3.5" />
            </Link>
          ) : (
            <div />
          )}
          <Button variant="secondary" size="sm" onClick={onClose} className="w-full sm:w-auto">
            {t("ledger.detail.close")}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
