"use client";

import * as React from "react";
import { formatIDR } from "@/lib/money";
import { formatDateTime } from "@/lib/date";
import type { CheckoutResponse } from "../types";

export interface ThermalReceiptProps {
  receipt: CheckoutResponse;
  merchantName?: string;
  merchantAddress?: string;
}

export function ThermalReceipt({
  receipt,
  merchantName = "AL-BARAKAH MART",
  merchantAddress = "Jl. Terusan Buah Batu No. 45, Bandung",
}: ThermalReceiptProps) {
  return (
    <div
      id="thermal-receipt-print"
      className="print-only hidden font-mono text-[11px] leading-tight text-black bg-white p-4 max-w-[80mm] mx-auto select-none"
    >
      {/* Merchant Header */}
      <div className="text-center pb-2 border-b border-dashed border-black space-y-0.5">
        <h2 className="text-sm font-bold tracking-wider">{merchantName}</h2>
        <p className="text-[10px] text-gray-700">{merchantAddress}</p>
        <p className="text-[9px] text-gray-500">NPWP: 01.234.567.8-429.000</p>
      </div>

      {/* Transaction Meta */}
      <div className="py-2 border-b border-dashed border-black space-y-1 text-[10px]">
        <div className="flex justify-between">
          <span>No. Bukti:</span>
          <span className="font-bold">{receipt.transaction_number}</span>
        </div>
        <div className="flex justify-between">
          <span>Waktu:</span>
          <span>{formatDateTime(receipt.created_at)}</span>
        </div>
        <div className="flex justify-between">
          <span>Kasir ID:</span>
          <span>{receipt.cashier_id.substring(0, 8)}...</span>
        </div>
        <div className="flex justify-between">
          <span>Metode:</span>
          <span>TUNAI (CASH)</span>
        </div>
      </div>

      {/* Items List */}
      <div className="py-2 border-b border-dashed border-black space-y-1.5">
        {receipt.items.map((item, idx) => (
          <div key={idx} className="space-y-0.5">
            <div className="font-semibold truncate">{item.name || item.product_name || "Produk"}</div>
            <div className="flex justify-between text-[10px]">
              <span>
                {item.quantity} x {formatIDR(item.unit_price)}
              </span>
              <span className="font-bold">{formatIDR(item.subtotal ?? item.subtotal_amount ?? 0)}</span>
            </div>
          </div>
        ))}
      </div>

      {/* Financial Totals */}
      <div className="py-2 border-b border-dashed border-black space-y-1 text-[10px]">
        <div className="flex justify-between">
          <span>Subtotal:</span>
          <span>{formatIDR(receipt.subtotal_amount)}</span>
        </div>
        <div className="flex justify-between">
          <span>Pajak (0%):</span>
          <span>{formatIDR(receipt.tax_amount)}</span>
        </div>
        <div className="flex justify-between">
          <span>Diskon:</span>
          <span>{formatIDR(receipt.discount_amount)}</span>
        </div>
        <div className="flex justify-between text-xs font-bold pt-1 border-t border-black">
          <span>TOTAL:</span>
          <span>{formatIDR(receipt.total_amount)}</span>
        </div>
        <div className="flex justify-between pt-0.5">
          <span>Tunai Diterima:</span>
          <span>{formatIDR(receipt.cash_tendered)}</span>
        </div>
        <div className="flex justify-between font-bold">
          <span>Kembalian:</span>
          <span>{formatIDR(receipt.change_amount)}</span>
        </div>
      </div>

      {/* Sharia Compliance & Footer */}
      <div className="pt-3 text-center text-[9px] text-gray-600 space-y-1">
        <p className="font-semibold text-black">Jazakumullahu Khairan Katsiran</p>
        <p>Barang yang sudah dibeli dapat ditukar dalam 1x24 jam dengan struk asli.</p>
        <p className="text-[8px] text-gray-400">Tenet Commerce · POS Engine</p>
      </div>
    </div>
  );
}
