"use client";

import * as React from "react";
import { CheckCircle2, Printer, PlusCircle } from "lucide-react";
import { formatIDR } from "@/lib/money";
import { formatDateTime } from "@/lib/date";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import type { CheckoutResponse } from "../types";

export interface ReceiptModalProps {
  isOpen: boolean;
  onClose: () => void;
  receipt: CheckoutResponse | null;
  onNewTransaction: () => void;
}

export function ReceiptModal({
  isOpen,
  onClose,
  receipt,
  onNewTransaction,
}: ReceiptModalProps) {
  if (!receipt) return null;

  const handlePrint = () => {
    window.print();
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      maxWidth="md"
      hideCloseButton
      className="p-0 overflow-hidden"
    >
      {/* Success Top Banner */}
      <div className="bg-[var(--color-status-success-bg)] p-6 text-center border-b border-[var(--color-status-success-border)]">
        <div className="w-12 h-12 rounded-full bg-emerald-100 text-[var(--color-status-success-text)] flex items-center justify-center mx-auto mb-2.5">
          <CheckCircle2 className="w-7 h-7" />
        </div>
        <h3 className="text-lg font-bold text-[var(--color-status-success-text)]">
          Pembayaran Berhasil
        </h3>
        <p className="text-xs text-[var(--color-text-secondary)] mt-0.5 font-mono">
          {receipt.transaction_number}
        </p>
      </div>

      {/* Receipt Details Body */}
      <div className="p-6 space-y-4 max-h-[55vh] overflow-y-auto">
        {/* Transaction Summary Card */}
        <div className="p-4 rounded-[16px] bg-[var(--color-surface-muted)] border border-[var(--color-border-hairline)] space-y-2 text-xs">
          <div className="flex justify-between">
            <span className="text-[var(--color-text-secondary)]">Waktu Transaksi</span>
            <span className="font-mono text-[var(--color-text-primary)]">
              {formatDateTime(receipt.created_at)}
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-[var(--color-text-secondary)]">Status</span>
            <span className="font-semibold text-[var(--color-status-success-text)]">
              SELESAI (COMPLETED)
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-[var(--color-text-secondary)]">Metode</span>
            <span className="font-medium text-[var(--color-text-primary)]">TUNAI</span>
          </div>
        </div>

        {/* Itemized Breakdown */}
        <div className="space-y-2">
          <h5 className="text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider">
            Rincian Produk
          </h5>
          <div className="divide-y divide-[var(--color-border-hairline)] border-y border-[var(--color-border-hairline)] py-1">
            {receipt.items.map((item, idx) => (
              <div key={idx} className="py-2 flex items-center justify-between text-xs">
                <div className="min-w-0 pr-2">
                  <p className="font-medium text-[var(--color-text-primary)] truncate">
                    {item.name || item.product_name || "Produk"}
                  </p>
                  <p className="text-[10px] font-mono text-[var(--color-text-muted)]">
                    {item.quantity} x {formatIDR(item.unit_price)}
                  </p>
                </div>
                <span className="font-mono font-semibold text-[var(--color-text-primary)]">
                  {formatIDR(item.subtotal ?? item.subtotal_amount ?? 0)}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Financial Numbers */}
        <div className="space-y-1.5 text-xs text-[var(--color-text-secondary)]">
          <div className="flex justify-between">
            <span>Subtotal</span>
            <span className="font-mono">{formatIDR(receipt.subtotal_amount)}</span>
          </div>
          <div className="flex justify-between">
            <span>Pajak (0%)</span>
            <span className="font-mono">{formatIDR(receipt.tax_amount)}</span>
          </div>
          <div className="flex justify-between text-sm font-bold text-[var(--color-text-primary)] pt-2 border-t border-[var(--color-border-hairline)]">
            <span>Total Transaksi</span>
            <span className="font-mono text-base">{formatIDR(receipt.total_amount)}</span>
          </div>
          <div className="flex justify-between pt-1">
            <span>Tunai Diterima</span>
            <span className="font-mono font-semibold text-[var(--color-text-primary)]">
              {formatIDR(receipt.cash_tendered)}
            </span>
          </div>
          <div className="flex justify-between font-semibold text-[var(--color-status-success-text)]">
            <span>Kembalian</span>
            <span className="font-mono text-sm">{formatIDR(receipt.change_amount)}</span>
          </div>
        </div>
      </div>

      {/* Modal Actions */}
      <div className="p-5 border-t border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] flex gap-3">
        <Button
          type="button"
          variant="secondary"
          onClick={handlePrint}
          className="flex-1 rounded-[14px] h-12 text-xs font-semibold flex items-center justify-center gap-2"
        >
          <Printer className="w-4 h-4" />
          <span>Cetak Struk</span>
        </Button>

        <Button
          type="button"
          onClick={onNewTransaction}
          className="flex-1 rounded-[14px] h-12 text-xs font-semibold flex items-center justify-center gap-2"
        >
          <PlusCircle className="w-4 h-4" />
          <span>Transaksi Baru</span>
        </Button>
      </div>
    </Modal>
  );
}
