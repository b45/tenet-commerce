"use client";

import * as React from "react";
import { Trash2, Plus, Minus, ShoppingCart, ArrowRight } from "lucide-react";
import { formatIDR } from "@/lib/money";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { CartItem } from "../types";

export interface CartPanelProps {
  items: CartItem[];
  totals: {
    subtotal: number;
    tax: number;
    discount: number;
    total: number;
    totalItems: number;
  };
  onUpdateQuantity: (sku: string, quantity: number) => void;
  onDecrementItem: (sku: string) => void;
  onRemoveItem: (sku: string) => void;
  onClearCart: () => void;
  onOpenTender: () => void;
}

export function CartPanel({
  items,
  totals,
  onUpdateQuantity,
  onDecrementItem,
  onRemoveItem,
  onClearCart,
  onOpenTender,
}: CartPanelProps) {
  const isCartEmpty = items.length === 0;

  return (
    <div className="flex flex-col h-full bg-[var(--color-surface-base)] rounded-[22px] border border-[var(--color-border-hairline)] shadow-[var(--shadow-card)] overflow-hidden">
      {/* Panel Header */}
      <div className="p-4 sm:p-5 border-b border-[var(--color-border-hairline)] flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-full bg-[var(--color-surface-muted)] flex items-center justify-center text-[var(--color-action-primary)]">
            <ShoppingCart className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">
              Keranjang Kasir
            </h3>
            <span className="text-[11px] text-[var(--color-text-muted)]">
              {totals.totalItems} item dipilih
            </span>
          </div>
        </div>

        {!isCartEmpty && (
          <button
            type="button"
            onClick={onClearCart}
            className="text-xs text-[var(--color-status-danger-text)] hover:underline font-medium"
          >
            Kosongkan
          </button>
        )}
      </div>

      {/* Items Scrollable List */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {isCartEmpty ? (
          <div className="h-full min-h-[220px] flex flex-col items-center justify-center text-center p-6 select-none">
            <div className="w-12 h-12 rounded-full bg-[var(--color-surface-muted)] flex items-center justify-center text-[var(--color-text-muted)] mb-3">
              <ShoppingCart className="w-5 h-5" />
            </div>
            <p className="text-sm font-medium text-[var(--color-text-secondary)]">
              Belum ada produk di keranjang
            </p>
            <p className="text-xs text-[var(--color-text-muted)] mt-1 max-w-[200px]">
              Klik kartu produk atau scan barcode untuk menambahkan.
            </p>
          </div>
        ) : (
          items.map((item) => {
            const isMaxStock = item.quantity >= item.product.stock_quantity;

            return (
              <div
                key={item.product.sku}
                className="p-3.5 rounded-[16px] bg-[var(--color-surface-muted)]/70 border border-[var(--color-border-hairline)] flex flex-col justify-between space-y-2.5 transition-all"
              >
                {/* Item Top: Name, SKU, and Delete */}
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1 min-w-0">
                    <h5 className="text-xs font-semibold text-[var(--color-text-primary)] truncate">
                      {item.product.name}
                    </h5>
                    <p className="text-[10px] font-mono text-[var(--color-text-muted)] mt-0.5">
                      {item.product.sku} · {formatIDR(item.product.unit_price)}
                    </p>
                  </div>

                  <button
                    type="button"
                    onClick={() => onRemoveItem(item.product.sku)}
                    aria-label={`Hapus ${item.product.name}`}
                    className="text-[var(--color-text-muted)] hover:text-[var(--color-status-danger-text)] transition-colors p-1"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>

                {/* Item Bottom: Quantity Control & Line Subtotal */}
                <div className="flex items-center justify-between pt-1">
                  {/* Quantity Stepper */}
                  <div className="flex items-center bg-[var(--color-surface-base)] rounded-full border border-[var(--color-border-hairline)] shadow-xs p-0.5">
                    <button
                      type="button"
                      onClick={() => onDecrementItem(item.product.sku)}
                      aria-label="Kurangi jumlah"
                      className="w-6 h-6 rounded-full flex items-center justify-center text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)] active:scale-95 transition-all"
                    >
                      <Minus className="w-3 h-3" />
                    </button>

                    <span className="w-8 text-center text-xs font-mono font-semibold text-[var(--color-text-primary)]">
                      {item.quantity}
                    </span>

                    <button
                      type="button"
                      disabled={isMaxStock}
                      onClick={() => onUpdateQuantity(item.product.sku, item.quantity + 1)}
                      aria-label="Tambah jumlah"
                      className={cn(
                        "w-6 h-6 rounded-full flex items-center justify-center text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)] active:scale-95 transition-all",
                        isMaxStock && "opacity-40 cursor-not-allowed hover:bg-transparent"
                      )}
                    >
                      <Plus className="w-3 h-3" />
                    </button>
                  </div>

                  {/* Line Subtotal */}
                  <span className="text-xs font-mono font-bold text-[var(--color-text-primary)] tracking-tight">
                    {formatIDR(item.subtotal)}
                  </span>
                </div>
              </div>
            );
          })
        )}
      </div>

      {/* Cart Summary & Checkout Action */}
      <div className="p-4 sm:p-5 border-t border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] space-y-3">
        <div className="space-y-1.5 text-xs text-[var(--color-text-secondary)]">
          <div className="flex justify-between">
            <span>Subtotal</span>
            <span className="font-mono text-[var(--color-text-primary)]">
              {formatIDR(totals.subtotal)}
            </span>
          </div>

          <div className="flex justify-between">
            <span>Pajak (0%)</span>
            <span className="font-mono text-[var(--color-text-primary)]">
              {formatIDR(totals.tax)}
            </span>
          </div>

          <div className="flex justify-between">
            <span>Diskon</span>
            <span className="font-mono text-[var(--color-text-primary)]">
              {formatIDR(totals.discount)}
            </span>
          </div>
        </div>

        {/* Grand Total */}
        <div className="pt-2 border-t border-[var(--color-border-hairline)] flex items-baseline justify-between">
          <span className="text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider">
            Total Bayar
          </span>
          <span className="text-xl font-bold font-mono text-[var(--color-text-primary)] tracking-tight">
            {formatIDR(totals.total)}
          </span>
        </div>

        {/* Primary Checkout CTA */}
        <Button
          type="button"
          disabled={isCartEmpty}
          onClick={onOpenTender}
          className="w-full h-12 text-sm font-semibold rounded-[14px] shadow-sm flex items-center justify-center gap-2"
        >
          <span>Lanjut Pembayaran</span>
          <ArrowRight className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
}
