"use client";

import * as React from "react";
import { Search, Barcode, RotateCw, PackageX } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTranslation } from "@/lib/i18n";
import { ProductCard } from "./product-card";
import type { Product, Category } from "../types";

export interface CatalogGridProps {
  products: Product[];
  categories: Category[];
  selectedCategory: string;
  onSelectCategory: (catName: string) => void;
  searchQuery: string;
  onSearchChange: (query: string) => void;
  onAddToCart: (product: Product) => void;
  isLoading: boolean;
  error: string | null;
  onRetry: () => void;
}

export function CatalogGrid({
  products,
  categories,
  selectedCategory,
  onSelectCategory,
  searchQuery,
  onSearchChange,
  onAddToCart,
  isLoading,
  error,
  onRetry,
}: CatalogGridProps) {
  const { t } = useTranslation();
  const searchInputRef = React.useRef<HTMLInputElement>(null);

  // Handle barcode scanner input: exact match auto-add on Enter
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      const raw = searchQuery.trim();
      if (!raw) return;

      // Check exact barcode match first
      const barcodeMatch = products.find(
        (p) => p.barcode === raw && p.stock_quantity > 0
      );
      if (barcodeMatch) {
        onAddToCart(barcodeMatch);
        onSearchChange("");
        return;
      }

      // If only one product in search result and stock > 0, auto-add
      if (products.length === 1 && products[0].stock_quantity > 0) {
        onAddToCart(products[0]);
        onSearchChange("");
      }
    }
  };

  return (
    <div className="flex min-w-0 flex-col space-y-4">
      {/* Top Controls: Search Bar & Barcode Scanner Indicator */}
      <div className="flex items-center gap-2.5">
        <div className="relative flex-1 flex items-center">
          <Search className="absolute start-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-muted)] pointer-events-none" />
          <input
            ref={searchInputRef}
            type="text"
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            onKeyDown={handleKeyDown}
            aria-label={t("pos.catalog.searchPlaceholder")}
            placeholder={t("pos.catalog.searchPlaceholder")}
            className={cn(
              "w-full h-12 ps-10 pe-10 text-base bg-[var(--color-surface-base)] text-[var(--color-text-primary)]",
              "rounded-[14px] border border-[var(--color-border-subtle)] shadow-xs",
              "placeholder:text-[var(--color-text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--color-action-focus-ring)]",
              "transition-all leading-normal"
            )}
          />
          <div className="absolute end-3.5 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)] pointer-events-none flex items-center" title={t("pos.catalog.barcode")}>
            <Barcode className="w-4 h-4" />
          </div>
        </div>

        <button
          type="button"
          onClick={onRetry}
          aria-label={t("common.actions.refresh")}
          title={t("common.actions.refresh")}
          className="h-12 min-w-12 px-3 flex items-center justify-center gap-1.5 rounded-[14px] bg-[var(--color-surface-base)] border border-[var(--color-border-subtle)] text-xs font-semibold text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-surface-muted)] shadow-xs active:scale-95 transition-all shrink-0"
        >
          <RotateCw className={cn("w-4 h-4", isLoading && "animate-spin text-[var(--color-action-primary)]")} />
        </button>
      </div>

      {/* Category Pills (Horizontal Scrollable) */}
      <div className="flex items-center gap-2 overflow-x-auto py-1.5 scrollbar-none select-none">
        <button
          type="button"
          onClick={() => onSelectCategory("ALL")}
          className={cn(
            "h-12 px-4 rounded-full text-sm font-medium whitespace-nowrap shrink-0 transition-all duration-150 inline-flex items-center justify-center",
            selectedCategory === "ALL"
              ? "bg-[var(--color-action-primary)] text-white shadow-xs"
              : "bg-[var(--color-surface-base)] text-[var(--color-text-secondary)] border border-[var(--color-border-hairline)] hover:bg-[var(--color-surface-muted)]"
          )}
        >
          {t("pos.catalog.allCategories")}
        </button>

        {categories.map((cat) => (
          <button
            key={cat.id}
            type="button"
            onClick={() => onSelectCategory(cat.name)}
            className={cn(
              "h-12 px-3.5 rounded-full text-sm font-medium whitespace-nowrap shrink-0 transition-all duration-150 inline-flex items-center gap-2",
              selectedCategory === cat.name
                ? "bg-[var(--color-action-primary)] text-white shadow-xs"
                : "bg-[var(--color-surface-base)] text-[var(--color-text-secondary)] border border-[var(--color-border-hairline)] hover:bg-[var(--color-surface-muted)]"
            )}
          >
            <span>{cat.name}</span>
            {cat.product_count !== undefined && (
              <span
                className={cn(
                  "text-[10px] px-1.5 py-0.5 rounded-full font-mono font-semibold leading-none",
                  selectedCategory === cat.name
                    ? "bg-white/25 text-white"
                    : "bg-[var(--color-surface-muted)] text-[var(--color-text-muted)]"
                )}
              >
                {cat.product_count}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Product Grid Area */}
      <div className="min-w-0">
        {isLoading ? (
          <div className="grid grid-cols-[repeat(auto-fill,minmax(min(100%,14rem),1fr))] gap-3.5 sm:gap-4">
            {Array.from({ length: 8 }).map((_, i) => (
              <div
                key={i}
                className="h-44 rounded-[18px] bg-[var(--color-surface-base)] border border-[var(--color-border-hairline)] p-4 animate-pulse flex flex-col justify-between"
              >
                <div className="space-y-2">
                  <div className="w-16 h-3 bg-gray-200 rounded-full" />
                  <div className="w-3/4 h-4 bg-gray-200 rounded" />
                  <div className="w-1/2 h-3 bg-gray-200 rounded" />
                </div>
                <div className="flex justify-between items-end pt-3 border-t border-[var(--color-border-hairline)]">
                  <div className="w-20 h-4 bg-gray-200 rounded" />
                  <div className="w-8 h-8 rounded-full bg-gray-200" />
                </div>
              </div>
            ))}
          </div>
        ) : error ? (
          <div className="h-64 flex flex-col items-center justify-center text-center p-6 bg-[var(--color-surface-base)] rounded-[20px] border border-[var(--color-border-hairline)]">
            <PackageX className="w-10 h-10 text-[var(--color-status-danger-text)] mb-2.5" />
            <h4 className="text-sm font-semibold text-[var(--color-text-primary)]">{t("common.status.failed")}</h4>
            <p className="text-xs text-[var(--color-text-secondary)] mt-1 max-w-sm">{error}</p>
            <button
              type="button"
              onClick={onRetry}
              className="mt-4 px-4 py-2 rounded-xl text-xs font-medium bg-[var(--color-action-primary)] text-white hover:bg-[var(--color-action-primary-hover)] transition-colors"
            >
              {t("common.actions.retry")}
            </button>
          </div>
        ) : products.length === 0 ? (
          <div className="h-64 flex flex-col items-center justify-center text-center p-6 bg-[var(--color-surface-base)] rounded-[20px] border border-[var(--color-border-hairline)]">
            <Search className="w-10 h-10 text-[var(--color-text-muted)] mb-2" />
            <h4 className="text-sm font-semibold text-[var(--color-text-primary)]">{t("pos.catalog.noMatch")}</h4>
            <p className="text-xs text-[var(--color-text-secondary)] mt-1">
              {searchQuery ? `"${searchQuery}"` : t("pos.catalog.emptyCatalog")}
            </p>
            {searchQuery && (
              <button
                type="button"
                onClick={() => onSearchChange("")}
                className="mt-3 text-xs text-[var(--color-action-primary)] font-medium hover:underline"
              >
                {t("common.actions.cancel")}
              </button>
            )}
          </div>
        ) : (
          <div className="grid grid-cols-[repeat(auto-fill,minmax(min(100%,14rem),1fr))] gap-3.5 sm:gap-4 pb-6">
            {products.map((product) => (
              <ProductCard
                key={product.id}
                product={product}
                onAddToCart={onAddToCart}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
