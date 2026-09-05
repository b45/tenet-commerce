"use client";

import * as React from "react";
import { Plus, CheckCircle2, AlertTriangle } from "lucide-react";
import { formatIDR } from "@/lib/money";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { Product } from "../types";

export interface ProductCardProps {
  product: Product;
  onAddToCart: (product: Product) => void;
  disabled?: boolean;
}

export function ProductCard({
  product,
  onAddToCart,
  disabled = false,
}: ProductCardProps) {
  const isOutOfStock = product.stock_quantity <= 0;
  const isLowStock = product.stock_quantity > 0 && product.stock_quantity <= 5;

  const handleClick = () => {
    if (isOutOfStock || disabled) return;
    onAddToCart(product);
  };

  return (
    <div
      onClick={handleClick}
      role="button"
      tabIndex={isOutOfStock ? -1 : 0}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          handleClick();
        }
      }}
      aria-disabled={isOutOfStock}
      className={cn(
        "group relative flex flex-col justify-between p-4 sm:p-5 text-left select-none transition-all duration-200",
        "bg-[var(--color-surface-base)] rounded-[18px] border border-[var(--color-border-hairline)]",
        "shadow-sm hover:shadow-[var(--shadow-card)] hover:border-[var(--color-border-subtle)]",
        "active:scale-[0.985] cursor-pointer focus:outline-none focus:ring-2 focus:ring-[var(--color-action-focus-ring)]",
        isOutOfStock && "opacity-60 cursor-not-allowed bg-[var(--color-surface-muted)] hover:scale-100 active:scale-100 hover:shadow-none"
      )}
    >
      <div>
        {/* Top Badges: Category & Halal Certification */}
        <div className="flex items-center justify-between gap-1.5 mb-2.5">
          <span className="text-[11px] font-medium text-[var(--color-text-muted)] tracking-wider uppercase truncate max-w-[140px]">
            {product.category_name || "Umum"}
          </span>

          {product.is_halal_certified && (
            <Badge
              variant="success"
              className="text-[10px] px-1.5 py-0.5 gap-1 font-semibold border border-[var(--color-status-success-border)]"
            >
              <CheckCircle2 className="w-2.5 h-2.5" />
              HALAL
            </Badge>
          )}
        </div>

        {/* Product Name & SKU */}
        <h4 className="text-[14px] font-semibold text-[var(--color-text-primary)] line-clamp-2 leading-snug group-hover:text-[var(--color-action-primary)] transition-colors">
          {product.name}
        </h4>

        <p className="text-[11px] font-mono text-[var(--color-text-muted)] mt-1 tracking-tight">
          {product.sku}
        </p>
      </div>

      {/* Bottom Area: Price & Stock Status */}
      <div className="mt-4 pt-3 border-t border-[var(--color-border-hairline)] flex items-end justify-between">
        <div>
          <span className="text-[15px] font-bold font-mono text-[var(--color-text-primary)] tracking-tight">
            {formatIDR(product.unit_price)}
          </span>

          <div className="flex items-center gap-1.5 mt-0.5">
            {isOutOfStock ? (
              <span className="text-[11px] font-medium text-[var(--color-status-danger-text)]">
                Stok Habis
              </span>
            ) : isLowStock ? (
              <span className="inline-flex items-center gap-1 text-[11px] font-medium text-[var(--color-status-warning-text)]">
                <AlertTriangle className="w-3 h-3" />
                Sisa {product.stock_quantity}
              </span>
            ) : (
              <span className="text-[11px] text-[var(--color-text-muted)]">
                Stok {product.stock_quantity}
              </span>
            )}
          </div>
        </div>

        {/* Quick Add Button */}
        <button
          type="button"
          disabled={isOutOfStock || disabled}
          aria-label={`Tambah ${product.name} ke keranjang`}
          className={cn(
            "w-8 h-8 rounded-full flex items-center justify-center transition-all duration-150",
            "bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)] border border-[var(--color-border-hairline)]",
            "group-hover:bg-[var(--color-action-primary)] group-hover:text-white group-hover:border-transparent",
            "active:scale-90",
            isOutOfStock && "hidden"
          )}
        >
          <Plus className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}
