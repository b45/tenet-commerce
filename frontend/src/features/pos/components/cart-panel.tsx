"use client";

import * as React from "react";
import { Trash2, Plus, Minus, ShoppingCart, ArrowRight } from "lucide-react";
import { formatIDR } from "@/lib/money";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useTranslation } from "@/lib/i18n";
import type { CartItem, CartTotals } from "../types";

export interface CartPanelProps {
  items: CartItem[];
  totals: CartTotals;
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
  const { t } = useTranslation();
  const isCartEmpty = items.length === 0;

  return (
    <div className="flex flex-col bg-[var(--color-surface-base)] rounded-[22px] border border-[var(--color-border-subtle)] shadow-xs overflow-hidden">
      {/* Panel Header */}
      <div className="p-4 sm:p-5 border-b border-[var(--color-border-hairline)] flex flex-wrap gap-2 items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-full bg-[var(--color-surface-muted)] flex items-center justify-center text-[var(--color-action-primary)]">
            <ShoppingCart className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">
              {t("pos.cart.title")}
            </h3>
            <span className="text-[11px] text-[var(--color-text-muted)]">
              {totals.totalItems} {t("pos.cart.totalItems")}
            </span>
          </div>
        </div>

        {!isCartEmpty && (
          <button
            type="button"
            onClick={onClearCart}
            className="min-h-12 px-2 text-sm text-[var(--color-status-danger-text)] hover:underline font-medium focus-visible:outline focus-visible:outline-2"
          >
            {t("pos.cart.clear")}
          </button>
        )}
      </div>

      {/* Items Scrollable List */}
      <div className="p-4 space-y-3">
        {isCartEmpty ? (
          <div className="h-full min-h-[220px] flex flex-col items-center justify-center text-center p-6 select-none">
            <div className="w-12 h-12 rounded-full bg-[var(--color-surface-muted)] flex items-center justify-center text-[var(--color-text-muted)] mb-3">
              <ShoppingCart className="w-5 h-5" />
            </div>
            <p className="text-sm font-medium text-[var(--color-text-secondary)]">
              {t("pos.cart.emptyTitle")}
            </p>
            <p className="text-xs text-[var(--color-text-muted)] mt-1 max-w-[200px]">
              {t("pos.cart.emptyDescription")}
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
                    <h5 className="break-words [overflow-wrap:anywhere] text-base font-semibold text-[var(--color-text-primary)]">
                      {item.product.name}
                    </h5>
                    <p className="break-all text-sm font-mono text-[var(--color-text-secondary)] mt-1">
                      <bdi>{item.product.sku}</bdi><br /><bdi>{formatIDR(item.product.unit_price)}</bdi>
                    </p>
                  </div>

                  <button
                    type="button"
                    onClick={() => onRemoveItem(item.product.sku)}
                    aria-label={t("pos.cart.removeProduct", { name: item.product.name })}
                    className="text-[var(--color-text-muted)] hover:text-[var(--color-status-danger-text)] flex h-12 w-12 shrink-0 items-center justify-center rounded-xl focus-visible:outline focus-visible:outline-2"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>

                {/* Item Bottom: Quantity Control & Line Subtotal */}
                <div className="flex flex-wrap items-center justify-between gap-3 pt-1">
                  {/* Quantity Stepper */}
                  <div className="flex max-w-full flex-wrap gap-2 items-center bg-[var(--color-surface-base)] rounded-full border border-[var(--color-border-hairline)] shadow-xs p-0.5">
                    <button
                      type="button"
                      disabled={item.quantity <= 1}
                      onClick={() => { if (item.quantity > 1) onDecrementItem(item.product.sku); }}
                      aria-label={t("pos.cart.decreaseProduct", { name: item.product.name })}
                      className="h-12 w-12 shrink-0 rounded-full flex items-center justify-center text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)] disabled:opacity-40 focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-action-primary)]"
                    >
                      <Minus className="w-3 h-3" />
                    </button>

                    <span className="min-w-8 text-center text-base font-mono font-semibold text-[var(--color-text-primary)]">
                      {item.quantity}
                    </span>

                    <button
                      type="button"
                      disabled={isMaxStock}
                      onClick={() => onUpdateQuantity(item.product.sku, item.quantity + 1)}
                      aria-label={t("pos.cart.increaseProduct", { name: item.product.name })}
                      className={cn(
                        "h-12 w-12 shrink-0 rounded-full flex items-center justify-center text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)] disabled:opacity-40 focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-action-primary)]",
                        isMaxStock && "opacity-40 cursor-not-allowed hover:bg-transparent"
                      )}
                    >
                      <Plus className="w-3 h-3" />
                    </button>
                  </div>

                  {/* Line Subtotal */}
                  <span className="break-all text-base font-mono font-bold text-[var(--color-text-primary)] tracking-tight">
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
        <div className="space-y-1.5 text-sm text-[var(--color-text-secondary)]">
          <div className="flex flex-wrap justify-between gap-2">
            <span>{t("pos.cart.subtotal")}</span>
            <span className="break-all font-mono text-[var(--color-text-primary)]">
              {formatIDR(totals.subtotal)}
            </span>
          </div>

          <div className="flex flex-wrap justify-between gap-2">
            <span>{t("pos.cart.tax")}</span>
            <span className="break-all font-mono text-[var(--color-text-primary)]">
              {formatIDR(totals.tax)}
            </span>
          </div>
        </div>

        {/* Grand Total */}
        <div className="pt-2 border-t border-[var(--color-border-hairline)] flex flex-wrap gap-2 items-baseline justify-between">
          <span className="text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider">
            {t("pos.cart.total")}
          </span>
          <span className="break-all text-xl font-bold font-mono text-[var(--color-text-primary)] tracking-tight">
            {formatIDR(totals.total)}
          </span>
        </div>

        {/* Primary Checkout CTA */}
        <Button
          type="button"
          disabled={isCartEmpty}
          onClick={onOpenTender}
          className="w-full min-h-12 h-auto py-3 text-sm font-semibold rounded-[14px] shadow-sm flex items-center justify-center gap-2"
        >
          <span>{t("pos.cart.checkout")}</span>
          <ArrowRight className="w-4 h-4 rtl:rotate-180" aria-hidden="true" />
        </Button>
      </div>
    </div>
  );
}
