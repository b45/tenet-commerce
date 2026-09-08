"use client";

import * as React from "react";
import { Plus, CheckCircle2, AlertTriangle } from "lucide-react";
import { formatIDR } from "@/lib/money";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { useTranslation } from "@/lib/i18n";
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
  const { t } = useTranslation();
  const isOutOfStock = product.stock_quantity <= 0;
  const isLowStock = product.stock_quantity > 0 && product.stock_quantity <= 5;

  const handleClick = () => {
    if (isOutOfStock || disabled) return;
    onAddToCart(product);
  };

  return (
    <article
      className={cn(
        "group relative flex flex-col justify-between min-w-0 p-4 sm:p-5 text-start motion-safe:transition-shadow",
        "bg-[var(--color-surface-base)] rounded-[18px] border border-[var(--color-border-hairline)]",
        "shadow-sm hover:shadow-[var(--shadow-card)] hover:border-[var(--color-border-subtle)]",

        isOutOfStock && "opacity-60 cursor-not-allowed bg-[var(--color-surface-muted)] hover:scale-100 active:scale-100 hover:shadow-none"
      )}
    >
      <div>
        {/* Top Badges: Category & Halal Certification */}
        <div className="flex flex-wrap items-center gap-2 mb-3">
          <span className="break-all text-sm font-medium text-[var(--color-text-secondary)]">
            {product.category_name || t("pos.catalog.generalCategory")}
          </span>

          {product.is_halal_certified && (
            <Badge
              variant="success"
              className="text-xs px-2 py-1 gap-1 font-semibold border border-[var(--color-status-success-border)]"
            >
              <CheckCircle2 className="w-2.5 h-2.5" />
              {t("pos.catalog.halalCertified")}
            </Badge>
          )}
        </div>

        {/* Product Name & SKU */}
        <h4 className="break-words [overflow-wrap:anywhere] text-base font-semibold text-[var(--color-text-primary)] leading-snug">
          {product.name}
        </h4>

        <p className="break-all text-sm font-mono text-[var(--color-text-secondary)] mt-2">
          <bdi>{product.sku}</bdi>
        </p>
      </div>

      {/* Bottom Area: Price & Stock Status */}
      <div className="mt-4 pt-3 border-t border-[var(--color-border-hairline)] flex flex-col gap-3">
        <div>
          <span className="break-all text-base font-bold font-mono text-[var(--color-text-primary)] tracking-tight">
            {formatIDR(product.unit_price)}
          </span>

          <div className="flex items-center gap-1.5 mt-0.5">
            {isOutOfStock ? (
              <span className="text-sm font-medium text-[var(--color-status-danger-text)]">
                {t("pos.catalog.stockEmpty")}
              </span>
            ) : isLowStock ? (
              <span className="inline-flex items-center gap-1 text-sm font-medium text-[var(--color-status-warning-text)]">
                <AlertTriangle className="w-3 h-3" />
                {t("pos.catalog.stockLow")} {product.stock_quantity}
              </span>
            ) : (
              <span className="text-sm text-[var(--color-text-muted)]">
                {t("pos.catalog.stockAvailable")} {product.stock_quantity}
              </span>
            )}
          </div>
        </div>

        {/* Quick Add Button */}
        <button
          type="button"
          onClick={handleClick}
          disabled={isOutOfStock || disabled}
          aria-label={t("pos.catalog.addProduct", { name: product.name })}
          className={cn(
            "min-h-12 w-full gap-2 rounded-xl flex items-center justify-center text-sm font-semibold focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-action-primary)]",
            "bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)] border border-[var(--color-border-hairline)]",
            "group-hover:bg-[var(--color-action-primary)] group-hover:text-white group-hover:border-transparent",
            "disabled:cursor-not-allowed disabled:opacity-60",

          )}
        >
          <Plus className="w-4 h-4" aria-hidden="true" />
          <span>{t("pos.catalog.addToCart")}</span>
        </button>
      </div>
    </article>
  );
}
