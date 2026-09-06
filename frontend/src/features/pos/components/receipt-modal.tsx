"use client";

import { CheckCircle2, Printer, PlusCircle } from "lucide-react";
import { formatIDR } from "@/lib/money";
import { formatDateTime } from "@/lib/date";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import type { CheckoutResponse } from "../types";

export interface ReceiptModalProps {
  isOpen: boolean;
  onClose: () => void;
  receipt: CheckoutResponse | null;
  onNewTransaction: () => void;
}

export function ReceiptModal({ isOpen, onClose, receipt, onNewTransaction }: ReceiptModalProps) {
  const { t } = useTranslation();
  if (!receipt) return null;

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={t("receipt.modalTitle")} maxWidth="md">
      <div className="space-y-5">
        <section className="rounded-xl border border-[var(--color-status-success-border)] bg-[var(--color-status-success-bg)] p-4">
          <div className="flex items-center gap-2 text-[var(--color-status-success-text)]">
            <CheckCircle2 className="h-5 w-5 shrink-0" aria-hidden="true" />
            <p className="text-base font-semibold">{t("common.status.completed")}</p>
          </div>
          <p className="mt-2 break-all font-mono text-sm"><bdi>{receipt.transaction_number}</bdi></p>
          <dl className="mt-4 space-y-3">
            <div className="flex flex-wrap justify-between gap-2">
              <dt>{t("receipt.total")}</dt>
              <dd className="break-all font-mono font-semibold"><bdi>{formatIDR(receipt.total_amount)}</bdi></dd>
            </div>
            <div>
              <dt className="text-sm font-semibold">{t("receipt.change")}</dt>
              <dd className="mt-1 break-all font-mono text-2xl font-bold text-[var(--color-status-success-text)]"><bdi>{formatIDR(receipt.change_amount)}</bdi></dd>
            </div>
          </dl>
        </section>

        <div className="flex flex-col gap-3 sm:flex-row">
          <Button type="button" variant="secondary" onClick={() => window.print()}
            className="h-auto min-h-12 flex-1 gap-2 py-3 text-sm">
            <Printer className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span>{t("receipt.printAction")}</span>
          </Button>
          <Button type="button" onClick={onNewTransaction}
            className="h-auto min-h-12 flex-1 gap-2 py-3 text-sm">
            <PlusCircle className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span>{t("receipt.newSaleAction")}</span>
          </Button>
        </div>

        <dl className="space-y-2 text-sm">
          {[
            [t("receipt.date"), formatDateTime(receipt.created_at)],
            [t("receipt.subtotal"), formatIDR(receipt.subtotal_amount)],
            [t("receipt.tax"), formatIDR(receipt.tax_amount)],
            [t("receipt.cash"), formatIDR(receipt.cash_tendered)],
          ].map(([label, value]) => (
            <div key={label} className="flex flex-wrap justify-between gap-2">
              <dt className="text-[var(--color-text-secondary)]">{label}</dt>
              <dd className="break-all font-mono"><bdi>{value}</bdi></dd>
            </div>
          ))}
        </dl>

        <details className="rounded-xl border border-[var(--color-border-hairline)] p-4">
          <summary className="min-h-12 cursor-pointer py-3 text-base font-semibold">{t("receipt.itemHeader")}</summary>
          <ul className="divide-y divide-[var(--color-border-hairline)]">
            {receipt.items.map((item, index) => (
              <li key={item.id || index} className="space-y-2 py-3">
                <p className="break-words [overflow-wrap:anywhere] text-base font-medium">{item.name || item.product_name || item.sku}</p>
                <p className="break-all text-sm"><bdi>{item.sku}</bdi></p>
                <p className="break-all font-mono text-sm"><bdi>{item.quantity} × {formatIDR(item.unit_price)}</bdi></p>
                <p className="break-all font-mono text-base font-semibold"><bdi>{formatIDR(item.subtotal ?? item.subtotal_amount ?? 0)}</bdi></p>
              </li>
            ))}
          </ul>
        </details>
      </div>
    </Modal>
  );
}
