"use client";

import * as React from "react";
import { Search, Barcode, RotateCw, PackageX } from "lucide-react";
import { cn } from "@/lib/utils";
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
    <div className="flex flex-col h-full space-y-4">
      {/* Top Controls: Search Bar & Barcode Scanner Indicator */}
      <div className="flex items-center gap-2.5">
        <div className="relative flex-1 flex items-center">
          <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-muted)] pointer-events-none" />
          <input
            ref={searchInputRef}
            type="text"
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Cari nama, SKU, atau scan barcode..."
            className={cn(
              "w-full h-11 pl-10 pr-10 text-xs sm:text-sm bg-[var(--color-surface-base)] text-[var(--color-text-primary)]",
              "rounded-[14px] border border-[var(--color-border-subtle)] shadow-xs",
              "placeholder:text-[var(--color-text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--color-action-focus-ring)]",
              "transition-all leading-normal"
            )}
          />
          <div className="absolute right-3.5 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)] pointer-events-none flex items-center" title="Barcode Scanner Aktif">
            <Barcode className="w-4 h-4" />
          </div>
        </div>

        <button
          type="button"
          onClick={onRetry}
          title="Segarkan Katalog"
          className="h-11 px-3.5 flex items-center justify-center gap-1.5 rounded-[14px] bg-[var(--color-surface-base)] border border-[var(--color-border-subtle)] text-xs font-semibold text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-surface-muted)] shadow-xs active:scale-95 transition-all shrink-0"
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
            "h-9 px-4 rounded-full text-xs font-medium whitespace-nowrap shrink-0 transition-all duration-150 inline-flex items-center justify-center",
            selectedCategory === "ALL"
              ? "bg-[var(--color-action-primary)] text-white shadow-xs"
              : "bg-[var(--color-surface-base)] text-[var(--color-text-secondary)] border border-[var(--color-border-hairline)] hover:bg-[var(--color-surface-muted)]"
          )}
        >
          Semua Kategori
        </button>

        {categories.map((cat) => (
          <button
            key={cat.id}
            type="button"
            onClick={() => onSelectCategory(cat.name)}
            className={cn(
              "h-9 px-3.5 rounded-full text-xs font-medium whitespace-nowrap shrink-0 transition-all duration-150 inline-flex items-center gap-2",
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
      <div className="flex-1 overflow-y-auto pr-1">
        {isLoading ? (
          <div className="grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-3.5 sm:gap-4">
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
            <h4 className="text-sm font-semibold text-[var(--color-text-primary)]">Gagal Memuat Produk</h4>
            <p className="text-xs text-[var(--color-text-secondary)] mt-1 max-w-sm">{error}</p>
            <button
              type="button"
              onClick={onRetry}
              className="mt-4 px-4 py-2 rounded-xl text-xs font-medium bg-[var(--color-action-primary)] text-white hover:bg-[var(--color-action-primary-hover)] transition-colors"
            >
              Coba Lagi
            </button>
          </div>
        ) : products.length === 0 ? (
          <div className="h-64 flex flex-col items-center justify-center text-center p-6 bg-[var(--color-surface-base)] rounded-[20px] border border-[var(--color-border-hairline)]">
            <Search className="w-10 h-10 text-[var(--color-text-muted)] mb-2" />
            <h4 className="text-sm font-semibold text-[var(--color-text-primary)]">Produk Tidak Ditemukan</h4>
            <p className="text-xs text-[var(--color-text-secondary)] mt-1">
              Tidak ada produk yang cocok dengan pencarian &ldquo;{searchQuery}&rdquo;.
            </p>
            {searchQuery && (
              <button
                type="button"
                onClick={() => onSearchChange("")}
                className="mt-3 text-xs text-[var(--color-action-primary)] font-medium hover:underline"
              >
                Hapus filter pencarian
              </button>
            )}
          </div>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-3.5 sm:gap-4 pb-6">
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
